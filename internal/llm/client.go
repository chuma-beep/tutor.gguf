package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL string
}

type CompletionRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Stream      bool    `json:"stream,omitempty"`
}

type StreamChunk struct {
	Content string `json:"content"`
	Stop    bool   `json:"stop"`
	Tokens  []int  `json:"tokens_evaluated,omitempty"`
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
	return c.CompleteWithContext(context.Background(), prompt)
}

func (c *Client) CompleteWithContext(ctx context.Context, prompt string) (string, error) {
	req := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   512,
		Temperature: 0.1,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/completion", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
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

// CompleteStream streams tokens via SSE (llama-server ?stream=true). onDelta is
// called for each content delta; the final accumulated string is returned.
// It handles both SSE `data: {...}` lines and `[DONE]` terminator.
func (c *Client) CompleteStream(ctx context.Context, prompt string, onDelta func(string) error) (string, error) {
	req := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   512,
		Temperature: 0.1,
		Stream:      true,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/completion", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http post stream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("generation server stream error (status %d): %s", resp.StatusCode, string(data))
	}
	var acc strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	// increase buffer for long lines (4096 default may be small for prompt)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Try generic map fallback for alternative field names
			var m map[string]interface{}
			if err2 := json.Unmarshal([]byte(payload), &m); err2 == nil {
				if v, ok := m["content"].(string); ok {
					chunk.Content = v
				}
				if v, ok := m["stop"].(bool); ok {
					chunk.Stop = v
				}
			} else {
				continue
			}
		}
		if chunk.Content != "" {
			acc.WriteString(chunk.Content)
			if onDelta != nil {
				if err := onDelta(chunk.Content); err != nil {
					return acc.String(), err
				}
			}
		}
		if chunk.Stop {
			break
		}
		select {
		case <-ctx.Done():
			return acc.String(), ctx.Err()
		default:
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		// If we already accumulated, return it; otherwise surface error
		if acc.Len() > 0 {
			return acc.String(), nil
		}
		return "", fmt.Errorf("stream scan: %w", err)
	}
	return acc.String(), nil
}
