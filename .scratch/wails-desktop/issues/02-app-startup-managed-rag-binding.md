# 02: App.Startup Managed RAG direct binding + health

**What to build:** The deep module behind a small `Ask` interface: `internal/desktop/app.go` `App{ctx,mgr,db,collection,retriever,genClient}` where `Startup(ctx)` does `startManaged(ctx)` (freePort dynamic, `waitHealthy` 300ms/5m, `LogsDir` `gen.log`/`embed.log`, `DiscoverLlamaServer` precedence `TUTOR_LLAMA_SERVER > ~/.tutor/bin > $PATH`) → `chromem.NewPersistentDB(DBPath)` (`resolveDBPath` precedence `flag > TUTOR_DB_PATH > ./data/chromem > ~/.tutor/chromem`) → `NewRetriever` (`search_query` prefix, `QueryEmbedding topK*4→topN 3`, fallback `other`) → `NewClient(genURL)`; `Ask(ctx, problem) → {content,answer,subdomain,category,chunks,prompt}` via `BuildPrompt` + `parse.Extract` + `PromptCategory`; `Shutdown` → `mgr.Stop()`; plus `GetPaths`/`GetStatus` (`TutorHome`, `DBPath`, `ModelsDir`). Add minimal `GET /health` in `serve.go` for `wails dev` proxy. Frontend can `invoke("Ask","find derivative of x^2")` and receive `content` without HTTP hop, while `POST /v1/complete` still serves `evals/promptfooconfig.yaml`.

**Blocked by:** 01 (needs `wails.json` + `cmd/desktop` + Svelte build)

**Status:** ready-for-agent

- [ ] `App.Startup` starts gen+embed `llama-server` and opens DB/collection without editing `cmd/tutor`
- [ ] `App.Ask` returns `{content,answer (`\boxed` → fallback), subdomain, category, chunks[3], prompt}` verified against `tp_001` `tp_002` in `metadata.json`
- [ ] `GET /health` responds 200 on both RAG handler and managed loopback
- [ ] `make eval` against `localhost:8082` still passes (HTTP seam not broken)
- [ ] `App.Shutdown` kills children, re-running `Startup` is idempotent
