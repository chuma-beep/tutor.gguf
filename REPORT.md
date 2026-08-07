# Technical Report — Tutor.gguf: On-Device Math Tutor

**Team ID:** chuma-beep  
**Domain:** math_scientific_reasoning  
**Model:** Qwen2.5-Math-1.5B-Instruct-Q4_K_M + RAG  
**Submission:** ADTC 2026 — The Laptop LLM (Gate 1). See [COMPLIANCE.md](COMPLIANCE.md) for the requirements matrix, `metadata.json` field map, and scoring worksheet.

---

## Problem

Nigerian CS undergraduates at distance-learning institutions need step-by-step math help (Discrete Mathematics, Calculus I/II, Linear Algebra) but face blocked access to cloud AI: API fees in naira, unstable connectivity, and unreliable power. This submission is a fully on-device math tutor: a RAG pipeline over the GSM8K / Hendrycks MATH / OpenStax / Rosen corpus, served by llama.cpp, with zero cloud dependencies. The same laptop that runs the student's coursework runs the tutor — offline, at no marginal cost.

## Design Decisions

- **Base model:** Qwen2.5-Math-1.5B-Instruct, chosen for its math specialization (trained to reason step-by-step and emit `\boxed{}` final answers) at a size that fits comfortably in the 7 GB RAM budget with RAG + embeddings resident.
- **Quantization:** GGUF Q4_K_M. Measured peak RSS is 1.71 GB — about 24% of the 7 GB budget, leaving room for the OS, browser, and RAG index. Q8_0 was rejected: it doubles model memory (~2 GB) for negligible accuracy gain on this task; Q2_K was rejected after sample evals showed degraded multi-step reasoning.
- **Retrieval:** chromem-go vector store (in-memory + persistent), nomic-embed-text-v1.5 Q4_K_M embeddings, subdomain keyword classifier (algebra/calculus/discrete_math/geometry/probability/number_theory) that filters retrieval and selects domain-specific prompt instructions.
- **Runtime:** llama.cpp (llama-server) — CPU-only, AVX2, no GPU required, matching the ADTC Standard Laptop's integrated-graphics constraint.
- **Alternatives considered and rejected:** 7B-class models (Mistral-7B, Qwen2.5-7B) exceeded the latency/memory envelope on 8 GB integrated graphics; cloud API fallbacks were rejected by the no-cloud-dependency requirement.

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

- Target: 8 GB DDR4 RAM, integrated GPU, Intel i5 10th-12th gen / AMD Ryzen 5 3000-5000, Ubuntu 22.04 LTS.
- No GPU acceleration — pure CPU inference via llama.cpp.
- No connectivity: retrieval, embedding, and generation must all run locally.
- Data constraints: corpus must be licensed/open — GSM8K, Hendrycks MATH (MIT), OpenStax (CC BY), Rosen discrete math solutions.

## Benchmarks

Measured on the participant's development machine via the official ADTC profiler (`adtc-profiler run --mode participant`).

| Metric | Value |
|---|---|
| Machine | AMD Ryzen 9 6900HX, 29.1 GB RAM, no GPU, Arch Linux |
| Peak RAM (RSS) | 1.71 GB |
| Steady-state RAM (RSS) | 1.61 GB |
| Generation speed | 45.78 tokens/s |
| Time to first token | ~1.98 s (512-token prompt, 128-token generation) |
| CPU utilization (p99) | 56.3% |
| Core temp peak | 20.0°C (sensor read; no throttling flag) |
| Thermal throttling | None observed (`throttled: false`) |

Note: These are self-reported development benchmarks on a machine stronger than the ADTC Standard Laptop. Official scores are measured by the ADTC profiler on the standard evaluation machine. With a 15.0 t/s reference and a 7 GB RAM budget, the design has substantial headroom on both axes. Mapping to the official scoring formula (`0.5·S_acc + 0.3·S_perf + 0.2·S_eff − P_thermal`): 45.78 t/s caps S_perf at 100; 1.71 GB peak yields S_eff ≈ 75.6; no thermal penalty observed. See [COMPLIANCE.md](COMPLIANCE.md) for the full worksheet.

## Accuracy & Eval Methodology

- **Benchmarks:** eval set of 30 problems sampled (seed 42) from the Hendrycks MATH training split, scored two ways: (1) deterministic answer matching (extracts `\boxed{}` / final answer, normalizes LaTeX and whitespace, numeric fallback for equivalent forms) and (2) model-graded rubric using a local Qwen2.5-3B-Instruct judge (JSON-schema-constrained output) — all grading on-device. Latest run: **18/30 cases fully passed**; matcher and judge agreed on all failures.
- **Qualitative:** a separate 10-case set covering step-by-step tutoring quality and African-context problems (JAMB-style exam items, market-trader arithmetic), graded by the same local judge. Latest run: **6/10 passed**; the failures are actionable tutoring gaps (final answers not stated clearly, weak induction structure) rather than harness errors.
- **Tooling:** promptfoo eval harness against the local RAG server (`localhost:8082`), `adtc-profiler` for official throughput/memory/thermal measurements.

## African Use Case

The tutor is built around the Nigerian undergraduate context: JAMB/WASSCE-style exam problems, naira-denominated word problems, curriculum-aligned RAG corpus, and a zero-cost offline UX designed for shared/low-power hardware and unreliable connectivity. See the qualitative eval set (African-context cases).

## Submission Status

- **Gate 1 pending items:** screenshots / short demo clips and the 2-minute demo video — add under `docs/` before the deadline.
- **Validation:** developed on Arch Linux; run a pass on Ubuntu 22.04 LTS before the official audit.
- **Repro:** every command lives in the [README](README.md) / Makefile; `adtc-profiler` participant mode reproduces `submission.json`.
- **Compliance:** full requirements matrix and scoring worksheet in [COMPLIANCE.md](COMPLIANCE.md).
