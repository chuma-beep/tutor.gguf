<script>
  export let ready = false
  export let checking = true
  export let setupRunning = false
  export let onClear
  export let onToggleRail
  export let railCollapsed = false
</script>

<header>
  <div class="brand">
    <span class="mark" aria-hidden="true">∑</span>
    <h1>tutor<span class="ext">.gguf</span></h1>
    <span class="tagline">on-device math tutor</span>
  </div>
  <div class="chrome">
    <button class="ghost" on:click={onToggleRail} title="Toggle history">
      {railCollapsed ? '☰ History' : 'Hide History'}
    </button>
    <span class="offline" title="No internet needed — everything runs on this laptop">● Offline</span>
    {#if checking}
      <span class="status checking">● Checking…</span>
    {:else if setupRunning}
      <span class="status amber">● Setting up</span>
    {:else if ready}
      <span class="status ok">● Ready</span>
    {:else}
      <span class="status err">● Not ready</span>
    {/if}
    <button class="ghost" on:click={onClear} title="Clear history (Ctrl+L)">Clear</button>
  </div>
</header>

<style>
  header {
    grid-column: 1 / -1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    height: 56px;
    background: var(--slate);
    border-bottom: 1px solid var(--green-30);
    gap: 16px;
  }

  .brand { display: flex; align-items: baseline; gap: 10px; min-width: 0; }
  .mark {
    font: 700 22px/1 'JetBrains Mono', monospace;
    color: var(--green);
  }
  h1 { font: 600 16px/1 'Inter', sans-serif; color: var(--chalk-bright); margin: 0; letter-spacing: -0.011em; }
  .ext { color: var(--green); }
  .tagline { font: 400 12px/1 'Inter', sans-serif; color: var(--chalk-faint); white-space: nowrap; }

  .chrome { display: flex; align-items: center; gap: 14px; }
  .offline {
    font: 400 11px/1 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    background: var(--slate-elev);
    border: 1px solid var(--slate-line);
    padding: 4px 8px;
    border-radius: 99px;
  }
  .status { font: 500 12px/1 'Inter', sans-serif; }
  .status.ok { color: var(--green); }
  .status.amber { color: var(--amber); }
  .status.err { color: var(--error); }
  .status.checking { color: var(--chalk-muted); }

  .ghost {
    background: transparent;
    border: 1px solid var(--slate-line2);
    color: var(--chalk-muted);
    border-radius: var(--radius-sm);
    padding: 6px 10px;
    font: 500 12px/1 'Inter', sans-serif;
    cursor: pointer;
    transition: border-color 160ms ease, color 160ms ease;
  }
  .ghost:hover { border-color: var(--green-30); color: var(--chalk-bright); }
</style>
