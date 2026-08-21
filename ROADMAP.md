# Tutor.gguf — RAG Pipeline Roadmap

## Current State

| Component | Status |
|-----------|--------|
| RAG pipeline (`internal/rag`, `internal/llm`, `internal/prompt`) | Working — index / retrieve / prompt / generate, served on `:8082` |
| `cmd/serve` | Working — `/v1/complete` RAG endpoint |
| `cmd/tutor` | Working — ingestion + out-of-band query CLI |
| LaTeX → AST parser (`internal/renderer/`) | Done — spans, fractions, scripts, sqrt, big-ops, `\left\right`, binom, decorations, degrade-to-passthrough |
| Terminal renderer (`internal/renderer/layout.go`) | Done — stacked frac/root/limits boxes, Unicode glyphs + ASCII fallback (`make tui-ascii`) |
| Bubble Tea TUI (`internal/tui/`, `cmd/tui`) | Done — input, streaming transcript, spinner, scroll, error turns |
| OpenStax PDF chunker (`internal/rag/openstax.go`) | Not started |

---

## Step 1 — Add Dependencies

Update `go.mod` and add:

- `chromem-go` — in-memory/persistent vector database with cosine similarity search
- `nomic-embed-text` — local embedding model (or use llama.cpp's embedding endpoint)
- `bubbletea` — TUI framework for interactive terminal UI
- `chroma` — for PDF parsing (OpenStax corpus)

Run `go mod tidy` after adding imports.

---

## Step 2 — Build a RAG Indexer

Create a command (e.g., `cmd/indexer/main.go`) that:

1. Walks `data/raw/hendrycks_math/train/` and calls `LoadHendrycksFile` for each JSON file
2. Walks `data/raw/gsm8k/` and calls `LoadGSM8KFile` for each JSONL file
3. For each chunk, generates an embedding vector (via llama.cpp embedding endpoint or a local embedding model like nomic-embed-text)
4. Stores chunks + embeddings in a `chromem-go` collection
5. Persists the collection to `data/index/`

**File:** `cmd/indexer/main.go`

---

## Step 3 — Build a Retriever

Create `internal/rag/retriever.go`:

- Takes a user query string
- Generates an embedding for the query (same model used during indexing)
- Searches the chromem-go collection for the top-k most similar chunks
- Returns the matching `Chunk` structs with similarity scores

```go
package rag

type Retriever struct {
    collection *chromem.Collection
}

func NewRetriever(collection *chromem.Collection) *Retriever

func (r *Retriever) Retrieve(query string, topK int) ([]Chunk, error)
```

---

## Step 4 — Build PromptBuilder

Create `internal/prompt/builder.go`:

- Takes the user's query and a list of retrieved `Chunk` structs
- Formats them into a structured prompt with system instructions, retrieved context, and the user's question
- The system instruction should tell the model to act as a step-by-step math tutor
- Retrieved chunks are inserted as context (with subdomain/source tags for provenance)

```go
package prompt

type Builder struct{}

func NewBuilder() *Builder

func (b *Builder) Build(query string, chunks []rag.Chunk) string
```

**Prompt template structure:**
```
You are a step-by-step math tutor. Use the following context to answer the user's question.

Context:
[chunk 1 text] (Source: hendrycks_math, Topic: algebra)
[chunk 2 text] (Source: gsm8k, Topic: calculus)

Question: [user query]

Provide a clear, step-by-step solution.
```

---

## Step 5 — Build ResponseParser

Create `internal/parser/parser.go`:

- Takes the raw LLM response string
- Extracts the step-by-step reasoning (split on numbered steps, bullet points, or "Step N:" patterns)
- Returns a structured `Response` with individual steps

```go
package parser

type Step struct {
    Number  int
    Content string
}

type Response struct {
    Steps      []Step
    FinalAnswer string
}

func Parse(raw string) Response
```

---

## Step 6 — Build LaTeX→ASCII Renderer

Create `internal/renderer/renderer.go`:

- Takes a string containing LaTeX math expressions (inline `$...$` and display `$$...$$`)
- Converts common LaTeX patterns to ASCII approximations:
  - `\frac{a}{b}` → `a/b`
  - `\sqrt{x}` → `sqrt(x)`
  - `\int_{a}^{b}` → `∫[a..b]`
  - `\sum`, `\prod`, `\lim`, etc.
  - Greek letters: `\alpha` → `α`, `\beta` → `β`, etc.
  - `x^2` → `x²` (using superscript Unicode)
  - `x_n` → `xₙ` (using subscript Unicode)
- Returns a plain-text string suitable for terminal display

```go
package renderer

func RenderMath(input string) string
```

---

## Step 7 — Add Chunkers for OpenStax and Rosen

### OpenStax PDF Chunker

Create `internal/rag/openstax.go`:

- Uses a PDF text extraction library (e.g., `github.com/ledongthuc/pdf` or `github.com/unidoc/unipdf`) to extract text from OpenStax PDFs
- Splits the extracted text into sections (by chapter/section headers)
- Returns a `[]Chunk` with `Source: "openstax"` and appropriate `Subdomain` (e.g., `"calculus"`, `"algebra"`)

```go
package rag

func LoadOpenStaxPDF(filePath string, subdomain string) ([]Chunk, error)
```

### Rosen Discrete Math Chunker

Create `internal/rag/rosen.go`:

- Walks the `data/raw/rosen/book/` directory
- Reads the Markdown/TeX solution files
- Splits by chapter/section
- Returns `[]Chunk` with `Source: "rosen_discrete_math"` and `Subdomain: "discrete_math"`

```go
package rag

func LoadRosenDir(rootDir string) ([]Chunk, error)
```

---

## Step 8 — Wire Everything in main.go

Update `cmd/tutor/main.go` to:

1. On first run (or when `data/index/` is empty), run the indexing pipeline
2. On each user query:
   - Embed the query
   - Retrieve top-k chunks from the vector index
   - Build a prompt with context via PromptBuilder
   - Send to llama.cpp via `llm.Client`
   - Parse the response via ResponseParser
   - Render the output via the renderer
3. Run an interactive TUI loop (bubbletea) for continuous Q&A

---

## Step 9 — Add a TUI

Create `internal/tui/tui.go` using bubbletea:

- Text input area for the user's math question
- Display area for the assistant's step-by-step response
- Support for multi-turn conversation (history)
- Keyboard shortcuts (Ctrl+C to quit, Enter to submit)

---

## Summary

| Step | File(s) | What it does |
|------|---------|-------------|
| 1 | `go.mod`, `go.sum` | Add chromem-go, nomic-embed-text, bubbletea |
| 2 | `cmd/indexer/main.go` | Load corpus → embed → persist vector index |
| 3 | `internal/rag/retriever.go` | Query → embed → search → return chunks |
| 4 | `internal/prompt/builder.go` | Format query + chunks into a prompt |
| 5 | `internal/parser/parser.go` | Parse LLM response into steps |
| 6 | `internal/renderer/renderer.go` | Convert LaTeX to ASCII for terminal |
| 7 | `internal/rag/openstax.go`, `internal/rag/rosen.go` | Chunkers for PDFs and Rosen |
| 8 | `cmd/tutor/main.go` | Wire everything together |
| 9 | `internal/tui/tui.go` | Interactive bubbletea TUI |