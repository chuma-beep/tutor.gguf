package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	chromem "github.com/philippgille/chromem-go"
)

// Embedder wraps a llama.cpp server running in embedding mode
// (llama-server --embedding) and produces vectors chromem-go can store
// and query against.
//
// Built specifically around nomic-embed-text, which is asymmetric:
// it was trained with different prefixes for the text being indexed
// vs. the text being searched for, and skipping them measurably hurts
// retrieval quality. See EmbedDocument / EmbedQuery below.
type Embedder struct {
	baseURL    string
	httpClient *http.Client
	normalize  bool
}

// TaskType controls which nomic-embed-text prefix gets prepended.
type TaskType string

const (
	TaskDocument TaskType = "search_document: "
	TaskQuery    TaskType = "search_query: "
)

type EmbedderOption func(*Embedder)

func WithHTTPClient(c *http.Client) EmbedderOption {
	return func(e *Embedder) { e.httpClient = c }
}

func WithNormalize(normalize bool) EmbedderOption {
	return func(e *Embedder) { e.normalize = normalize }
}

// NewEmbedder points at a running llama.cpp server, e.g.
// "http://localhost:8080", started separately from the embedding
// model — keep this distinct from the Qwen2.5-Math generation model,
// since they serve different purposes and likely run as separate
// processes/ports.
func NewEmbedder(baseURL string, opts ...EmbedderOption) *Embedder {
	e := &Embedder{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		normalize: true,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// llamaEmbeddingRequest matches llama.cpp server's /embedding request body.
type llamaEmbeddingRequest struct {
	Content string `json:"content"`
}

// llamaEmbeddingResult matches one element of llama.cpp server's
// /embedding response. The response is a top-level JSON array (one
// element per input, though we only ever send one), and Embedding is
// itself doubly-nested — an array of rows rather than a flat vector.
// In practice with pooling enabled server-side there's exactly one row,
// so we take Embedding[0] as the actual vector.
type llamaEmbeddingResult struct {
	Index     int         `json:"index"`
	Embedding [][]float32 `json:"embedding"`
}

// EmbedDocument embeds a corpus chunk for indexing, using nomic's
// "search_document" prefix. Register this as the collection's
// EmbeddingFunc so ingestion (AddDocument) prefixes correctly:
//
//	collection, err := db.CreateCollection("tutor-corpus", nil, embedder.EmbedDocument)
func (e *Embedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text, TaskDocument)
}

// EmbedQuery embeds a student's question for retrieval, using nomic's
// "search_query" prefix. chromem-go's Collection.Query(text) always
// reuses the collection's EmbeddingFunc (EmbedDocument above), which
// would apply the wrong prefix — so don't call Query(ctx, queryText,...)
// directly. Instead call EmbedQuery yourself, then pass the resulting
// vector into Collection.QueryEmbedding(ctx, vector, ...).
func (e *Embedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text, TaskQuery)
}

func (e *Embedder) embed(ctx context.Context, text string, task TaskType) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("embed: empty input text")
	}
	prefixed := string(task) + text

	reqBody, err := json.Marshal(llamaEmbeddingRequest{Content: prefixed})
	if err != nil {
		return nil, fmt.Errorf("embed: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embedding", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("embed: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: request to llama.cpp server failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed: server returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed []llamaEmbeddingResult
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("embed: decode response: %w", err)
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("embed: server returned empty result array")
	}
	if len(parsed[0].Embedding) == 0 {
		return nil, fmt.Errorf("embed: server returned empty embedding rows")
	}

	vector := parsed[0].Embedding[0]
	if len(vector) == 0 {
		return nil, fmt.Errorf("embed: server returned empty vector")
	}

	if e.normalize {
		normalizeInPlace(vector)
	}

	return vector, nil
}

// EmbedBatch embeds multiple corpus chunks sequentially for indexing
// (always document-prefixed — batching is a Phase 2 ingestion concern,
// never something you'd do for a live query). llama.cpp's embedding
// endpoint doesn't reliably batch across all server builds, so this
// stays simple rather than assuming a /embeddings (plural) route exists.
// Swap to concurrent calls with a worker pool if indexing throughput
// becomes the bottleneck.
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for i, t := range texts {
		vec, err := e.EmbedDocument(ctx, t)
		if err != nil {
			return nil, fmt.Errorf("embed batch: item %d: %w", i, err)
		}
		out = append(out, vec)
	}
	return out, nil
}

func normalizeInPlace(v []float32) {
	var sumSquares float64
	for _, x := range v {
		sumSquares += float64(x) * float64(x)
	}
	norm := math.Sqrt(sumSquares)
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
}

// compile-time check that EmbedDocument satisfies chromem-go's
// expected EmbeddingFunc signature — this is the func you register
// with the collection at creation time.
var _ chromem.EmbeddingFunc = (*Embedder)(nil).EmbedDocument
