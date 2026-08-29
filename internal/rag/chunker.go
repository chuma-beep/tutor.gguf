package rag

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Chunk represents a single, self-contained mathematical unit.
type Chunk struct {
	Text      string `json:"text"`
	Subdomain string `json:"subdomain"`
	Source    string `json:"source"`
	Level     string `json:"level,omitempty"` // Only used by Hendrycks
}

// HendrycksItem maps to the expected JSON structure of a Hendrycks MATH file.
type HendrycksItem struct {
	Problem  string `json:"problem"`
	Solution string `json:"solution"`
	Type     string `json:"type"`
	Level    string `json:"level"`
}

// GSM8KItem maps to a single line in the GSM8K JSONL file.
type GSM8KItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// LoadHendrycksFile processes a single Hendrycks JSON file. It returns one
// chunk, or several when the problem+solution text exceeds
// hendrycksMaxChunkChars — dense LaTeX solutions (e.g. Asymptote geometry)
// can run > 2000 tokens, so splitting keeps every chunk safely inside the
// embedding server's physical batch.
func LoadHendrycksFile(filePath string) ([]Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Hendrycks file: %w", err)
	}
	defer file.Close()
	var item HendrycksItem
	if err := json.NewDecoder(file).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to decode Hendrycks JSON: %w", err)
	}
	text := fmt.Sprintf("Problem: %s\nSolution: %s", item.Problem, item.Solution)
	subdomain := mapTypeToSubdomain(item.Type)
	var chunks []Chunk
	for _, seg := range splitOnLines(text, hendrycksMaxChunkChars) {
		chunks = append(chunks, Chunk{
			Text:      seg,
			Subdomain: subdomain,
			Source:    "hendrycks_math",
			Level:     item.Level,
		})
	}
	return chunks, nil
}

// LoadGSM8KFile processes an entire GSM8K JSONL file, returning one chunk per
// line (or more when a line's problem+solution exceeds gsm8kMaxChunkChars).
func LoadGSM8KFile(filePath string) ([]Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GSM8K file: %w", err)
	}
	defer file.Close()
	var chunks []Chunk
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item GSM8KItem
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, fmt.Errorf("failed to decode GSM8K line: %w", err)
		}
		text := fmt.Sprintf("Problem: %s\nSolution: %s", item.Question, item.Answer)
		for _, seg := range splitOnLines(text, gsm8kMaxChunkChars) {
			chunks = append(chunks, Chunk{
				Text:      seg,
				Subdomain: "calculus", // GSM8K consists of word problems
				Source:    "gsm8k",
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading GSM8K file: %w", err)
	}
	return chunks, nil
}

// LoadRosenDir walks a directory of Rosen solution files (recursively) and
// returns one chunk per file. Accepted formats: .md, .txt, and .tex (raw LaTeX
// exercise solutions — retrieval handles the markup fine). Files longer than
// rosenMaxChunkChars are split on line boundaries so no chunk overflows the
// embedding model's context window.
func LoadRosenDir(dirPath string) ([]Chunk, error) {
	var chunks []Chunk
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path == dirPath {
			return nil
		}
		switch ext := filepath.Ext(d.Name()); ext {
		case ".md", ".txt", ".tex":
		default:
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read Rosen file %s: %w", d.Name(), err)
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			return nil
		}
		for _, seg := range splitOnLines(text, rosenMaxChunkChars) {
			chunks = append(chunks, Chunk{
				Text:      seg,
				Subdomain: "discrete_math",
				Source:    "rosen",
				Level:     "undergraduate",
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read Rosen directory: %w", err)
	}
	return chunks, nil
}

// Chunk-size budgets in characters. Rosen keeps its proven 2000-char budget
// (≈600 dense-LaTeX tokens — safely inside the 8192-token physical batch).
// Hendrycks and GSM8K use a larger budget since their texts are one
// problem+solution unit; the 8192 batch leaves ample headroom (6000 chars
// ≈ 2000 tokens + the search_document prefix).
const (
	rosenMaxChunkChars     = 2000
	hendrycksMaxChunkChars = 6000
	gsm8kMaxChunkChars     = 6000
)

// splitOnLines cuts s into segments of at most max characters, preferring to
// break after a newline so solutions don't split mid-equation.
func splitOnLines(s string, max int) []string {
	if len(s) <= max {
		return []string{s}
	}
	var segs []string
	for len(s) > max {
		cut := strings.LastIndexByte(s[:max+1], '\n')
		if cut < max/2 {
			cut = max // no sane newline nearby; hard-cut
		}
		segs = append(segs, strings.TrimSpace(s[:cut]))
		s = strings.TrimSpace(s[cut:])
	}
	if s != "" {
		segs = append(segs, s)
	}
	return segs
}

// mapTypeToSubdomain standardizes raw dataset types to your internal categories.
func mapTypeToSubdomain(rawType string) string {
	switch rawType {
	case "Algebra":
		return "algebra"

	case "Intermediate Algebra":
		return "algebra"

	case "Prealgebra":
		return "arithmetic"

	case "Precalculus":
		return "precalculus"

	case "Geometry":
		return "geometry"

	case "Counting & Probability":
		return "probability"

	case "Number Theory":
		return "number_theory"

	default:
		return "general_math"
	}
}
