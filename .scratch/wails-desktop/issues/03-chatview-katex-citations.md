# 03: ChatView KaTeX chat loop + citations + prompt viewer

**What to build:** The student-facing chat: Svelte `ChatView` that replicates `internal/tui/tui.go:54 Model` behavior over the `App.Ask` seam — input `maxlength 512` placeholder `Ask a math question…` + `Enter` guard (empty + `loading`), clear atomically, spinner `Dot` "thinking" + cursor `▍`, pinned `-1` overscan scroll with `PgUp/PgDn` parity, `Esc`/`Ctrl+C` quit, `Ctrl+L` clear → `localStorage` capped ~100. Rendering: reuse `extractSpans` precedence (`$$` display > `$` inline > `\(` inline > `\[` display, unclosed → math through EOF) → `segText` escaped, `segInline`/`segDisplay` via `katex.renderToString({throwOnError:false, displayMode})`, `\boxed` native. Each turn shows `Prompt Category` pill + `subdomainInstructions`, collapsible Sources `[1..3]` (`ScoredChunk{Text,Subdomain,Similarity}`), copy-to-clipboard for `answer`, and a collapsible "View prompt sent to Qwen" panel. TUI `renderer/render_test.go` goldens render equivalently via KaTeX.

**Blocked by:** 02 (needs `App.Ask` deep seam)

**Status:** ready-for-agent

- [ ] Asking "Find the derivative of x^2" shows spinner, then KaTeX `2x` with `\frac{1}{2}` etc and `Prompt Category: calculus` pill
- [ ] Sources `[1..3]` display citations sorted by Similarity, not `Collection.Query` prefix leak
- [ ] "View prompt" shows `ChatML` with retrieved context `[%d] (subdomain)` blocks
- [ ] String `malformed \frac{` or unclosed `\(\frac{2002}{` does not throw (KaTeX `throwOnError:false`, streaming tolerance)
- [ ] `localStorage` persists ~100 turns, `Clear` empties, `Esc` closes window; TUI `make tui` still identical
