package cli

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	tutor "github.com/chuma-beep/tutor.gguf"
)

// webUIDist returns the embedded frontend/dist tree (built by `npm run build`
// in frontend/). It is the same FS the Wails desktop shell serves; `tutor serve`
// reuses those bytes so localhost gets a real UI with no extra download.
func webUIDist() (fs.FS, error) {
	return fs.Sub(tutor.FrontendDist, "frontend/dist")
}

// registerWebUI mounts the static browser UI at GET /. API routes (/health,
// /v1/*) must be registered first — the Go 1.22+ mux prefers the more specific
// pattern, so "/" only handles what the API does not.
func registerWebUI(mux *http.ServeMux) error {
	dist, err := webUIDist()
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(dist))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		if handleCORS(w, r) {
			return
		}
		clean := path.Clean(r.URL.Path)
		// Serve real files (/, /assets/*, ...) directly with cache policy.
		// Note: clean == "/" is served as the directory index — dist.Open("")
		// would fail, so it is special-cased.
		if clean == "/" {
			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, r)
			return
		}
		if f, err := dist.Open(strings.TrimPrefix(clean, "/")); err == nil {
			_ = f.Close()
			if strings.HasPrefix(clean, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		// Unknown extensionless path: SPA fallback to index.html so a
		// browser refresh on a client-side route still loads the app.
		// Paths with a file extension 404 instead of masking typos.
		// index.html is written directly (rather than rewriting the request
		// path) because FileServer redirects /index.html to ./.
		if path.Ext(clean) != "" {
			http.NotFound(w, r)
			return
		}
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
	})
	return nil
}
