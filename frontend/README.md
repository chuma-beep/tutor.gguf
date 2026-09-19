# Frontend (Wails desktop)

The Svelte + Vite UI for the tutor.gguf desktop app. It is embedded in the Go binary through
`//go:embed`. It talks to the backend over Wails Go bindings, so there is no HTTP hop.

## Develop

```bash
npm install --legacy-peer-deps
npm run dev      # vite dev server
npm run build    # writes frontend/dist
```

The Wails build reads `frontend/dist`. See `../README.md` and `../cmd/desktop/main.go`.

## Notes

- Math is rendered with self-hosted KaTeX. No CDN is used and the CSP is `default-src 'self'`.
- The UI calls the backend through `frontend/src/lib/wails.js`.
- Main components: `ChatView`, `SetupView`, `SourcesRail` and `Turn`.
