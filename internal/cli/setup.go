package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	Phase      string `json:"phase"`
	Message    string `json:"message"`
	Downloaded int64  `json:"downloaded,omitempty"`
	Total      int64  `json:"total,omitempty"`
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

// reportBytes emits a progress event carrying byte counts for the UI bar.
func reportBytes(p ProgressFn, phase string, downloaded, total int64) {
	if p == nil {
		return
	}
	p(SetupProgress{Phase: phase, Downloaded: downloaded, Total: total})
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
		if err := ensureLlamaServer(ctx, opts.Force, opts.Progress); err != nil {
			return err
		}
		report(opts.Progress, "gen-model", "download %s", "~1.1 GB (generation)")
		if err := ensureGGUF(ctx, genModelURL, runtime.GenModelPath(), "~1.1 GB (generation)", "gen-model", opts.Force, opts.Progress); err != nil {
			return err
		}
		report(opts.Progress, "embed-model", "download %s", "~90 MB (embeddings)")
		if err := ensureGGUF(ctx, embedModelURL, runtime.EmbedModelPath(), "~90 MB (embeddings)", "embed-model", opts.Force, opts.Progress); err != nil {
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

func ensureLlamaServer(ctx context.Context, force bool, p ProgressFn) error {
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
	if _, err := fetch.EnsureFileProgress(ctx, url, archive, func(d, t int64) {
		reportBytes(p, "llama", d, t)
	}); err != nil {
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
	bin, err := runtime.DiscoverLlamaServer()
	if err != nil {
		return fmt.Errorf("installed llama-server but still cannot find it: %w", err)
	}
	if err := verifyLlamaServer(bin); err != nil {
		return err
	}
	fmt.Printf("✓ llama-server installed: %s\n", bin)
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

func ensureGGUF(ctx context.Context, url, dest, label, phase string, force bool, p ProgressFn) error {
	if !force {
		if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
			fmt.Printf("✓ %s (%s)\n", dest, humanSize(info.Size()))
			return nil
		}
	}
	fmt.Printf("↓ downloading %s → %s\n", label, dest)
	did, err := fetch.EnsureFileProgress(ctx, url, dest, func(d, t int64) {
		reportBytes(p, phase, d, t)
	})
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
// flat per-file layout our non-recursive chunker expects. Per-config
// idempotent: a config that already has files on disk is skipped, so a
// partial failure can be resumed with a plain `tutor setup` re-run.
// A config that keeps failing after retries is skipped with a warning — the
// corpus stays partial rather than aborting setup — unless nothing at all
// was fetched.
func ensureHendrycks(ctx context.Context, force bool) error {
	dest := runtime.HendrycksTrainDir()
	return ensureHendrycksURLs(ctx, "https://datasets-server.huggingface.co/rows", dest, force, nil)
}

// ensureHendrycksURLs is ensureHendrycks with injectable base URL, dest dir
// and config list (nil = the standard seven). Used by tests.
func ensureHendrycksURLs(ctx context.Context, baseURL, dest string, force bool, configs []string) error {
	if configs == nil {
		configs = hendrycksConfigs
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	fmt.Println("↓ downloading Hendrycks MATH train split…")
	total := 0
	var skipped []string
	for _, cfg := range configs {
		if !force {
			if n, err := countFilesPrefixed(dest, ".json", cfg+"-"); err == nil && n > 0 {
				fmt.Printf("  ✓ %s: %d problems (already on disk)\n", cfg, n)
				total += n
				continue
			}
		}
		n, err := fetchHendrycksConfigURL(ctx, baseURL, cfg, dest, hendrycksDataset)
		if err != nil {
			// Warn and continue with a partial corpus. Setup still
			// completes; the missing subdomain just won't retrieve.
			fmt.Printf("  ⚠ %s: failed after retries: %v\n", cfg, err)
			fmt.Printf("    continuing with a partial corpus — re-run `tutor setup` to retry this config\n")
			skipped = append(skipped, cfg)
			continue
		}
		fmt.Printf("  ✓ %s: %d problems\n", cfg, n)
		total += n
	}
	if total == 0 {
		return errors.New("no problems fetched — datasets-server response may have changed; check network or HF status")
	}
	if len(skipped) > 0 {
		fmt.Printf("⚠ Hendrycks MATH corpus partial: %d problems (missing %s)\n", total, strings.Join(skipped, ", "))
	} else {
		fmt.Printf("✓ Hendrycks MATH corpus: %d problems\n", total)
	}
	return nil
}

type hendrycksRow struct {
	Problem  string `json:"problem"`
	Solution string `json:"solution"`
	Level    string `json:"level"`
	Type     string `json:"type"`
}

// hendrycksClient is the shared HTTP client for datasets-server pages.
var hendrycksClient = &http.Client{Timeout: 60 * time.Second}

// Retry knobs — variables so tests can shrink the backoff.
var (
	hendrycksMaxAttempts = 5
	hendrycksBackoff     = time.Second
)

// fetchHendrycksConfig downloads one subdomain's problems page by page.
// It guards HTTP status (datasets-server can answer 429/500 with an HTML
// error page), sets a real User-Agent, retries transient failures up to 5
// times with exponential backoff (honoring Retry-After), and paces pages so
// anonymous rate limits are not tripped.
func fetchHendrycksConfig(ctx context.Context, config, destDir string) (int, error) {
	return fetchHendrycksConfigURL(ctx, "https://datasets-server.huggingface.co/rows", config, destDir, hendrycksDataset)
}

// fetchHendrycksConfigURL downloads one subdomain's problems page by page.
// It guards HTTP status (datasets-server can answer 429/500 with an HTML
// error page), sets a real User-Agent, retries transient failures up to 5
// times with exponential backoff (honoring Retry-After), and paces pages so
// anonymous rate limits are not tripped. baseURL and dataset are injectable
// for tests.
func fetchHendrycksConfigURL(ctx context.Context, baseURL, config, destDir, dataset string) (int, error) {
	const page = 100
	written := 0
	for offset := 0; ; offset += page {
		u, err := url.Parse(baseURL)
		if err != nil {
			return 0, err
		}
		q := u.Query()
		q.Set("dataset", dataset)
		q.Set("config", config)
		q.Set("split", "train")
		q.Set("offset", strconv.Itoa(offset))
		q.Set("length", strconv.Itoa(page))
		u.RawQuery = q.Encode()

		rows, numTotal, err := fetchHendrycksPage(ctx, u.String(), config, offset)
		if err != nil {
			return written, err
		}

		for i, r := range rows {
			out, err := json.Marshal(r)
			if err != nil {
				return written, err
			}
			name := filepath.Join(destDir, fmt.Sprintf("%s-%05d.json", config, offset+i))
			if err := os.WriteFile(name, out, 0o644); err != nil {
				return written, err
			}
			written++
		}
		if len(rows) == 0 || offset+len(rows) >= numTotal {
			return written, nil
		}
		// Pace anonymous requests — ~21 pages total across configs; keep
		// well under the ~100 req/min anonymous bucket.
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// fetchHendrycksPage performs one paginated GET with retries. Transient
// errors (429/500/502/503) retry up to 5 times with exponential backoff,
// honoring a Retry-After header when present.
func fetchHendrycksPage(ctx context.Context, urlStr, config string, offset int) ([]hendrycksRow, int, error) {
	maxAttempts := hendrycksMaxAttempts
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * hendrycksBackoff
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", "tutor.gguf/0.1.0 (ADTC; +https://github.com/chuma-beep/tutor.gguf)")
		req.Header.Set("Accept", "application/json")
		if tok := os.Getenv("HF_TOKEN"); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}

		resp, err := hendrycksClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
			lastErr = fmt.Errorf("datasets-server %s at offset %d (config %s): %q",
				resp.Status, offset, config, string(snippet))
			// Transient statuses retry; anything else is a hard failure.
			if resp.StatusCode != http.StatusTooManyRequests &&
				resp.StatusCode != http.StatusInternalServerError &&
				resp.StatusCode != http.StatusBadGateway &&
				resp.StatusCode != http.StatusServiceUnavailable {
				return nil, 0, lastErr
			}
			// 429 may carry Retry-After (seconds) — respect it once.
			if resp.StatusCode == http.StatusTooManyRequests {
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if secs, err := strconv.Atoi(ra); err == nil && secs > 0 && secs <= 30 {
						select {
						case <-ctx.Done():
							return nil, 0, ctx.Err()
						case <-time.After(time.Duration(secs) * time.Second):
						}
						// One Retry-After wait per attempt loop iteration.
						continue
					}
				}
			}
			continue
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
			lastErr = fmt.Errorf("decode rows at offset %d (config %s): %w", offset, config, err)
			continue
		}
		rows := make([]hendrycksRow, 0, len(body.Rows))
		for _, r := range body.Rows {
			rows = append(rows, r.Row)
		}
		return rows, body.NumRowsTotal, nil
	}
	return nil, 0, fmt.Errorf("giving up after %d attempts: %w", maxAttempts, lastErr)
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
	return countFilesPrefixed(dir, ext, "")
}

// countFilesPrefixed counts files in dir with the given extension whose name
// starts with prefix ("" matches all).
func countFilesPrefixed(dir, ext, prefix string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), prefix) && filepath.Ext(d.Name()) == ext {
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
