<script>
  import { renderMath } from '../lib/renderMath.js'

  export let streaming = null // { question, answer, category, subdomain }
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
    <div class="answer">{@html renderMath(streaming.answer)} <span class="cursor" aria-hidden="true">▍</span></div>
  </div>
{/if}

<style>
  .turn { margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--slate-line); }
  .q { display: flex; align-items: baseline; gap: 8px; margin-bottom: 10px; }
  .q-mark {
    font: 700 12px/1 'JetBrains Mono', monospace;
    color: var(--green);
    border: 1px solid var(--green-30);
    border-radius: 6px;
    padding: 2px 6px;
    flex-shrink: 0;
  }
  .q-text { font: 600 13px/1.5 'JetBrains Mono', monospace; color: var(--chalk-bright); }
  .pill {
    font: 700 10px/1 'Inter', sans-serif;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--green);
    background: var(--green-12);
    border: 1px solid var(--green-30);
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

  .cursor { animation: blink 1s steps(1) infinite; color: var(--green); }
  @keyframes blink { 50% { opacity: 0; } }
</style>
