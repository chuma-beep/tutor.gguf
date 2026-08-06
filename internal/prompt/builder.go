package prompt

import (
	"strings"

	"github.com/chuma-beep/tutor.gguf/internal/rag"
)

// Builder constructs the final prompt sent to llama-server.
type Builder struct {
	Templates map[string]string
}

// NewBuilder returns a Builder pre-loaded with subdomain-specific CoT instructions.
func NewBuilder() *Builder {
	return &Builder{
		Templates: map[string]string{
			"discrete_math":  "Show your logical reasoning step by step. State each inference rule or proof technique used.",
			"calculus":       "Show each differentiation, integration, or algebraic step. State the rule applied at each step.",
			"linear_algebra": "Show matrix operations row by row. State the operation applied at each step.",
			"other":          "Solve step by step, showing your reasoning clearly.",
		},
	}
}

// Build constructs the full prompt: instruction + retrieved examples + question + answer anchor.
func (b *Builder) Build(problem string, subdomain string, chunks []rag.Chunk) string {
	var sb strings.Builder

	instruction, ok := b.Templates[subdomain]
	if !ok {
		instruction = b.Templates["other"]
	}
	sb.WriteString(instruction)
	sb.WriteString("\n\n")

	if len(chunks) > 0 {
		sb.WriteString("Here are similar solved problems for reference:\n\n")
		for _, c := range chunks {
			sb.WriteString(c.Text)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("Now solve this problem and put your final answer in \\boxed{}:\n")
	sb.WriteString(problem)

	return sb.String()
}

// DetectSubdomain does crude keyword matching to guess the problem's subdomain.
func DetectSubdomain(problem string) string {
	lower := strings.ToLower(problem)
	switch {
	case strings.Contains(lower, "prove") || strings.Contains(lower, "graph") ||
		strings.Contains(lower, "set") || strings.Contains(lower, "permutation") ||
		strings.Contains(lower, "combinatoric"):
		return "discrete_math"
	case strings.Contains(lower, "derivative") || strings.Contains(lower, "integral") ||
		strings.Contains(lower, "limit") || strings.Contains(lower, "log") ||
		strings.Contains(lower, "sec") || strings.Contains(lower, "csc"):
		return "calculus"
	case strings.Contains(lower, "matrix") || strings.Contains(lower, "vector") ||
		strings.Contains(lower, "eigen"):
		return "linear_algebra"
	default:
		return "other"
	}
}
