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
	sb.WriteString("\n\nAnswer step by step, referencing the material above where relevant. End your response with the final answer alone inside \\boxed{...} on the last line.<|im_end|>\n")
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
	"discrete_math":  "Reason step by step, naming the proof technique or counting rule used at each step." + answerAnchor,
	"calculus":       "Reason step by step. After each algebraic manipulation, state what the equation now says before continuing." + answerAnchor,
	"linear_algebra": "Reason step by step, showing each row or vector operation explicitly before moving on." + answerAnchor,
	"geometry":       "State the formula you use and substitute the given values, including the value of π given in the problem." + answerAnchor,
	"other":          "Reason step by step." + answerAnchor,
}

// answerAnchor is the identical format mandate appended to every instruction.
// Keeping it constant matters: the eval's answer parser relies on \boxed{...}
// appearing just before the response ends, so the phrasing must not vary.
const answerAnchor = " End with the final answer alone inside \\boxed{...} on the very last line."

// defaultInstruction is used for categories without dedicated text.
const defaultInstruction = "Reason step by step." + answerAnchor
