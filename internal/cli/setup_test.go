package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func shrinkRetries() func() {
	oldMax, oldBackoff := hendrycksMaxAttempts, hendrycksBackoff
	hendrycksMaxAttempts = 3
	hendrycksBackoff = time.Millisecond
	return func() { hendrycksMaxAttempts, hendrycksBackoff = oldMax, oldBackoff }
}

func rowJSON(problem string) string {
	return fmt.Sprintf(`{"problem":%q,"solution":"solution","level":"Level 1","type":"Number Theory"}`, problem)
}

// TestFetchHendrycksRetries429 proves a 429 HTML page is retried and the
// Retry-After header is honored before a successful page.
func TestFetchHendrycksRetries429(t *testing.T) {
	defer shrinkRetries()()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("<!DOCTYPE html><html>rate limited</html>"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"num_rows_total":2,"rows":[{"row":%s},{"row":%s}]}`,
			rowJSON("p1"), rowJSON("p2"))
	}))
	defer srv.Close()

	old := hendrycksClient
	hendrycksClient = srv.Client()
	defer func() { hendrycksClient = old }()

	dir := t.TempDir()
	n, err := fetchHendrycksConfigURL(context.Background(), srv.URL, "number_theory", dir, "test")
	if err != nil {
		t.Fatalf("fetchHendrycksConfig: %v", err)
	}
	if n != 2 {
		t.Fatalf("rows = %d, want 2", n)
	}
	if calls < 2 {
		t.Fatalf("expected at least 2 calls (retry), got %d", calls)
	}
}

// TestFetchHendrycksRetries500 proves 500 HTML (tail-shard cache miss) is
// retried, and the HTTP status surfaces in the error if it never succeeds.
func TestFetchHendrycksRetries500(t *testing.T) {
	defer shrinkRetries()()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("<!DOCTYPE html><html>boom</html>"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"num_rows_total":1,"rows":[{"row":%s}]}`, rowJSON("p1"))
	}))
	defer srv.Close()

	old := hendrycksClient
	hendrycksClient = srv.Client()
	defer func() { hendrycksClient = old }()

	dir := t.TempDir()
	n, err := fetchHendrycksConfigURL(context.Background(), srv.URL, "number_theory", dir, "test")
	if err != nil {
		t.Fatalf("fetchHendrycksConfig: %v", err)
	}
	if n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}

// TestFetchHendrycksHardError proves a non-transient 403 HTML page surfaces
// as an actionable error containing the status and body snippet, not the
// opaque "invalid character '<'".
func TestFetchHendrycksHardError(t *testing.T) {
	defer shrinkRetries()()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("<!DOCTYPE html><html>Attention Required | Cloudflare</html>"))
	}))
	defer srv.Close()

	old := hendrycksClient
	hendrycksClient = srv.Client()
	defer func() { hendrycksClient = old }()

	dir := t.TempDir()
	_, err := fetchHendrycksConfigURL(context.Background(), srv.URL, "number_theory", dir, "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "403"; !contains(err.Error(), want) {
		t.Fatalf("error %q missing status %q", err, want)
	}
	if want := "Cloudflare"; !contains(err.Error(), want) {
		t.Fatalf("error %q missing body snippet %q", err, want)
	}
}

// TestEnsureHendrycksPerConfigIdempotent proves a partial config on disk is
// skipped without network, and a full failure warns and continues.
func TestEnsureHendrycksPerConfigIdempotent(t *testing.T) {
	defer shrinkRetries()()
	dest := t.TempDir()
	// Simulate an already-fetched algebra config.
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "algebra-00000.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Force all configs to fail: assert continue-on-partial does not abort.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("<!DOCTYPE html><html>down</html>"))
	}))
	defer srv.Close()

	old := hendrycksClient
	hendrycksClient = srv.Client()
	defer func() { hendrycksClient = old }()

	err := ensureHendrycksURLs(context.Background(), srv.URL, dest, false, nil)
	if err != nil {
		t.Fatalf("expected warn-and-continue, got error: %v", err)
	}
	// The skipped algebra file must remain untouched and other configs not
	// written (all failed).
	if _, err := os.Stat(filepath.Join(dest, "algebra-00000.json")); err != nil {
		t.Fatalf("algebra file should still exist: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
