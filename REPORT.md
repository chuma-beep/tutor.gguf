# Technical Report — Tutor.gguf: On-Device Math Tutor

**Team ID:** chuma-beep  
**Domain:** math_scientific_reasoning  
**Model:** Qwen2.5-Math-1.5B-Instruct-Q4_K_M + RAG  
**Submission:** ADTC 2026 — The Laptop LLM (Gate 1). See [COMPLIANCE.md](COMPLIANCE.md) for the requirements matrix, the `metadata.json` field map and the scoring worksheet.

---

## Problem

Nigerian CS undergraduates at distance-learning institutions need step-by-step math help. They
study Discrete Mathematics, Calculus I/II and Linear Algebra. But cloud AI is out of reach:
API fees come in naira, connectivity is unstable and power is unreliable.

This submission is a fully on-device math tutor. It is a RAG pipeline over the GSM8K /
Hendrycks MATH / OpenStax / Rosen corpus, served by llama.cpp with zero cloud dependencies.
The same laptop that runs the student's coursework runs the tutor. It works offline at no
marginal cost.

## Design Decisions

- **Base model.** Qwen2.5-Math-1.5B-Instruct. It is math-specialized: it reasons step by step
  and emits `\boxed{}` final answers. At 1.5B it fits the 7 GB RAM budget with RAG embeddings
  resident.
- **Quantization.** GGUF Q4_K_M. Measured peak RSS is 1.10 GB, about 16% of the 7 GB budget.
  That leaves room for the OS, the browser and the RAG index. Q8_0 was rejected: it doubles
  model memory (~2 GB) for negligible accuracy gain. Q2_K was rejected after sample evals
  showed degraded multi-step reasoning.
- **Retrieval.** chromem-go vector store (in-memory and persistent), nomic-embed-text-v1.5
  Q4_K_M embeddings and a subdomain keyword classifier
  (algebra/calculus/discrete_math/geometry/probability/number_theory). The classifier filters
  retrieval and selects domain-specific prompt instructions.
- **Runtime.** llama.cpp (`llama-server`). CPU-only and AVX2. No GPU required. This matches
  the ADTC Standard Laptop's integrated-graphics constraint.
- **Alternatives rejected.** 7B-class models (Mistral-7B, Qwen2.5-7B) exceeded the
  latency/memory envelope on 8 GB integrated graphics. Cloud API fallbacks were ruled out by
  the no-cloud-dependency requirement.

### Tools used and why

| Tool | Role |
|---|---|
| llama.cpp (`llama-server`) | mandatory runtime per ADTC rules — GGUF, CPU-only inference |
| Qwen2.5-Math-1.5B + GGUF Q4_K_M | math-specialized base model at a size that fits the 7 GB budget |
| nomic-embed-text-v1.5 | local embeddings with asymmetric document/query prefixes |
| chromem-go | persistent in-memory vector store with cosine similarity |
| Go | single static binaries for the RAG server and indexing CLI |
| promptfoo | eval harness against the local server (deterministic + LLM-rubric) |
| Qwen2.5-3B-Instruct | on-device grading judge (JSON-schema constrained) |

## Constraints

- Target: 8 GB DDR4 RAM, integrated GPU, Intel i5 10th-12th gen / AMD Ryzen 5 3000-5000,
  Ubuntu 22.04 LTS.
- No GPU acceleration. Inference is pure CPU through llama.cpp.
- No connectivity. Retrieval, embedding and generation must all run locally.
- Data. The corpus must be licensed or open: GSM8K, Hendrycks MATH (MIT), OpenStax (CC BY)
  and Rosen discrete math solutions.

## Benchmarks

Measured with the official ADTC profiler in **audit-profile mode**. That is the profiler's own
Docker image (`adtc-profiler:latest`) under `--memory=7.5g --cpus=4` with its baseline
(no-AVX2) llama.cpp build. It is the closest local proxy for the Standard Laptop / audit VM.
Full tuning log in `docs/tuning.md`.

| Metric | Value |
|---|---|
| Machine | host AMD Ryzen 9 6900HX, 29.1 GB RAM, no GPU; 4-vCPU / 7.5 GB container profile |
| Peak RAM (RSS) | 1.10 GB |
| Steady-state RAM (RSS) | 1.03 GB |
| Generation speed | ~13–14 tokens/s (llama-bench default threads); 16.6 t/s at `-t 4` |
| Time to first token | ~23 s (512-token prompt, 128-token generation) |
| CPU utilization (p99) | 34.9% |
| Core temp peak | 20.0°C (sensor read; no throttling flag) |
| Thermal throttling | None observed (`throttled: false`) |

These are self-reported audit-profile values. The official audit runs the same profiler on the
Standard Laptop. Mapping to the official scoring formula
(`0.5·S_acc + 0.3·S_perf + 0.2·S_eff − P_thermal`): ~13.5 t/s yields S_perf ≈ 90 (16.6 t/s at
`-t 4` caps at 100). 1.10 GB peak yields S_eff ≈ 84.3. No thermal penalty was observed. See
[COMPLIANCE.md](COMPLIANCE.md) for the full worksheet. The in-repo `submission.json` and
`audit.json` reproduce the numbers through `adtc-profiler compare` (verdict **pass**).

## Accuracy & Eval Methodology

- **Benchmarks.** The eval set is 30 problems sampled (seed 42) from the Hendrycks MATH
  training split. Each is scored two ways. First, deterministic answer matching: it extracts
  the `\boxed{}` / final answer, normalizes LaTeX and whitespace and falls back to numeric
  comparison for equivalent forms. Second, a model-graded rubric using a local
  Qwen2.5-3B-Instruct judge with JSON-schema-constrained output. All grading runs on-device.
  Latest run: **18/30 cases fully passed**. The matcher and judge agreed on all failures.
- **Qualitative.** A separate 10-case set covers step-by-step tutoring quality and
  African-context problems (JAMB-style exam items, market-trader arithmetic). The same local
  judge grades it. Latest run: **6/10 passed**. The failures are actionable tutoring gaps
  (final answers not stated clearly, weak induction structure), not harness errors.
- **Tooling.** The promptfoo eval harness runs against the local RAG server
  (`localhost:8082`). `adtc-profiler` handles the official throughput/memory/thermal
  measurements.

## African Use Case

The tutor is built around the Nigerian undergraduate context. It uses JAMB/WASSCE-style exam
problems, naira-denominated word problems and a curriculum-aligned RAG corpus. The offline UX
is zero-cost and designed for shared or low-power hardware with unreliable connectivity. See
the qualitative eval set (African-context cases).

## Submission Status

- **Gate 1 pending items.** None on the docs side. Screenshots (8 captures) and a 104 s silent
  demo video are in-repo (`docs/tutor-gguf-demo.mp4`, script in `docs/media-kit.md`). A
  narrated re-cut is optional polish, not a blocker.
- **Validation.** `download_model.sh` was verified on Ubuntu 22.04 (fresh `ubuntu:22.04`
  container: clean download to the exact expected byte size, valid GGUF header, idempotent
  re-run).
- **Repro.** Every command lives in the [README](README.md), the Makefile or `docs/tuning.md`.
  The Docker audit profile reproduces `submission.json` and `audit.json` (`compare` → PASS).
- **Compliance.** Full requirements matrix and scoring worksheet in
  [COMPLIANCE.md](COMPLIANCE.md).
