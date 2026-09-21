<script>
  // Right rail — "what it pulled from". Top-K retrieved chunks as source cards.
  export let turn = null // the active turn whose chunks to show
  export let onCopy = () => {}

  const LIMIT = 300
  let expanded = new Set()

  function toggle(i) {
    if (expanded.has(i)) expanded.delete(i)
    else expanded.add(i)
    expanded = expanded
  }

  function chunkText(c) {
    return c.Text || c.text || ''
  }
  function chunkSub(c) { return c.Subdomain || c.subdomain || 'other' }
  function chunkSrc(c) { return c.Source || c.source || 'corpus' }
  function chunkSim(c) { return c.Similarity ?? c.similarity ?? 0 }
  function shown(c, i) {
    const t = chunkText(c)
    if (expanded.has(i) || t.length <= LIMIT) return t
    return t.slice(0, LIMIT) + '…'
  }
</script>

<aside class="rail">
  <div class="section-head"><span>02</span><h2>Retrieved</h2></div>
  {#if turn && turn.chunks && turn.chunks.length}
    <p class="hint">Top {turn.chunks.length} chunks the tutor used for this question.</p>
    {#each turn.chunks as c, i (i)}
      <div class="card">
        <div class="card-head">
          <span class="idx">[{i + 1}]</span>
          <span class="sub">{chunkSub(c)}</span>
          <span class="sim">{(chunkSim(c) * 100).toFixed(1)}%</span>
        </div>
        <div class="src">{chunkSrc(c)}</div>
        <p class="snippet">{shown(c, i)}</p>
        <div class="card-actions">
          {#if chunkText(c).length > LIMIT}
            <button class="link" on:click={() => toggle(i)}>{expanded.has(i) ? 'Show less' : 'Show more'}</button>
          {/if}
          <button class="link" on:click={() => onCopy(chunkText(c))}>Copy</button>
        </div>
      </div>
    {/each}
  {:else}
    <p class="hint">Sources appear here after you ask a question — the tutor's retrieval, in the open.</p>
  {/if}
</aside>

<style>
  .rail {
    grid-area: right;
    background: var(--slate);
    border-left: 1px solid var(--slate-line);
    overflow-y: auto;
    padding: 12px 10px;
  }
  h2 {
    font: 500 13px/1 'Inter', sans-serif;
    margin: 0;
    color: var(--chalk-bright);
  }
  .section-head {
    display: flex;
    align-items: baseline;
    gap: 10px;
    border-bottom: 1px solid var(--slate-line2);
    padding: 4px 6px 10px;
    margin-bottom: 10px;
  }
  .section-head > span {
    font: 400 10px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.14em;
    color: var(--chalk-faint);
  }
  .hint { font: 400 12px/1.5 'Inter', sans-serif; color: var(--chalk-faint); padding: 0 6px; }
  .card {
    background: var(--slate-elev);
    border: 1px solid var(--slate-line);
    border-radius: var(--radius-sm);
    padding: 10px;
    margin: 0 0 10px;
  }
  .card-head { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
  .idx { font: 700 12px/1 'JetBrains Mono', monospace; color: var(--chalk-bright); }
  .sub {
    font: 700 10px/1 'Inter', sans-serif;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--chalk-muted);
  }
  .sim { font: 400 10px/1 'JetBrains Mono', monospace; color: var(--chalk-faint); margin-left: auto; }
  .src { font: 400 10px/1 'JetBrains Mono', monospace; color: var(--chalk-faint); margin-bottom: 6px; }
  .snippet {
    font: 400 11.5px/1.55 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    margin: 0;
    white-space: pre-wrap;
  }
  .card-actions { display: flex; gap: 12px; margin-top: 6px; }
  .link {
    background: none;
    border: none;
    color: var(--chalk-muted);
    font: 500 11px/1 'Inter', sans-serif;
    text-decoration: underline;
    padding: 2px 0;
    cursor: pointer;
  }
</style>
