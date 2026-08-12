// Package prompt builds the ChatML prompt fed to Qwen2.5-Math: a coarse
// subdomain-based instruction, retrieved reference chunks, and the student
// question with an answer anchor. It is the sole place prompt formatting and
// instruction text live; rag.BuildPrompt is a thin adapter over it.
package prompt

import (
	"fmt"
	"strings"
)

// Source is a retrievable corpus chunk. Only the fields needed for prompt
// assembly are carried here so the package stays independent of rag.
type Source struct {
	Text      string
	Subdomain string
}

// Builder constructs the final prompt sent to llama-server.
type Builder struct {
	instructions map[string]string
}

// NewBuilder returns a Builder pre-loaded with the coarse-category CoT
// instructions. The default ("other") instruction is applied when a category
// has no dedicated text.
func NewBuilder() *Builder {
	return &Builder{
		instructions: subdomainInstructions,
	}
}

// Build assembles the ChatML-formatted prompt: domain instruction + retrieved
// examples + student question + answer anchor.
func (b *Builder) Build(query string, chunks []Source, subdomain string) string {
	var sb strings.Builder
	sb.WriteString("<|im_start|>system\n")

	instr, ok := b.instructions[PromptCategory(subdomain)]
	if !ok {
		instr = defaultInstruction
	}
	sb.WriteString(instr)
	sb.WriteString("<|im_end|>\n")

	sb.WriteString("<|im_start|>user\n")

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

// subdomainToPromptCategory maps the classifier's fine-grained subdomains
// down to the coarse keys that subdomainInstructions has dedicated text for.
// Retrieval filtering keeps using the fine-grained values in rag; only
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

// subdomainInstructions holds the dedicated reasoning instruction per coarse
// category. Edit these when redesigning the CoT instructions — remember to
// update the golden fixture in builder_test.go too.
var subdomainInstructions = map[string]string{
	"discrete_math":  "Please reason step by step, stating each inference rule or proof technique used, and put your final answer within \\boxed{}.",
	"calculus":       "Please reason step by step, stating the rule applied at each differentiation, integration, or algebraic step, and put your final answer within \\boxed{}.",
	"linear_algebra": "Please reason step by step, showing matrix operations row by row, and put your final answer within \\boxed{}.",
	"geometry":       "Please reason step by step, citing the relevant geometric theorem or property (Pythagorean theorem, triangle inequality, circle properties, etc.), and put your final answer within \\boxed{}.",
}

// defaultInstruction is used for categories without dedicated text.
const defaultInstruction = "Please reason step by step, and put your final answer within \\boxed{}."
