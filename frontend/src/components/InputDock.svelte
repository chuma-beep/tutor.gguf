<script>
  export let input = ''
  export let loading = false
  export let onInput
  export let onSubmit
  export let onClear

  function handleKey(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      onSubmit()
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      onInput('')
    }
  }
</script>

<div class="dock">
  <div class="row">
    {#if loading}<span class="spinner" aria-hidden="true"></span>{/if}
    <input
      autofocus
      value={input}
      on:input={(e) => onInput(e.target.value)}
      on:keydown={handleKey}
      placeholder="Find the derivative of x^2…"
      maxlength="512"
      disabled={loading}
      aria-label="Ask a math question"
    />
    <button class="send" on:click={onSubmit} disabled={loading || !input.trim()}>Send</button>
    <button class="ghost" on:click={onClear}>Clear</button>
  </div>
  <div class="hint">Enter to send · Shift+Enter for a new line · Esc clears · 512 chars max</div>
</div>

<style>
  .dock {
    grid-area: input;
    position: sticky;
    bottom: 0;
    background: linear-gradient(to top, var(--slate) 72%, transparent);
    border-top: 1px solid var(--slate-line);
    padding: 14px 20px 18px;
  }
  .row { display: flex; gap: 10px; align-items: center; }
  .spinner {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--green);
    flex-shrink: 0;
    animation: pulse 1.2s ease infinite;
  }
  @keyframes pulse { 50% { opacity: 0.35; } }

  input {
    flex: 1;
    padding: 12px 14px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--slate-line2);
    background: var(--slate-elev);
    color: var(--chalk-bright);
    font: 400 14px/1.5 'JetBrains Mono', monospace;
    transition: border-color 160ms ease;
  }
  input::placeholder { color: var(--chalk-faint); }
  input:focus { border-color: var(--green-30); outline: none; }
  input:disabled { opacity: 0.55; cursor: not-allowed; }

  button {
    padding: 12px 18px;
    border-radius: var(--radius-sm);
    font: 600 13px/1 'Inter', sans-serif;
    cursor: pointer;
    transition: background 160ms ease, border-color 160ms ease;
  }
  .send { background: var(--green); border: 1px solid var(--green); color: var(--chalk-bright); }
  .send:hover { background: #469a68; }
  .send:disabled { opacity: 0.5; cursor: not-allowed; }
  .ghost { background: transparent; border: 1px solid var(--slate-line2); color: var(--chalk-muted); }
  .ghost:hover { border-color: var(--green-30); color: var(--chalk-bright); }

  .hint {
    font: 400 11px/1 'Inter', sans-serif;
    color: var(--chalk-faint);
    margin-top: 8px;
    padding-left: 2px;
  }
</style>
