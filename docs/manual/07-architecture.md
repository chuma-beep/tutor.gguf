---
title: Architecture
order: 7
---

# Architecture

One request flows through five stages:

```
problem → classify subdomain → embed query → retrieve top-K chunks
  → build RAG prompt → llama-server (Qwen2.5-Math) → content + parsed answer
```

## Components

1. **HTTP layer** (`internal/cli/serve.go`) — decodes the request, orchestrates retrieval, prompt and generation, encodes JSON. Supervises llama-servers in managed mode.
2. **Retriever** (`internal/rag`) — a keyword classifier sorts each question (algebra, calculus, discrete math, geometry, probability, number theory), embeds it with the `search_query` prefix and queries chromem-go directly by embedding (top-K, default 3, with unfiltered fallback when a subdomain slice is thin).
3. **Prompt builder** (`internal/prompt`) — ChatML framing, coarse-category chain-of-thought instructions, RAG context block, answer anchor. The coarse category derives from the fine subdomain label.
4. **LLM client** (`internal/llm`) — posts to llama.cpp `/completion` (blocking) or streams token-by-token for SSE.
5. **Answer parser** (`internal/parse`) — `\boxed{}` → `final answer:` → `####` → whole-output fallback.

## Frontends

The TUI (`internal/tui`, Bubble Tea) and the Wails desktop / browser UI (Svelte, KaTeX) are two shells over the same stack — neither owns the other. The TUI renders math as terminal art (Unicode with ASCII fallback); the web frontends render KaTeX. History is in-memory in the TUI and `localStorage` (100 turns) on the web.

## Evaluation

promptfoo runs accuracy (30 sampled Hendrycks cases, deterministic matcher plus local LLM rubric judge) and quality (10 bespoke cases) suites against a local server. Current self-reported status: 18/30 accuracy, 6/10 quality — see `REPORT.md` for failure analysis. Measured audit-profile performance: ~1.1 GB peak RSS, ~13–14 tok/s CPU-only.
