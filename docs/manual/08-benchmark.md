---
title: Benchmark
order: 8
---

# Benchmark

How the tutor actually performs under the target constraints. For how the
pipeline works, see [Architecture](architecture).

## Test environment

All numbers below were measured with the official ADTC profiler in
**audit-profile mode**: the profiler's own Docker image
(`adtc-profiler:latest`) under `--memory=7.5g --cpus=4`, with its baseline
(no-AVX2) llama.cpp build — the closest local proxy for the Standard Laptop /
audit VM.

| Fact | Detail |
|---|---|
| Host machine | AMD Ryzen 9 6900HX, 29.1 GB RAM, no GPU |
| Container profile | 4 vCPU / 7.5 GB |
| Runtime | llama.cpp, CPU only |
| Model | Qwen2.5-Math-1.5B, GGUF Q4_K_M |

## Performance

| Metric | Result |
|---|---|
| Throughput | ~13–14 tokens/s (default threads) |
| 4-thread throughput | 16.6 tokens/s (`-t 4`) |
| Time to first token | ~23 s (512-token prompt) |
| CPU utilisation (p99) | 34.9% |
| Core temp peak | 20.0 °C |
| Thermal throttling | None observed |

### Memory

| Metric | Result |
|---|---|
| Peak RSS | 1.10 GB |
| Steady-state RSS | 1.03 GB |
| Budget share | ~16% of the 7 GB limit |

## Evaluation

### Accuracy

**15 / 30** (`evals/results_new_2.json`; earlier reruns 14–16/30). Thirty problems sampled (seed 42) from the Hendrycks MATH
training split, each scored two ways: deterministic answer matching
(`\boxed{}` extraction, LaTeX/whitespace normalization, numeric fallback for
equivalent forms) and a model-graded rubric from a local Qwen2.5-3B-Instruct
judge. Dominant failure mode is truncation before the `\boxed{}` finale; see
REPORT.md `## Failure analysis` for the per-case breakdown.

### Quality

**6 / 10** (`evals/results_quality_new.json`). Ten bespoke cases — JAMB-style exam items, market-trader
arithmetic, induction, integration — graded by the same local judge on step
clarity and tutoring quality. All four failures hold the correct `\boxed{}`
answer and fail on judge-overstrictness (method phrasing, not math); see
REPORT.md `## Failure analysis`.

## Methodology

Throughput, memory and thermals come from `adtc-profiler` (see
`docs/tuning.md` for the full log); accuracy and quality come from the
promptfoo harness running against a local RAG server. These are
**self-reported audit-profile values** — the official audit runs the same
profiler on the Standard Laptop. Full analysis in
[REPORT.md](https://github.com/chuma-beep/tutor.gguf/blob/main/REPORT.md).

Scoring context (our estimate, not an official score): reproducing the
official formula gives S_perf ≈ 90 and S_eff ≈ 84.3 with no thermal penalty —
see [COMPLIANCE.md](https://github.com/chuma-beep/tutor.gguf/blob/main/COMPLIANCE.md)
for the worksheet.

## Limitations

- The host (Ryzen 9 6900HX) is not the Standard Laptop; the Docker profile is a proxy.
- The baseline no-AVX2 llama.cpp build understates tuned hardware.
- Accuracy rests on a 30-case sample, quality on 10 bespoke cases.
- Evals run single-concurrency against one local server.
- All values are self-reported until the official audit.
