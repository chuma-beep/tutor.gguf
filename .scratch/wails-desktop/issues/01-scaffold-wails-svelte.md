# 01: Scaffold Wails v2 Svelte shell

**What to build:** End-to-end scaffolding for the desktop shell that keeps TUI intact: `wails.json` (`name:tutor-gguf, frontend:dir=frontend, main:cmd/desktop/main.go, outputfilename:tutor-desktop`), `frontend/` Vite Svelte + TS + Tailwind, `cmd/desktop/main.go` stub running `wails.Run` with empty `App` bound, `.gitignore` `frontend/dist, node_modules, build/bin`, `Makefile` `dev-desktop`/`build-desktop` placeholders, and `AGENTS.md` still points to local issue tracker. Running `wails dev -tags desktop` opens an empty window titled Tutor.gguf without touching `cmd/tutor` or `internal/tui`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `wails.json` exists with `Svelte` frontend config and `cmd/desktop/main.go` as main
- [ ] `frontend/` builds (`npm run build` → `frontend/dist` via `base:'./'`, embedded placeholder)
- [ ] `cmd/desktop/main.go` launches Wails window; `wails dev -tags desktop` opens empty Svelte page
- [ ] `cmd/tutor` (`setup|chat|serve|index`) and `make build` (`CGO_ENABLED=0`) still pass `go vet ./... && go test ./...`
- [ ] `git status` shows `skills/` ignored, new files are opt-in tracked
