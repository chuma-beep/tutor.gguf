<script>
  export let input = ''
  export let loading = false
  export let onInput
  export let onSubmit
  export let onClear
  export let onStop

  const MAX = 2000

  let ta = null

  function autoresize() {
    if (!ta) return
    ta.style.height = 'auto'
    ta.style.height = Math.min(ta.scrollHeight, 160) + 'px'
  }

  function handleInput(e) {
    onInput(e.target.value.slice(0, MAX))
    autoresize()
  }

  function handleKey(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      onSubmit()
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      onInput('')
      autoresize()
    }
  }

  // Keep the box collapsed after submit clears the input.
  $: if (ta && input === '') {
    ta.style.height = 'auto'
  }

  $: remaining = MAX - (input ? input.length : 0)
</script>

<div class="dock">
  <div class="row">
    {#if loading}<span class="spinner" aria-hidden="true"></span>{/if}
    <textarea
      bind:this={ta}
      rows="1"
      value={input}
      on:input={handleInput}
      on:keydown={handleKey}
      placeholder="Find the derivative of x^2…"
      maxlength={MAX}
      disabled={loading}
      aria-label="Ask a math question"
    ></textarea>
    {#if loading}
      <button class="stop" on:click={onStop}>Stop</button>
    {:else}
      <button class="send" on:click={onSubmit} disabled={!input.trim()}>Send</button>
    {/if}
    <button class="ghost" on:click={onClear}>Clear</button>
  </div>
  <div class="hint">
    <span>Enter to send · Shift+Enter for a new line · Esc clears</span>
    <span class:low={remaining < 200}>{remaining} left</span>
  </div>
</div>

<style>
  .dock {
    grid-area: input;
    position: sticky;
    bottom: 0;
    background: var(--slate-elev);
    border-top: 1px solid var(--slate-line2);
    padding: 14px 20px 16px;
  }
  .row { display: flex; gap: 10px; align-items: flex-end; }
  .spinner {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--chalk-faint);
    flex-shrink: 0;
    align-self: center;
    animation: pulse 1.2s ease infinite;
  }
  @keyframes pulse { 50% { opacity: 0.35; } }

  textarea {
    flex: 1;
    padding: 12px 14px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--slate-line2);
    background: var(--slate-elev);
    color: var(--chalk-bright);
    font: 400 14px/1.5 'JetBrains Mono', monospace;
    resize: none;
    overflow-y: auto;
    max-height: 160px;
    transition: border-color 160ms ease;
  }
  textarea::placeholder { color: var(--chalk-faint); }
  textarea:focus { border-color: var(--slate-line2); outline: none; }
  textarea:disabled { opacity: 0.55; cursor: not-allowed; }

  button {
    padding: 12px 18px;
    border-radius: var(--radius-sm);
    font: 600 13px/1 'Inter', sans-serif;
    cursor: pointer;
    transition: background 160ms ease, border-color 160ms ease;
    flex-shrink: 0;
  }
  .send { background: var(--chalk-bright); border: 1px solid var(--chalk-bright); color: var(--slate-elev); }
  .send:hover { filter: brightness(1.12); }
  .send:disabled { opacity: 0.5; cursor: not-allowed; }
  .stop { background: transparent; border: 1px solid var(--error); color: var(--error); }
  .stop:hover { background: var(--error); color: var(--chalk-bright); }
  .ghost { background: transparent; border: 1px solid var(--slate-line2); color: var(--chalk-muted); }
  .ghost:hover { border-color: var(--slate-line2); color: var(--chalk-bright); }

  .hint {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 4px 8px;
    font: 400 10px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--chalk-faint);
    margin-top: 8px;
    padding: 0 2px;
  }
  .hint .low { color: var(--amber); }
</style>
