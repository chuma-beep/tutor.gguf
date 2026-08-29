<script>
  export let status
  export let setupLog = []
  export let setupRunning = false
  export let paths = null
  export let retryLoading = false
  export let onRunSetup
  export let onRetry
  export let onRefresh
  export let onCancel

  // Latest byte progress from tutor:setup:progress events.
  $: current = setupLog[setupLog.length - 1] || {}
  $: pct = current && current.total > 0
    ? Math.round((current.downloaded / current.total) * 100)
    : 0
  $: hasProgress = current && current.total > 0

  function human(n) {
    if (!n) return '0 B'
    if (n >= 1 << 30) return (n / (1 << 30)).toFixed(2) + ' GB'
    if (n >= 1 << 20) return (n / (1 << 20)).toFixed(1) + ' MB'
    if (n >= 1 << 10) return (n / (1 << 10)).toFixed(1) + ' KB'
    return n + ' B'
  }

  const STEPS = ['llama', 'gen-model', 'embed-model', 'gsm8k', 'hendrycks', 'rosen', 'index']
  $: donePhases = new Set(setupLog.map((e) => e.phase).filter((p) => STEPS.includes(p)))
  $: stepIndex = STEPS.findIndex((s) => !donePhases.has(s))

  function friendlyError(msg) {
    if (!msg) return 'Something went wrong. Tap Retry.'
    if (msg.includes('no internet') || msg.includes('network') || msg.includes('429') || msg.includes('503')) {
      return 'No internet — check your data or Wi-Fi, then tap Retry.'
    }
    if (msg.includes('no space') || msg.includes('ENOSPC')) {
      return 'Not enough disk space — free up ~3 GB, then tap Retry.'
    }
    if (msg.includes('setup canceled') || msg.includes('context canceled')) {
      return 'Setup paused. Open Tutor again anytime — it resumes.'
    }
    return msg
  }
</script>

<div class="panel">
  <h3>{setupRunning ? 'Setting up — one time only' : 'Setup required'}</h3>
  <p class="muted">
    {#if setupRunning}
      Tutor works without internet after this one download (~1.2 GB).
      Keep the laptop charging — if NEPA takes light, just open again later; it resumes.
    {:else}
      {friendlyError(status.startErr)}
    {/if}
  </p>

  {#if setupRunning}
    <div class="bar-wrap">
      <div class="bar" style="width:{Math.max(hasProgress ? pct : 4, 4)}%"></div>
    </div>
    <div class="bar-meta">
      {#if hasProgress}
        <span>{human(current.downloaded)} / {human(current.total)} ({pct}%)</span>
      {:else}
        <span>Working…</span>
      {/if}
      {#if stepIndex >= 0}
        <span class="step">Step {Math.min(stepIndex + 1, STEPS.length)} / {STEPS.length}: {STEPS[stepIndex]}</span>
      {/if}
    </div>
  {:else if setupLog.length}
    <div class="bar-wrap"><div class="bar" style="width:100%"></div></div>
    <div class="bar-meta"><span>Ready to chat — {status.dbChunks} examples indexed.</span></div>
  {/if}

  <ul class="status">
    {#each STEPS as s, i}
      <li class:done={donePhases.has(s)} class:active={!donePhases.has(s) && i === stepIndex}>
        {donePhases.has(s) ? '✓' : (i === stepIndex && setupRunning ? '●' : '○')} {s}
      </li>
    {/each}
  </ul>

  <div class="row">
    {#if setupRunning}
      <button class="ghost" on:click={onCancel}>Cancel</button>
    {:else}
      <button class="send" on:click={onRunSetup}>Download &amp; Install (~1.2 GB)</button>
      <button on:click={onRetry} disabled={retryLoading}>{retryLoading ? 'Retrying…' : 'Retry'}</button>
    {/if}
    <button class="ghost" on:click={onRefresh}>Refresh</button>
  </div>

  {#if setupLog.length}
    <div class="log" role="log" aria-live="polite">
      {#each setupLog.slice(-40) as e, i}
        <div class:err={e.phase === 'error'}>[{e.phase}] {e.message}</div>
      {/each}
    </div>
  {/if}

  {#if paths}
    <details><summary>Advanced</summary>
      <pre class="paths">{JSON.stringify(paths, null, 2)}</pre>
    </details>
  {/if}
</div>

<style>
  .panel {
    background: var(--slate-elev);
    border: 1px solid var(--green-30);
    border-radius: var(--radius);
    padding: 18px 20px;
    margin: 20px;
    max-width: 720px;
  }
  h3 { font: 600 16px/1.4 'Inter', sans-serif; color: var(--chalk-bright); margin: 0 0 8px; }
  .muted { font: 400 12.5px/1.6 'Inter', sans-serif; color: var(--chalk-muted); margin: 0 0 10px; }

  .bar-wrap {
    height: 8px;
    background: var(--slate);
    border: 1px solid var(--slate-line);
    border-radius: 99px;
    overflow: hidden;
    margin: 8px 0 6px;
  }
  .bar {
    height: 100%;
    background: var(--green);
    border-radius: 99px;
    transition: width 160ms ease;
  }
  .bar-meta { display: flex; justify-content: space-between; font: 400 11px/1.5 'JetBrains Mono', monospace; color: var(--chalk-muted); }
  .step { color: var(--amber); }

  .status { font: 400 11.5px/1.8 'JetBrains Mono', monospace; color: var(--chalk-faint); margin: 10px 0 14px; padding-left: 0; list-style: none; display: flex; flex-wrap: wrap; gap: 4px 14px; }
  .status .done { color: var(--green); }
  .status .active { color: var(--amber); }

  .row { display: flex; gap: 10px; align-items: center; margin-bottom: 12px; flex-wrap: wrap; }
  button { padding: 10px 16px; border-radius: var(--radius-sm); font: 600 13px/1 'Inter', sans-serif; cursor: pointer; }
  .send { background: var(--green); border: 1px solid var(--green); color: var(--chalk-bright); }
  .ghost { background: transparent; border: 1px solid var(--slate-line2); color: var(--chalk-muted); }
  button:disabled { opacity: 0.5; cursor: not-allowed; }

  .log {
    font: 400 11px/1.6 'JetBrains Mono', monospace;
    background: var(--slate);
    border: 1px solid var(--slate-line);
    border-radius: var(--radius-sm);
    padding: 10px;
    color: var(--green);
    max-height: 140px;
    overflow-y: auto;
  }
  .log .err { color: var(--error); }

  details { margin-top: 12px; }
  summary { font: 500 12px/1 'Inter', sans-serif; color: var(--chalk-muted); cursor: pointer; }
  .paths { font: 400 11px/1.5 'JetBrains Mono', monospace; color: var(--chalk-faint); background: var(--slate); border-radius: var(--radius-sm); padding: 10px; overflow-x: auto; }
</style>
