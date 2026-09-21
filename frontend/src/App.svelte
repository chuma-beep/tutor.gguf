<script>
  import { onMount, afterUpdate } from 'svelte'
  import 'katex/dist/katex.min.css'

  import Header from './components/Header.svelte'
  import LeftRail from './components/LeftRail.svelte'
  import SourcesRail from './components/SourcesRail.svelte'
  import Turn from './components/Turn.svelte'
  import StreamingTurn from './components/StreamingTurn.svelte'
  import InputDock from './components/InputDock.svelte'
  import SetupPanel from './components/SetupPanel.svelte'
  import DocsView from './components/DocsView.svelte'
  import { hasWails, hasRuntime, askStream, getStatus, getPaths, retryInit, runSetup as wailsSetup, onEvent } from './lib/wails.js'
  import { getTheme, cycleTheme } from './lib/theme.js'

  let turns = []
  let input = ''
  let loading = false
  let streaming = null
  let showPromptFor = null
  let errorMsg = ''
  let status = null
  let checking = true
  let retryLoading = false
  let paths = null
  let setupLog = []
  let setupRunning = false
  let railCollapsed = false
  let sourcesOpen = true
  let view = 'chat'
  let docsSlug = null
  let activeIndex = -1
  let numberWordsEnabled = false
  let theme = getTheme()

  function onCycleTheme() {
    theme = cycleTheme(theme)
  }

  let streamCleanup = null
  let streamAbort = null
  let streamIsWails = false
  let stopRequested = false

  // Scroll pinning: follow new output only while the user sits at the bottom.
  let transcriptEl = null
  let pinned = true

  function trackPin() {
    if (!transcriptEl) return
    const gap = transcriptEl.scrollHeight - transcriptEl.scrollTop - transcriptEl.clientHeight
    pinned = gap < 48
  }

  function scrollToBottom() {
    pinned = true
    if (transcriptEl) transcriptEl.scrollTop = transcriptEl.scrollHeight
  }

  afterUpdate(() => {
    if (pinned && transcriptEl) transcriptEl.scrollTop = transcriptEl.scrollHeight
  })

  const examples = [
    'Find the derivative of x^2.',
    'Prove by induction 1 + 2 + ... + n = n(n+1)/2.',
    'Solve 2x + y = 7, x - y = 2 using matrix row reduction.',
    'A Lagos water tank has circumference 66 m. Using π = 22/7, find the radius.'
  ]

  function streamBackend(problem, { onMeta, onDelta, onDone, onError }) {
    if (hasWails() && hasRuntime()) {
      streamIsWails = true
      const offMeta = onEvent('tutor:stream:meta', (e) => onMeta && onMeta(e))
      const offChunk = onEvent('tutor:stream:chunk', (e) => onDelta && onDelta(e.content || ''))
      const offDone = onEvent('tutor:stream:done', (e) => { cleanup(); onDone && onDone({ content: e.content || '', answer: e.answer || '', answerRule: e.answer_rule || e.answerRule || 'none' }) })
      const offErr = onEvent('tutor:stream:error', (e) => { cleanup(); onError && onError(e.error || String(e)) })
      const cleanup = () => [offMeta, offChunk, offDone, offErr].forEach((f) => f && f())
      streamCleanup = cleanup
      streamAbort = cleanup
      askStream(problem).then(() => {}).catch((e) => { cleanup(); onError && onError(String(e)) })
      return
    }
    // HTTP fallback (plain vite dev, or browser UI served by `tutor serve`).
    // Relative URLs: same-origin under `tutor serve`, proxied to :8082 in vite dev.
    const controller = new AbortController()
    streamIsWails = false
    streamAbort = () => controller.abort()
    fetch('/v1/complete/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ problem }),
      signal: controller.signal
    }).then(async (resp) => {
      if (!resp.ok) throw new Error(`server error ${resp.status}`)
      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      let buf = ''
      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buf += decoder.decode(value, { stream: true })
        let idx
        while ((idx = buf.indexOf('\n\n')) >= 0) {
          const block = buf.slice(0, idx)
          buf = buf.slice(idx + 2)
          for (const line of block.split('\n')) {
            const l = line.trim()
            if (!l.startsWith('data:')) continue
            const payload = l.slice(5).trim()
            if (payload === '[DONE]') continue
            let ev
            try { ev = JSON.parse(payload) } catch { continue }
            if (ev.error) { onError && onError(ev.error); continue }
            if (ev.subdomain || ev.category || ev.chunks || ev.prompt) { onMeta && onMeta(ev); continue }
            if (ev.done) { onDone && onDone({ content: ev.content || '', answer: ev.answer || '', answerRule: ev.answer_rule || ev.answerRule || 'none' }); continue }
            if (ev.content) onDelta && onDelta(ev.content)
          }
        }
      }
      onDone && onDone({ content: '', answer: '' })
    }).catch((e) => onError && onError(String(e)))
  }

  function saveHistory() {
    try { localStorage.setItem('tutor-history', JSON.stringify(turns.slice(-100))) } catch {}
  }
  function loadHistory() {
    try {
      const raw = localStorage.getItem('tutor-history')
      if (raw) turns = JSON.parse(raw)
    } catch {}
  }

  function onSetupProgress(e) {
    setupLog = [...setupLog.slice(-200), e]
  }

  let autoSetupFired = false

  onMount(async () => {
    loadHistory()
    // Ctrl+L clears the transcript (matches the Clear button tooltip).
    // Esc closes overlay rails on narrow screens (ignored while typing).
    const onKey = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'l') {
        e.preventDefault()
        clearHistory()
      }
      if (e.key === 'Escape') {
        const tag = (document.activeElement && document.activeElement.tagName) || ''
        if (tag === 'TEXTAREA' || tag === 'INPUT') return
        railCollapsed = true
        sourcesOpen = false
      }
    }
    window.addEventListener('keydown', onKey)
    // Restore number-words toggle
    try {
      const saved = localStorage.getItem('tutor-number-words')
      if (saved !== null) numberWordsEnabled = saved === 'true'
      if (hasWails()) {
        try {
          const remote = await window.go.desktop.App.GetNumberWords()
          numberWordsEnabled = remote
        } catch {}
        // Apply to backend
        try { await window.go.desktop.App.SetNumberWords(numberWordsEnabled) } catch {}
      }
    } catch {}
    if (hasRuntime()) {
      onEvent('tutor:setup:progress', onSetupProgress)
      onEvent('tutor:setup:error', (e) => { setupRunning = false; setupLog = [...setupLog, { phase: 'error', message: e.error || String(e) }] })
      onEvent('tutor:setup:done', async () => { setupRunning = false; await checkStatus() })
    }
    await checkStatus()
    // Auto-provision on first launch: no terminal, no "Run Setup" click.
    if (hasWails() && status && !status.ready && !autoSetupFired) {
      autoSetupFired = true
      setTimeout(() => { if (status && !status.ready && !setupRunning) runSetup() }, 1000)
    }
  })

  function toggleNumberWords() {
    numberWordsEnabled = !numberWordsEnabled
    try { localStorage.setItem('tutor-number-words', String(numberWordsEnabled)) } catch {}
    if (hasWails()) {
      try { window.go.desktop.App.SetNumberWords(numberWordsEnabled) } catch {}
    }
  }

  async function runSetup() {
    setupRunning = true
    setupLog = []
    try {
      if (hasWails()) {
        await wailsSetup(false, false, false)
      } else {
        setupLog = [{ phase: 'info', message: 'Not under Wails — run `tutor setup` in a terminal, then refresh.' }]
        setupRunning = false
      }
    } catch (e) {
      setupRunning = false
      setupLog = [...setupLog, { phase: 'error', message: String(e) }]
    }
  }

  function cancelSetup() {
    try {
      if (hasWails() && window.go.desktop.App.CancelSetup) {
        window.go.desktop.App.CancelSetup()
      }
    } catch {}
    setupRunning = false
    setupLog = [...setupLog, { phase: 'info', message: 'Setup paused — resumes on next launch.' }]
  }

  async function checkStatus() {
    checking = true
    try {
      if (hasWails()) {
        status = await getStatus()
        try { paths = await getPaths() } catch {}
      } else {
        // Browser UI (served by `tutor serve` or vite dev proxy): ask the
        // backend for its status. Relative URL = same origin in both cases.
        try {
          let r = await fetch('/v1/status')
          if (!r.ok) r = await fetch('/health')
          if (!r.ok) throw new Error(`backend error ${r.status}`)
          const s = await r.json().catch(() => ({}))
          status = {
            ready: s.ready ?? true,
            startErr: s.ready === false ? 'backend not ready' : '',
            dbChunks: s.dbChunks ?? 0,
            genModelExists: true,
            embedExists: true
          }
        } catch {
          status = { ready: false, startErr: 'backend offline — start `tutor serve`, then refresh.', dbChunks: 0 }
        }
      }
    } catch (e) {
      status = { ready: false, startErr: String(e), dbChunks: 0 }
    } finally {
      checking = false
    }
  }
  async function retry() {
    retryLoading = true
    try {
      if (hasWails()) status = await retryInit()
      else await checkStatus()
    } catch (e) {
      errorMsg = String(e)
    } finally {
      retryLoading = false
    }
  }

  function finalizeStream(q, { content, answer, answerRule, subdomain, category }) {
    const turn = {
      question: q,
      title: '',
      answer: content || (streaming && streaming.answer) || '',
      boxed: answer || '',
      answerRule: answerRule || 'none',
      subdomain: subdomain || (streaming && streaming.subdomain) || 'other',
      category: category || (streaming && streaming.category) || 'other',
      chunks: (streaming && streaming.chunks) || [],
      prompt: (streaming && streaming.prompt) || ''
    }
    turns = [...turns, turn]
    saveHistory()
    activeIndex = turns.length - 1
    streaming = null
    loading = false
  }

  function submit() {
    const q = input.trim()
    if (!q || loading) return
    input = ''
    askQuestion(q)
  }

  function askQuestion(q) {
    if (!q || loading) return
    loading = true
    errorMsg = ''
    stopRequested = false
    streaming = { question: q, answer: '', subdomain: 'other', category: 'other', chunks: [], prompt: '' }
    scrollToBottom()
    let done = false
    streamBackend(q, {
      onMeta: (m) => {
        if (streaming) streaming = { ...streaming, subdomain: m.subdomain || streaming.subdomain, category: m.category || streaming.category, chunks: m.chunks || streaming.chunks, prompt: m.prompt || streaming.prompt }
      },
      onDelta: (d) => {
        if (streaming) streaming = { ...streaming, answer: streaming.answer + d }
      },
      onDone: (e) => {
        if (done) return
        done = true
        streamAbort = null
        finalizeStream(q, e)
      },
      onError: (err) => {
        if (done) return
        done = true
        streamAbort = null
        // Intentional Stop keeps the partial answer instead of an error turn.
        if (stopRequested) {
          stopRequested = false
          const partial = streaming && streaming.answer
          if (partial) finalizeStream(q, { content: partial, answer: '' })
          else {
            streaming = null
            loading = false
          }
          return
        }
        turns = [...turns, { question: q, title: '', error: String(err) }]
        saveHistory()
        activeIndex = turns.length - 1
        streaming = null
        loading = false
      }
    })
  }

  function stopStream() {
    if (!loading || !streaming) return
    stopRequested = true
    if (streamIsWails) {
      // Wails has no server-side cancel: detach listeners, keep the partial.
      try { streamAbort && streamAbort() } catch {}
      try { streamCleanup && streamCleanup() } catch {}
      streamAbort = null
      streamCleanup = null
      const q = streaming.question
      const partial = streaming.answer
      stopRequested = false
      if (partial) finalizeStream(q, { content: partial, answer: '' })
      else {
        streaming = null
        loading = false
      }
    } else {
      try { streamAbort && streamAbort() } catch {}
      // The abort surfaces through onError above, which finalizes the
      // partial. Safety net in case the fetch never settles:
      setTimeout(() => {
        if (streaming && stopRequested) {
          stopRequested = false
          const q = streaming.question
          const partial = streaming.answer
          if (partial) finalizeStream(q, { content: partial, answer: '' })
          else {
            streaming = null
            loading = false
          }
        }
      }, 1000)
    }
  }

  function clearHistory() {
    turns = []
    activeIndex = -1
    showPromptFor = null
    localStorage.removeItem('tutor-history')
    if (streamCleanup) { streamCleanup(); streamCleanup = null }
  }

  async function copyText(text) {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      errorMsg = 'Copy failed — select the text and copy manually.'
      setTimeout(() => { errorMsg = '' }, 2500)
    }
  }

  function deleteTurn(i) {
    turns = turns.filter((_, idx) => idx !== i)
    if (showPromptFor === i) showPromptFor = null
    else if (showPromptFor !== null && showPromptFor > i) showPromptFor -= 1
    if (activeIndex === i) activeIndex = -1
    else if (activeIndex > i) activeIndex -= 1
    saveHistory()
  }

  function retryTurn(i) {
    const t = turns[i]
    if (!t || loading) return
    askQuestion(t.question)
  }

  function askExample(q) {
    if (loading) return
    input = ''
    askQuestion(q)
  }

  function selectTurn(i) {
    activeIndex = i
    // On narrow screens the rails are overlays: selecting a turn returns to
    // the transcript instead of covering it.
    if (window.matchMedia('(max-width: 1120px)').matches) {
      railCollapsed = true
      sourcesOpen = false
    }
  }

  function renameTurn(i, title) {
    if (title && turns[i]) {
      turns = turns.map((t, idx) => (idx === i ? { ...t, title } : t))
      saveHistory()
    }
  }

  function togglePrompt(i) {
    showPromptFor = showPromptFor === i ? null : i
  }
