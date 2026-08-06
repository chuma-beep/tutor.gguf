package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"

	chromem "github.com/philippgille/chromem-go"
)

// ScoredChunk pairs a corpus Chunk (defined in chunker.go) with the
// retrieval-time metadata chromem-go returns — an ID and a similarity
// score. These don't belong on Chunk itself since they only exist once
// a chunk has been embedded and queried against, not while it's just
// sitting in the corpus.
type ScoredChunk struct {
	Chunk
	ID         string
	Similarity float64
}

// Retriever wraps a chromem-go collection plus a lightweight domain
// classifier used to narrow retrieval when the query is identifiable.
type Retriever struct {
	collection *chromem.Collection
	classifier *SubdomainClassifier
	embedder   *Embedder
	topK       int
}

func NewRetriever(collection *chromem.Collection, classifier *SubdomainClassifier, embedder *Embedder) *Retriever {
	return &Retriever{
		collection: collection,
		classifier: classifier,
		embedder:   embedder,
		topK:       3,
	}
}

// Retrieve returns the top-K chunks for a query. If the query's subdomain
// can be confidently identified, results are filtered to that subdomain
// first; if filtering leaves too few results, it falls back to the
// unfiltered pool so we never starve the prompt of context.

// Embed with the query prefix ourselves — the collection was created
// with EmbedDocument as its EmbeddingFunc (for ingestion), so calling
// collection.Query(ctx, query, ...) would silently re-embed this text
// with the wrong nomic prefix. QueryEmbedding bypasses that and takes
// our correctly-prefixed vector directly

func (r *Retriever) Retrieve(ctx context.Context, query string) ([]ScoredChunk, string, error) {
	subdomain, confident := r.classifier.Identify(query)

	queryVec, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, "", fmt.Errorf("embed query: %w", err)
	}

	results, err := r.collection.QueryEmbedding(ctx, queryVec, r.topK*4, nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("chromem query failed: %w", err)
	}

	chunks := toScoredChunks(results)

	if confident {
		filtered := filterBySubdomain(chunks, subdomain)
		if len(filtered) >= r.topK {
			return topN(filtered, r.topK), subdomain, nil
		}
		return topN(append(filtered, chunks...), r.topK), subdomain, nil
	}

	return topN(chunks, r.topK), subdomain, nil
}

func toScoredChunks(results []chromem.Result) []ScoredChunk {
	chunks := make([]ScoredChunk, 0, len(results))
	for _, res := range results {
		chunks = append(chunks, ScoredChunk{
			Chunk: Chunk{
				Text:      res.Content,
				Subdomain: res.Metadata["subdomain"],
				Source:    res.Metadata["source"],
				Level:     res.Metadata["level"],
			},
			ID:         res.ID,
			Similarity: float64(res.Similarity),
		})
	}
	return chunks
}

func filterBySubdomain(chunks []ScoredChunk, subdomain string) []ScoredChunk {
	out := make([]ScoredChunk, 0, len(chunks))
	for _, c := range chunks {
		if c.Subdomain == subdomain {
			out = append(out, c)
		}
	}
	return out
}

// topN dedupes by ID (in case filter+fallback overlapped), sorts by
// similarity descending, and truncates to n.
func topN(chunks []ScoredChunk, n int) []ScoredChunk {
	seen := make(map[string]bool, len(chunks))
	deduped := make([]ScoredChunk, 0, len(chunks))
	for _, c := range chunks {
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		deduped = append(deduped, c)
	}

	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Similarity > deduped[j].Similarity
	})

	if len(deduped) > n {
		deduped = deduped[:n]
	}
	return deduped
}

// SubdomainClassifier does cheap keyword-based domain identification so
// we don't need a second model call just to route retrieval. Swap this
// for an embedding-similarity classifier later if precision matters more
// than latency.
//
// Keys here must exactly match the Subdomain values your chunker
// assigns (see chunker.go: mapTypeToSubdomain and the loader functions) —
// "algebra", "arithmetic", "precalculus", "geometry", "probability",
// "number_theory", "general_math", plus the hardcoded "calculus" (GSM8K)
// and "discrete_math" (Rosen) tags.
type SubdomainClassifier struct {
	keywords map[string][]string // subdomain -> trigger terms
	minHits  int                 // hits required before we trust the call
}

