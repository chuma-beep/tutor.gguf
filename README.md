# Tutor.gguf — On-device Math Tutor

A fully offline, on-device math tutor for Nigerian CS undergraduates at distance-learning
institutions. Given a math problem, it retrieves relevant worked examples from a local corpus
(RAG) and generates a step-by-step solution — no GPU, no internet, no cloud API fees.

> **Team:** chuma-beep · **Domain:** math_scientific_reasoning · **Language:** en
> **Claims:** African Alpha use case, budget-laptop compatible

## What it does

- **Model:** Qwen2.5-Math-1.5B-Instruct (GGUF Q4_K_M) generates step-by-step solutions with
  LaTeX `\boxed{}` final answers.
- **Retrieval:** nomic-embed-text-v1.5 embeddings + chromem-go vector store + a lightweight
  keyword subdomain classifier (algebra / calculus / discrete math / geometry / probability /
  number theory) that filters retrieved context and selects domain-specific prompt instructions.
- **Corpus:** GSM8K, Hendrycks MATH, Rosen Discrete Math solutions, and OpenStax textbooks.
- **Runtime:** 100% local llama.cpp (`llama-server`) — CPU-only, no GPU required.

Covers Discrete Mathematics, Calculus I/II, Linear Algebra, and Geometry-style problems in the
style Nigerian students meet in JAMB/WASSCE and first-year CS courses.

## Requirements

Target platform is the **ADTC 2026 Standard Laptop**:

| Constraint | Spec |
|---|---|
| CPU | Intel Core i5 10th–12th gen or AMD Ryzen 5 3000–5000 (x86-64) |
| RAM | 8 GB DDR4 — **max 7 GB managed working set** (hard limit) |
| Graphics | Integrated only (Intel UHD / Iris Xe, AMD Radeon integrated). **No discrete GPU.** |
| OS | Ubuntu 22.04 LTS reference |
| Runtime | **llama.cpp + GGUF only** (competition rule) |
| Connectivity | **100% offline** — zero outbound network calls during eval |

The tutor runs CPU-only via llama.cpp and measures ~1.7 GB peak RSS, well inside the 7 GB
budget. You can develop on any machine; only the final artifact is measured against the
profile above.

Toolchain:

