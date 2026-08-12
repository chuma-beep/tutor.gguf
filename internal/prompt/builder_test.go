package prompt

import (
	"strings"
	"testing"
)

func TestBuildChatMLStructure(t *testing.T) {
	got := NewBuilder().Build("query", []Source{{Text: "c1", Subdomain: "algebra"}}, "algebra")

	wantOrder := []string{
		"<|im_start|>system\n",
		"<|im_start|>user\n",
		"Student question:",
		"Answer step by step, referencing the material above where relevant. End your response with the final answer alone inside \\boxed{...} on the last line.<|im_end|>\n",
		"<|im_start|>assistant\n",
	}
	pos := -1
	for _, token := range wantOrder {
		idx := strings.Index(got, token)
		if idx < 0 {
			t.Fatalf("prompt missing %q:\n%s", token, got)
		}
		if idx <= pos {
			t.Fatalf("token %q appears before previous token (pos %d <= %d):\n%s", token, idx, pos, got)
		}
		pos = idx
	}

	if !strings.HasSuffix(got, "<|im_start|>assistant\n") {
		t.Fatalf("prompt must end with the assistant token:\n%s", got)
	}
}

func TestBuildInstructionSelection(t *testing.T) {
	cases := map[string]string{
		"number_theory": subdomainInstructions["discrete_math"],
		"probability":   subdomainInstructions["discrete_math"],
		"algebra":       subdomainInstructions["calculus"],
		"precalculus":   subdomainInstructions["calculus"],
		"arithmetic":    subdomainInstructions["calculus"],
		"geometry":      subdomainInstructions["geometry"],
	}
	for subdomain, wantInstr := range cases {
		got := NewBuilder().Build("q", nil, subdomain)
		if !strings.Contains(got, wantInstr) {
			t.Errorf("subdomain %q: prompt missing instruction %q:\n%s", subdomain, wantInstr, got)
		}
		if strings.Contains(got, defaultInstruction) {
			t.Errorf("subdomain %q: prompt should not use the default instruction:\n%s", subdomain, got)
		}
	}
}

func TestBuildDefaultInstruction(t *testing.T) {
	for _, subdomain := range []string{"totally_unknown", "general_math", ""} {
		got := NewBuilder().Build("q", nil, subdomain)
		if !strings.Contains(got, defaultInstruction) {
			t.Errorf("subdomain %q: expected the default instruction:\n%s", subdomain, got)
		}
	}
}

func TestPromptCategory(t *testing.T) {
	cases := map[string]string{
		"algebra":       "calculus",
		"precalculus":   "calculus",
		"arithmetic":    "calculus",
		"geometry":      "geometry",
		"probability":   "discrete_math",
		"number_theory": "discrete_math",
		"general_math":  "other",
		"":              "other",
	}
	for in, want := range cases {
		if got := PromptCategory(in); got != want {
			t.Errorf("PromptCategory(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildContextRendering(t *testing.T) {
	chunks := []Source{
		{Text: "Problem: What is 2+2?\nSolution: 4", Subdomain: "arithmetic"},
		{Text: "Problem: derivative\nSolution: 2x", Subdomain: "algebra"},
	}
	got := NewBuilder().Build("q", chunks, "algebra")

	for _, want := range []string{
		"Relevant reference material:\n\n",
		"[1] (arithmetic)\nProblem: What is 2+2?\nSolution: 4\n\n",
		"[2] (algebra)\nProblem: derivative\nSolution: 2x\n\n",
		"---\n\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("context block missing %q:\n%s", want, got)
		}
	}

	noChunks := NewBuilder().Build("q", nil, "algebra")
	if strings.Contains(noChunks, "Relevant reference material") || strings.Contains(noChunks, "---") {
		t.Errorf("prompt with no chunks must not contain a context block:\n%s", noChunks)
	}
}

func TestBuildQuestionAndAnchor(t *testing.T) {
	q := "Find the derivative of x^2."
	got := NewBuilder().Build(q, nil, "algebra")

	if !strings.Contains(got, "Student question: "+q+"\n\n") {
		t.Errorf("missing question anchor:\n%s", got)
	}
	if !strings.Contains(got, "Answer step by step, referencing the material above where relevant. End your response with the final answer alone inside \\boxed{...} on the last line.<|im_end|>\n") {
		t.Errorf("missing answer anchor:\n%s", got)
	}
}

// TestBuildGoldenPrompt pins the exact prompt output so the refactor from the
// old rag.BuildPrompt into prompt.Builder stays byte-identical. When you change
// instruction text (CoT redesign), regenerate this fixture deliberately.
func TestBuildGoldenPrompt(t *testing.T) {
	got := NewBuilder().Build("Find the derivative of x^2.", []Source{
		{Text: "Problem: What is 2+2?\nSolution: 4", Subdomain: "algebra"},
		{Text: "Problem: derivative\nSolution: 2x", Subdomain: "calculus"},
	}, "algebra")

	want := `<|im_start|>system
Reason step by step. After each algebraic manipulation, state what the equation now says before continuing. End with the final answer alone inside \boxed{...} on the very last line.<|im_end|>
<|im_start|>user
Relevant reference material:

[1] (algebra)
Problem: What is 2+2?
Solution: 4

[2] (calculus)
Problem: derivative
Solution: 2x

---

Student question: Find the derivative of x^2.

Answer step by step, referencing the material above where relevant. End your response with the final answer alone inside \boxed{...} on the last line.<|im_end|>
<|im_start|>assistant
`

	if got != want {
		t.Errorf("golden prompt mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