func NewSubdomainClassifier() *SubdomainClassifier {
	return &SubdomainClassifier{
		keywords: map[string][]string{
			"algebra": {
				"equation", "polynomial", "quadratic", "factor",
				"root", "variable", "expression",
			},

			"arithmetic": {
				"sum", "difference", "product", "quotient",
				"percent", "ratio", "fraction", "decimal",
			},

			"precalculus": {
				"function", "domain", "range",
				"trigonometric", "sine", "cosine", "tangent",
				"logarithm", "exponential",
			},

			"geometry": {
				"triangle", "circle", "angle", "radius",
				"diameter", "perimeter", "area", "volume",
				"polygon", "coordinate",
			},

			"probability": {
				"probability", "random", "expected value",
				"coin", "dice", "cards", "combinations",
				"permutations",
			},

			"number_theory": {
				"prime", "divisor", "multiple", "gcd",
				"lcm", "mod", "modulo", "congruence",
				"integer",
			},
		},
		minHits: 1,
	}
}

// Identify returns the best-matching subdomain and whether confidence
// clears the threshold. Ties go to the first subdomain with the most
// hits in map iteration order — fine for now since keyword sets barely
// overlap; revisit if that changes.
func (c *SubdomainClassifier) Identify(query string) (string, bool) {
	q := strings.ToLower(query)

	best, bestHits := "", 0
	for subdomain, terms := range c.keywords {
		hits := 0
		for _, term := range terms {
			if strings.Contains(q, term) {
				hits++
			}
		}
		if hits > bestHits {
			best, bestHits = subdomain, hits
		}
	}

	return best, bestHits >= c.minHits
}

// subdomainToPromptCategory maps the classifier's fine-grained subdomains
// down to the coarse keys that subdomainInstructions has dedicated text
// for. Retrieval filtering keeps using the fine-grained values; only
// prompt-instruction selection goes through this bridge. Geometry maps to
// itself because it gets its own instruction text.
var subdomainToPromptCategory = map[string]string{
	"algebra":       "calculus",
	"precalculus":   "calculus",
	"arithmetic":    "calculus",
	"geometry":      "geometry",
	"probability":   "discrete_math",
	"number_theory": "discrete_math",
}

// PromptCategory converts a classifier subdomain into the coarse category
// used for prompt-instruction selection, or "other" when unmapped.
func PromptCategory(subdomain string) string {
	if cat, ok := subdomainToPromptCategory[subdomain]; ok {
		return cat
	}
	return "other"
}

// BuildPrompt assembles retrieved chunks into the context block fed to
// Qwen2.5-Math. Kept separate from Retrieve so you can swap prompt
// formatting without touching retrieval logic.

var subdomainInstructions = map[string]string{
	"discrete_math":  "Please reason step by step, stating each inference rule or proof technique used, and put your final answer within \\boxed{}.",
	"calculus":       "Please reason step by step, stating the rule applied at each differentiation, integration, or algebraic step, and put your final answer within \\boxed{}.",
	"linear_algebra": "Please reason step by step, showing matrix operations row by row, and put your final answer within \\boxed{}.",
	"geometry":       "Please reason step by step, citing the relevant geometric theorem or property (Pythagorean theorem, triangle inequality, circle properties, etc.), and put your final answer within \\boxed{}.",
}

func BuildPrompt(query string, chunks []ScoredChunk, subdomain string) string {
	var sb strings.Builder
	sb.WriteString("<|im_start|>system\n")

	instr, ok := subdomainInstructions[PromptCategory(subdomain)]
	if !ok {
		instr = "Please reason step by step, and put your final answer within \\boxed{}."
	}
	sb.WriteString(instr)
	sb.WriteString("<|im_end|>\n")

	sb.WriteString("<|im_start|>user\n") // ← add this back

	if len(chunks) > 0 {
		sb.WriteString("Relevant reference material:\n\n")
		for i, c := range chunks {
			sb.WriteString(fmt.Sprintf("[%d] (%s)\n%s\n\n", i+1, c.Subdomain, c.Text))
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("Student question: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer step by step, referencing the material above where relevant.<|im_end|>\n")
	sb.WriteString("<|im_start|>assistant\n")

	return sb.String()
}
