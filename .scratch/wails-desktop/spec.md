# Spec: Wails Desktop + Keep TUI

Status: verified (2026-09-09 full pass — see issues/01-05)

## Problem Statement

Nigerian CS undergraduates at distance-learning institutions need step-by-step math help
(Discrete Mathematics, Calculus I/II, Linear Algebra). But Tutor.gguf is TUI-only
(`internal/tui/tui.go:23` + `internal/renderer/render.go:16` terminal art). LaTeX (`\frac`,
`\sqrt`, `\boxed{}`, `\binom`, `\int`) degrades to stacked `─`/`|` ASCII boxes. Streaming is
blocking-only (`llm/client.go:36` + `tui.go:173` spinner). Setup (`internal/cli/setup.go:49`)
runs 8 phases over ~1.2 GB (gen 1.1 GB, embed 90 MB, llama-server b10612, GSM8K, Hendrycks 7
configs, Rosen, chromem index) and blocks.

The ADTC 2026 Standard Laptop is Ubuntu 22.04 with 8 GB / 7 GB managed, integrated GPU only
and `llama.cpp + GGUF` only (`metadata.json: runtime=llama.cpp`). It must run 100% offline at
eval. Any desktop must preserve this and keep `bin/tutor` evaluatable (`make profile` PASS).

## Solution

Add a Svelte + Vite + Tailwind + KaTeX WebView desktop app, `tutor-desktop` (Wails v2). It is
a **second frontend** over the same Managed RAG (`internal/runtime.Manager` +
`retriever.Retrieve` + `prompt.Builder.Build` + `llm.Client.Complete` + `parse.Extract`).

Keep the `tutor chat` TUI intact (`cmd/tutor/main.go:34`) for SSH, headless and the dev loop.
Both frontends share `Tutor Home` (`~/.tutor`, `runtime/paths.go:22`) models, the `chromem` DB
(`DBPath`) and `~/.tutor/logs`.

Desktop uses direct Go bindings (`desktop.App.Ask()` → `retriever.Retrieve` → `BuildPrompt` →
`Complete` → `Extract`), so there is no HTTP hop. `POST /v1/complete`
(`internal/cli/serve.go:50`) stays for `evals/promptfooconfig.yaml` and `adtc-profiler`.

Desktop SetupView runs in the background and emits `EventsEmit("setup:progress")`. v1 ships
blocking generation with a spinner. Streaming SSE is Phase 2. Distribution is Linux
`deb/AppImage`, `darwin/universal` and `windows/amd64` WebView2 through
`wails build -tags desktop`.

## User Stories

1. As a Nigerian undergrad, I want to ask "Find the derivative of x^2" in a native window and
   see KaTeX fractions, square roots and `\boxed{}` so math is readable on first try.
2. As a distance-learning student on unstable power, I want first launch to download llama.cpp
   b10612, Qwen2.5-Math-1.5B-Q4_K_M, nomic-embed-text-v1.5-Q4_K_M, GSM8K, Hendrycks MATH 7
   configs and Rosen, then build the index in the background, so I can chat immediately when
   `DBPath` already has chunks and models exist.
3. As a student, I want everything after setup to work 100% offline with zero outbound calls
   (`COMPLIANCE.md:23`), so `adtc-profiler run --mode audit` still PASS and peak RSS 1.1 GB
   stays within 7 GB.
4. As a TUI user on a lab with no desktop env, I want `tutor chat` (`make tui` / `make
   tui-ascii`, `internal/tui/tui_test.go`) to still work unchanged over SSH, so low-end labs
   are not forced into WebView.
5. As a student, I want citations: a `Prompt Category` pill with `subdomainInstructions` text,
   collapsible Sources `[1..3]` (`ScoredChunk{Text,Subdomain,Similarity}`) and a "View prompt
   sent to Qwen" debug panel (`cli/index.go:143`), so retrieval is transparent.
6. As a student, I want desktop-only history in `localStorage` with Clear (`Ctrl+L` parity
   `tui.go:103`), `Esc`/`Ctrl+C` quit, `PgUp/PgDn` scroll with pinned `-1` overscan
   (`tui.go:300` transcriptOverscan), plus a "thinking" spinner and `▍` cursor while loading
   (`tui.go:281`), so the UX feels familiar.
