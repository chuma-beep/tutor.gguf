# Tutor.gguf — On-device Math Tutor

Tutor.gguf is a fully offline math tutor. It runs on the laptop you already own. It is built
for Nigerian CS undergraduates at distance-learning institutions. Give it a math problem. It
finds similar worked examples in a local corpus (RAG). Then it writes a step-by-step solution.
No GPU. No internet. No cloud API fees.

> **Team:** chuma-beep · **Domain:** math_scientific_reasoning · **Language:** en
> **Claims:** African Alpha use case, budget-laptop compatible

**[Project homepage](https://chuma-beep.github.io/tutor.gguf/)** ·
[releases](https://github.com/chuma-beep/tutor.gguf/releases) ·
[technical report](REPORT.md) · [compliance record](COMPLIANCE.md)

## What it does

- **Model.** Qwen2.5-Math-1.5B-Instruct (GGUF Q4_K_M) writes step-by-step solutions. Final
  answers use LaTeX `\boxed{}`.
- **Retrieval.** nomic-embed-text-v1.5 embeddings feed a chromem-go vector store. A small
  keyword classifier sorts each question into algebra / calculus / discrete math / geometry /
  probability / number theory. It then filters the retrieved context and picks the right
  prompt instructions.
- **Corpus.** GSM8K, Hendrycks MATH, Rosen Discrete Math solutions and OpenStax textbooks.
- **Runtime.** 100% local llama.cpp (`llama-server`). CPU-only. No GPU needed.

It covers Discrete Mathematics, Calculus I/II, Linear Algebra and Geometry-style problems.
These match what Nigerian students see in JAMB/WASSCE and first-year CS courses.

## Requirements

The target platform is the **ADTC 2026 Standard Laptop**:

| Constraint | Spec |
|---|---|
| CPU | Intel Core i5 10th–12th gen or AMD Ryzen 5 3000–5000 (x86-64) |
| RAM | 8 GB DDR4 — **max 7 GB managed working set** (hard limit) |
| Graphics | Integrated only (Intel UHD / Iris Xe, AMD Radeon integrated). **No discrete GPU.** |
| OS | Ubuntu 22.04 LTS reference |
| Runtime | **llama.cpp + GGUF only** (competition rule) |
| Connectivity | **100% offline** — zero outbound network calls during eval |

Tutor.gguf runs CPU-only through llama.cpp. It uses about 1.1 GB peak RSS, well under the
7 GB budget. You can develop on any machine. Only the final artifact is measured against the
profile above.

Toolchain:

- Go 1.26+ for the Go components
- [llama.cpp](https://github.com/ggml-org/llama.cpp) built with `llama-server` and
  `llama-bench`. Keep it on your PATH or point the Makefile at it.
- Python 3.11+ and [promptfoo](https://promptfoo.dev) for the eval workflow
- The three models and the corpus data (see below)

> **Note on the 3-model setup.** Three local models do three different jobs. Only one of them
> is the scored submission model:
>
> | Model | Role in the system | Status |
> |---|---|---|
> | Qwen2.5-Math-1.5B | generation — **the submission model** | measured by the ADTC profiler |
> | nomic-embed-text | RAG retrieval dependency (index + query) | supporting (not scored directly) |
> | Qwen2.5-3B | eval-only judge (`llm-rubric`), never used at inference time | tooling |
>
> `download_model.sh` fetches only the scored model. The embedder and judge are dev
> dependencies the pipeline needs locally.

### Models

| Model | Role | Notes |
|---|---|---|
| `qwen2.5-math-1.5b-instruct-q4_k_m.gguf` | generation | `./download_model.sh` fetches it into `model/` |
| `nomic-embed-text-v1.5.Q4_K_M.gguf` | embeddings | embedding server is separate from generation |
| `qwen2.5-3b-instruct-q4_k_m.gguf` | grading judge | used only by the eval harness |

Model and server paths live at the top of the `Makefile`. Adjust them to your environment.

### Corpus (`data/raw/`, git-ignored)

| Source | Contents | License |
|---|---|---|
| `hendrycks_math/` | 7 subdomain MATH JSONs (train split) | MIT |
| `gsm8k/train.jsonl` | arithmetic word problems | MIT |
| `rosen/` | discrete math solutions (.md/.txt) | open |
| `openstax/` | college algebra / calculus PDFs | CC BY |

`internal/rag/chunker.go` ingests Hendrycks MATH, GSM8K and Rosen. OpenStax PDFs are
scaffolded in `ROADMAP.md` but not loaded yet.

## Quick start

### Just want to try it? (no toolchain needed)

Grab a prebuilt binary from [Releases](https://github.com/chuma-beep/tutor.gguf/releases) or
run `install.sh`. Then:

```bash
tutor setup   # one-time: downloads llama.cpp, models (~1.2 GB) and corpus, then builds the index
tutor chat    # interactive shell — starts the whole stack and tears it down on exit
```

`setup` is idempotent. Re-run it any time: it skips what already exists. Artifacts live in
`~/.tutor/` (`models/`, `corpus/`, `bin/llama-server`, `chromem/`, `logs/`). After setup,
everything runs 100% offline. `TUTOR_HOME` moves the directory. `TUTOR_LLAMA_SERVER`,
`TUTOR_THREADS`, `TUTOR_CTX` and `TUTOR_DB_PATH` override runtime pieces.

`tutor serve` (HTTP API on :8082) and `tutor index` work the same way. With no URL flags they
start their own llama-servers on free ports. Pass `-gen-url`/`-embedder-url` to use external
servers instead.

Windows: download `tutor-windows-amd64.exe` from Releases. Run `tutor setup`, then
`tutor chat`.

Desktop (Wails): download one of these from Releases.

- `tutor-desktop-linux-amd64.deb`
- `tutor-desktop-linux-amd64.AppImage`
- `tutor-desktop-darwin-universal.dmg`
- `tutor-desktop-windows-amd64-installer.exe`

**Double-click to install. Then just chat.** On first launch it downloads the ~1.2 GB model
and corpus (about 10 min on 10 Mbps, longer on slow data) and shows a progress bar. After
that it runs 100% offline forever. No terminal. No commands. Lost power? Open it again — it
resumes. No internet lab? Use the `tutor-offline.tar.gz` USB pack instead (campus Wi-Fi once,
then copy). Linux needs `webkit2gtk-4.1` (`sudo apt install libwebkit2gtk-4.1-0` on Ubuntu
24.04; 22.04 uses the 4.0 build).

### Developer flow (repo checkout)

The Makefile targets below run the three llama-server processes by hand. Same components,
manual orchestration. Useful when you tune or run evals.

#### 1. Get the models

```bash
./download_model.sh                                   # Qwen2.5-Math into model/
# + place your nomic-embed-text and qwen2.5-3b judge GGUFs at the Makefile paths
```

#### 2. Start the three local llama-server processes (one terminal each)

```bash
make serve-gen       # generation server  -> :8080
make serve-embed     # embedding server   -> :8081  (runs --embeddings)
make serve-judge     # judge model        -> :8083  (only needed for evals)
```

Three separate model processes for three purposes. Keep the ports straight.

#### 3. Index the corpus

```bash
make index
```

This embeds and indexes every chunk from Hendrycks / GSM8K / Rosen into the persistent
`data/chromem` store. Then it runs the test query. Indexing is sequential (one embedding
request per chunk). You can re-run it at any time.

#### 4. Run the tutor server

```bash
make serve-tutor
```

Starts the Go RAG server on `:8082`. It exposes the `/v1/complete` endpoint.

#### 5. Ask it something

```bash
make run Q="find the derivative of x^2"        # CLI: prints retrieval + final prompt (no server)
curl -s localhost:8082/v1/complete \
  -H 'Content-Type: application/json' \
  -d '{"problem":"find the derivative of x^2"}'
```

`make run` uses the already-indexed DB. It skips ingestion when you pass no corpus sources.
It is handy for inspecting which chunks were retrieved and what prompt was sent, without a
running server.

#### 6. Ask it from the terminal (interactive TUI)

```bash
make tui            # Bubble Tea shell, Unicode math (>≡ π … ≤, tall brackets, boxed answers)
make tui-ascii      # same shell with ASCII-only fallbacks (x^2, sqrt()-style, +/- borders)
```

Both open a Bubble Tea alternate screen on `:8082`. Type a question and press Enter. The
model's output streams into the transcript. LaTeX spans (`\(...\)`, `\[...\]`, `$...$`) become
terminal art: stacked fractions, square/cube roots, sum/integral limits, binom, `\boxed`
borders and `\alpha` → α. The parser degrades gracefully. Anything it does not model (rare
`\begin{matrix}` synthetic-division tables) falls back to a linear passthrough, never blank.
Unicode mode is the prettier default. ASCII mode trades the glyphs for wider terminal safety
(useful for screenshots on exotic fonts).

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

`answer` is a lightweight runtime parse of the model's final answer (`internal/parse`). It
mirrors the eval matcher's extractor: `\boxed{...}` → `final answer:` → `####` → whole output.
It is omitted when nothing can be parsed. The eval configs use only `json.content`, so adding
`answer` does not affect scoring.

The server retrieves the top-K (default 3) chunks for the problem. It classifies the
subdomain, builds the RAG prompt and waits for generation before returning. The eval configs
run it single-concurrency (`maxConcurrency: 1`).

## Per-subdomain smoke matrix

Quick sanity checks. Run one query per coarse prompt category. `make run` shows the selected
system instruction. `curl /v1/complete` shows the parsed `answer`. Expect the geometry case
to fall back to unfiltered retrieval when the geometry corpus slice is thin:

| Category | Query | Expected instruction key |
|---|---|---|
| calculus | `Find the derivative of x^2.` | calculus |
| calculus (integral) | `Integrate 2x * e^(x^2).` | calculus |
| discrete_math | Prove by induction `1 + 2 + ... + n = n(n+1)/2` | discrete_math |
| linear_algebra | `Solve 2x + y = 7, x - y = 2 using matrix row reduction.` | linear_algebra |
| geometry | Lagos water tank, circumference 66, `π = 22/7` | geometry |
| other | `Why does 0.999... equal 1? Explain clearly.` | other (default) |

Each answer parsed correctly on the ADTC dev environment. The subdomain text comes from
`internal/prompt/subdomainInstructions`.

## Architecture

```
                    classify subdomain ──▶ domain instruction
                                     ▼
problem ──▶ tutor serve ──▶ Retriever ──▶ chromem-go "tutor-corpus" (vector store)
             (HTTP)           │              ▲
                              │              │ EmbedQuery (search_query prefix)
                              └── top-K chunks
                                     │
                    rag.BuildPrompt ──▶ system instruction + context + question
                                     ▼
                   internal/llm.Client ──▶ llama-server (Qwen2.5-Math) ──▶ content
```

Components, in the order a request flows through them:

1. `internal/cli/serve.go` — HTTP layer (`tutor serve` subcommand). It decodes the request,
   calls retrieval, builds the prompt, calls generation and encodes the JSON response. With no
   URL flags it also supervises the llama-server processes through `internal/runtime`.
2. `internal/rag/retriever.go` — `Retriever.Retrieve`. It classifies the subdomain, embeds the
   query (**with the `search_query` prefix**) and queries the vector store through
   `QueryEmbedding`. It picks top-K and falls back to the unfiltered pool if a subdomain filter
   leaves too few results.
3. `internal/llm/client.go` — posts the built prompt to llama.cpp `/completion` and returns text.
4. `internal/rag/embedder.go` — llama.cpp `/embedding` client. It splits the two nomic
   prefixes: `search_document` for corpus indexing and `search_query` for queries. chromem-go's
   collection-level `Query(text)` always reuses the document prefix. So it is bypassed on
   purpose: queries embed through `EmbedQuery` and hit `QueryEmbedding` directly.
5. `internal/rag/chunker.go` — loaders that map each corpus format to a homogeneous `Chunk`
   (problem + solution text plus subdomain/source/level metadata for the store).

### Subdomain classifier

A cheap keyword heuristic (`minHits = 1`) in `retriever.go` maps a question to algebra /
arithmetic / precalculus / geometry / probability / number_theory. It then does two jobs:

- It narrows retrieval so the prompt's context stays on-topic. It falls back to the full pool
  when the filter would starve the prompt (topK=3).
- It selects the domain-specific instruction text (for example "reason step by step, citing
  the relevant geometric theorem or property") that is prepended to the user turn.

### Prompt builder (`internal/prompt`)

`prompt.Builder` is the canonical prompt builder (ChatML framing, coarse-category CoT
instructions, RAG context block, answer anchor). `rag.BuildPrompt` is a thin adapter over it.
`internal/prompt` also owns the subdomain → instruction mapping. Unit tests live in
`internal/prompt/builder_test.go` (structure, instruction selection and a golden prompt).

## Evaluation

The eval harness (promptfoo) runs against the local tutor server and grades answers twice:

1. **Deterministic match** (`evals/answer_assert.js`). It extracts the model's `\boxed{...}` /
   final answer, normalizes LaTeX/whitespace (fractions, sqrt, lists) and falls back to numeric
   comparison for algebraically-equivalent forms. No network needed.
2. **LLM rubric** (`llm-rubric`, local `qwen2.5-3b-instruct` judge on `:8083`). It returns
   JSON-schema `{pass, reason, score}` judgments on the expected answer and tutoring-quality
   criteria.

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

Current status is self-reported: 18/30 accuracy and 6/10 quality. Failure details are in
**REPORT.md**.

## Performance snapshot

Measured with the official ADTC profiler in **audit-profile mode**. That means the profiler's
own Docker image under `--memory=7.5g --cpus=4` with its baseline (no-AVX2) llama.cpp build.
It is the closest local proxy for the Standard Laptop / audit VM:

| Metric | Value |
|---|---|
| Peak RAM (RSS) | 1.10 GB |
| Steady-state RAM (RSS) | 1.03 GB |
| Generation speed | ~13–14 tokens/s (llama-bench default threads; 16.6 t/s at `-t 4`) |
| First-token latency | ~23 s (512-token prompt) |
| CPU p99 | 34.9% |
| Core temp / throttling | 20 °C / `throttled: false` |

> Earlier dev-machine numbers (45.78 t/s, 1.71 GB, native build, 16 cores) were
> non-representative. This snapshot reflects the Docker audit profile. Full methodology and
> tuning log in **docs/tuning.md**.

### Scoring context

ADTC 2026 scoring is `S_total = 0.5·S_acc + 0.3·S_perf + 0.2·S_eff − P_thermal`. Applying the
official formulas to the numbers above:

| Component | Formula | This submission |
|---|---|---|
| Throughput | `min(TPS / 15.0, 1.0) · 100` | 13.5 t/s → **≈ 90** (16.6 t/s at `-t 4` → capped 100) |
| Efficiency | `max(0, (7.0 − peak_rss_gb) / 7.0) · 100` | 1.10 GB → **≈ 84.3** |
| Thermal penalty | `−10` if throttled or core temp > 85 °C | none observed (`throttled: false`) |

These are self-reported audit-profile values. The official audit runs on the Standard Laptop.
Reproduce locally with:

```bash
make profile        # adtc-profiler run --mode participant (on this machine)
make profile-audit  # adtc-profiler run --mode audit (gsm8k accuracy sample)
# Docker audit profile (same env as this table):
docker run --rm --memory=7.5g --cpus=4 -v "$PWD":/submission:ro \
  -v ~/Projects/models:/home/wisdom/Projects/models:ro \
  -e GIT_CONFIG_COUNT=1 -e GIT_CONFIG_KEY_0=safe.directory -e GIT_CONFIG_VALUE_0='*' \
  -v "$PWD"/results:/artifacts \
  adtc-profiler:latest run --submission /submission --mode audit \
  --output /artifacts/audit.json --skip-accuracy
```

## ADTC 2026 compliance

This project is entered in the **Africa Deep Tech Challenge 2026 — The Laptop LLM**
(`math_scientific_reasoning` domain). The full requirements matrix, `metadata.json` field map
and scoring worksheet live in **[COMPLIANCE.md](COMPLIANCE.md)**. Summary:

- **Rules met.** llama.cpp + GGUF only. Fully offline. Fits the 7 GB RAM budget (1.10 GB
  peak). No discrete GPU. Exactly 2 test prompts in `metadata.json`. `download_model.sh` is
  idempotent and writes to `_runtime.model_path`. No weights committed.
- **Bonus claims.** `african_alpha_claim: true` (JAMB/WASSCE-style Nigerian context, naira
  word problems). `budget_laptop_claim: true`.
- **Gate-1 package status.** Repo, REPORT.md, screenshots and demo video are done
  (`docs/tutor-gguf-demo.mp4`, 104 s silent v1 — narrated re-cut optional).

> Official references: [challenge page](https://africadeeptech.org/challenge-2026/) ·
> [submission template](https://github.com/Africa-Deep-Tech-Foundation/adtc-2026-submission-template) ·
> [profiler](https://github.com/Africa-Deep-Tech-Foundation/adtc-profiler) ·
> [DevPost](https://adtc-2026.devpost.com/)

## Project layout

```
cmd/
  serve/main.go          # RAG HTTP server (:8082)
  tutor/main.go          # ingestion + out-of-band query CLI
  tui/main.go            # Bubble Tea shell (make tui / make tui-ascii)
internal/
  rag/                   # retriever, embedder, chunker, prompt builder
  llm/client.go          # llama.cpp completion client
  parse/                 # final-answer extractor (runtime `answer` field)
  renderer/              # LaTeX → AST → terminal-art renderer (Unicode + ASCII)
  tui/                   # Bubble Tea model: input, streaming transcript, scroll
evals/                   # promptfoo configs, sampler, matcher, results
data/raw/                # corpus (git-ignored)
data/chromem/            # persistent vector store (git-ignored)
site/                    # GitHub Pages homepage (Vite + React + TanStack Router)
packaging/               # .deb / .AppImage packaging for the desktop build
Makefile                 # everything below
download_model.sh        # model fetch
REPORT.md                # technical report (problem, design, benchmarks)
COMPLIANCE.md            # ADTC 2026 requirements matrix + scoring worksheet
promptfooconfig.yaml     # generic promptfoo playground (root-level leftovers)
metadata.json            # submission metadata
```

## Troubleshooting

- **Port conflicts.** The Makefile pins 8080 / 8081 / 8083 / 8082 for the four servers. Change
  the `*_PORT` vars.
- **No chunks indexed.** Confirm the corpus exists under `data/raw/`. It is git-ignored and NOT
  downloaded by `download_model.sh`.
- **Stale store.** `data/chromem/*.db` is generated on index. Delete it and re-index if the
  store looks stale. It is git-ignored.
- **`make run` prints nothing about indexing.** Expected when corpus sources are absent. It
  assumes an existing store.
- **Judge / gen / embed mixups.** `serve-embed` must run `--embeddings`. Never point
  `serve-tutor` at the judge port.

## Further reading

- **[REPORT.md](REPORT.md)** — design choices, measured benchmarks, eval methodology
- **[COMPLIANCE.md](COMPLIANCE.md)** — ADTC 2026 requirements matrix, `metadata.json` field map, scoring worksheet
- **[ROADMAP.md](ROADMAP.md)** — planned work (e.g. OpenStax chunker, TUI, LaTeX→text renderer)
- promptfoo docs: https://promptfoo.dev/docs
