# Tutor.gguf

> On-device math tutor for African CS students — runs fully offline on an 8 GB laptop.

**ADTC 2026 · Math & Scientific Reasoning · chuma-beep**

---

## What it does

Tutor.gguf is a terminal-native math tutoring assistant that runs entirely on commodity hardware with no internet connection. Ask it a question in Discrete Mathematics, Calculus, or Linear Algebra and it walks you through the solution step by step.

It is built for students at distance-learning institutions like NOUN Awka — where stable internet is unreliable, cloud AI subscriptions are unaffordable, and a working 8 GB laptop is the only tool available.

---

## Domains covered

| Domain | Scope |
|---|---|
| Discrete Mathematics | Proofs, combinatorics, graph theory, logic, recurrences |
| Calculus I & II | Limits, derivatives, integrals, series |
| Linear Algebra | Matrices, systems of equations, eigenvalues, vector spaces |

---

## Stack

| Component | Technology |
|---|---|
| Inference engine | llama.cpp (mandatory per ADTC rules) |
| Model | Qwen2.5-Math-1.5B-Instruct — Q4_K_M quantization |
| Orchestration | Go |
| RAG pipeline | chromem-go + nomic-embed-text |
| Corpus | GSM8K, Hendrycks MATH, OpenStax Calculus, OpenStax Linear Algebra, Rosen's Discrete Math |
| UI | Terminal-native TUI (Bubble Tea) |
| Math rendering | Custom LaTeX → ASCII renderer |

---

## Requirements

- Linux (Ubuntu 22.04 LTS on reference hardware)
- 8 GB RAM minimum
- No GPU required — CPU-only inference
- No internet connection required after setup

---

## Setup

**1. Download the model**
```bash
bash download_model.sh
```

**2. Build the binary**
```bash
go build -o tutor ./cmd/tutor
```

**3. Start llama-server**
```bash
llama-server -m model/qwen2.5-math-1.5b-instruct-q4_k_m.gguf --port 8080
```

**4. Run Tutor.gguf**
```bash
./tutor
```

---

## Benchmarks

> Measured via adtc-profiler in Docker audit mode (`--memory=7.5g --cpus=4`)

| Metric | Value |
|---|---|
| Tokens per second | TBD (Phase 1) |
| Peak RAM | TBD (Phase 1) |
| Thermal | TBD (Phase 1) |

---

## Project structure

```
tutor.gguf/
├── cmd/tutor/          ← entry point
├── internal/
│   ├── llm/            ← llama-server HTTP client
│   ├── rag/            ← chromem-go retrieval pipeline
│   ├── prompt/         ← PromptBuilder
│   ├── parser/         ← ResponseParser
│   └── renderer/       ← LaTeX → ASCII renderer
├── corpus/             ← raw corpus files (not committed)
├── model/              ← model weights (not committed)
├── metadata.json       ← ADTC submission metadata
├── download_model.sh   ← model download script
└── REPORT.md           ← technical writeup
```

---

## ADTC submission

- **Domain:** `math_scientific_reasoning`
- **Runtime:** `llama.cpp`
- **Quantization:** `GGUF Q4_K_M`
- **Parameters:** 1.5B
- **Packaging:** `binary_bundle`

---

## Author

Wisdom Anwaegbu — [@chuma-beep](https://github.com/chuma-beep)  
300-level CS · NOUN Awka · Africa Deep Tech Challenge 2026
