package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
}

type CompletionRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type CompletionResponse struct {
	Content string           `json:"content"`
	Error   *CompletionError `json:"error,omitempty"`
}

type CompletionError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

func (c *Client) Complete(prompt string) (string, error) {
	req := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   512,
		Temperature: 0.1,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(c.BaseURL+"/completion", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result CompletionResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || result.Error != nil {
		msg := "unknown"
		if result.Error != nil && result.Error.Message != "" {
			msg = result.Error.Message
		}
		return "", fmt.Errorf("generation server error (status %d): %s", resp.StatusCode, msg)
	}

	return result.Content, nil
}
