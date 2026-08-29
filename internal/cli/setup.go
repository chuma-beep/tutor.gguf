package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chuma-beep/tutor.gguf/assets"
	"github.com/chuma-beep/tutor.gguf/internal/fetch"
	"github.com/chuma-beep/tutor.gguf/internal/rag"
	"github.com/chuma-beep/tutor.gguf/internal/runtime"
	chromem "github.com/philippgille/chromem-go"
)

const (
	genModelURL   = "https://huggingface.co/bartowski/Qwen2.5-Math-1.5B-Instruct-GGUF/resolve/main/Qwen2.5-Math-1.5B-Instruct-Q4_K_M.gguf"
	embedModelURL = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf"
	gsm8kTrainURL = "https://raw.githubusercontent.com/openai/grade-school-math/master/grade_school_math/data/train.jsonl"

	defaultLlamaTag = "b10612"

	// hendrycksDataset is served as paginated JSON by the HF datasets-server
	// (the original Berkeley MATH.tar now 403s). Fields match the chunker.
	hendrycksDataset = "EleutherAI/hendrycks_math"
)

var hendrycksConfigs = []string{
	"algebra", "counting_and_probability", "geometry", "intermediate_algebra",
	"number_theory", "prealgebra", "precalculus",
}

// SetupProgress is a phase-level progress event emitted during setup.
type SetupProgress struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

// SetupOptions controls one SetupWithProgress run.
type SetupOptions struct {
	Force       bool
	SkipModels  bool
	SkipCorpus  bool
	Progress    func(SetupProgress)
}

func report(p ProgressFn, phase, format string, a ...interface{}) {
	if p == nil {
		return
	}
	p(SetupProgress{Phase: phase, Message: fmt.Sprintf(format, a...)})
}

// Setup provisions everything needed to run offline: the llama.cpp server
// binary, both GGUF models, the corpus snapshot (Rosen ships embedded in this
// binary), and the vector-store index. Every step is idempotent.
func Setup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	home := fs.String("home", "", "tutor home directory (default $TUTOR_HOME or ~/.tutor)")
	force := fs.Bool("force", false, "re-download and re-index even when artifacts exist")
	skipModels := fs.Bool("skip-models", false, "skip llama-server + GGUF downloads")
	skipCorpus := fs.Bool("skip-corpus", false, "skip corpus downloads")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *home != "" {
		os.Setenv("TUTOR_HOME", *home)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return SetupWithProgress(ctx, SetupOptions{
		Force:      *force,
		SkipModels: *skipModels,
		SkipCorpus: *skipCorpus,
		Progress: func(p SetupProgress) {
			fmt.Printf("[%s] %s\n", p.Phase, p.Message)
		},
	})
}

// ProgressFn receives phase progress during setup.
type ProgressFn func(SetupProgress)

