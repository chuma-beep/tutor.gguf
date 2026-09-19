# Performance Tuning Log

Every benchmark below comes from **Docker audit mode** (`adtc-profiler:latest` image,
`--memory=7.5g --cpus=4`). No raw host numbers. The audit container builds llama.cpp as
baseline x86-64 with no AVX2 or FMA and presents a 4-vCPU / 7.5 GB profile. That is the
closest local proxy for the ADTC Standard Laptop and the official audit VM.

Run protocol. After every major change (threads, context, prompts, LoRA) re-run the Docker
audit plus `compare`. Append a date-stamped section here. Copy the JSON snapshots into
`results/`.

Key references from the tuning reading list:
[Drepper, *What Every Programmer Should Know About Memory*](https://www.akkadia.org/drepper/cpumemory.pdf)
and the [llama.cpp performance docs](https://github.com/ggml-org/llama.cpp/blob/master/PERFORMANCE.md).

---

## 2026-08-13 — Baseline + thread tuning in Docker audit mode

Container: `adtc-profiler:latest` (baseline x86-64 llama.cpp build). Host: AMD Ryzen 9 6900HX,
16 cores, 29 GB. Model: qwen2.5-math-1.5b Q4_K_M (941 MB). Bench shape mirrors the profiler:
`llama-bench -p 512 -n 128`.

### Environment probe (`n_threads` resolution)

- The container's llama-bench resolves `n_threads=8` by default in this llama.cpp version. It
  does not use hardware concurrency (the host shows 16). Note that `--cpus=4`, `--cpuset-cpus`
  and `taskset` do NOT change `/proc/cpuinfo`.
- Under a `--cpus=4` quota the default 8 threads give **14.8 t/s**. A real 4-vCPU VM reports
  `hw_concurrency=4`, so the official audit should land near the `-t 4` row.

### Thread matrix (`--memory=7.5g --cpus=4`, p=512 n=128)

| `-t` | pp t/s | gen t/s | note |
|---|---|---|---|
| 1 | 6.40 | 4.88 | baseline single-thread |
| 2 | 12.26 | 9.00 | |
| **4** | **22.86** | **16.62** | **best under 4-core quota** |
| 6 | 21.81 | 15.24 | oversubscribed |
| 8 | 20.83 | 13.36 | worse |
| auto(8) | ~22.4 | ~14.8 | default llama-bench |

Conclusion: `-t 4` is optimal for a 4-vCPU environment. It gives 16.6 t/s, so S_perf is capped
at 100 because 16.6 is above the 15.0 reference. Beyond 4 threads under a 4-CPU quota,
throughput drops.

Action: default the Makefile `THREADS` var so constrained 4-core deploys can pass `-t 4`. Leave
auto for the dev machine or a Standard Laptop with more cores.

---

## 2026-08-13 — In-container participant + audit, compare verdict

Method. Both runs use an identical environment (`--memory=7.5g --cpus=4`, `--skip-accuracy`).
The official profiler image runs with the repo mounted read-only. The model symlink resolves
through the `~/Projects/models` mount. `GIT_CONFIG` sets `safe.directory` so the git SHA
resolves. `_runtime.docker_image=adtc-profiler:latest` provides the digest.

| Metric | participant | audit | Δ | verdict |
|---|---|---|---|---|
| peak_rss_mb | 1096.86 | 1097.23 | +0.0% | pass |
| steady_state_rss_mb | 1035.46 | 1034.74 | −0.1% | pass |
| tokens/s (gen) | 14.10 | 13.52 | −4.1% | pass |
| first_token_latency_ms | 22839.67 | 23624.90 | +3.4% | pass |

**Verdict: PASS** → `results/verdict.json`.

Notes:
- TPS is ~13–14 t/s (llama-bench default 8 threads under a 4-CPU quota). On a real 4-vCPU VM
  `hw_concurrency=4`, so llama-bench resolves 4 threads and reaches ~16.6 t/s (the `-t 4` row).
  Either way S_perf is capped at 100.
- Peak RSS is **1.10 GB**, so S_eff ≈ **84.3**. The container or baseline build measures lower
  than the old dev-machine 1.71 GB. The whole stack still has ~6 GB headroom.
- `submission.json` was updated to the container participant measurement so a future audit
  compare is coherent. The dev-machine 45.78 t/s could never pass.
- git_commit_sha `24835df95839`, docker_image_digest `sha256:517bf562...`.

---

## 2026-08-13 — Context window sizing

RAG prompt lengths were measured across the 30-case accuracy set with the gen tokenizer.
Median was 643 tokens and worst case was **1122 tokens**. The Go client caps generation at 512
tokens (`internal/llm/client.go`), so the worst case is prompt + gen ≈ **1634 tokens**.

Matcher-only sweep (`promptfoo`, 30 cases, deterministic `answer_assert.js`, no judge):

| `-c` | passed | peak RSS (warm, dev host) | note |
|---|---|---|---|
| 4096 (old default) | 16/30 | 1815 MB | baseline |
| **2048 (new default)** | **17/30** | **1756 MB** | same accuracy, −60 MB |
| 1536 | would truncate | — | worst-case prompt alone is 1122; too tight |

Conclusion: **`-c 2048`** is the safe minimum. It fits the observed worst case with headroom.
It has no accuracy regression and saves ~60 MB of KV cache against 4096. `Makefile` default:
`CTX ?= 2048`.

---

## 2026-08-13 — RAM profiling + 7.5 GB container validation

### Live product stack under `--memory=7.5g --cpus=4` (baseline llama.cpp build)

The full stack ran inside the audit container: serve-gen (`-c 2048 -t 4`), serve-embed, the RAG
`serve` binary and the chromem store. It was warmed with 8 real `/v1/complete` requests.

| process | RSS | what it is |
|---|---|---|
| gen-model | 1113 MB | weights ~0.96 GB file-backed + KV cache (~59 MB @ 2048) + compute buffers + thread stacks |
| embed-model | 123 MB | nomic 81 MB + buffers |
| rag-server | 55 MB | Go heap incl. ~7,124 chromem vectors (~22 MB) + metadata |
| **total** | **~1.29 GB** | **~6.2 GB headroom under the 7.5 GB cap; no OOM** |

The judge model (Qwen2.5-3B) is eval-only. It is not resident in the live path.

### Non-model memory consumers (host, native build, 16 threads, `-c 4096`)

- The gen server's anonymous memory reached **~1.7 GB** after sustained load, against a ~0.96 GB
  model file. Contributors are the KV cache, growable compute/work buffers, 16 thread stacks
  and the hybrid-memory arena. This is build/thread/context dependent. The baseline 4-thread
  container server in the row above uses ~1.1 GB total, so this is not a budget risk.
- RAM scales with threads (`-t`), context (`-c`) and batch size. The tuned profile (2048 ctx
  and `-t 4` on 4-core hardware) is the memory-minimal configuration.

---

## Reading reference

- Drepper memory paper (above)
- llama.cpp PERFORMANCE.md (above)
