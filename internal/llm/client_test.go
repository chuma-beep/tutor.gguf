package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCompleteStreamSSE exercises CompleteStream against an httptest server
// that speaks llama-server's SSE format. It verifies deltas accumulate and
// the [DONE] terminator stops the stream.
func TestCompleteStreamSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/completion" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		// Echo back the prompt to prove request body round-trips
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"stream":true`) {
			t.Fatalf("expected stream flag in request, got %s", string(body))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"content\":\"The derivative is \\\\(2x\\\\)\"}\n\n"))
		w.Write([]byte("data: {\"content\":\".\",\"stop\":false}\n\n"))
		w.Write([]byte("data: {\"content\":\" \\\\boxed{2x}\"}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	var got []string
	out, err := c.CompleteStream(context.Background(), "prompt", func(d string) error {
		got = append(got, d)
		return nil
	})
	if err != nil {
		t.Fatalf("CompleteStream: %v", err)
	}
	want := `The derivative is \(2x\). \boxed{2x}`
	if out != want {
		t.Fatalf("accumulated = %q, want %q", out, want)
	}
	if len(got) != 3 {
		t.Fatalf("delta count = %d, want 3", len(got))
	}
}

// TestCompleteStreamStop verifies the server can end with a stop:true chunk
// before [DONE].
func TestCompleteStreamStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data: {\"content\":\"42\"}\n\n"))
		w.Write([]byte("data: {\"content\":\"\",\"stop\":true}\n\n"))
	}))
	defer srv.Close()

	out, err := NewClient(srv.URL).CompleteStream(context.Background(), "p", nil)
	if err != nil {
		t.Fatalf("CompleteStream: %v", err)
	}
	if out != "42" {
		t.Fatalf("got %q want 42", out)
	}
}

// TestCompleteStreamError checks non-200 responses surface an error.
func TestCompleteStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"oom"}`, 500)
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL).CompleteStream(context.Background(), "p", nil)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}
