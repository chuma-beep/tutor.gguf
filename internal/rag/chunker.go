package rag

import (
	"bufio"
	"encoding/json"
	"fmt"
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

// LoadHendrycksFile processes a single Hendrycks JSON file into one chunk.
func LoadHendrycksFile(filePath string) (Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Chunk{}, fmt.Errorf("failed to open Hendrycks file: %w", err)
	}
	defer file.Close()
	var item HendrycksItem
	if err := json.NewDecoder(file).Decode(&item); err != nil {
		return Chunk{}, fmt.Errorf("failed to decode Hendrycks JSON: %w", err)
	}
	return Chunk{
		Text:      fmt.Sprintf("Problem: %s\nSolution: %s", item.Problem, item.Solution),
		Subdomain: mapTypeToSubdomain(item.Type),
		Source:    "hendrycks_math",
		Level:     item.Level,
	}, nil
}

// LoadGSM8KFile processes an entire GSM8K JSONL file, returning one chunk per line.
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
		chunks = append(chunks, Chunk{
			Text:      fmt.Sprintf("Problem: %s\nSolution: %s", item.Question, item.Answer),
			Subdomain: "calculus", // GSM8K consists of word problems
			Source:    "gsm8k",
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading GSM8K file: %w", err)
	}
	return chunks, nil
}

// LoadRosenDir walks a directory of Rosen solution files and returns one chunk per file.
// Each file is expected to be a self-contained problem+solution (.md or .txt).
func LoadRosenDir(dirPath string) ([]Chunk, error) {
	var chunks []Chunk
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Rosen directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".md" && ext != ".txt" {
			continue
		}
		path := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read Rosen file %s: %w", entry.Name(), err)
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			continue
		}
		chunks = append(chunks, Chunk{
			Text:      text,
			Subdomain: "discrete_math",
			Source:    "rosen",
			Level:     "undergraduate",
		})
	}
	return chunks, nil
}

// mapTypeToSubdomain standardizes raw dataset types to your internal categories.
func mapTypeToSubdomain(rawType string) string {
	switch rawType {
	case "Counting & Probability", "Number Theory", "Prealgebra":
		return "discrete_math"
	case "Algebra", "Intermediate Algebra", "Precalculus":
		return "calculus"
	case "Geometry":
		return "linear_algebra"
	default:
		return "general_math"
	}
}
