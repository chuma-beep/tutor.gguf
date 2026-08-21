package renderer

import (
	"strings"
	"testing"
)

func TestFrac(t *testing.T) {
	cases := []struct{ in, want string }{
		{`\frac{1}{2}`, "1\n─\n2"},
		{`\frac{2002}{k(k+1)}`, " 2002 \n──────\nk(k+1)"},
		{`x + \frac{1}{2}`, "    1\nx + ─\n    2"},
	}
	for _, c := range cases {
		if got := RenderSpan(c.in, false); got != c.want {
			t.Errorf("%s:\n got:\n%s\nwant:\n%s", c.in, got, c.want)
		}
	}
}

func TestSqrt(t *testing.T) {
	if got, want := RenderSpan(`\sqrt{x+1}`, false), "√(x+1)"; got != want {
		t.Errorf("sqrt flat: got %q want %q", got, want)
	}
	if got := RenderSpan(`\sqrt{\frac{1}{2}}`, false); !strings.Contains(got, "√") || !strings.Contains(got, "─") {
		t.Errorf("sqrt of frac missing radical or bar: %q", got)
	}
	if got := RenderSpan(`\sqrt[3]{8}`, false); !strings.Contains(got, "∛") {
		t.Errorf("cube root should use ∛, got %q", got)
	}
}

func TestScripts(t *testing.T) {
	if got := RenderSpan(`x^2`, false); got != "x²" {
		t.Errorf("x^2: got %q", got)
	}
	if got := RenderSpan(`a_n`, false); got != "aₙ" {
		t.Errorf("a_n: got %q", got)
	}
	// Script operand must not swallow later atoms: \log_2 x bounds "2".
	if got := RenderSpan(`\log_2 x`, false); got != "log₂ x" {
		t.Errorf("\\log_2 x: got %q", got)
	}
	if got := RenderSpan(`x^2 + y`, false); got != "x² + y" {
		t.Errorf("x^2 + y: got %q", got)
	}
	// Both sub and sup stack.
	if got := RenderSpan(`x_i^2`, false); got != "2\nx\ni" {
		t.Errorf("x_i^2: got %q", got)
	}
}

func TestBigOps(t *testing.T) {
	got := RenderSpan(`\sum_{i=1}^{n} i`, false)
	if !strings.Contains(got, "∑") || !strings.Contains(got, "i=1") || !strings.Contains(got, "n") {
		t.Errorf("sum limits missing: %q", got)
	}
	got = RenderSpan(`\int_a^b f(x)\,dx`, false)
	if !strings.Contains(got, "∫") || !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("int limits missing: %q", got)
	}
}

func TestNestBinom(t *testing.T) {
	if got := RenderSpan(`\left(x+1\right)`, false); got != "(x+1)" {
		t.Errorf("flat nest: got %q", got)
	}
	got := RenderSpan(`\left(\frac{1}{2}\right)+1`, false)
	if !strings.Contains(got, "1") || !strings.Contains(got, "2") || !strings.Contains(got, "+1") {
		t.Errorf("tall nest: got %q", got)
	}
	// The +1 sits on the fraction bar row (baseline).
	if got := RenderSpan(`\left(\frac{1}{2}\right)+1`, false); got != "⎛1⎞  \n⎜─⎟+1\n⎝2⎠  " {
		t.Errorf("tall nest alignment: got\n%s", got)
	}
	if got := RenderSpan(`\binom{n}{k}`, false); got != "⎛n⎞\n⎝k⎠" {
		t.Errorf("binom: got %q", got)
	}
}

func TestDecorations(t *testing.T) {
	if got := RenderSpan(`\boxed{16}`, false); got != "┌────┐\n│ 16 │\n└────┘" {
		t.Errorf("boxed: got\n%s", got)
	}
	if got := RenderSpan(`\overline{AB}`, false); !strings.Contains(got, "──") || !strings.Contains(got, "AB") {
		t.Errorf("overline: got %q", got)
	}
}

func TestErrorsDegrade(t *testing.T) {
	// Unknown environments linearize; never blank.
	got := RenderSpan(`\begin{matrix}1&2\\3&4\end{matrix}`, false)
	if strings.TrimSpace(got) == "" {
		t.Errorf("matrix degrade to empty: %q", got)
	}
	// Unclosed \left degrades to its flat body.
	if got := RenderSpan(`\left(a`, false); strings.TrimSpace(got) != "a" {
		t.Errorf("unclosed nest: got %q", got)
	}
	// Pathological input must not panic.
	for _, bad := range []string{`\frac{`, `{_{^`, `\left`, `^`, ``} {
		RenderSpan(bad, false)
		RenderSpan(bad, true)
	}
}

func TestRenderWholeText(t *testing.T) {
	got := Render(`The value is \(\frac{1}{2}\) and \[\sum_{i=1}^{n} i\].`, false)
	for _, want := range []string{"The value is", "1\n─\n2", "∑", "i=1", "."} {
		if !strings.Contains(got, want) {
			t.Errorf("Render missing %q in:\n%s", want, got)
		}
	}
	// Display spans float on their own lines.
	if i := strings.Index(got, "∑"); i < 0 || !strings.Contains(got[:i], "The value is") {
		t.Errorf("prose before display span lost:\n%s", got)
	}
}

func TestASCIIMode(t *testing.T) {
	// ASCII output should contain no non-ASCII glyph box-drawing or math.
	got := RenderSpan(`\frac{1}{2}`, true)
	if got != "1\n-\n2" {
		t.Errorf("ascii frac: got\n%s", got)
	}
	got = RenderSpan(`\alpha \le \beta`, true)
	if strings.Contains(got, "α") || strings.Contains(got, "β") || strings.Contains(got, "≤") {
		t.Errorf("ascii leaks unicode: %q", got)
	}
	got = RenderSpan(`\left(\frac{1}{2}\right)`, true)
	if !strings.Contains(got, "|") {
		t.Errorf("ascii tall nest should use | bars: %q", got)
	}
}

func TestStreamingTolerance(t *testing.T) {
	// Partial deltas must not panic and should keep prior progress.
	got := Render(`to find \(\frac{2002}{`, false)
	if strings.TrimSpace(got) == "" {
		t.Errorf("partial math rendered empty: %q", got)
	}
	got2 := Render(got, false)
	if got2 == "" {
		t.Errorf("re-rendering failed: %q", got)
	}
}
