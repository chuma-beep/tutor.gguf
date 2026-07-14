package rag

import (
	"context"
	"fmt"
	chromem "github.com/philippgille/chromem-go"
	"sort"
	"strings"
)

// Chunk represents a corpus chunk stored in chromem-go, enriched with a
// subdomain tag (e.g. "linear-algebra", "calculus", "discrete-math").
type Chunk struct {
	ID         string
	Text       string
	Subdomain  string
	Similarity float64
}

// Retriever wraps a chromem-go collection plus a lightweight domain
// classifier used to narrow retrieval when the query is identifiable.
type Retriever struct {
	collection *chromem.Collection
	classifier *SubdomainClassifier
	topK       int
}

func NewRetriever(collection *chromem.Collection, classifier *SubdomainClassifier) *Retriever {
	return &Retriever{
		collection: collection,
		classifier: classifier,
		topK:       3,
	}
}

// Retrieve returns the top-K chunks for a query. If the query's subdomain
// can be confidently identified, results are filtered to that subdomain
// first; if filtering leaves too few results, it falls back to the
// unfiltered pool so we never starve the prompt of context.
func (r *Retriever) Retrieve(ctx context.Context, query string) ([]Chunk, error) {
	subdomain, confident := r.classifier.Identify(query)

	results, err := r.collection.Query(ctx, query, r.topK*4, nil, nil) // over-fetch for filtering headroom
	if err != nil {
		return nil, fmt.Errorf("chromem query failed: %w", err)
	}

	chunks := toChunks(results)

	if confident {
		filtered := filterBySubdomain(chunks, subdomain)
		if len(filtered) >= r.topK {
			return topN(filtered, r.topK), nil
		}
		// Not enough subdomain-specific matches — merge filtered results
		// first (they're more relevant), then pad with the rest.
		return topN(append(filtered, chunks...), r.topK), nil
	}

	return topN(chunks, r.topK), nil
}

func toChunks(results []chromem.Result) []Chunk {
	chunks := make([]Chunk, 0, len(results))
	for _, res := range results {
		chunks = append(chunks, Chunk{
			ID:         res.ID,
			Text:       res.Content,
			Subdomain:  res.Metadata["subdomain"],
			Similarity: float64(res.Similarity),
		})
	}
	return chunks
}

func filterBySubdomain(chunks []Chunk, subdomain string) []Chunk {
	out := make([]Chunk, 0, len(chunks))
	for _, c := range chunks {
		if c.Subdomain == subdomain {
			out = append(out, c)
		}
	}
	return out
}

// topN dedupes by ID (in case filter+fallback overlapped), sorts by
// similarity descending, and truncates to n.
func topN(chunks []Chunk, n int) []Chunk {
	seen := make(map[string]bool, len(chunks))
	deduped := make([]Chunk, 0, len(chunks))
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
type SubdomainClassifier struct {
	keywords map[string][]string // subdomain -> trigger terms
	minHits  int                 // hits required before we trust the call
}

func NewSubdomainClassifier() *SubdomainClassifier {
	return &SubdomainClassifier{
		keywords: map[string][]string{
			"linear-algebra": {"matrix", "vector", "eigenvalue", "eigenvector", "determinant", "basis", "span", "rank"},
			"calculus":       {"derivative", "integral", "limit", "gradient", "partial derivative", "chain rule"},
			"discrete-math":  {"graph", "proof", "induction", "combinatorics", "set theory", "logic", "boolean"},
			"probability":    {"probability", "distribution", "expectation", "variance", "bayes", "random variable"},
			"statistics":     {"hypothesis", "confidence interval", "regression", "p-value", "standard deviation"},
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

// BuildPrompt assembles retrieved chunks into the context block fed to
// Qwen2.5-Math. Kept separate from Retrieve so you can swap prompt
// formatting without touching retrieval logic.
func BuildPrompt(query string, chunks []Chunk) string {
	var sb strings.Builder

	if len(chunks) > 0 {
		sb.WriteString("Relevant reference material:\n\n")
		for i, c := range chunks {
			sb.WriteString(fmt.Sprintf("[%d] (%s)\n%s\n\n", i+1, c.Subdomain, c.Text))
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("Student question: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer step by step, referencing the material above where relevant.")

	return sb.String()
}