- Go 1.26+ (for the Go components)
- [llama.cpp](https://github.com/ggml-org/llama.cpp) build with `llama-server`
  (and `llama-bench`), on PATH or pointed at in the Makefile
- Python 3.11+ and [promptfoo](https://promptfoo.dev) for the eval workflow
- The three models and the corpus data (see below)

> **Note on the 3-model setup.** Three local models play three different roles —
> only one of them is the scored submission model:
>
> | Model | Role in the system | Status |
> |---|---|---|
> | Qwen2.5-Math-1.5B | generation — **the submission model** | measured by the ADTC profiler |
> | nomic-embed-text | RAG retrieval dependency (index + query) | supporting (not scored directly) |
> | Qwen2.5-3B | eval-only judge (`llm-rubric`), never used at inference time | tooling |
>
> `download_model.sh` fetches only the scored model; the embedder and judge are
> dev-loop dependencies the pipeline needs locally.

### Models

| Model | Role | Notes |
|---|---|---|
| `qwen2.5-math-1.5b-instruct-q4_k_m.gguf` | generation | `./download_model.sh` fetches it into `model/` |
| `nomic-embed-text-v1.5.Q4_K_M.gguf` | embeddings | embedding server is separate from generation |
| `qwen2.5-3b-instruct-q4_k_m.gguf` | grading judge | used only by the eval harness |

Model and server paths are defined at the top of the `Makefile` — adjust to your environment.

### Corpus (`data/raw/`, git-ignored)

| Source | Contents | License |
|---|---|---|
| `hendrycks_math/` | 7 subdomain MATH JSONs (train split) | MIT |
| `gsm8k/train.jsonl` | arithmetic word problems | MIT |
| `rosen/` | discrete math solutions (.md/.txt) | open |
| `openstax/` | college algebra / calculus PDFs | CC BY |

Ingestion for Hendrycks MATH, GSM8K, and Rosen is implemented in `internal/rag/chunker.go`;
OpenStax PDFs are scaffolded in `ROADMAP.md` but not yet loaded.

## Quick start

### 1. Get the models

```bash
./download_model.sh                                   # Qwen2.5-Math into model/
# + place your nomic-embed-text and qwen2.5-3b judge GGUFs at the Makefile paths
```

### 2. Start the three local llama-server processes (one terminal each)

```bash
make serve-gen       # generation server  -> :8080
make serve-embed     # embedding server   -> :8081  (runs --embeddings)
make serve-judge     # judge model        -> :8083  (only needed for evals)
```

Three separate model processes for three distinct uses — keep the ports straight.

### 3. Index the corpus

```bash
make index
```

Embeds and indexes every chunk from Hendrycks / GSM8K / Rosen into the persistent
`data/chromem` store, then runs the test query. Indexing is sequential (one embedding
request per chunk) and can be re-run at any time.

### 4. Run the tutor server

```bash
make serve-tutor
```

Start the Go RAG server on `:8082`. It exposes the `/v1/complete` endpoint.

### 5. Ask it something

```bash
make run Q="find the derivative of x^2"        # CLI: prints retrieval + final prompt (no server)
curl -s localhost:8082/v1/complete \
  -H 'Content-Type: application/json' \
  -d '{"problem":"find the derivative of x^2"}'
```

`make run` uses the already-indexed DB (skips ingestion when no corpus sources are passed) and
is handy for inspecting exactly which chunks were retrieved and what prompt is sent, without a
running server.

## API

### `POST /v1/complete`

```json
{
  "problem": "find the derivative of x^2",
  "max_tokens": 512,
  "temperature": 0.1
}
```

`max_tokens` and `temperature` are optional (server defaults 512 / 0.1). Response:

```json
{ "content": "<model-generated solution>", "answer": "<parsed final answer>" }
```

`answer` is a lightweight runtime parse of the model's final answer (`internal/parse`,
mirroring the eval matcher's extractor: `\boxed{...}` → `final answer:` → `####` → whole
output). It is omitted when nothing can be parsed. The eval configs keep using only
`json.content`, so adding `answer` does not affect scoring.

The server retrieves the top-K (default 3) chunks for the problem, classifies its subdomain,
builds the RAG prompt, and blocks on generation before returning. The eval configs run it
single-concurrency (`maxConcurrency: 1`).

## Per-subdomain smoke matrix

Quick sanity checks — one query per coarse prompt category (`make run` shows the selected
system instruction; `curl /v1/complete` shows the parsed `answer`). Expect the geometry case
to fall back to unfiltered retrieval when the geometry corpus slice is thin:

| Category | Query | Expected instruction key |
|---|---|---|
| calculus | `Find the derivative of x^2.` | calculus |
| calculus (integral) | `Integrate 2x * e^(x^2).` | calculus |
| discrete_math | Prove by induction `1 + 2 + ... + n = n(n+1)/2` | discrete_math |
| linear_algebra | `Solve 2x + y = 7, x - y = 2 using matrix row reduction.` | linear_algebra |
| geometry | Lagos water tank, circumference 66, `π = 22/7` | geometry |
| other | `Why does 0.999... equal 1? Explain clearly.` | other (default) |

Each answer parsed correctly on the ADTC dev environment; subdomain text comes from
`internal/prompt/subdomainInstructions`.

## Architecture

```
                    classify subdomain ──▶ domain instruction
                                     ▼
problem ──▶ cmd/serve ──▶ Retriever ──▶ chromem-go "tutor-corpus" (vector store)
              (HTTP)          │              ▲
                              │              │ EmbedQuery (search_query prefix)
                              └── top-K chunks
                                     │
                    rag.BuildPrompt ──▶ system instruction + context + question
                                     ▼
                   internal/llm.Client ──▶ llama-server (Qwen2.5-Math) ──▶ content
```

Components, in the order a request flows through them:

1. `cmd/serve/main.go` — HTTP layer. Decodes the request, calls retrieval, builds the prompt,
   calls generation, encodes the JSON response.
2. `internal/rag/retriever.go` — `Retriever.Retrieve`: classifies the subdomain, embeds the
   query (**with the `search_query` prefix**), queries the vector store via `QueryEmbedding`,
   filters / picks top-K, and falls back to the unfiltered pool if a subdomain filter leaves
   too few results.
3. `internal/llm/client.go` — posts the built prompt to llama.cpp `/completion`, returns text.
4. `internal/rag/embedder.go` — llama.cpp `/embedding` client. Splits the two nomic prefixes:
   `search_document` for corpus indexing vs `search_query` for queries. chromem-go's
   collection-level `Query(text)` always reuses the document prefix for embedding, so it is
   bypassed on purpose: queries embed via `EmbedQuery` and hit `QueryEmbedding` directly.
5. `internal/rag/chunker.go` — loaders that map each corpus format to a homogeneous `Chunk`
   (problem + solution text, plus subdomain/source/level metadata for the store).

### Subdomain classifier

A cheap keyword heuristic (`minHits = 1`) in `retriever.go` maps a question to
algebra / arithmetic / precalculus / geometry / probability / number_theory, then two uses:

- narrow retrieval so the prompt's context is on-topic, occasionally falling back to the full
  pool if the filter would starve the prompt (topK=3);
- selects the domain-specific instruction text (e.g. "reason step by step, citing the relevant
  geometric theorem or property") prepended to the user turn.

### Prompt builder (`internal/prompt`)

`prompt.Builder` is the canonical prompt builder (ChatML framing, coarse-category CoT
instructions, RAG context block, answer anchor). `rag.BuildPrompt` is a thin adapter over it,
and `internal/prompt` also owns the subdomain → instruction mapping. Unit tests live in
`internal/prompt/builder_test.go` (structure, instruction selection, and a golden prompt).

## Evaluation

The eval harness (promptfoo) runs against the local tutor server and grades answers twice:

1. **Deterministic match** (`evals/answer_assert.js`): extracts the model's `\boxed{...}` /
   final answer, normalizes LaTeX/whitespace (fractions, sqrt, lists), and falls back to
   numeric comparison for algebraically-equivalent forms. No network needed.
2. **LLM rubric** (`llm-rubric`, local `qwen2.5-3b-instruct` judge on `:8083`): JSON-schema
   `{pass, reason, score}` judgments on the expected answer / tutoring-quality criteria.

Workflow:

```bash
make eval-sample     # regenerate the 30-case accuracy set (seed 42) from Hendrycks MATH
make eval            # run the accuracy eval (respects promptfoo cache)
make eval-fresh      # run the accuracy eval with no disk cache
make eval-quality    # run the 10-case qualitative + illustrative African set
make eval-view       # open the interactive promptfoo results
```

- `evals/promptfooconfig.yaml` — the accuracy eval: 30 cases, dual-graded.
- `evals/quality.yaml` — the qualitative eval: 10 bespoke cases (JAMB-style, market trader,
  induction, integration) graded by rubric.
- Result artifacts land in `evals/results_*.json`.

Current status (self-reported, 18/30 accuracy and 6/10 quality — failure details in
**REPORT.md**).

## Performance snapshot

Measured with the official ADTC profiler in **participant mode** on the development machine
(AMD Ryzen 9 6900HX, 29.1 GB RAM, no GPU, Arch Linux):

| Metric | Value |
|---|---|
| Peak RAM (RSS) | 1.71 GB |
| Steady-state RAM (RSS) | 1.61 GB |
| Generation speed | ~45.8 tokens/s |
| First-token latency | ~2.0 s (512-token prompt) |
| CPU p99 | 56.3% |

Full methodology and notes in **REPORT.md**.

### Scoring context

ADTC 2026 scoring is `S_total = 0.5·S_acc + 0.3·S_perf + 0.2·S_eff − P_thermal`.
Applying the official formulas to the numbers above:

| Component | Formula | This submission |
|---|---|---|
| Throughput | `min(TPS / 15.0, 1.0) · 100` | 45.8 t/s → **capped at 100** |
| Efficiency | `max(0, (7.0 − peak_rss_gb) / 7.0) · 100` | 1.71 GB → **≈ 75.6** |
| Thermal penalty | `−10` if throttled or core temp > 85 °C | none observed (`throttled: false`) |

These are self-reported development-machine values; the official audit runs on the
Standard Laptop. Reproduce locally with:

```bash
make profile        # adtc-profiler run --mode participant
make profile-audit  # adtc-profiler run --mode audit (gsm8k accuracy sample)
```

## ADTC 2026 compliance

This project is entered in the **Africa Deep Tech Challenge 2026 — The Laptop LLM**
(`math_scientific_reasoning` domain). The full requirements matrix, `metadata.json`
field map, and scoring worksheet live in **[COMPLIANCE.md](COMPLIANCE.md)**. Summary:

- **Rules met:** llama.cpp + GGUF only; fully offline; fits the 7 GB RAM budget
  (~1.7 GB peak); no discrete GPU; exactly 2 test prompts in `metadata.json`;
  `download_model.sh` idempotent and writing to `_runtime.model_path`; no weights committed.
- **Bonus claims:** `african_alpha_claim: true` (JAMB/WASSCE-style Nigerian context,
  naira word problems), `budget_laptop_claim: true`.
- **Gate-1 package status:** repo + REPORT.md done; **screenshots and the 2-minute demo
  video are still pending** — add `docs/` clips before submission.

> Official references: [challenge page](https://africadeeptech.org/challenge-2026/) ·
> [submission template](https://github.com/Africa-Deep-Tech-Foundation/adtc-2026-submission-template) ·
> [profiler](https://github.com/Africa-Deep-Tech-Foundation/adtc-profiler) ·
> [DevPost](https://adtc-2026.devpost.com/)

## Project layout

```
cmd/
  serve/main.go          # RAG HTTP server (:8082)
  tutor/main.go          # ingestion + out-of-band query CLI
internal/
  rag/                   # retriever, embedder, chunker, prompt builder
  llm/client.go          # llama.cpp completion client
  parse/                 # final-answer extractor (runtime `answer` field)
evals/                   # promptfoo configs, sampler, matcher, results
data/raw/                # corpus (git-ignored)
data/chromem/            # persistent vector store (git-ignored)
Makefile                 # everything below
download_model.sh        # model fetch
REPORT.md                # technical report (problem, design, benchmarks)
COMPLIANCE.md            # ADTC 2026 requirements matrix + scoring worksheet
promptfooconfig.yaml     # generic promptfoo playground (root-level leftovers)
metadata.json            # submission metadata
```

## Troubleshooting

- **Port conflicts** — Makefile pins 8080 / 8081 / 8083 / 8082 for the four servers; change
  the `*_PORT` vars.
- **No chunks indexed** — confirm the corpus exists under `data/raw/` (git-ignored, NOT
  downloaded by `download_model.sh`).
- **Stale store** — `data/chromem/*.db` is generated on index; delete it and re-index if the
  store looks stale (it is git-ignored).
- **`make run` prints nothing about indexing** — expected when corpus sources are absent;
  it assumes an existing store.
- **Judge / gen / embed mixups** — `serve-embed` must run `--embeddings`; never point
  `serve-tutor` at the judge port, etc.

## Further reading

- **[REPORT.md](REPORT.md)** — design choices, measured benchmarks, eval methodology
- **[COMPLIANCE.md](COMPLIANCE.md)** — ADTC 2026 requirements matrix, `metadata.json` field map, scoring worksheet
- **[ROADMAP.md](ROADMAP.md)** — planned work (e.g. OpenStax chunker, TUI, LaTeX→text renderer)
- promptfoo docs: https://promptfoo.dev/docs