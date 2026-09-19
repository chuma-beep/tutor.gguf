# Tutor.gguf

An offline math tutor for Nigerian CS undergraduates. It runs on-device. It uses RAG over
GSM8K / Hendrycks MATH / Rosen and Qwen2.5-Math-1.5B GGUF on llama.cpp. CPU-only. 7 GB budget.
100% offline at eval.

## Language

**Tutor**
The end-to-end product: retrieval, prompt build, generation and math rendering. Avoid "bot" or
"assistant" as a noun for the product.

**Managed RAG**
The `internal/runtime.Manager` + `internal/rag.Retriever` + `internal/llm.Client` +
`internal/prompt.Builder` stack. It turns a `Problem` string into `Content` + `Answer`. It also
owns the llama-server lifecycle (gen and embed), freePort and waitHealthy.

**Scored Chunk**
`internal/rag.ScoredChunk`. A corpus `Chunk` plus `ID` + `Similarity` after `QueryEmbedding`.
Do not confuse it with `Chunk` itself, which is pre-embedding and holds
`Text/Subdomain/Source/Level` from `chunker.go`.

**Chunk**
The corpus unit from `LoadHendrycksFile` / `LoadGSM8KFile` / `LoadRosenDir`. It holds
`Text/Subdomain/Source/Level`. Chunks are homogeneous and embedded with the `search_document`
prefix.

**Subdomain**
The fine label assigned by the chunker (`algebra`, `precalculus`, `arithmetic`, `geometry`,
`probability`, `number_theory`, `calculus` for GSM8K, `discrete_math` for Rosen). It drives the
chunk filter and the prompt instruction selector.

**Prompt Category**
A coarse bucket (`calculus`, `discrete_math`, `linear_algebra`, `geometry`, `other`). It is
derived from `Subdomain` through `prompt.PromptCategory` / `subdomainToPromptCategory`. It
selects the `subdomainInstructions` text.

**Tutor Home**
`TUTOR_HOME` or `~/.tutor`. It is the root for `models/`, `bin/llama-server`, `corpus/`, the
`chromem` DB (`DBPath`) and `logs/`. Env overrides: `TUTOR_LLAMA_SERVER`, `TUTOR_THREADS`,
`TUTOR_CTX`, `TUTOR_DB_PATH`.

**GGUF / Quantization**
The `Q4_K_M` model files `qwen2.5-math-1.5b-instruct-q4_k_m.gguf` (`GenModelFile`) and
`nomic-embed-text-v1.5.Q4_K_M.gguf` (`EmbedModelFile`). The runtime is `llama.cpp` only.

**Answer**
`parse.Extract(content)`. It tries boxed `\\boxed{}`, then `final answer:`, then `####`, then
the trimmed output. It is shown as a badge next to `Content`.

**Wails Desktop**
A native desktop shell (`cmd/desktop` + `internal/desktop.App`) that wraps Managed RAG through
direct Go bindings (no HTTP hop). The WebView frontend is Svelte + Vite + Tailwind + KaTeX. It
sits alongside the TUI (`tutor chat` through `internal/tui`): dual frontend, shared backend and
shared `chromem` DB. Setup runs in the background (`EventsEmit("setup:progress")`). v1 uses
blocking `Complete` plus a spinner. Streaming SSE is Phase 2. Distribution is Linux
`deb/AppImage`, `darwin/universal` and `windows/amd64` WebView2 through
`wails build -tags desktop`.

**TUI**
The Bubble Tea shell `internal/tui.Model` with `transcriptLines`/`transcriptView` and `askCmd`
(`AnswerMsg{id,delta,done,err}`). It renders terminal art through `renderer.Render` (stacked
frac/root, `glyph` superscripts). Kept for headless/SSH and the dev loop. `ascii=true` is
TUI-only; desktop uses KaTeX only. TUI history is in-memory. Desktop history is `localStorage`
capped at ~100 with `Ctrl+L` to clear.

## Relationships

- Managed RAG holds a Retriever (which needs an Embedder, a Subdomain Classifier and a chromem
  Collection) and a Prompt Builder plus an LLM Client.
- TUI and Wails Desktop are two frontends over the same Managed RAG. Neither owns the other.
- Prompt Category is derived from Subdomain. Subdomain is assigned by the Chunker.
- Tutor Home contains DBPath, LogsDir, ModelsDir, BinDir and CorpusDir.
