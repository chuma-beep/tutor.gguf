# Spec: Wails Desktop + Keep TUI

Status: ready-for-agent

## Problem Statement

Nigerian CS undergraduates at distance-learning institutions need step-by-step math help (Discrete Mathematics, Calculus I/II, Linear Algebra) but the current Tutor.gguf is TUI-only (`internal/tui/tui.go:23` + `internal/renderer/render.go:16` terminal art). LaTeX (`\frac`, `\sqrt`, `\boxed{}`, `\binom`, `\int`) degrades to stacked `─`/`|` ASCII boxes and streaming is blocking-only (`llm/client.go:36` + `tui.go:173 spinner`). Setup (`internal/cli/setup.go:49`) is 8 phases (~1.2 GB: gen 1.1 GB + embed 90 MB + llama-server b10612 + GSM8K + Hendrycks 7 configs + Rosen + chromem index) and blocks. ADTC 2026 Standard Laptop is Ubuntu 22.04 8 GB / 7 GB managed, integrated GPU only, `llama.cpp + GGUF` only (`metadata.json: runtime=llama.cpp`), 100% offline at eval — any desktop must preserve this and keep `bin/tutor` evaluatable (`make profile` PASS).

## Solution

Add a Svelte + Vite + Tailwind + KaTeX WebView desktop `tutor-desktop` (Wails v2) as a **second frontend** over the same Managed RAG (`internal/runtime.Manager` + `retriever.Retrieve` + `prompt.Builder.Build` + `llm.Client.Complete` + `parse.Extract`). Keep `tutor chat` TUI intact (`cmd/tutor/main.go:34`) for SSH/headless/dev loop; both frontends share `Tutor Home` (`~/.tutor`, `runtime/paths.go:22`) models, `chromem` DB (`DBPath`), and `~/.tutor/logs`. Direct Go bindings (`desktop.App.Ask()` → `retriever.Retrieve` → `BuildPrompt` → `Complete` → `Extract`) eliminate the HTTP hop for desktop, while `POST /v1/complete` (`internal/cli/serve.go:50`) stays for `evals/promptfooconfig.yaml` + `adtc-profiler`. Desktop SetupView runs background non-blocking with `EventsEmit("setup:progress")`; v1 ships blocking generation with spinner, streaming SSE is Phase 2. Distribution: Linux `deb/AppImage`, `darwin/universal`, `windows/amd64` WebView2 via `wails build -tags desktop`.

## User Stories

1. As a Nigerian undergrad, I want to ask "Find the derivative of x^2" in a native window and see KaTeX fractions, square roots, and `\boxed{}` so math is readable on first try.

2. As a distance-learning student on unstable power, I want first launch to download llama.cpp b10612 + Qwen2.5-Math-1.5B-Q4_K_M + nomic-embed-text-v1.5-Q4_K_M + GSM8K + Hendrycks MATH 7 configs + Rosen + index in background while I can chat immediately if `DBPath` already has chunks and models exist, so I am not blocked.

3. As a student, I want everything after setup to work 100% offline with zero outbound (`COMPLIANCE.md:23`) so `adtc-profiler run --mode audit` still PASS and `peak RSS 1.1 GB` stays within 7 GB.

4. As a TUI user on a lab with no desktop env, I want `tutor chat` (`make tui` / `make tui-ascii`, `internal/tui/tui_test.go`) to still work unchanged over SSH so low-end labs are not forced into WebView.

5. As a student, I want citations: `Prompt Category` pill + `subdomainInstructions` text, collapsible Sources `[1..3]` (`ScoredChunk{Text,Subdomain,Similarity}`), and a "View prompt sent to Qwen" debug panel (`cli/index.go:143`) so retrieval is transparent.

6. As a student, I want history desktop-only in `localStorage` with Clear (`Ctrl+L` parity `tui.go:103`), `Esc`/`Ctrl+C` quit, `PgUp/PgDn` scroll with pinned `-1` overscan (`tui.go:300 transcriptOverscan`), spinner "thinking" and cursor `▍` while loading (`tui.go:281`) so UX is familiar.

