---
title: Troubleshooting
order: 8
---

# Troubleshooting

## Port conflicts

`EADDRINUSE` on `:8082` means another server holds the port (check `ss -ltnp | grep 8082`). Either stop it or serve elsewhere: `tutor serve -port 8099`. The Makefile's hand-run ports (`8080`/`8081`/`8083`) move via `*_PORT` vars.

## Backend offline

The browser UI reports "backend offline — start `tutor serve`, then refresh" when nothing answers `/v1/status`. In vite dev (`npm run dev` in `frontend/`), API calls proxy to `:8082` — start the server first.

## No chunks indexed

The corpus is git-ignored and never downloaded by `download_model.sh`. Confirm sources exist under `data/raw/` (or `~/.tutor/corpus/`), then re-run indexing. `tutor index -query ...` with no corpus prints retrieval diagnostics against the existing store.

## Stale store

`data/chromem/*.db` (or `~/.tutor/chromem/`) is generated. Delete it and re-index if results look stale after corpus changes.

## Judge / gen / embed mixups

The embedding server must run with `--embeddings`. Never point the tutor at the judge port (`:8083`) — generation and grading models serve different endpoints. Managed mode (`tutor serve` / `tutor chat` with no URLs) wires this correctly on its own.

## Setup resumes, not restarts

Interrupted downloads keep `.part` files and completed phases are skipped — just run `tutor setup` again. If the desktop app loses power mid-setup, reopen it; it resumes.

## Still stuck?

File an issue with the command run, the last log lines (`~/.tutor/logs/`), and your OS/CPU/RAM: `https://github.com/chuma-beep/tutor.gguf/issues/new`.
