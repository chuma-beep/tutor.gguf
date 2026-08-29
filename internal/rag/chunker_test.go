package rag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRosenDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "book", "exercises", "ch1")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(dir, "README.md"):         "# Rosen solutions",
		filepath.Join(dir, "notes.txt"):         "counting basics",
		filepath.Join(nested, "ch01_1-sol.tex"): `\section{Solution} Prove that $A \cup (B \cap C) = \ldots$`,
		filepath.Join(nested, "figure.pdf"):     "%PDF should be ignored",
		filepath.Join(nested, "empty.md"):       "   ",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	chunks, err := LoadRosenDir(dir)
	if err != nil {
		t.Fatalf("LoadRosenDir: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks (md+txt+tex, skipping pdf and empty), got %d", len(chunks))
	}
	for _, c := range chunks {
		if c.Subdomain != "discrete_math" || c.Source != "rosen" {
			t.Errorf("unexpected chunk metadata: %+v", c)
		}
	}
}

func TestLoadRosenDirMissing(t *testing.T) {
	if _, err := LoadRosenDir(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoadRosenDirSplitsOversizedFiles(t *testing.T) {
	dir := t.TempDir()
	big := make([]string, 0, 200)
	for i := 0; i < 400; i++ { // ~4000+ chars of LaTeX-ish lines
		big = append(big, fmt.Sprintf("\\item exercise %d with $x_{%d}^2$ and more text to pad the line length out", i, i))
	}
	if err := os.WriteFile(filepath.Join(dir, "big.tex"), []byte(strings.Join(big, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	chunks, err := LoadRosenDir(dir)
	if err != nil {
		t.Fatalf("LoadRosenDir: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected oversized file split into multiple chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if len(c.Text) > rosenMaxChunkChars+10 {
			t.Errorf("chunk %d too long: %d chars (limit %d)", i, len(c.Text), rosenMaxChunkChars)
		}
	}
}

func TestSplitOnLines(t *testing.T) {
	got := splitOnLines("aaaa\nbb\ncc", 4)
	want := []string{"aaaa", "bb", "cc"}
	if len(got) != len(want) {
		t.Fatalf("want %q, got %q", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %q, got %q", want, got)
		}
	}
	if segs := splitOnLines("short", 10); len(segs) != 1 || segs[0] != "short" {
		t.Fatalf("short input should pass through, got %q", segs)
	}
}

// TestLoadHendrycksFileSplitsOversized proves a problem+solution that would
// exceed the embedding batch (the 2209-token case) is split into multiple
// chunks, each under hendrycksMaxChunkChars, with metadata preserved.
func TestLoadHendrycksFileSplitsOversized(t *testing.T) {
	dir := t.TempDir()
	big := make([]string, 0, 120)
	for i := 0; i < 120; i++ {
		big = append(big, fmt.Sprintf("\\item Asymptote diagram step %d with $\\frac{x_{%d}}{2}$ and plenty more LaTeX padding the solution body out to exceed the character budget", i, i))
	}
	item := HendrycksItem{
		Problem:  "Find the area of the inscribed polygon.",
		Solution: strings.Join(big, "\n"),
		Type:     "Geometry",
		Level:    "Level 5",
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "74.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	chunks, err := LoadHendrycksFile(path)
	if err != nil {
		t.Fatalf("LoadHendrycksFile: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected oversized problem+solution split into multiple chunks, got %d", len(chunks))
	}
	// Text must round-trip (splitting keeps all content, no truncation).
	joined := ""
	for _, c := range chunks {
		joined += c.Text
	}
	if !strings.Contains(joined, "Asymptote diagram step 0") || !strings.Contains(joined, "step 59") {
		t.Fatal("split chunks should retain the full text")
	}
	for i, c := range chunks {
		if len(c.Text) > hendrycksMaxChunkChars+10 {
			t.Errorf("chunk %d too long: %d chars (limit %d)", i, len(c.Text), hendrycksMaxChunkChars)
		}
		if c.Subdomain != "geometry" || c.Source != "hendrycks_math" || c.Level != "Level 5" {
			t.Errorf("chunk %d metadata lost: %+v", i, c)
		}
	}
}

// TestLoadGSM8KFileNoSplitSmall proves small lines pass through unchanged.
func TestLoadGSM8KFileNoSplitSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "train.jsonl")
	line := `{"question":"If you double a number and add 5, you get 20. What is it?","answer":"2*x+5=20 so x=7.5\n#### 7.5"}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chunks, err := LoadGSM8KFile(path)
	if err != nil {
		t.Fatalf("LoadGSM8KFile: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("small line should yield one chunk, got %d", len(chunks))
	}
	if chunks[0].Subdomain != "calculus" || chunks[0].Source != "gsm8k" {
		t.Fatalf("unexpected chunk metadata: %+v", chunks[0])
	}
}
