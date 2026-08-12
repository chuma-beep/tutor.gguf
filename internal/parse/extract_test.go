package parse

import "testing"

func TestExtract(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"boxed simple", "The answer is \\boxed{49}.", "49"},
		{"boxed nested", "Therefore \\boxed{\\frac{65}{9}}. Done.", "\\frac{65}{9}"},
		{"boxed at end", "x = \\boxed{2}", "2"},
		{"boxed takes first", "\\boxed{5} wait \\boxed{7}", "5"},
		{"final answer is", "reasoning... final answer is 17", "17"},
		{"answer is verb", "answer is 3.5", "3.5"},
		{"answer colon", "answer: 3.5", "3.5"},
		{"gsm8k marker", "some work\n#### 4", "4"},
		{"bare content fallback", "just text with no answer", "just text with no answer"},
		{"empty", "", ""},
		{"whitespace only", "   \n \t ", ""},
	}
	for _, c := range cases {
		if got := Extract(c.in); got != c.want {
			t.Errorf("%s: Extract(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}