7. As an evaluator, I want `POST /v1/complete` (`evals/promptfooconfig.yaml:5`
   `transformResponse: json.content`, `make eval` `maxConcurrency:1`) and `download_model.sh`
   (`model/*.gguf`) to still work against `localhost:8082`, so ADTC scoring
   (`S_total = 0.5 S_acc + 0.3 S_perf + 0.2 S_eff`) is unchanged.
8. As a maintainer, I want Settings to show `TUTOR_THREADS` (0=auto, 4 optimal per
   `docs/tuning.md`) and `TUTOR_CTX` (2048). Advanced values (`TUTOR_HOME`,
   `TUTOR_LLAMA_SERVER`, `TUTOR_DB_PATH`, `LlamaServerPath`) sit behind "Open Config Folder",
   so tuning is not lost.
9. As a release manager, I want `make build` (`CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"
   -o bin/tutor ./cmd/tutor`) to still emit `tutor-linux-amd64` and `wails build -tags desktop
   -platform linux/amd64,windows/amd64,darwin/universal` (`CGO_ENABLED=1` +
   `webkit2gtk-4.0`) to emit `tutor-desktop-*` with `SHA256SUMS` merged in `release.yml`, so
   both ship.
10. As a new user, I want the Svelte frontend to render `\( \)` / `\[ \]` / `$` / `$$` spans
    through `extractSpans` precedence plus unclosed streaming tolerance. `\boxed{16}` is boxed
    natively by KaTeX (`throwOnError:false`), so model output is never dropped.

## Implementation Decisions

- **Modules.** New `frontend/` (Svelte + Vite + TS, `base:'./'`). New `wails.json`
  (`name:tutor-gguf`, `frontend:dir=frontend`, `main:cmd/desktop/main.go`,
  `outputfilename:tutor-desktop`). New `cmd/desktop/main.go` (`wails.Run` with
  `OnStartup`/`OnShutdown`). New `internal/desktop/app.go`, a deep module
  `App{ctx,mgr,db,collection,retriever,genClient}`. Small edits: add `GET /health` (and CORS
  for the `wails dev` proxy) to `internal/cli/serve.go`, export `ResolveDBPath` from
  `internal/cli/managed.go`, add `build-desktop`/`dev-desktop` Makefile targets and ignore
  `frontend/dist/`, `frontend/node_modules/` and `build/bin/`. Leave `cmd/tutor/main.go`,
  `internal/tui/*`, `internal/renderer/*` and the module path unchanged.

- **Interfaces.** External `App.Ask(ctx, problem string) → {content, answer, subdomain,
  category, chunks []ScoredChunk, prompt string}`. HTTP `requestBody{Problem,MaxTokens,
  Temperature}` / `responseBody{Content,Answer}` (`serve.go:21`). `App.Ask` is small and deep.
  It hides the Manager lifecycle, the embedding prefixes (`search_query` vs `search_document`),
  the `topK*4→topN 3` dedup, the `ChatML` prompt framing and the `parse.Extract` regex chain
  (`boxedRE→markerRE→gsm8kRE`). Internal `retriever.Retrieve(ctx, query)` and
  `prompt.Builder.Build(query, []Source, subdomain)` stay behind `Ask` and are not exposed
  to JS.

- **Seams.** The highest seam is `App.Ask` (primary). HTTP `POST /v1/complete` is the secondary
  seam for eval parity. Do not add a `chromem.Collection` adapter or call
  `embedder.EmbedDocument` directly. One deep seam, not shallow per-file interfaces.