</script>

<main class:rail-collapsed={railCollapsed} class:has-sources={activeIndex >= 0 && sourcesOpen}>
  <Header
    ready={!!status && !!status.ready}
    {checking}
    {setupRunning}
    onClear={clearHistory}
    onToggleRail={() => (railCollapsed = !railCollapsed)}
    {railCollapsed}
    onToggleSources={() => (sourcesOpen = !sourcesOpen)}
    {sourcesOpen}
    {numberWordsEnabled}
    onToggleNumberWords={toggleNumberWords}
    {theme}
    onCycleTheme={onCycleTheme}
    {view}
    onToggleView={() => (view = view === 'docs' ? 'chat' : 'docs')}
  />

  {#if view === 'docs'}
    <DocsView slug={docsSlug} onNavigate={(s) => (docsSlug = s)} />
  {:else if checking}
    <div class="center-wrap">
      <div class="checking-panel"><span class="spin" aria-hidden="true"></span> Checking setup…</div>
    </div>
  {:else if status && !status.ready}
    <div class="center-wrap"><SetupPanel {status} {setupLog} {setupRunning} {paths} {retryLoading} onRunSetup={runSetup} onRetry={retry} onRefresh={checkStatus} onCancel={cancelSetup} /></div>
  {:else}
    {#if !railCollapsed}
      <LeftRail {turns} {activeIndex} onSelectTurn={selectTurn} onRename={renameTurn} onDelete={deleteTurn} />
    {/if}

    <section class="transcript" role="log" aria-live="polite" aria-label="Math tutor conversation" bind:this={transcriptEl} on:scroll={trackPin}>
      <div class="record-head">
        <div>
          <p class="archive-label">Index / TG-001 · Local session</p>
          <h2 class="record-title">tutor.gguf <span>Archive</span></h2>
        </div>
        <dl class="facts">
          <div><dt>Status</dt><dd>{status && status.ready ? 'Ready' : setupRunning ? 'Setting up' : 'Not ready'}</dd></div>
          <div><dt>Examples</dt><dd>{status && status.dbChunks != null ? status.dbChunks + ' indexed' : '—'}</dd></div>
          <div><dt>Model</dt><dd>Qwen2.5-Math-1.5B</dd></div>
          <div><dt>Runtime</dt><dd>CPU · Offline</dd></div>
        </dl>
      </div>

      {#each turns as t, idx}
        <Turn
          turn={t}
          index={idx}
          showPrompt={showPromptFor === idx}
          onTogglePrompt={() => togglePrompt(idx)}
          onCopy={(text) => copyText(text)}
          onRetry={() => retryTurn(idx)}
          onDelete={() => deleteTurn(idx)}
        />
      {/each}

      <StreamingTurn {streaming} />

      {#if turns.length === 0 && !streaming}
        <div class="empty">
          <p class="ghost-line">Find the derivative of x^2</p>
          <p class="ghost-sub">Ask a math question — Enter to send, Shift+Enter for a new line.</p>
          <div class="examples">
            {#each examples as q}
              <button class="example" on:click={() => askExample(q)}>{q}</button>
            {/each}
          </div>
        </div>
      {/if}

      <footer class="record-foot">
        <span>Archive reference / TG-001-LOCAL</span>
        <span>Qwen2.5-Math-1.5B · llama.cpp · chromem</span>
        <span>100% offline</span>
      </footer>

      {#if !pinned && (turns.length > 0 || streaming)}
        <button class="latest" on:click={scrollToBottom} title="Jump to latest output">↓ Latest</button>
      {/if}
    </section>

    {#if activeIndex >= 0 && turns[activeIndex] && sourcesOpen}
      <div class="sources-wrap" class:open={sourcesOpen}>
        <SourcesRail turn={turns[activeIndex]} onCopy={(text) => copyText(text)} />
      </div>
    {/if}
    {#if (!railCollapsed || (sourcesOpen && activeIndex >= 0 && turns[activeIndex])) && status && status.ready}
      <button class="scrim" aria-label="Close side panels" on:click={() => { railCollapsed = true; sourcesOpen = false; }}></button>
    {/if}
  {/if}

  {#if status && status.ready}
    <InputDock {input} {loading} onInput={(v) => (input = v)} onSubmit={submit} onClear={clearHistory} onStop={stopStream} />
  {/if}

  {#if errorMsg}
    <div class="error-msg">{errorMsg}</div>
  {/if}
</main>

<style>
  main {
    display: grid;
    grid-template-rows: auto minmax(0, 1fr) auto;
    grid-template-columns: 220px minmax(0, 1fr) 320px;
    grid-template-areas:
      "header header header"
      "left center right"
      "input input input";
    height: 100vh;
    background: var(--slate);
    color: var(--chalk);
  }

  /* Hide history: left rail collapses, center + right stay put. */
  main.rail-collapsed {
    grid-template-columns: 1fr 320px;
    grid-template-areas:
      "header header"
      "center right"
      "input input";
  }
  /* Hide history AND no active sources: center takes the full width. */
  main.rail-collapsed:not(.has-sources) {
    grid-template-columns: 1fr;
    grid-template-areas:
      "header"
      "center"
      "input";
  }
  /* No active sources but history shown: left + center only. */
  main:not(.rail-collapsed):not(.has-sources) {
    grid-template-columns: 220px 1fr;
    grid-template-areas:
      "header header"
      "left center"
      "input input";
  }

  .transcript { grid-area: center; }

  .transcript {
    background: var(--slate-elev);
    border: 1px solid var(--slate-line);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    margin: clamp(8px, 2.5vw, 20px);
    overflow-y: auto;
    padding: clamp(12px, 3vw, 24px);
  }

  /* Overlay rails + scrim: hidden on desktop, slide-overs on narrow. */
  .sources-wrap { display: contents; }
  .scrim { display: none; }

  @media (max-width: 1120px) {
    /* Single column: transcript owns the scroll region, rails overlay it. */
    main,
    main.rail-collapsed,
    main:not(.rail-collapsed):not(.has-sources) {
      grid-template-columns: minmax(0, 1fr);
      grid-template-areas:
        "header"
        "center"
        "input";
    }
    .transcript { margin: clamp(8px, 2vw, 14px); }
    .sources-wrap { display: none; }
    .sources-wrap.open {
      display: block;
      position: fixed;
      top: 56px;
      bottom: 0;
      right: 0;
      width: min(340px, 90vw);
      z-index: 20;
      background: var(--slate-elev);
      border-left: 1px solid var(--slate-line2);
      box-shadow: var(--shadow);
      overflow-y: auto;
    }
    .sources-wrap.open > :global(aside) {
      border: none;
      height: 100%;
    }
    .scrim {
      display: block;
      position: fixed;
      top: 56px;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 15;
      background: rgba(0, 0, 0, 0.35);
      border: none;
      padding: 0;
      cursor: default;
    }
  }
  @media (max-width: 760px) {
    .transcript { margin: 8px; padding: 12px; }
    .record-foot { margin: 24px -12px -12px; padding: 14px 12px; }
  }

  .center-wrap {
    grid-column: 1 / -1;
    display: flex;
    align-items: flex-start;
    justify-content: center;
  }

  .checking-panel {
    display: flex;
    align-items: center;
    gap: 10px;
    font: 400 13px/1 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    margin: 40px;
  }
  .spin {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--chalk-faint);
    animation: pulse 1.2s ease infinite;
  }
  @keyframes pulse { 50% { opacity: 0.35; } }

  .empty { text-align: center; padding: 40px 0; }
  .ghost-line {
    font: 400 15px/1 'JetBrains Mono', monospace;
    color: var(--chalk-faint);
    margin: 0 0 10px;
  }
  .ghost-sub { font: 400 12.5px/1 'Inter', sans-serif; color: var(--chalk-faint); margin: 0; }

  .examples {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: center;
    margin-top: 20px;
  }
  .example {
    background: transparent;
    border: 1px solid var(--slate-line2);
    color: var(--chalk-muted);
    border-radius: var(--radius-sm);
    padding: 10px 14px;
    font: 400 12px/1.4 'JetBrains Mono', monospace;
    cursor: pointer;
    max-width: 320px;
    transition: border-color 160ms ease, color 160ms ease;
  }
  .example:hover { border-color: var(--slate-line2); color: var(--chalk-bright); }

  .transcript { position: relative; }

  /* Archive record language (mirrors site/src/routes/index.tsx). */
  .archive-label {
    font: 400 10px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--chalk-faint);
    margin: 0;
  }
  .record-head {
    display: grid;
    gap: 20px;
    border-bottom: 1px solid var(--slate-line2);
    padding: 8px 4px 20px;
    margin-bottom: 8px;
  }
  @media (min-width: 900px) {
    .record-head { grid-template-columns: 1fr 320px; align-items: start; }
  }
  .record-title {
    font: 500 clamp(28px, 5vw, 40px)/0.95 'Inter', sans-serif;
    letter-spacing: -0.01em;
    text-transform: uppercase;
    color: var(--chalk-bright);
    margin: 12px 0 0;
  }
  .record-title span { color: var(--chalk-faint); }
  .facts {
    display: grid;
    grid-template-columns: 1fr 1fr;
    margin: 0;
    border-top: 1px solid var(--slate-line2);
    border-left: 1px solid var(--slate-line2);
    font: 400 10px/1.5 'JetBrains Mono', monospace;
    text-transform: uppercase;
  }
  .facts > div {
    border-bottom: 1px solid var(--slate-line2);
    border-right: 1px solid var(--slate-line2);
    padding: 10px 12px;
  }
  .facts dt { color: var(--chalk-faint); margin: 0; }
  .facts dd { color: var(--chalk-bright); margin: 4px 0 0; }
  .record-foot {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 20px;
    justify-content: space-between;
    border-top: 1px solid var(--slate-line2);
    background: var(--slate);
    margin: 24px -24px -20px;
    padding: 14px 24px;
    font: 400 10px/1 'JetBrains Mono', monospace;
    text-transform: uppercase;
    color: var(--chalk-faint);
  }
  .latest {
    position: sticky;
    bottom: 12px;
    float: right;
    background: var(--slate-elev);
    border: 1px solid var(--slate-line2);
    color: var(--chalk-bright);
    border-radius: 99px;
    padding: 8px 14px;
    font: 600 12px/1 'Inter', sans-serif;
    cursor: pointer;
    box-shadow: var(--shadow);
  }

  .error-msg {
    position: fixed;
    bottom: 12px;
    left: 50%;
    transform: translateX(-50%);
    background: var(--slate-elev);
    border: 1px solid var(--error);
    color: var(--error);
    font: 400 12px/1 'JetBrains Mono', monospace;
    padding: 8px 14px;
    border-radius: var(--radius-sm);
    z-index: 10;
  }

</style>
