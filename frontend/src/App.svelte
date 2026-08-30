<script>
  import { onMount } from 'svelte'
  import 'katex/dist/katex.min.css'

  import Header from './components/Header.svelte'
  import LeftRail from './components/LeftRail.svelte'
  import SourcesRail from './components/SourcesRail.svelte'
  import Turn from './components/Turn.svelte'
  import StreamingTurn from './components/StreamingTurn.svelte'
  import InputDock from './components/InputDock.svelte'
  import SetupPanel from './components/SetupPanel.svelte'
  import { hasWails, hasRuntime, askStream, getStatus, getPaths, retryInit, runSetup as wailsSetup, onEvent } from './lib/wails.js'

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
  let activeIndex = -1
  let numberWordsEnabled = false

  let streamCleanup = null

  function streamBackend(problem, { onMeta, onDelta, onDone, onError }) {
    if (hasWails() && hasRuntime()) {
      const offMeta = onEvent('tutor:stream:meta', (e) => onMeta && onMeta(e))
      const offChunk = onEvent('tutor:stream:chunk', (e) => onDelta && onDelta(e.content || ''))
      const offDone = onEvent('tutor:stream:done', (e) => { cleanup(); onDone && onDone(e) })
      const offErr = onEvent('tutor:stream:error', (e) => { cleanup(); onError && onError(e.error || String(e)) })
      const cleanup = () => [offMeta, offChunk, offDone, offErr].forEach((f) => f && f())
      streamCleanup = cleanup
      askStream(problem).then(() => {}).catch((e) => { cleanup(); onError && onError(String(e)) })
      return
    }
    // HTTP SSE fallback (plain vite dev)
    fetch('http://127.0.0.1:8082/v1/complete/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ problem })
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
            if (ev.subdomain || ev.category) { onMeta && onMeta({ subdomain: ev.subdomain, category: ev.category }); continue }
            if (ev.done) { onDone && onDone({ content: ev.content || '', answer: ev.answer || '' }); continue }
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
        try {
          const r = await fetch('http://127.0.0.1:8082/health')
          status = { ready: r.ok, startErr: r.ok ? '' : 'offline', dbChunks: 0, genModelExists: true, embedExists: true }
        } catch {
          status = { ready: true, startErr: '', dbChunks: 0, genModelExists: true, embedExists: true }
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

  function finalizeStream(q, { content, answer, subdomain, category }) {
    const turn = {
      question: q,
      title: '',
      answer: content || (streaming && streaming.answer) || '',
      boxed: answer || '',
      subdomain: subdomain || (streaming && streaming.subdomain) || 'other',
      category: category || (streaming && streaming.category) || 'other',
      chunks: [],
      prompt: ''
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
    loading = true
    errorMsg = ''
    streaming = { question: q, answer: '', subdomain: 'other', category: 'other' }
    let done = false
    streamBackend(q, {
      onMeta: (m) => {
        if (streaming) streaming = { ...streaming, subdomain: m.subdomain || streaming.subdomain, category: m.category || streaming.category }
      },
      onDelta: (d) => {
        if (streaming) streaming = { ...streaming, answer: streaming.answer + d }
      },
      onDone: (e) => {
        if (done) return
        done = true
        finalizeStream(q, e)
      },
      onError: (err) => {
        if (done) return
        done = true
        turns = [...turns, { question: q, title: '', error: String(err) }]
        saveHistory()
        streaming = null
        loading = false
      }
    })
  }

  function clearHistory() {
    turns = []
    activeIndex = -1
    localStorage.removeItem('tutor-history')
    if (streamCleanup) { streamCleanup(); streamCleanup = null }
  }

  function selectTurn(i) {
    activeIndex = i
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

<main class:rail-collapsed={railCollapsed} class:has-sources={activeIndex >= 0}>
  <Header
    ready={!!status && !!status.ready}
    {checking}
    {setupRunning}
    onClear={clearHistory}
    onToggleRail={() => (railCollapsed = !railCollapsed)}
    {railCollapsed}
    {numberWordsEnabled}
    onToggleNumberWords={toggleNumberWords}
  />

  {#if checking}
    <div class="center-wrap">
      <div class="checking-panel"><span class="spin" aria-hidden="true"></span> Checking setup…</div>
    </div>
  {:else if status && !status.ready}
    <div class="center-wrap"><SetupPanel {status} {setupLog} {setupRunning} {paths} {retryLoading} onRunSetup={runSetup} onRetry={retry} onRefresh={checkStatus} onCancel={cancelSetup} /></div>
  {:else}
    {#if !railCollapsed}
      <LeftRail {turns} {activeIndex} onSelectTurn={selectTurn} onRename={renameTurn} />
    {/if}

    <section class="transcript" role="log" aria-live="polite" aria-label="Math tutor conversation">
      {#each turns as t, idx}
        <Turn
          turn={t}
          showPrompt={showPromptFor === idx}
          onTogglePrompt={() => togglePrompt(idx)}
        />
      {/each}

      <StreamingTurn {streaming} />

      {#if turns.length === 0 && !streaming}
        <div class="empty">
          <p class="ghost-line">Find the derivative of x^2</p>
          <p class="ghost-sub">Ask a math question — enter to send. Shift+Enter for a new line.</p>
        </div>
      {/if}
    </section>

    {#if activeIndex >= 0 && turns[activeIndex]}
      <SourcesRail turn={turns[activeIndex]} />
    {/if}
  {/if}

  {#if status && status.ready}
    <InputDock {input} {loading} onInput={(v) => (input = v)} onSubmit={submit} onClear={clearHistory} />
  {/if}

  {#if errorMsg}
    <div class="error-msg">{errorMsg}</div>
  {/if}
</main>

<style>
  main {
    display: grid;
    grid-template-rows: 56px 1fr auto;
    grid-template-columns: 220px 1fr 320px;
    grid-template-areas:
      "header header header"
      "left center right"
      "input input input";
    min-height: 100vh;
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

  header { grid-area: header; }
  .transcript { grid-area: center; }

  .transcript {
    background: var(--slate-elev);
    border: 1px solid var(--slate-line);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    margin: 16px 20px;
    overflow-y: auto;
    padding: 20px 24px;
  }

  @media (max-width: 1120px) {
    /* Spec: right rail off by default on narrow windows. */
    main {
      grid-template-columns: 200px 1fr;
      grid-template-areas:
        "header header"
        "left center"
        "input input";
    }
    main.rail-collapsed {
      grid-template-columns: 1fr;
      grid-template-areas:
        "header"
        "center"
        "input";
    }
    .transcript { margin: 12px 14px; }
  }
  @media (max-width: 760px) {
    main {
      grid-template-columns: 1fr;
      grid-template-areas:
        "header"
        "center"
        "input";
    }
    .transcript { margin: 10px 8px; }
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
    background: var(--green);
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

  @media (max-width: 1120px) {
    main { grid-template-columns: 200px 1fr; }
    .transcript { margin: 12px 14px; }
  }
  @media (max-width: 760px) {
    main { grid-template-columns: 1fr; }
    .transcript { margin: 10px 8px; }
  }
</style>
