# 05: Build, CI, and release distribution for desktop

**What to build:** Production distribution that keeps headless evaluatable: `Makefile` `build-desktop: wails build -clean -tags desktop -ldflags "-s -w" -platform linux/amd64,windows/amd64,darwin/universal` (Linux `CGO_ENABLED=1` + `webkit2gtk-4.0` prereq), `dev-desktop: wails dev -tags desktop -frontendDevUrl http://localhost:34115`, `frontend/dist` via `//go:embed all:frontend/dist` (`base:'./'`, `getAsset`); `.gitignore` `frontend/dist`/`frontend/node_modules`/`build/bin` preserved; `README.md` desktop install section (`.deb/.AppImage`/`.dmg`/`nsis` vs `install.sh:9 curl|bash` for headless); `release.yml` parallel `desktop` job (`setup-node`, `wails` install, `CGO_ENABLED=1`, `wails build`), merge artifacts into `SHA256SUMS` with `tutor-*` headless; `adtc-profiler run --mode participant` still PASS with desktop closed (WebView not measured); docs `docs/screenshots/` gets desktop ChatView + SetupView captures in addition to TUI shots.

**Blocked by:** 03, 04 (needs ChatView + SetupView demoable)

**Status:** ready-for-agent

- [ ] `wails build` on linux produces `build/bin/tutor-desktop` that `go vet ./... && go test ./...` still pass and `make build` headless still `CGO_ENABLED=0` stripped
- [ ] `release.yml` desktop job publishes `tutor-desktop-linux-amd64`/`tutor-desktop-darwin-universal`/`tutor-desktop-windows-amd64.exe` + `SHA256SUMS`
- [ ] `wails dev` hot reloads Svelte, no `CORS` error, `GET /health` responds on loopback
- [ ] Fresh Ubuntu 22.04 VM installs `.deb` and runs offline with `peak RSS` headroom intact (`~6 GB` left)
- [ ] `README.md` + `COMPLIANCE.md` note webview `webkit2gtk-4.1` vs `4.0` pin and CSP `default-src 'self'` with KaTeX self-hosted (no CDN)
