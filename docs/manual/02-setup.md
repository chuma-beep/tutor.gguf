---
title: Setup
order: 2
---

# Setup

`tutor setup` provisions everything the tutor needs: the llama.cpp server, both GGUF models, the corpus and the vector index. It is idempotent — re-run it any time and it skips what already exists.

```bash
tutor setup
```

## What it fetches

| Artifact | Location | Size |
|---|---|---|
| `llama-server` (llama.cpp) | `~/.tutor/bin/` | ~tens of MB |
| Qwen2.5-Math-1.5B generation model | `~/.tutor/models/` | ~1 GB |
| nomic-embed-text-v1.5 embedding model | `~/.tutor/models/` | ~100 MB |
| Corpus (GSM8K, Hendrycks MATH, Rosen) | `~/.tutor/corpus/` | varies |
| Vector index (chromem) | `~/.tutor/chromem/` | varies |
| Logs | `~/.tutor/logs/` | — |

## Moving the directory

Set `TUTOR_HOME` to relocate everything (for example to a larger drive). The remaining overrides are documented under [Configuration](config).

## No internet lab?

Download the `tutor-offline.tar.gz` USB pack once on campus Wi-Fi, unpack it on the target machine, then run `tutor setup` — it detects the pre-staged artifacts and skips downloading.

## If setup is interrupted

Just run it again (or reopen the desktop app). Downloads resume from partial files and completed phases are skipped. Lost power mid-index? Delete `~/.tutor/chromem/*.db` and re-run to rebuild the index cleanly.

## Next

Start asking questions in [Using the tutor](using).