- **Architecture.** `App.Startup(ctx)` → `startManaged(ctx)` checks `DiscoverLlamaServer`
  precedence (`TUTOR_LLAMA_SERVER > ~/.tutor/bin/llama-server > $PATH`) and confirms
  `GenModelPath`/`EmbedModelPath` exist. It then builds
  `runtime.New(Config{GenModel,EmbedModel,Threads:envInt("TUTOR_THREADS",0),
  Ctx:envInt("TUTOR_CTX",2048), LogDir:LogsDir, Mode:ModeBoth})`, calls `mgr.Start(ctx)`,
  polls `GET /health` (300ms, 5m timeout), captures `gen.log`/`embed.log` and runs
  `go func(){<-ctx.Done(); mgr.Stop()}`. Next it opens
  `chromem.NewPersistentDB(DBPath,false)`, calls
  `GetOrCreateCollection("tutor-corpus", nil, EmbedDocument)`, then builds `NewRetriever` and
  `NewClient(genURL)`. `App.Shutdown` calls `mgr.Stop()` and is idempotent. `freePort` uses
  `net.Listen 127.0.0.1:0` and is dynamic, not hardcoded `8080/8081`. `ResolveDBPath`
  precedence is `flag > TUTOR_DB_PATH > ./data/chromem (if dir or go.mod) > ~/.tutor/chromem`.
  The setup wizard `Setup(ctx, force, skipModels, skipCorpus)` delegates to the 8 phases in
  `setup.go`: dirs → `ensureLlamaServer` (asset `ubuntu-x64` etc through `llamaAssetSuffix`)
  → `fetch.EnsureFile` (`.part` + `ExtractTarGz/Zip` normalize) → `ensureGGUF` gen/embed →
  `ensureGSM8K` → `ensureHendrycks` (paginated `datasets-server`) → `ensureRosen`
  (`assets.Rosen()`) → `runSetupIndex` (`ModeEmbedOnly` + `RunIndex` at 1 concurrency). It
  injects a `progressWriter` that emits `EventsEmit("setup:progress", {phase,downloaded,total})`
  and writes the sentinel `~/.tutor/.setup-complete`. It guards network access when
  `AUDIT_MODE` is set.

- **Decisions from grilling.** Use Svelte. Keep both frontends. Blocking v1. Background setup.
  Drop ASCII on desktop. Ship Linux, Darwin and Windows in v1. Use Svelte KaTeX. Keep
  desktop-only `localStorage` capped at ~100. Test both seams.

## Testing Decisions

- **What makes a good test.** Verify behavior through public seams, not internals. Expected
  values come from independent literals (known `tp_001`/`tp_002` boxed answers, golden prose),
  not recomputed ones. Prefer vertical slices: one test to one implementation.

- **Which modules are tested.** `desktop.App.Ask` (blocking, citation ordering by `Similarity`,
  subdomain fallback `other`, `boxed` badge through `parse.Extract`). HTTP `POST /v1/complete`
  (golden `evals/promptfooconfig.yaml` `json.content`, error `500` → red). No direct adapter
  tests at `chromem` or `embedder`.

- **Prior art.** `internal/tui/tui_test.go:70 TestModelFlow` and `121 TestStreamingAppend`
  (id guard, `loading`→turn, `▍` cursor, `scroll -1` pin). `internal/renderer/render_test.go`
  (Frac/Sqrt/Scripts `x²/log₂`/BigOps `∑∫`/Nest `⎛⎜⎝`/`boxed` `┌──┐`/`overline`,
  `TestStreamingTolerance` unclosed `\(\frac{2002}{`, `TestASCIIMode`). These are mirrored as
  frontend Vitest/Playwright visual equivalence (KaTeX `throwOnError:false`).
  `internal/parse/extract_test.go` covers `\boxed` → `Answer`.

- **Seams under test.** `App.Ask` is primary. HTTP `POST /v1/complete` is secondary. Both were
  confirmed in grilling Round 2.

## Out of Scope

- Phase 2 SSE streaming (`POST /v1/complete/stream`, `llm.Client.CompleteStream`, `answerMsg`
  delta incremental KaTeX)
- OpenStax PDF chunker (`internal/rag/openstax.go`, `ROADMAP.md:13`)
- Corpus migration to `tantivy`/`qdrant`
- Rust/Tauri path (ADR 0001)
- ASCII fallback on desktop
- Shared `~/.tutor/history.json` between TUI and desktop
- Java/Go WASM inference
- `wayfinder` map (the way is clear)
- `triage` queue for this feature

## Further Notes

- Source: this conversation plus exploration reports (Manager/Retriever/Prompt/Renderer/Build),
  the `CONTEXT.md` glossary and `docs/adr/0001`.
- The way is clear. The prototype detour (`skills/skills/engineering/prototype/UI.md`, 3 KaTeX
  variants toggleable through `?variant=`) is optional and not gating. If used, commit it to a
  `prototype/wails-ui` branch as the primary source.
- After this spec: `to-tickets` tracer bullets with `Blocked by`, then loop `implement` per
  ticket driving `/tdd` at the agreed seams plus `/code-review` before commit. The issue
  tracker is local markdown at `.scratch/wails-desktop/issues/`.
