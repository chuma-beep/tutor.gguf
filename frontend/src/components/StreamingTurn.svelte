<script>
  import { onDestroy } from 'svelte'
  import { renderMath } from '../lib/renderMath.js'

  export let streaming = null // { question, answer, category, subdomain }

  // Throttle KaTeX re-renders while tokens stream in: the prop updates on
  // every delta, but the expensive renderMath() runs at most ~7x/sec.
  let shown = ''
  const timer = setInterval(() => {
    if (streaming && streaming.answer !== shown) shown = streaming.answer
  }, 150)
  onDestroy(() => clearInterval(timer))

  $: if (streaming) {
    // Short answers render immediately (no perceptible throttle lag).
    if (streaming.answer.length < 400) shown = streaming.answer
    // Always catch up fully the moment streaming ends (component unmounts,
    // but a final sync avoids showing a stale tail).
    if (!streaming.answer.startsWith(shown)) shown = streaming.answer
  } else {
    shown = ''
  }
</script>

{#if streaming}
  <div class="turn streaming">
    <div class="q">
      <span class="q-mark">Q</span>
      <span class="q-text">{streaming.question}</span>
      {#if streaming.category || streaming.subdomain}
        <span class="pill loading">Reading {streaming.subdomain || streaming.category} sources…</span>
      {/if}
    </div>
    <div class="answer">{@html renderMath(shown)} <span class="cursor" aria-hidden="true">▍</span></div>
  </div>
{/if}

<style>
  .turn {
    margin: 16px 0 0;
    padding: 0 14px 20px;
    border: 1px solid var(--slate-line2);
    background: var(--slate-elev);
  }
  .q { display: flex; align-items: baseline; gap: 8px; margin: 14px 0 10px; }
  .q-mark {
    font: 700 12px/1 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    border: 1px solid var(--slate-line2);
    border-radius: 6px;
    padding: 2px 6px;
    flex-shrink: 0;
  }
  .q-text { font: 600 13px/1.5 'JetBrains Mono', monospace; color: var(--chalk-bright); }
  .pill {
    font: 700 10px/1 'Inter', sans-serif;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--chalk-muted);
    background: transparent;
    border: 1px solid var(--slate-line2);
    border-radius: 99px;
    padding: 3px 8px;
    flex-shrink: 0;
  }
  .pill.loading { animation: pulse 1.2s ease infinite; }
  @keyframes pulse { 50% { opacity: 0.5; } }

  .answer { font: 400 14.5px/1.7 'Inter', system-ui, sans-serif; color: var(--chalk); }
  .answer :global(.prose) { white-space: pre-wrap; }
  .answer :global(.math-display) { margin: 10px 0; overflow-x: auto; }
  .answer :global(.katex) { font-size: 1.05em; color: var(--chalk-bright); }

  .cursor { animation: blink 1s steps(1) infinite; color: var(--chalk-muted); }
  @keyframes blink { 50% { opacity: 0; } }
</style>
