<script>
  import { renderMath } from '../lib/renderMath.js'
  import BoxedAnswer from './BoxedAnswer.svelte'

  export let turn
  export let showPrompt = false
  export let onTogglePrompt
</script>

<div class="turn">
  <div class="q"><span class="q-mark">Q</span> <span class="q-text">{turn.question}</span>
    {#if turn.subdomain && !turn.error}
      <span class="pill">{turn.category || turn.subdomain}</span>
    {/if}
  </div>

  {#if turn.error}
    <div class="error">error: {turn.error}</div>
  {:else}
    <div class="answer">{@html renderMath(turn.answer)}</div>

    {#if turn.boxed}
      <BoxedAnswer value={turn.boxed} />
    {/if}

    {#if turn.prompt}
      <button class="link" on:click={onTogglePrompt}>{showPrompt ? 'Hide' : 'View'} prompt sent to Qwen</button>
      {#if showPrompt}
        <pre class="prompt">{turn.prompt}</pre>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .turn {
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--slate-line);
  }
  .turn:last-child { border-bottom: none; }

  .q { display: flex; align-items: baseline; gap: 8px; margin-bottom: 10px; }
  .q-mark {
    font: 700 12px/1 'JetBrains Mono', monospace;
    color: var(--green);
    border: 1px solid var(--green-30);
    border-radius: 6px;
    padding: 2px 6px;
    flex-shrink: 0;
  }
  .q-text {
    font: 600 13px/1.5 'JetBrains Mono', monospace;
    color: var(--chalk-bright);
  }
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

  .answer {
    font: 400 14.5px/1.7 'Inter', system-ui, sans-serif;
    color: var(--chalk);
  }
  .answer :global(.prose) { white-space: pre-wrap; }
  .answer :global(.math-display) { margin: 10px 0; overflow-x: auto; scrollbar-width: thin; }
  .answer :global(.katex) { font-size: 1.05em; color: var(--chalk-bright); }

  .error { color: var(--error); font: 400 13px/1.5 'JetBrains Mono', monospace; }

  .link {
    background: none;
    border: none;
    color: var(--green);
    font: 500 12px/1 'Inter', sans-serif;
    text-decoration: underline;
    padding: 6px 0;
    cursor: pointer;
  }

  .prompt {
    font: 400 11px/1.5 'JetBrains Mono', monospace;
    white-space: pre-wrap;
    background: var(--slate);
    border: 1px solid var(--slate-line);
    padding: 10px;
    border-radius: var(--radius-sm);
    overflow-x: auto;
    color: var(--chalk-muted);
    margin: 8px 0 0;
  }
</style>
