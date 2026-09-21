package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	h, err := NewRAGHandler(context.Background(), "http://127.0.0.1:9", "http://127.0.0.1:9", t.TempDir())
	if err != nil {
		t.Fatalf("NewRAGHandler: %v", err)
	}
	return h
}

// GET / serves the embedded browser UI.
func TestWebUIServesIndex(t *testing.T) {
	h := newTestHandler(t)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET / = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if body := w.Body.String(); !strings.Contains(body, `<div id="app">`) {
		t.Fatalf("index body missing #app mount")
	}
}

// Unknown extensionless paths fall back to index.html (SPA refresh safe).
func TestWebUISPAFallback(t *testing.T) {
	h := newTestHandler(t)
	r := httptest.NewRequest(http.MethodGet, "/history", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET /history = %d, want 200", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `<div id="app">`) {
		t.Fatalf("fallback body missing #app mount")
	}
}

// Unknown paths with a file extension 404 instead of masking typos.
func TestWebUIMissingAsset404(t *testing.T) {
	h := newTestHandler(t)
	r := httptest.NewRequest(http.MethodGet, "/assets/nope.js", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /assets/nope.js = %d, want 404", w.Code)
	}
}

// API routes still win over the static fallback.
func TestWebUIDoesNotShadowAPI(t *testing.T) {
	h := newTestHandler(t)
	r := httptest.NewRequest(http.MethodPost, "/v1/complete", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/complete {} = %d, want 400", w.Code)
	}
}

// GET /v1/status reports readiness + DB chunk count.
func TestStatusEndpoint(t *testing.T) {
	h := newTestHandler(t)
	r := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET /v1/status = %d, want 200", w.Code)
	}
	var s struct {
		Ready    bool   `json:"ready"`
		DBChunks int    `json:"dbChunks"`
		GenURL   string `json:"genURL"`
	}
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !s.Ready {
		t.Fatalf("ready = false, want true")
	}
}
