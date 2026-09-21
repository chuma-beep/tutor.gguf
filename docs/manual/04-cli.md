---
title: CLI reference
order: 4
---

# CLI reference

`tutor <command> [flags]`. Run `tutor <command> -h` for exact flags; this page summarizes intent.

## setup

Download models and corpus, build the index. Idempotent. See [Setup](setup).

## chat

Interactive Bubble Tea shell. Flags:

- `-tutor-url` — RAG server base URL. Omit to start the whole stack (llama-servers plus in-process server) automatically; the shell tears it down on exit.
- `-ascii` — render math with ASCII fallbacks instead of Unicode.

## serve

Run the RAG HTTP server (retrieval plus generation) and the browser UI. Flags:

- `-port` — HTTP port (default `8082`).
- `-gen-url`, `-embedder-url` — external llama-servers. Pass both together to attach to existing servers, or neither to auto-start local ones.
- `-db-path` — chromem vector store path (default `./data/chromem` in a checkout, else `~/.tutor/chromem`).

With no URL flags it supervises local llama-servers and serves the UI at `/`, the API under `/v1/`.

## index

Ingest corpus sources into the vector store, then run a test query. Useful for inspecting retrieval and the built prompt without a running server:

```bash
tutor index -query "Find the derivative of x^2"
```

Re-run any time to refresh the index after corpus changes.
