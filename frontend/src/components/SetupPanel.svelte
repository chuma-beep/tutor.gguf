<script>
  export let status
  export let setupLog = []
  export let setupRunning = false
  export let paths = null
  export let retryLoading = false
  export let onRunSetup
  export let onRetry
  export let onRefresh
</script>

<div class="panel">
  <h3>Setup required — Tutor not ready</h3>
  <p class="muted">{status.startErr || 'Models or index missing. Run setup.'}</p>
  <ul class="status">
    <li>Llama: {status.llamaExists ? 'found ' + (status.llamaServer || '') : 'missing — run tutor setup'}</li>
    <li>Gen model: {status.genModelExists ? (status.genModelBytes / 1e9).toFixed(2) + ' GB' : 'missing'}</li>
    <li>Embed: {status.embedExists ? (status.embedBytes / 1e6).toFixed(0) + ' MB' : 'missing'}</li>
    <li>DB: {status.dbChunks} chunks @ {status.dbPath}</li>
  </ul>
  <div class="row">
    <button class="send" on:click={onRunSetup} disabled={setupRunning}>
      {setupRunning ? 'Setting up…' : 'Run Setup'}
    </button>
    <button on:click={onRetry} disabled={retryLoading || setupRunning}>{retryLoading ? 'Retrying…' : 'Retry'}</button>
    <button class="ghost" on:click={onRefresh}>Refresh</button>
  </div>
  {#if setupRunning || setupLog.length}
    <div class="progress" role="progressbar" aria-valuetext={setupRunning ? 'working…' : 'done'}>
      {#each setupLog as e, i}
        <div class:err={e.phase === 'error'}>[{e.phase}] {e.message}</div>
      {/each}
      {#if setupRunning}<span class="spin">●</span>{/if}
    </div>
  {:else}
    <p class="muted">First launch downloads ~1.2 GB (models + corpus), builds the index, then enables chat automatically. Runs in the background — cancel-safe.</p>
  {/if}
  {#if paths}
    <details><summary>Paths</summary><pre class="paths">{JSON.stringify(paths, null, 2)}</pre></details>
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
  .status { font: 400 12px/1.7 'JetBrains Mono', monospace; color: var(--chalk-muted); margin: 0 0 14px; padding-left: 18px; }
  .row { display: flex; gap: 10px; align-items: center; margin-bottom: 12px; flex-wrap: wrap; }
  button {
    padding: 10px 16px;
    border-radius: var(--radius-sm);
    font: 600 13px/1 'Inter', sans-serif;
    cursor: pointer;
  }
  .send { background: var(--green); border: 1px solid var(--green); color: var(--chalk-bright); }
  .ghost { background: transparent; border: 1px solid var(--slate-line2); color: var(--chalk-muted); }
  button:disabled { opacity: 0.5; cursor: not-allowed; }

  .progress {
    font: 400 11px/1.6 'JetBrains Mono', monospace;
    background: var(--slate);
    border: 1px solid var(--slate-line);
    border-radius: var(--radius-sm);
    padding: 10px;
    color: var(--green);
    max-height: 180px;
    overflow-y: auto;
  }
  .progress .err { color: var(--error); }
  .spin { animation: pulse 1.2s ease infinite; }
  @keyframes pulse { 50% { opacity: 0.35; } }

  details { margin-top: 12px; }
  summary { font: 500 12px/1 'Inter', sans-serif; color: var(--chalk-muted); cursor: pointer; }
  .paths { font: 400 11px/1.5 'JetBrains Mono', monospace; color: var(--chalk-faint); background: var(--slate); border-radius: var(--radius-sm); padding: 10px; overflow-x: auto; }
</style>
