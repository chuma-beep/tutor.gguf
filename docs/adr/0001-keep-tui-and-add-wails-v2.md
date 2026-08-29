# ADR 0001: Keep TUI and add Wails v2 desktop

Date: 2026-08-29
Status: Accepted

## Context

Tutor.gguf is a Go monolith (`internal/rag`, `internal/llm`, `internal/prompt`, `internal/runtime.Manager`, `internal/renderer`, `internal/tui`). The question was Tauri vs Wails vs Fyne for a desktop app, and whether to keep the Bubble Tea TUI (`tutor chat`, `internal/tui/tui.go:23` + `internal/renderer/render.go:16`).

ADTC constraints: llama.cpp + GGUF only, 100% offline at eval, 8 GB / 7 GB managed, no GPU, peak 1.1 GB, `model/` and `data/` git-ignored, `metadata.json` exactly 2 prompts, headless `bin/tutor` remains evaluatable.

## Decision

- Use **Wails v2** (Go's Tauri) for desktop, not Tauri (Rust) nor Fyne/Gio.
- **Keep both frontends**: `tutor chat` TUI stays; new `tutor-desktop` Wails binary is additive.

## Consequences

- Wails binds directly to Go (`retriever.Retrieve`, `llm.Complete`, `Manager.Start`, `DBPath`) — no sidecar HTTP hop; **but** `POST /v1/complete` (`serve.go:50`, `evals/promptfooconfig.yaml:5`) is kept for `make eval`/`profile` parity (decision: keep both seams). Frontend is Svelte + Vite + Tailwind + KaTeX (`frontend/dist` via `//go:embed`, `base:'./'`). Bundle 5-10 MB + webview, KaTeX for math (replaces terminal stacked fractions/roots, `\boxed` native).
- TUI stays for SSH/headless/dev loop (`make tui`), webview for students needing KaTeX. Shared `~/.tutor` DB (`TUTOR_HOME`) so both read same `chromem` store. Setup wizard is **background non-blocking** (chat enabled if `indexedChunkCount>0` + models exist; else progress `gen.log`/`embed.log` + 8 phases in `EventsEmit`); v1 is blocking `Complete` + spinner (`tui.go:206`), streaming SSE is Phase 2.
- Build: headless `bin/tutor` stays `CGO_ENABLED=0`; desktop `wails build -tags desktop -platform linux/amd64,windows/amd64,darwin/universal` is `CGO_ENABLED=1` + `webkit2gtk` Linux / `WebView2` Windows. CI adds parallel `desktop` job, same `SHA256SUMS` release. Settings show `TUTOR_THREADS`/`TUTOR_CTX` only, `TUTOR_HOME` behind Advanced. History is desktop-only `localStorage` capped ~100. Citations (`ScoredChunk` Sources `[1..3]` + `Prompt Category` pill + prompt viewer) are now in scope.
- No ADR conflict. Tradeoff: webview dependency (~150 MB at runtime, within 7 GB headroom); offline setup wizard gates network.

## Alternatives considered

- Tauri: sidecar complexity, Rust toolchain, same webview — rejected because Go already owns RAG.
- Fyne/Gio: pure Go widgets, no KaTeX, manual math canvas — strictly worse for LaTeX.
