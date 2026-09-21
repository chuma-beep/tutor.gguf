<script>
  export let ready = false
  export let checking = true
  export let setupRunning = false
  export let onClear
  export let onToggleRail
  export let railCollapsed = false
  export let onToggleSources = () => {}
  export let sourcesOpen = false
  export let numberWordsEnabled = false
  export let onToggleNumberWords = () => {}
  export let theme = 'system'
  export let onCycleTheme = () => {}
  export let view = 'chat'
  export let onToggleView = () => {}

  $: themeLabel = theme === 'dark' ? '● Dark' : theme === 'system' ? '◐ System' : '○ Light'
</script>

<header>
  <div class="brand">
    <a href="#top" class="wordmark">tutor.gguf</a>
    <span class="index">Index / TG-001 · Local</span>
  </div>
  <div class="chrome">
    <button class="ghost" on:click={onToggleRail} title="Toggle history">
      {railCollapsed ? '☰ History' : 'Hide History'}
    </button>
    <button class="ghost" on:click={onToggleSources} title="Toggle retrieved sources">
      {sourcesOpen ? 'Hide Sources' : '☰ Sources'}
    </button>
    <button class="ghost" on:click={onToggleView} title="Toggle the offline manual">
      {view === 'docs' ? '☰ Chat' : '☰ Docs'}
    </button>
    <label class="toggle" title="Convert word numbers to digits before sending to model (e.g. one plus one → 1 + 1)">
      <input type="checkbox" checked={numberWordsEnabled} on:change={onToggleNumberWords} />
      <span>1+1</span>
    </label>
    <button class="ghost" on:click={onCycleTheme} title="Color theme: light archive, dark chalkboard, or system">
      {themeLabel}
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
    grid-area: header;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    min-height: 56px;
    background: var(--slate-elev);
    border-bottom: 1px solid var(--slate-line2);
    gap: 16px;
  }

  .brand { display: flex; align-items: baseline; gap: 12px; min-width: 0; }
  .wordmark {
    font: 600 14px/1 'Inter', sans-serif;
    letter-spacing: 0.02em;
    text-transform: uppercase;
    color: var(--chalk-bright);
    text-decoration: none;
  }
  .index {
    font: 400 10px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--chalk-faint);
    white-space: nowrap;
  }

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
  .status.ok { color: var(--chalk-bright); }
  .status.amber { color: var(--amber); }
  .status.err { color: var(--error); }
  .status.checking { color: var(--chalk-muted); }

  .ghost {
    background: transparent;
    border: 1px solid var(--slate-line2);
    color: var(--chalk-muted);
    border-radius: var(--radius-sm);
    padding: 10px 12px;
    font: 500 12px/1 'Inter', sans-serif;
    cursor: pointer;
    white-space: nowrap;
    transition: border-color 160ms ease, color 160ms ease;
  }
  .ghost:hover { border-color: var(--slate-line2); color: var(--chalk-bright); }

  .toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    font: 500 11px/1 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    background: var(--slate-elev);
    border: 1px solid var(--slate-line);
    border-radius: 99px;
    padding: 4px 10px;
    cursor: pointer;
    user-select: none;
  }
  .toggle:has(input:checked) { border-color: var(--slate-line2); color: var(--chalk-bright); background: transparent; }
  .toggle input { accent-color: var(--chalk-muted); }

  @media (max-width: 760px) {
    header { padding: 0 12px; gap: 10px; }
    .index, .offline { display: none; }
    .brand { gap: 0; }
    .chrome {
      gap: 8px;
      overflow-x: auto;
      scrollbar-width: none;
      -webkit-overflow-scrolling: touch;
    }
    .chrome::-webkit-scrollbar { display: none; }
    .status { white-space: nowrap; }
  }
</style>
