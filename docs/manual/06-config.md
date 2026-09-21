---
title: Configuration
order: 6
---

# Configuration

## Tutor home

`TUTOR_HOME` (default `~/.tutor`) roots `models/`, `bin/llama-server`, `corpus/`, the chromem DB and `logs/`.

## Environment overrides

| Variable | Effect |
|---|---|
| `TUTOR_HOME` | Relocate all artifacts |
| `TUTOR_LLAMA_SERVER` | Use a specific llama-server binary |
| `TUTOR_THREADS` | Generation threads (default: auto) |
| `TUTOR_CTX` | Prompt context size (default: 2048) |
| `TUTOR_DB_PATH` | chromem DB path (overrides auto-resolution) |

## Serve flags

`tutor serve -port` sets the HTTP port (default `8082`). The Makefile pins `8080`/`8081`/`8083` for hand-run gen/embed/judge servers — change the `*_PORT` vars on conflict. Managed mode picks random free ports for its own llama-servers, so only `:8082` is stable.

## Models

| File | Role |
|---|---|
| `qwen2.5-math-1.5b-instruct-q4_k_m.gguf` | Generation (the scored model) |
| `nomic-embed-text-v1.5.Q4_K_M.gguf` | Retrieval embeddings |
| `qwen2.5-3b-instruct-q4_k_m.gguf` | Eval judge only, never inference |

All GGUF `Q4_K_M` through llama.cpp only — a competition rule, not just a preference.

## Corpus

Git-ignored under `data/raw/`: Hendrycks MATH JSONs, `gsm8k/train.jsonl`, Rosen solutions, OpenStax PDFs. The chunker maps each format to homogeneous units (problem plus solution text, subdomain/source/level metadata); queries embed with the `search_query` prefix, documents with `search_document`.