// SetupWithProgress runs the full 8-phase provisioning with a progress
// callback. Shared by the CLI (Setup) and the desktop App.
func SetupWithProgress(ctx context.Context, opts SetupOptions) error {
	base := runtime.TutorHome()
	report(opts.Progress, "dirs", "installing into %s", base)
	for _, d := range []string{runtime.ModelsDir(), runtime.BinDir(), runtime.LogsDir(),
		filepath.Join(runtime.CorpusDir(), "gsm8k"),
		filepath.Join(runtime.CorpusDir(), "hendrycks_math", "train")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}

	if !opts.SkipModels {
		report(opts.Progress, "llama", "installing llama-server (tag %s)", envOr("TUTOR_LLAMA_TAG", defaultLlamaTag))
		if err := ensureLlamaServer(ctx, opts.Force); err != nil {
			return err
		}
		report(opts.Progress, "gen-model", "download %s", "~1.1 GB (generation)")
		if err := ensureGGUF(ctx, genModelURL, runtime.GenModelPath(), "~1.1 GB (generation)", opts.Force); err != nil {
			return err
		}
		report(opts.Progress, "embed-model", "download %s", "~90 MB (embeddings)")
		if err := ensureGGUF(ctx, embedModelURL, runtime.EmbedModelPath(), "~90 MB (embeddings)", opts.Force); err != nil {
			return err
		}
	}

	if !opts.SkipCorpus {
		report(opts.Progress, "gsm8k", "downloading GSM8K training problems")
		if err := ensureGSM8K(ctx, opts.Force); err != nil {
			return err
		}
		report(opts.Progress, "hendrycks", "downloading Hendrycks MATH train split (%d configs)", len(hendrycksConfigs))
		if err := ensureHendrycks(ctx, opts.Force); err != nil {
			return err
		}
		report(opts.Progress, "rosen", "writing embedded Rosen corpus")
		if err := ensureRosen(opts.Force); err != nil {
			return err
		}
	}

	report(opts.Progress, "index", "building vector index (sequential embeddings, may take a while)")
	if err := runSetupIndex(ctx, opts.Force); err != nil {
		return err
	}

	report(opts.Progress, "done", "setup complete — models: %s, index: %s", runtime.ModelsDir(), runtime.DBPath())
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ensureLlamaServer(ctx context.Context, force bool) error {
	if !force {
		if p, err := runtime.DiscoverLlamaServer(); err == nil {
			fmt.Printf("✓ llama-server: %s\n", p)
			return nil
		}
	}

	tag := os.Getenv("TUTOR_LLAMA_TAG")
	if tag == "" {
		tag = defaultLlamaTag
	}
	suffix, err := llamaAssetSuffix(goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://github.com/ggml-org/llama.cpp/releases/download/%s/llama-%s-bin-%s", tag, tag, suffix)

	fmt.Printf("↓ downloading llama.cpp %s (%s/%s)…\n", tag, goruntime.GOOS, goruntime.GOARCH)
	tmp, err := os.MkdirTemp("", "tutor-llama")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, filepath.Base(url))
	if _, err := fetch.EnsureFile(ctx, url, archive); err != nil {
		return fmt.Errorf("fetching prebuilt llama.cpp failed (%w) — build it from source and point $TUTOR_LLAMA_SERVER at llama-server instead", err)
	}

	if strings.HasSuffix(suffix, ".zip") {
		err = fetch.ExtractZip(archive, runtime.BinDir())
	} else {
		err = fetch.ExtractTarGz(archive, runtime.TutorHome(), nil)
	}
	if err != nil {
		return fmt.Errorf("extract %s: %w", archive, err)
	}

	// Upstream archive layouts vary between releases (flat llama-b<rev>/
	// dirs vs bin/+lib/ trees); normalize by locating the binary and
	// consolidating everything next to it into ~/.tutor/bin.
	if src := fetch.FindFileBelow(runtime.TutorHome(), "llama-server", runtime.BinDir()); src != "" && filepath.Dir(src) != runtime.BinDir() {
		bindir := filepath.Dir(src)
		if err := fetch.CopyDir(bindir, runtime.BinDir()); err != nil {
			return fmt.Errorf("consolidate %s into %s: %w", bindir, runtime.BinDir(), err)
		}
		if base := filepath.Base(bindir); strings.HasPrefix(base, "llama-") {
			os.RemoveAll(bindir)
		}
	}
	if err := os.Chmod(runtime.LlamaServerPath(), 0o755); err != nil {
		return err
	}
	p, err := runtime.DiscoverLlamaServer()
	if err != nil {
		return fmt.Errorf("installed llama-server but still cannot find it: %w", err)
	}
	if err := verifyLlamaServer(p); err != nil {
		return err
	}
	fmt.Printf("✓ llama-server installed: %s\n", p)
	return nil
}

func llamaAssetSuffix(goos, goarch string) (string, error) {
	switch {
	case goos == "linux" && goarch == "amd64":
		return "ubuntu-x64.tar.gz", nil
	case goos == "linux" && goarch == "arm64":
		return "ubuntu-arm64.tar.gz", nil
	case goos == "darwin" && goarch == "amd64":
		return "macos-x64.tar.gz", nil
	case goos == "darwin" && goarch == "arm64":
		return "macos-arm64.tar.gz", nil
	case goos == "windows" && goarch == "amd64":
		return "win-cpu-x64.zip", nil
	case goos == "windows" && goarch == "arm64":
		return "win-cpu-arm64.zip", nil
	default:
		return "", fmt.Errorf("no prebuilt llama.cpp CPU asset for %s/%s — install llama-server manually and set $TUTOR_LLAMA_SERVER", goos, goarch)
	}
}

func ensureGGUF(ctx context.Context, url, dest, label string, force bool) error {
	if !force {
		if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
			fmt.Printf("✓ %s (%s)\n", dest, humanSize(info.Size()))
			return nil
		}
	}
	fmt.Printf("↓ downloading %s → %s\n", label, dest)
	did, err := fetch.EnsureFile(ctx, url, dest)
	if err != nil {
		return err
	}
	_ = did
	return nil
}

func ensureGSM8K(ctx context.Context, force bool) error {
	dest := runtime.GSM8KTrainFile()
	if !force {
		if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
			fmt.Printf("✓ GSM8K corpus: %s\n", dest)
			return nil
		}
	}
	fmt.Println("↓ downloading GSM8K training problems…")
	_, err := fetch.EnsureFile(ctx, gsm8kTrainURL, dest)
	return err
}

