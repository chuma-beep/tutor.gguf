package desktop

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/chuma-beep/tutor.gguf/internal/llm"
	"github.com/chuma-beep/tutor.gguf/internal/parse"
	"github.com/chuma-beep/tutor.gguf/internal/prompt"
	"github.com/chuma-beep/tutor.gguf/internal/rag"
	"github.com/chuma-beep/tutor.gguf/internal/runtime"
	chromem "github.com/philippgille/chromem-go"
)

// App is the Wails-bound application struct. It wraps Managed RAG via
// direct Go bindings (no HTTP hop) while keeping the TUI intact.
type App struct {
	ctx        context.Context
	mgr        *runtime.Manager
	db         *chromem.DB
	collection *chromem.Collection
	retriever  *rag.Retriever
	genClient  *llm.Client
	dbPath     string
	ready      bool
	startErr   string
}

// NewApp creates a new App.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved so we can
// call the runtime methods and emit events. It attempts to start the
// Managed RAG stack in background; if models are missing it stays not-ready
// until Setup completes (background non-blocking per spec).
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.dbPath = resolveDBPath("")
	// Try to start managed stack, but don't crash if provisioning not done.
	if err := a.initRAG(ctx); err != nil {
		a.startErr = err.Error()
		log.Printf("desktop startup: RAG not ready: %v", err)
		return
	}
	a.ready = true
	log.Printf("desktop startup: RAG ready db=%s gen=%s embed=%s", a.dbPath, a.mgr.GenURL(), a.mgr.EmbedURL())
}

func (a *App) initRAG(ctx context.Context) error {
	if _, err := runtime.DiscoverLlamaServer(); err != nil {
		return err
	}
	genPath := runtime.GenModelPath()
	if _, err := os.Stat(genPath); err != nil {
		return fmt.Errorf("generation model missing at %s — run setup", genPath)
	}
	embedPath := runtime.EmbedModelPath()
	if _, err := os.Stat(embedPath); err != nil {
		return fmt.Errorf("embedding model missing at %s — run setup", embedPath)
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
		return fmt.Errorf("start llama-servers: %w", err)
	}
	embedder := rag.NewEmbedder(mgr.EmbedURL())
	db, err := chromem.NewPersistentDB(a.dbPath, false)
	if err != nil {
		mgr.Stop()
		return fmt.Errorf("open chromem db: %w", err)
	}
	collection, err := db.GetOrCreateCollection("tutor-corpus", nil, embedder.EmbedDocument)
	if err != nil {
		mgr.Stop()
		return fmt.Errorf("get collection: %w", err)
	}
	classifier := rag.NewSubdomainClassifier()
	retriever := rag.NewRetriever(collection, classifier, embedder)
	genClient := llm.NewClient(mgr.GenURL())

	a.mgr = mgr
	a.db = db
	a.collection = collection
	a.retriever = retriever
	a.genClient = genClient
	return nil
}

// Shutdown is called when the app terminates.
func (a *App) Shutdown(ctx context.Context) {
	if a.mgr != nil {
		a.mgr.Stop()
	}
}

// AskResult is the JSON shape returned to the Svelte frontend.
type AskResult struct {
	Content   string            `json:"content"`
	Answer    string            `json:"answer"`
	Subdomain string            `json:"subdomain"`
	Category  string            `json:"category"`
	Prompt    string            `json:"prompt"`
	Chunks    []rag.ScoredChunk `json:"chunks"`
}

// Ask performs retrieval + prompt build + generation via direct Go bindings.
// It mirrors internal/cli/serve.go NewRAGHandler logic without the HTTP hop.
func (a *App) Ask(problem string) (*AskResult, error) {
	if !a.ready || a.retriever == nil || a.genClient == nil {
		return nil, fmt.Errorf("RAG not ready: %s — run setup", a.startErr)
	}
	if problem == "" {
		return nil, fmt.Errorf("missing problem")
	}
	// Use background context derived from startup ctx if available, else new.
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	chunks, subdomain, err := a.retriever.Retrieve(ctx, problem)
	if err != nil {
		log.Printf("Ask retrieval failed: %v", err)
		chunks, subdomain = nil, "other"
	}
	promptStr := rag.BuildPrompt(problem, chunks, subdomain)
	content, err := a.genClient.Complete(promptStr)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}
	category := prompt.PromptCategory(subdomain)
	answer := parse.Extract(content)
	return &AskResult{
		Content:   content,
		Answer:    answer,
		Subdomain: subdomain,
		Category:  category,
		Prompt:    promptStr,
		Chunks:    chunks,
	}, nil
}

