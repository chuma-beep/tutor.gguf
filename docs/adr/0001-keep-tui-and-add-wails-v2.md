# ADR 0001: Keep TUI and add Wails v2 desktop

Date: 2026-08-29
Status: Accepted

## Context

Tutor.gguf is a Go monolith (`internal/rag`, `internal/llm`, `internal/prompt`,
`internal/runtime.Manager`, `internal/renderer`, `internal/tui`). We needed a desktop app. The
candidates were Tauri, Wails and Fyne. We also had to decide whether to keep the Bubble Tea TUI
(`tutor chat`, `internal/tui/tui.go:23` + `internal/renderer/render.go:16`).

ADTC constraints: llama.cpp + GGUF only, 100% offline at eval, 8 GB / 7 GB managed, no GPU,
peak 1.1 GB, `model/` and `data/` git-ignored, `metadata.json` with exactly 2 prompts and a
headless `bin/tutor` that stays evaluatable.

## Decision

- Use **Wails v2** (Go's Tauri) for desktop. Do not use Tauri (Rust) or Fyne/Gio.
- **Keep both frontends.** The `tutor chat` TUI stays. The new `tutor-desktop` Wails binary is
  additive.

## Consequences

- Wails binds directly to Go (`retriever.Retrieve`, `llm.Complete`, `Manager.Start`, `DBPath`).
  There is no sidecar HTTP hop. We still keep `POST /v1/complete` (`serve.go:50`,
  `evals/promptfooconfig.yaml:5`) for `make eval`/`profile` parity. Decision: keep both seams.
- The frontend is Svelte + Vite + Tailwind + KaTeX (`frontend/dist` through `//go:embed`,
  `base:'./'`). The bundle is 5–10 MB plus the webview. KaTeX handles math and replaces the
  terminal stacked fractions/roots; `\boxed` is native.
- The TUI stays for SSH, headless and the dev loop (`make tui`). The webview serves students who
  need KaTeX. Both share the `~/.tutor` DB (`TUTOR_HOME`) and read the same `chromem` store.
- The setup wizard is background and non-blocking. Chat is enabled when `indexedChunkCount>0`
  and models exist. Otherwise it emits `EventsEmit` progress across 8 phases with
  `gen.log`/`embed.log`. v1 uses blocking `Complete` plus a spinner (`tui.go:206`). Streaming
  SSE is Phase 2.
- Build: headless `bin/tutor` stays `CGO_ENABLED=0`. The desktop build uses `CGO_ENABLED=1`
  with `wails build -tags desktop -platform linux/amd64,windows/amd64,darwin/universal` and
  links `webkit2gtk` on Linux and `WebView2` on Windows. CI adds a parallel `desktop` job that
  shares the same `SHA256SUMS` release.
- Settings show `TUTOR_THREADS`/`TUTOR_CTX` only. `TUTOR_HOME` sits behind Advanced. History is
  desktop-only `localStorage` capped at ~100. Citations (`ScoredChunk` Sources `[1..3]` plus
  the `Prompt Category` pill and prompt viewer) are in scope.
- No ADR conflict. Tradeoff: the webview dependency is about 150 MB at runtime. That stays
  within the 7 GB headroom and the offline setup wizard gates network access.

## Alternatives considered

- Tauri: sidecar complexity, a Rust toolchain and the same webview. Rejected because Go already
  owns RAG.
- Fyne/Gio: pure Go widgets, no KaTeX, manual math canvas. Strictly worse for LaTeX.
