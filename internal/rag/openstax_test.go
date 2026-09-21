package rag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOpenStaxPDFMissing(t *testing.T) {
	if _, err := LoadOpenStaxPDF(filepath.Join(t.TempDir(), "nope.pdf"), "calculus"); err == nil {
		t.Fatal("expected error for missing PDF")
	}
}

func TestLoadOpenStaxPDFSections(t *testing.T) {
	body := "Preface to college algebra.\n" +
		"Chapter 1 Functions\n" +
		"Functions map inputs to outputs.\n" +
		"Chapter 2 Linear Equations\n" +
		"Solve 2x + 3 = 7.\n"
	path := filepath.Join(t.TempDir(), "college-algebra.pdf")
	if err := os.WriteFile(path, minimalPDF(body), 0o644); err != nil {
		t.Fatal(err)
	}
	chunks, err := LoadOpenStaxPDF(path, "algebra")
	if err != nil {
		t.Fatalf("LoadOpenStaxPDF: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected one chunk per chapter section, got %d", len(chunks))
	}
	for _, c := range chunks {
		if c.Subdomain != "algebra" || c.Source != "openstax" {
			t.Errorf("unexpected chunk metadata: %+v", c)
		}
	}
	joined := ""
	for _, c := range chunks {
		joined += c.Text
	}
	if !strings.Contains(joined, "Functions map inputs") || !strings.Contains(joined, "2x + 3 = 7") {
		t.Fatalf("split chunks should retain the full text, got %q", joined)
	}
}

func TestLoadOpenStaxPDFSplitsOversized(t *testing.T) {
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, fmt.Sprintf("worked example %d with $x_{%d}^2$ and padding text to grow the section", i, i))
	}
	path := filepath.Join(t.TempDir(), "calculus.pdf")
	if err := os.WriteFile(path, minimalPDF(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	chunks, err := LoadOpenStaxPDF(path, "calculus")
	if err != nil {
		t.Fatalf("LoadOpenStaxPDF: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected oversized section split into multiple chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if len(c.Text) > openstaxMaxChunkChars+10 {
			t.Errorf("chunk %d too long: %d chars (limit %d)", i, len(c.Text), openstaxMaxChunkChars)
		}
	}
}

func TestLoadOpenStaxDirInfersSubdomain(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"calculus-vol1.pdf":     "Chapter 1 Limits\nLimit of f(x) as x approaches 2.",
		"college-algebra.pdf":   "Chapter 1 Functions\nFunctions map inputs to outputs.",
		"notes.txt":             "should be ignored",
		"mystery-book.pdf":      "Chapter 1 Sets\nA set is a collection.",
		"empty-dir-check.pdf.x": "ignored by extension",
	}
	for name, body := range files {
		if strings.HasSuffix(name, ".pdf") {
			if err := os.WriteFile(filepath.Join(dir, name), minimalPDF(body), 0o644); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	chunks, err := LoadOpenStaxDir(dir, "general_math")
	if err != nil {
		t.Fatalf("LoadOpenStaxDir: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks (2 named + 1 fallback, skipping txt), got %d", len(chunks))
	}
	seen := map[string]bool{}
	for _, c := range chunks {
		if c.Source != "openstax" {
			t.Errorf("unexpected source: %+v", c)
		}
		seen[c.Subdomain] = true
	}
	for _, want := range []string{"calculus", "algebra", "general_math"} {
		if !seen[want] {
			t.Errorf("expected subdomain %q in %v", want, seen)
		}
	}
}

func TestLoadOpenStaxDirMissing(t *testing.T) {
	if _, err := LoadOpenStaxDir(filepath.Join(t.TempDir(), "nope"), "calculus"); err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestSplitSectionsNoHeadings(t *testing.T) {
	raw := "just some prose\nwith no chapter markers"
	got := splitSections(raw)
	if len(got) != 1 || got[0] != raw {
		t.Fatalf("expected single passthrough section, got %q", got)
	}
}

// minimalPDF builds a single-page PDF with one content line per input line,
// computing a valid xref table so ledongthuc/pdf can parse it in tests.
func minimalPDF(body string) []byte {
	var content strings.Builder
	content.WriteString("BT /F1 12 Tf 72 720 Td\n")
	lines := strings.Split(body, "\n")
	for i, ln := range lines {
		ln = strings.ReplaceAll(ln, `\`, `\\`)
		ln = strings.ReplaceAll(ln, `(`, `\(`)
		ln = strings.ReplaceAll(ln, `)`, `\)`)
		if i > 0 {
			content.WriteString("0 -20 Td\n")
		}
		fmt.Fprintf(&content, "(%s) Tj\n", ln)
	}
	content.WriteString("ET")
	stream := content.String()

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var buf strings.Builder
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects))
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xrefPos := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xrefPos)
	return []byte(buf.String())
}