// GetPaths exposes Tutor Home locations to the frontend.
func (a *App) GetPaths() map[string]string {
	return map[string]string{
		"home":        runtime.TutorHome(),
		"db":          runtime.DBPath(),
		"models":      runtime.ModelsDir(),
		"bin":         runtime.BinDir(),
		"logs":        runtime.LogsDir(),
		"corpus":      runtime.CorpusDir(),
		"genModel":    runtime.GenModelPath(),
		"embedModel":  runtime.EmbedModelPath(),
		"llamaServer": runtime.LlamaServerPath(),
		"dbResolved":  a.dbPath,
	}
}

// Status is the setup/readiness snapshot for SetupView.
type Status struct {
	Ready          bool   `json:"ready"`
	StartErr       string `json:"startErr"`
	DBPath         string `json:"dbPath"`
	DBChunks       int    `json:"dbChunks"`
	GenModelExists bool   `json:"genModelExists"`
	GenModelBytes  int64  `json:"genModelBytes"`
	EmbedExists    bool   `json:"embedExists"`
	EmbedBytes     int64  `json:"embedBytes"`
	LlamaServer    string `json:"llamaServer"`
	LlamaExists    bool   `json:"llamaExists"`
	GenURL         string `json:"genURL"`
	EmbedURL       string `json:"embedURL"`
}

// GetStatus returns the current provisioning and readiness snapshot.
func (a *App) GetStatus() Status {
	s := Status{
		Ready:    a.ready,
		StartErr: a.startErr,
		DBPath:   a.dbPath,
	}
	if a.mgr != nil {
		s.GenURL = a.mgr.GenURL()
		s.EmbedURL = a.mgr.EmbedURL()
	}
	if p, err := runtime.DiscoverLlamaServer(); err == nil {
		s.LlamaServer = p
		s.LlamaExists = true
	}
	if info, err := os.Stat(runtime.GenModelPath()); err == nil {
		s.GenModelExists = true
		s.GenModelBytes = info.Size()
	}
	if info, err := os.Stat(runtime.EmbedModelPath()); err == nil {
		s.EmbedExists = true
		s.EmbedBytes = info.Size()
	}
	// Best-effort DB chunk count.
	if a.collection != nil {
		s.DBChunks = a.collection.Count()
	} else {
		// Try open read-only to count without mgr.
		if db, err := chromem.NewPersistentDB(a.dbPath, false); err == nil {
			if col, err := db.GetOrCreateCollection("tutor-corpus", nil, func(ctx context.Context, s string) ([]float32, error) { return nil, nil }); err == nil {
				s.DBChunks = col.Count()
			}
		}
	}
	return s
}

// Health returns true when the RAG stack is ready and both llama-servers are healthy.
// It is exposed for the frontend SetupView polling and for wails dev proxy.
func (a *App) Health() bool {
	return a.ready && a.mgr != nil
}

// RetryInit attempts to re-initialize the RAG stack (used by SetupView retry).
// It is non-blocking in the sense the frontend can call it after provisioning.
func (a *App) RetryInit() (Status, error) {
	if a.ready {
		return a.GetStatus(), nil
	}
	if a.mgr != nil {
		a.mgr.Stop()
		a.mgr = nil
	}
	// Ensure ctx exists
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
		a.ctx = ctx
	}
	a.dbPath = resolveDBPath("")
	if err := a.initRAG(ctx); err != nil {
		a.startErr = err.Error()
		a.ready = false
		return a.GetStatus(), err
	}
	a.ready = true
	a.startErr = ""
	return a.GetStatus(), nil
}

// Greet is a placeholder bound method for smoke testing the Wails bridge.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Tutor.gguf desktop is ready!", name)
}

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
