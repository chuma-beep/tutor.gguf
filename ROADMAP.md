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
| OpenStax PDF chunker (`internal/rag/openstax.go`) | Not started |

---

## Remaining work

### 1. OpenStax PDF chunker

Create `internal/rag/openstax.go`:

- Extract text from the OpenStax PDFs with a Go PDF library (for example
  `github.com/ledongthuc/pdf`).
- Split the text into sections by chapter and section headings.
- Return `[]Chunk` with `Source: "openstax"` and a matching `Subdomain` (for example
  `"calculus"` or `"algebra"`).

```go
package rag

func LoadOpenStaxPDF(filePath string, subdomain string) ([]Chunk, error)
```

Wire it into `RunIndex` next to the Hendrycks, GSM8K and Rosen loaders.

### 2. Narrated demo video (optional polish)

`docs/tutor-gguf-demo.mp4` is a 104 s silent v1. Re-cut it with voiceover. The script is in
`docs/media-kit.md`.

### 3. Official audit run

The local Docker audit profile already reproduces `submission.json` and `audit.json`
(`adtc-profiler compare` → PASS). Run the official audit on the Standard Laptop when it is
available.
