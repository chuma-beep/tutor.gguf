# Tutor.gguf — Roadmap

Most of the original build plan is done. This file tracks what is left.

## Current state

| Component | Status |
|-----------|--------|
| RAG pipeline (`internal/rag`, `internal/llm`, `internal/prompt`) | Working. Index, retrieve, prompt and generate, served on `:8082` |
| `cmd/tutor` | Working. Ingestion and out-of-band query CLI |
| `cmd/serve` | Working. `/v1/complete` RAG endpoint |
| LaTeX → AST parser (`internal/renderer/`) | Done. Spans, fractions, scripts, sqrt, big-ops, `\left\right`, binom, decorations, degrade-to-passthrough |
| Terminal renderer (`internal/renderer/layout.go`) | Done. Stacked frac/root/limits boxes, Unicode glyphs and ASCII fallback (`make tui-ascii`) |
| Bubble Tea TUI (`internal/tui/`, `cmd/tui`) | Done. Input, streaming transcript, spinner, scroll, error turns |
| OpenStax PDF chunker (`internal/rag/openstax.go`) | Working. `LoadOpenStaxPDF` + `LoadOpenStaxDir`, wired into `RunIndex` |

---

## Remaining work

### 1. OpenStax PDF chunker — done

`internal/rag/openstax.go` extracts text with `github.com/ledongthuc/pdf`
(positioned `Content()` glyphs, so `Td`/`Tm` layouts keep line breaks), splits
on chapter/section headings and returns `[]Chunk` with `Source: "openstax"`.
`LoadOpenStaxDir` infers `Subdomain` from the file name (`calculus`, `algebra`,
etc). Wired into `RunIndex` via `-openstax-dir` and into `tutor setup` through
`runtime.OpenStaxDir()` (skipped when the directory is absent — PDFs are CC BY
and placed manually, there is no downloader yet).

### 2. Narrated demo video (optional polish)

`docs/tutor-gguf-demo.mp4` is a 104 s silent v1. Re-cut it with voiceover. The script is in
`docs/media-kit.md`.

### 3. Official audit run

The local Docker audit profile already reproduces `submission.json` and `audit.json`
(`adtc-profiler compare` → PASS). Run the official audit on the Standard Laptop when it is
available.
