package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/chuma-beep/tutor.gguf/internal/runtime"
)

// serverManager bundles supervised llama-server processes with the URLs the
// rest of the CLI consumes.
type serverManager struct {
	mgr      *runtime.Manager
	genURL   string
	embedURL string
}

func (s *serverManager) stop() {
	if s != nil && s.mgr != nil {
		s.mgr.Stop()
	}
}

// startManaged spawns local llama-server processes for generation and
// embeddings and waits until both are healthy. Models must already be
// provisioned by `tutor setup`.
func startManaged(ctx context.Context) (*serverManager, error) {
	if _, err := runtime.DiscoverLlamaServer(); err != nil {
		return nil, err
	}
	genPath := runtime.GenModelPath()
	if _, err := os.Stat(genPath); err != nil {
		return nil, fmt.Errorf("generation model missing at %s — run `tutor setup`", genPath)
	}
	embedPath := runtime.EmbedModelPath()
	if _, err := os.Stat(embedPath); err != nil {
		return nil, fmt.Errorf("embedding model missing at %s — run `tutor setup`", embedPath)
	}

	mgr := runtime.New(runtime.Config{
		GenModel:   genPath,
		EmbedModel: embedPath,
		Threads:    envInt("TUTOR_THREADS", 0),
		Ctx:        envInt("TUTOR_CTX", 2048),
		LogDir:     runtime.LogsDir(),
		Mode:       runtime.ModeBoth,
	})
	if err := mgr.Start(ctx); err != nil {
		mgr.Stop()
		return nil, err
	}
	fmt.Printf("llama-servers ready — gen: %s | embed: %s\n", mgr.GenURL(), mgr.EmbedURL())
	return &serverManager{mgr: mgr, genURL: mgr.GenURL(), embedURL: mgr.EmbedURL()}, nil
}

// resolveDBPath picks the vector store location:
// explicit flag > $TUTOR_DB_PATH > ./data/chromem inside a repo checkout >
// ~/.tutor/chromem for standalone installs.
func resolveDBPath(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv("TUTOR_DB_PATH"); v != "" {
		return v
	}
	if fi, err := os.Stat("./data/chromem"); err == nil && fi.IsDir() {
		return "./data/chromem"
	}
	if _, err := os.Stat("go.mod"); err == nil {
		return "./data/chromem"
	}
	return runtime.DBPath()
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
