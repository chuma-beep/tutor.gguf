# 04: SetupView background non-blocking with progress

**What to build:** First-launch provisioning as background, not blocking: `SetupView` that checks `indexedChunkCount` + `GenModelPath`/`EmbedModelPath` existence; if ready, enables Chat immediately while setup can re-run on demand; if not, shows 8-phase progress (dirs → `ensureLlamaServer` b10612 `ubuntu-x64` etc via `llamaAssetSuffix` → `ensureGGUF` gen 1.1 GB + embed 90 MB → `ensureGSM8K` → `ensureHendrycks` 7 configs paginated `datasets-server` → `ensureRosen` `assets.Rosen()` → `runSetupIndex` `ModeEmbedOnly` 1-concurrency) streamed via `EventsEmit("setup:progress", {phase,downloaded,total})` with cancel (`context.WithCancel` → `mgr.Stop()`, `signal.NotifyContext` analogue for Wails `OnBeforeClose`), resume idempotent (`.part` atomic, `Stat>0` skip), sentinel `~/.tutor/.setup-complete`, and `TUTOR_LLAMA_TAG` override. Settings shows `TUTOR_THREADS` (0=auto, 4 optimal per `tuning.md`) + `TUTOR_CTX` (2048) sliders with Advanced hidden (`TUTOR_HOME`, `TUTOR_LLAMA_SERVER` → "Open Config Folder").

**Blocked by:** 02 (needs `App` lifecycle + `TutorHome`/`DBPath`)

**Status:** ready-for-agent

- [ ] Fresh `~/.tutor` shows SetupView progress per phase; existing `~/.tutor` with chunks skips to ChatView instantly
- [ ] Progress emits per GGUF/config/index chunk; Cancel stops downloads and kills `embed` child without corrupting `.part`
- [ ] Re-running SetupView is no-op when all `Stat>0` + `col.Count>0` (no network)
- [ ] After setup completes, `tutor chat` and `tutor serve` share same `~/.tutor` DB without re-index
- [ ] Offline guard: when sentinel exists, no fetch to `huggingface.co`/`github.com/ggml-org`/`datasets-server` during `adtc-profiler` window
