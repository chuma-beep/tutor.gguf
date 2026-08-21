package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// askRequest mirrors cmd/serve's requestBody.
type askRequest struct {
	Problem     string  `json:"problem"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type askResponse struct {
	Content string `json:"content"`
	Answer  string `json:"answer,omitempty"`
}

// Ask posts a single blocking request to the tutor RAG server (/v1/complete)
// and returns the raw model content.
func Ask(tutorURL, problem string) (string, error) {
	body, err := json.Marshal(askRequest{Problem: problem})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	resp, err := http.Post(tutorURL+"/v1/complete", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request to %s: %w", tutorURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server error (status %d): %s", resp.StatusCode, data)
	}

	var out askResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return out.Content, nil
}
