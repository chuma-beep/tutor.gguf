package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/complete" {
			http.Error(w, "not found", 404)
			return
		}
		var body askRequest
		if err := decodeJSON(r, &body); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		if body.Problem == "" {
			http.Error(w, `{"error":"missing problem"}`, 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"content":%q,"answer":"1/2"}`, content)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func TestAsk(t *testing.T) {
	srv := testServer(t, `To find the derivative \frac{1}{2}`)
	got, err := Ask(srv.URL, "find derivative of x^2")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	want := `To find the derivative \frac{1}{2}`
	if got != want {
		t.Fatalf("content != want:\n got %q\nwant %q", got, want)
	}
}

func TestAskError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"generation failed"}`, 500)
	}))
	defer srv.Close()
	_, err := Ask(srv.URL, "q")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "server error") {
		t.Fatalf("error should mention server error, got: %v", err)
	}
}

// TestModelFlow drives the model through typing, submit, and a blocking
// answer message, asserting the transcript renders the reply.
func TestModelFlow(t *testing.T) {
	m := newModel(Options{
		TutorURL: "http://t.example",
		Render:   identity,
	})

	m, _ = upd(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	for _, r := range "find derivative of x^2" {
		m, _ = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(string(r))})
	}

	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.loading {
		t.Fatal("expected loading state after submit")
	}
	if m.input.Value() != "" {
		t.Fatalf("input should be cleared, got %q", m.input.Value())
	}
	if cmd == nil {
		t.Fatal("submit should return an ask command")
	}

	// In production this would be produced by a streaming/blocking cmd; feed
	// the equivalent answerMsg directly to keep the test terminal-free.
	m, _ = upd(m, answerMsg{id: m.nextID - 1, delta: "The derivative is \\(2x\\).", done: true})

	if m.loading {
		t.Fatal("expected loading to clear after final msg")
	}
	if len(m.turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(m.turns))
	}
	turn := m.turns[0]
	if turn.Question != "find derivative of x^2" {
		t.Fatalf("question mismatch: %q", turn.Question)
	}
	if !strings.Contains(turn.Answer, "2x") {
		t.Fatalf("answer mismatch: %q", turn.Answer)
	}

	view := m.View()
	for _, want := range []string{"tutor.gguf", "Q: find derivative of x^2", "The derivative is"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

// TestStreamingAppend verifies partial deltas accumulate and only the trailing
// done msg finalizes the turn.
func TestStreamingAppend(t *testing.T) {
	m := newModel(Options{TutorURL: "http://t.example", Render: identity})
	m, _ = upd(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m, _ = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !m.loading || m.streaming == nil {
		t.Fatalf("submit should start a streaming turn (loading=%v stream=%v)", m.loading, m.streaming != nil)
	}

	id := m.nextID - 1
	m, _ = upd(m, answerMsg{id: id, delta: "\\(\\frac{1}{"})
	if m.streaming == nil {
		t.Fatal("streaming turn should exist")
	}
	m, _ = upd(m, answerMsg{id: id, delta: "2}\\)", done: true})
	if m.streaming != nil {
		t.Fatal("streaming turn should finalize on done")
	}
	if len(m.turns) != 1 {
		t.Fatalf("expected 1 finalized turn, got %d", len(m.turns))
	}
	if m.turns[0].Answer != "\\(\\frac{1}{2}\\)" {
		t.Fatalf("accumulated answer mismatch: %q", m.turns[0].Answer)
	}
}

func upd(m Model, msg tea.Msg) (Model, tea.Cmd) {
	mm, cmd := m.Update(msg)
	return mm.(Model), cmd
}
