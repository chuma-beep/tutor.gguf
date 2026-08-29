# Tutor.gguf

Offline, on-device math tutor for Nigerian CS undergraduates. RAG over GSM8K / Hendrycks MATH / Rosen over Qwen2.5-Math-1.5B GGUF on llama.cpp, CPU-only, 7 GB budget, 100% offline at eval.

## Language

**Tutor**:
The end-to-end product: retrieval + prompt build + generation + math rendering. Avoid: bot, assistant as noun for the product.

**Managed RAG**:
The `internal/runtime.Manager` + `internal/rag.Retriever` + `internal/llm.Client` + `internal/prompt.Builder` stack that turns a `Problem` string into `Content` + `Answer`. Includes llama-server lifecycle (gen + embed), freePort, waitHealthy.

**Scored Chunk**:
`internal/rag.ScoredChunk` — a corpus `Chunk` plus `ID` + `Similarity` after `QueryEmbedding`. Not to be confused with `Chunk` itself (pre-embedding, holds `Text/Subdomain/Source/Level` from `chunker.go`).

**Chunk**:
Corpus unit from `LoadHendrycksFile` / `LoadGSM8KFile` / `LoadRosenDir` with `Text/Subdomain/Source/Level`. Homogeneous, embedded with `search_document` prefix.

**Subdomain**:
Fine label assigned by chunker (`algebra`, `precalculus`, `arithmetic`, `geometry`, `probability`, `number_theory`, `calculus` for GSM8K, `discrete_math` for Rosen). Chunk-filter + prompt instruction selector.

**Prompt Category**:
Coarse bucket (`calculus`, `discrete_math`, `linear_algebra`, `geometry`, `other`) derived from `Subdomain` via `prompt.PromptCategory` / `subdomainToPromptCategory`. Selects `subdomainInstructions` text.

**Tutor Home**:
`TUTOR_HOME` env or `~/.tutor` — root for `models/`, `bin/llama-server`, `corpus/`, `chromem` DB (`DBPath`), `logs/`. Env overrides: `TUTOR_LLAMA_SERVER`, `TUTOR_THREADS`, `TUTOR_CTX`, `TUTOR_DB_PATH`.

**GGUF / Quantization**:
`Q4_K_M` quantized model file `qwen2.5-math-1.5b-instruct-q4_k_m.gguf` (`GenModelFile`) and `nomic-embed-text-v1.5.Q4_K_M.gguf` (`EmbedModelFile`). Runtime is `llama.cpp` only.

**Answer**:
`parse.Extract(content)` — boxed `\\boxed{}` → `final answer:` → `####` → trimmed output. Shown as badge alongside `Content`.

**Wails Desktop**:
Native desktop shell (`cmd/desktop` + `internal/desktop.App`) wrapping Managed RAG via direct Go bindings (no HTTP hop), WebView frontend (Svelte + Vite + Tailwind + KaTeX). Keeps TUI (`tutor chat` via `internal/tui`) alongside — dual frontend, shared backend and shared `chromem` DB. Setup runs background non-blocking (`EventsEmit("setup:progress")`); v1 is blocking `Complete` + spinner, streaming SSE is Phase 2; distribution is Linux `deb/AppImage` + `darwin/universal` + `windows/amd64` WebView2 via `wails build -tags desktop`.

**TUI**:
Bubble Tea shell `internal/tui.Model` with `transcriptLines`/`transcriptView`, `askCmd` (`AnswerMsg{id,delta,done,err}`), `renderer.Render` terminal art (stacked frac/root, `glyph` superscripts). Kept for headless/SSH and dev loop; `ascii=true` stays TUI-only, desktop uses KaTeX only; history stays TUI in-memory, desktop history is `localStorage` capped ~100 with `Ctrl+L` clear.

## Relationships

- Managed RAG holds a Retriever (needs Embedder + Subdomain Classifier + chromem Collection) and a Prompt Builder + LLM Client
- TUI and Wails Desktop are two frontends over the same Managed RAG; neither owns the other
- Prompt Category is derived from Subdomain; Subdomain is assigned by Chunker
- Tutor Home contains DBPath, LogsDir, ModelsDir, BinDir, CorpusDir