7. As an evaluator, I want `POST /v1/complete` (`evals/promptfooconfig.yaml:5` `transformResponse: json.content`, `make eval` `maxConcurrency:1`) and `download_model.sh` (`model/*.gguf`) still work against `localhost:8082` so ADTC scoring (`S_total = 0.5 S_acc + 0.3 S_perf + 0.2 S_eff`) is unchanged.

8. As a maintainer, I want Settings to show `TUTOR_THREADS` (0=auto, 4 optimal per `docs/tuning.md`) + `TUTOR_CTX` (2048) with Advanced hidden (`TUTOR_HOME`, `TUTOR_LLAMA_SERVER`, `TUTOR_DB_PATH`, `LlamaServerPath`) behind "Open Config Folder" so tuning is not lost.

9. As a release manager, I want `make build` (`CGO_ENABLED=0` `go build -trimpath -ldflags "-s -w" -o bin/tutor ./cmd/tutor`) still emits `tutor-linux-amd64` and `wails build -tags desktop -platform linux/amd64,windows/amd64,darwin/universal` (`CGO_ENABLED=1` + `webkit2gtk-4.0`) emits `tutor-desktop-*` with `SHA256SUMS` merged in `release.yml` so both are shipped.

10. As a new user, I want the Svelte frontend to render `\( \)` / `\[ \]` / `$` / `$$` spans via `extractSpans` precedence plus unclosed streaming tolerance, with `\boxed{16}` boxed natively by KaTeX (`throwOnError:false`) so model output is never dropped.

## Implementation Decisions

- Modules built/modified: new `frontend/` (SvelteKit or Vite Svelte + TS, `base:'./'`), `wails.json` (`name:tutor-gguf, frontend:dir=frontend, main:cmd/desktop/main.go, outputfilename:tutor-desktop`), `cmd/desktop/main.go` (`wails.Run` with `OnStartup/OnShutdown`), `internal/desktop/app.go` (deep module: `App{ctx,mgr,db,collection,retriever,genClient}`); minimal edit `internal/cli/serve.go` add `GET /health` (and CORS if `wails dev` proxy); `internal/cli/managed.go` export `ResolveDBPath` for sharing or duplicate; `Makefile` `build-desktop`/`dev-desktop` targets; `.gitignore` add `frontend/dist/`, `frontend/node_modules/`, `build/bin/`; `README.md` desktop install section; `docs/agents` unchanged. Keep `cmd/tutor/main.go`, `internal/tui/*`, `internal/renderer/*`, `go.mod` module `github.com/chuma-beep/tutor.gguf` unchanged.

- Interfaces: external `App.Ask(ctx, problem string) → {content, answer, subdomain, category, chunks []ScoredChunk, prompt string}` + HTTP `requestBody{Problem,MaxTokens,Temperature}` / `responseBody{Content,Answer}` (`serve.go:21`). `App.Ask` is small, deep, hides Manager lifecycle, embedding prefixes (`search_query` vs `search_document`), `topK*4→topN 3` dedup, prompt `ChatML` framing, and `parse.Extract` regex `boxedRE→markerRE→gsm8kRE`. Internal `retriever.Retrieve(ctx, query)` + `prompt.Builder.Build(query, []Source, subdomain)` stay behind `Ask`, not exposed to JS.

- Seams: highest seam `App.Ask` (primary) and HTTP `POST /v1/complete` (secondary for eval parity). Not `chromem.Collection` adapter, not `embedder.EmbedDocument` directly. One deep seam, not per-file shallow interfaces.

