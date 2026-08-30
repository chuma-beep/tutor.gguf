package desktop

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/chuma-beep/tutor.gguf/internal/cli"
	"github.com/chuma-beep/tutor.gguf/internal/llm"
	"github.com/chuma-beep/tutor.gguf/internal/parse"
	"github.com/chuma-beep/tutor.gguf/internal/prompt"
	"github.com/chuma-beep/tutor.gguf/internal/rag"
	goruntime "github.com/chuma-beep/tutor.gguf/internal/runtime"
	chromem "github.com/philippgille/chromem-go"
)

// App is the Wails-bound application struct. It wraps Managed RAG via
// direct Go bindings (no HTTP hop) while keeping the TUI intact.
type App struct {
	ctx        context.Context
	mgr        *goruntime.Manager
	db         *chromem.DB
	collection *chromem.Collection
	retriever  *rag.Retriever
	genClient  *llm.Client
	dbPath     string
	ready      bool
	startErr   string

	setupCancel context.CancelFunc
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
	if _, err := goruntime.DiscoverLlamaServer(); err != nil {
		return err
	}
	genPath := goruntime.GenModelPath()
	if _, err := os.Stat(genPath); err != nil {
		return fmt.Errorf("generation model missing at %s — run setup", genPath)
	}
	embedPath := goruntime.EmbedModelPath()
	if _, err := os.Stat(embedPath); err != nil {
		return fmt.Errorf("embedding model missing at %s — run setup", embedPath)
	}
	mgr := goruntime.New(goruntime.Config{
		GenModel:   genPath,
		EmbedModel: embedPath,
		Threads:    envInt("TUTOR_THREADS", 0),
		Ctx:        envInt("TUTOR_CTX", 2048),
		LogDir:     goruntime.LogsDir(),
		Mode:       goruntime.ModeBoth,
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

// AskStream streams generation token-by-token to the frontend via Wails
// events (chunk), then a done event with the final answer. It mirrors
// /v1/complete/stream without the HTTP hop.
func (a *App) AskStream(problem string) error {
	log.Printf("AskStream called problem=%q", problem)
	if !a.ready || a.retriever == nil || a.genClient == nil {
		return fmt.Errorf("RAG not ready: %s — run setup", a.startErr)
	}
	if problem == "" {
		return fmt.Errorf("missing problem")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	chunks, subdomain, err := a.retriever.Retrieve(ctx, problem)
	if err != nil {
		log.Printf("AskStream retrieval failed: %v", err)
		chunks, subdomain = nil, "other"
	}
	promptStr := rag.BuildPrompt(problem, chunks, subdomain)
	category := prompt.PromptCategory(subdomain)

	// Emit meta event so the UI can show the pill immediately
	runtime.EventsEmit(ctx, "tutor:stream:meta", map[string]string{
		"subdomain": subdomain,
		"category":  category,
	})

	var accum strings.Builder
	_, err = a.genClient.CompleteStream(ctx, promptStr, func(delta string) error {
		accum.WriteString(delta)
		runtime.EventsEmit(ctx, "tutor:stream:chunk", map[string]string{"content": delta})
		return nil
	})
	if err != nil {
		runtime.EventsEmit(ctx, "tutor:stream:error", map[string]string{"error": err.Error()})
		return err
	}
	runtime.EventsEmit(ctx, "tutor:stream:done", map[string]string{
		"content": accum.String(),
		"answer":  parse.Extract(accum.String()),
	})
	return nil
}

// Setup runs the 8-phase provisioning, emitting tutor:setup:progress events
// (including byte counts for the progress bar). Returns an error if any phase
// fails; on success it re-initializes the RAG stack so chat is available.
func (a *App) Setup(force, skipModels, skipCorpus bool) error {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
		a.ctx = ctx
	}
	// Cancel any previous run, then give this run a cancellable context.
	a.CancelSetup()
	setupCtx, cancel := context.WithCancel(ctx)
	a.setupCancel = cancel
	defer func() { a.setupCancel = nil }()
	err := cli.SetupWithProgress(setupCtx, cli.SetupOptions{
		Force:      force,
		SkipModels: skipModels,
		SkipCorpus: skipCorpus,
		Progress: func(p cli.SetupProgress) {
			runtime.EventsEmit(ctx, "tutor:setup:progress", map[string]interface{}{
				"phase":      p.Phase,
				"message":    p.Message,
				"downloaded": p.Downloaded,
				"total":      p.Total,
			})
		},
	})
	if err != nil {
		runtime.EventsEmit(ctx, "tutor:setup:error", map[string]string{"error": err.Error()})
		return err
	}
	runtime.EventsEmit(ctx, "tutor:setup:done", map[string]string{"message": "setup complete"})
	// Re-init the RAG stack now that artifacts exist.
	if a.mgr != nil {
		a.mgr.Stop()
		a.mgr = nil
	}
	a.ready = false
	a.startErr = ""
	if err := a.initRAG(ctx); err != nil {
		a.startErr = err.Error()
		return err
	}
	a.ready = true
	return nil
}

// CancelSetup cancels an in-flight Setup. The next setup call will resume
// from where it stopped (idempotent phases + .part resume).
func (a *App) CancelSetup() {
	if a.setupCancel != nil {
		a.setupCancel()
		a.setupCancel = nil
	}
}

// GetPaths exposes Tutor Home locations to the frontend.
func (a *App) GetPaths() map[string]string {
	return map[string]string{
		"home":        goruntime.TutorHome(),
		"db":          goruntime.DBPath(),
		"models":      goruntime.ModelsDir(),
		"bin":         goruntime.BinDir(),
		"logs":        goruntime.LogsDir(),
		"corpus":      goruntime.CorpusDir(),
		"genModel":    goruntime.GenModelPath(),
		"embedModel":  goruntime.EmbedModelPath(),
		"llamaServer": goruntime.LlamaServerPath(),
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
	if p, err := goruntime.DiscoverLlamaServer(); err == nil {
		s.LlamaServer = p
		s.LlamaExists = true
	}
	if info, err := os.Stat(goruntime.GenModelPath()); err == nil {
		s.GenModelExists = true
		s.GenModelBytes = info.Size()
	}
	if info, err := os.Stat(goruntime.EmbedModelPath()); err == nil {
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

// SetNumberWords toggles word-number normalization (e.g. "one plus one" -> "1 + 1").
func (a *App) SetNumberWords(enabled bool) { prompt.SetNumberWords(enabled) }

// GetNumberWords returns whether word-number normalization is enabled.
func (a *App) GetNumberWords() bool { return prompt.NumberWordsEnabled }

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
	return goruntime.DBPath()
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