// ensureHendrycks fetches the MATH train split (7 subdomains) from the HF
// datasets-server rows API and writes one JSON file per problem, matching the
// flat per-file layout our non-recursive chunker expects.
func ensureHendrycks(ctx context.Context, force bool) error {
	dest := runtime.HendrycksTrainDir()
	if !force {
		if n, err := countFiles(dest, ".json"); err == nil && n > 0 {
			fmt.Printf("✓ Hendrycks MATH corpus: %s (%d files)\n", filepath.Dir(dest), n)
			return nil
		}
	}
	fmt.Println("↓ downloading Hendrycks MATH train split…")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	total := 0
	for _, cfg := range hendrycksConfigs {
		n, err := fetchHendrycksConfig(ctx, cfg, dest)
		if err != nil {
			return fmt.Errorf("%s: %w", cfg, err)
		}
		fmt.Printf("  ✓ %s: %d problems\n", cfg, n)
		total += n
	}
	if total == 0 {
		return errors.New("no problems fetched — datasets-server response may have changed")
	}
	fmt.Printf("✓ Hendrycks MATH corpus: %d problems\n", total)
	return nil
}

type hendrycksRow struct {
	Problem  string `json:"problem"`
	Solution string `json:"solution"`
	Level    string `json:"level"`
	Type     string `json:"type"`
}

func fetchHendrycksConfig(ctx context.Context, config, destDir string) (int, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	const page = 100
	written := 0
	for offset := 0; ; offset += page {
		u, err := url.Parse("https://datasets-server.huggingface.co/rows")
		if err != nil {
			return 0, err
		}
		q := u.Query()
		q.Set("dataset", hendrycksDataset)
		q.Set("config", config)
		q.Set("split", "train")
		q.Set("offset", strconv.Itoa(offset))
		q.Set("length", strconv.Itoa(page))
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		var body struct {
			NumRowsTotal int `json:"num_rows_total"`
			Rows         []struct {
				Row hendrycksRow `json:"row"`
			} `json:"rows"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil {
			return 0, fmt.Errorf("decode rows at offset %d: %w", offset, err)
		}

		for i, r := range body.Rows {
			out, err := json.Marshal(r.Row)
			if err != nil {
				return 0, err
			}
			name := filepath.Join(destDir, fmt.Sprintf("%s-%05d.json", config, offset+i))
			if err := os.WriteFile(name, out, 0o644); err != nil {
				return 0, err
			}
			written++
		}
		if len(body.Rows) == 0 || offset+len(body.Rows) >= body.NumRowsTotal {
			return written, nil
		}
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}
	}
}

// ensureRosen writes the embedded Rosen discrete-math corpus to disk so the
// standard directory-based indexer can pick it up.
func ensureRosen(force bool) error {
	dest := runtime.RosenDir()
	if !force {
		if _, err := os.Stat(filepath.Join(dest, "book")); err == nil {
			fmt.Printf("✓ Rosen corpus: %s\n", dest)
			return nil
		}
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(assets.Rosen(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := fs.ReadFile(assets.Rosen(), path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// runSetupIndex starts the embedding server and builds the vector store once.
// An existing non-empty index is left untouched unless force is set.
func runSetupIndex(ctx context.Context, force bool) error {
	dbPath := runtime.DBPath()
	if !force {
		if count, err := indexedChunkCount(dbPath); err == nil && count > 0 {
			fmt.Printf("✓ vector index: %s (%d chunks)\n", dbPath, count)
			return nil
		}
	}

	server, err := runtime.DiscoverLlamaServer()
	if err != nil {
		return fmt.Errorf("embedding server: %w", err)
	}
	fmt.Println("↓ starting embedding server and building vector index…")
	mgr := runtime.New(runtime.Config{
		LlamaServer: server,
		EmbedModel:  runtime.EmbedModelPath(),
		Threads:     envInt("TUTOR_THREADS", 0),
		LogDir:      runtime.LogsDir(),
		Mode:        runtime.ModeEmbedOnly,
	})
	if err := mgr.Start(ctx); err != nil {
		mgr.Stop()
		return fmt.Errorf("embedding server: %w", err)
	}
	defer mgr.Stop()

	return RunIndex(ctx, IndexOptions{
		EmbedderURL:  mgr.EmbedURL(),
		DBPath:       dbPath,
		HendrycksDir: runtime.HendrycksTrainDir(),
		GSM8KFile:    runtime.GSM8KTrainFile(),
		RosenDir:     runtime.RosenDir(),
		RunQuery:     false,
	})
}

// indexedChunkCount opens the persistent store read-only-ish to see whether a
// previous setup already populated it.
func indexedChunkCount(dbPath string) (int, error) {
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return 0, err
	}
	embedder := rag.NewEmbedder("")
	col, err := db.GetOrCreateCollection("tutor-corpus", nil, embedder.EmbedDocument)
	if err != nil {
		return 0, err
	}
	return col.Count(), nil
}

func countFiles(dir, ext string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ext {
			n++
		}
		return nil
	})
	return n, err
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// verifyLlamaServer runs `llama-server --version` as a smoke test.
func verifyLlamaServer(path string) error {
	out, err := exec.Command(path, "--version").CombinedOutput()
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if err != nil {
		return fmt.Errorf("%s --version failed: %v (%s)", path, err, line)
	}
	fmt.Printf("  version: %s\n", line)
	return nil
}