- Architecture: `App.Startup(ctx)` → `startManaged(ctx)` (checks `DiscoverLlamaServer` precedence `TUTOR_LLAMA_SERVER > ~/.tutor/bin/llama-server > $PATH`, `GenModelPath`/`EmbedModelPath` exist, `runtime.New(Config{GenModel,EmbedModel,Threads:envInt("TUTOR_THREADS",0), Ctx:envInt("TUTOR_CTX",2048), LogDir:LogsDir, Mode:ModeBoth})` → `mgr.Start(ctx)` polls `GET /health` 300ms/5m, captures `gen.log`/`embed.log`, `go func(){<-ctx.Done(); mgr.Stop()}`) → `chromem.NewPersistentDB(DBPath,false)` + `GetOrCreateCollection("tutor-corpus", nil, EmbedDocument)` → `NewRetriever` + `NewClient(genURL)`. `App.Shutdown` → `mgr.Stop()` idempotent. `freePort` (`net.Listen 127.0.0.1:0`) dynamic, not hardcoded `8080/8081`. `ResolveDBPath` precedence `flag > TUTOR_DB_PATH > ./data/chromem (if dir or go.mod) > ~/.tutor/chromem`. Setup wizard: `Setup(ctx, force, skipModels, skipCorpus)` delegates to `Setup.go` 8 phases (dirs → `ensureLlamaServer` asset `ubuntu-x64` etc via `llamaAssetSuffix` → `fetch.EnsureFile` `.part` + `ExtractTarGz/Zip` normalize → `ensureGGUF` gen/embed → `ensureGSM8K` → `ensureHendrycks` paginated `datasets-server` → `ensureRosen` `assets.Rosen()` → `runSetupIndex` `ModeEmbedOnly` + `RunIndex` 1-concurrency); inject `progressWriter` → `EventsEmit("setup:progress", {phase,downloaded,total})`; sentinel `~/.tutor/.setup-complete`; guard network when `AUDIT_MODE`.

- Decisions from grilling: Svelte, keep both, blocking v1, background setup, ASCII dropped on desktop, Linux+Darwin+Windows v1, Svelte KaTeX, desktop-only `localStorage` capped ~100, both seams tested.

## Testing Decisions

- What makes a good test: verify behavior through public seams, not internals; expected values from independent literals (known `tp_001` `tp_002` boxed answers, golden prose), not recomputed; vertical slices (one test → one implementation).

- Which modules will be tested: `desktop.App.Ask` (blocking, citation ordering by `Similarity`, subdomain fallback `other`, `boxed` badge via `parse.Extract`) and HTTP `POST /v1/complete` (golden `evals/promptfooconfig.yaml` `json.content`, error `500` → red). No direct adapter tests at `chromem` or `embedder`.

- Prior art: `internal/tui/tui_test.go:70 TestModelFlow` + `121 TestStreamingAppend` (id guard, `loading`→turn, `▍` cursor, `scroll -1` pin), `internal/renderer/render_test.go` (Frac/Sqrt/Scripts `x²/log₂`/BigOps `∑∫`/Nest `⎛⎜⎝`/`boxed` `┌──┐`/`overline`, `TestStreamingTolerance` unclosed `\(\frac{2002}{`, `TestASCIIMode`) mirrored as frontend Vitest/Playwright visual equivalence (KaTeX `throwOnError:false`); `internal/parse/extract_test.go` for `\boxed` → `Answer`.

- Seams under test: external `App.Ask` primary, HTTP `POST /v1/complete` secondary (both confirmed by grilling Round 2).

## Out of Scope

- Phase 2 SSE streaming (`POST /v1/complete/stream`, `llm.Client.CompleteStream`, `answerMsg delta` incremental KaTeX), OpenStax PDF chunker (`internal/rag/openstax.go`, `ROADMAP.md:13`), corpus migration to `tantivy`/`qdrant`, Rust/Tauri path (ADR 0001), ASCII fallback on desktop, shared `~/.tutor/history.json` between TUI and desktop, Java/Go WASM inference, `wayfinder` map (way is clear), `triage` queue for this feature.

## Further Notes

- Source: this conversation + exploration reports (Manager/Retriever/Prompt/Renderer/Build), `CONTEXT.md` glossary, `docs/adr/0001`.
- Way is clear; prototype detour (`skills/skills/engineering/prototype/UI.md` 3 KaTeX variants toggleable via `?variant=`) is optional, not gating. If used, commit to `prototype/wails-ui` branch as primary source.
- After this spec: `to-tickets` tracer bullets with `Blocked by`, then loop `implement` per ticket driving `/tdd` at agreed seams + `/code-review` before commit. Issue tracker is local markdown `.scratch/wails-desktop/issues/`.
