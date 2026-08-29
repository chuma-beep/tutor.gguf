<script>
  // Right rail — "what it pulled from". Top-K retrieved chunks as source cards.
  export let turn = null // the active turn whose chunks to show

  function chunkText(c) {
    const t = c.Text || c.text || ''
    return t.length > 300 ? t.slice(0, 300) + '…' : t
  }
  function chunkSub(c) { return c.Subdomain || c.subdomain || 'other' }
  function chunkSrc(c) { return c.Source || c.source || 'corpus' }
  function chunkSim(c) { return c.Similarity ?? c.similarity ?? 0 }
</script>

<aside class="rail">
  <h2>Retrieved</h2>
  {#if turn && turn.chunks && turn.chunks.length}
    <p class="hint">Top {turn.chunks.length} chunks the tutor used for this question.</p>
    {#each turn.chunks as c, i (i)}
      <div class="card">
        <div class="card-head">
          <span class="idx">[{i + 1}]</span>
          <span class="sub">{chunkSub(c)}</span>
          <span class="sim">{chunkSim(c).toFixed(3)}</span>
        </div>
        <div class="src">{chunkSrc(c)}</div>
        <p class="snippet">{chunkText(c)}</p>
      </div>
    {/each}
  {:else}
    <p class="hint">Sources appear here after you ask a question — the tutor's retrieval, in the open.</p>
  {/if}
</aside>

<style>
  .rail {
    background: var(--slate);
    border-left: 1px solid var(--slate-line);
    overflow-y: auto;
    padding: 12px 10px;
  }
  h2 {
    font: 700 11px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--chalk-faint);
    margin: 0 0 8px 6px;
  }
  .hint { font: 400 12px/1.5 'Inter', sans-serif; color: var(--chalk-faint); padding: 0 6px; }
  .card {
    background: var(--slate-elev);
    border: 1px solid var(--slate-line);
    border-left: 3px solid var(--green);
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
    color: var(--green);
  }
  .sim { font: 400 10px/1 'JetBrains Mono', monospace; color: var(--chalk-faint); margin-left: auto; }
  .src { font: 400 10px/1 'JetBrains Mono', monospace; color: var(--chalk-faint); margin-bottom: 6px; }
  .snippet {
    font: 400 11.5px/1.55 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    margin: 0;
    white-space: pre-wrap;
  }
</style>
