<script>
  import { renderMath } from '../lib/renderMath.js'
  import BoxedAnswer from './BoxedAnswer.svelte'

  export let turn
  export let index = 0
  export let showPrompt = false
  export let onTogglePrompt
  export let onCopy = () => {}
  export let onRetry = () => {}
  export let onDelete = () => {}

  const BADGE_MAX = 200

  // The Final badge is for short extracted answers only. Whole-output
  // fallbacks (rule 'none', or legacy history without a rule and a long
  // value) stay out of the badge — the full text is already above.
  function badgeValue(t) {
    if (!t.boxed) return ''
    if (t.answerRule && t.answerRule === 'none') return ''
    if (!t.answerRule && t.boxed.length > BADGE_MAX) return ''
    return t.boxed.length > BADGE_MAX ? t.boxed.slice(0, BADGE_MAX) + '…' : t.boxed
  }
</script>

<div class="turn">
  <div class="plate-head">
    <span class="archive-label">Plate {String(index + 1).padStart(2, '0')}</span>
    {#if turn.subdomain && !turn.error}
      <span class="pill">{turn.category || turn.subdomain}</span>
    {/if}
  </div>
  <div class="q"><span class="q-mark">Q</span> <span class="q-text">{turn.question}</span></div>

  {#if turn.error}
    <div class="error">error: {turn.error}</div>
    <div class="actions">
      <button class="link" on:click={onRetry}>Retry</button>
      <button class="link danger" on:click={onDelete}>Delete</button>
    </div>
  {:else}
    <div class="answer">{@html renderMath(turn.answer)}</div>

    {#if badgeValue(turn)}
      <BoxedAnswer value={badgeValue(turn)} />
    {/if}

    <div class="actions">
      <button class="link" on:click={() => onCopy(turn.answer)}>Copy answer</button>
      <button class="link" on:click={() => onCopy(turn.question)}>Copy question</button>
      {#if turn.prompt}
        <button class="link" on:click={onTogglePrompt}>{showPrompt ? 'Hide' : 'View'} prompt sent to Qwen</button>
      {/if}
      <button class="link" on:click={onRetry}>Retry</button>
      <button class="link danger" on:click={onDelete}>Delete</button>
    </div>
    {#if turn.prompt && showPrompt}
      <pre class="prompt">{turn.prompt}</pre>
    {/if}
  {/if}
</div>

<style>
  .turn {
    margin: 16px 0 0;
    padding: 0 0 20px;
    border: 1px solid var(--slate-line2);
    border-top-width: 1px;
    background: var(--slate-elev);
  }
  .turn:last-of-type { margin-bottom: 8px; }

  .plate-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    border-bottom: 1px solid var(--slate-line);
    padding: 10px 14px;
    margin-bottom: 14px;
  }
  .archive-label {
    font: 400 10px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--chalk-faint);
  }

  .q { display: flex; align-items: baseline; gap: 8px; margin: 0 14px 10px; }
  .turn .answer, .turn .error { margin-left: 14px; margin-right: 14px; }
  .turn :global(.boxed-wrap) { margin-left: 14px !important; }
  .turn .actions { margin-left: 14px; margin-right: 14px; }
  .turn .prompt { margin-left: 14px; margin-right: 14px; }
  .q-mark {
    font: 700 12px/1 'JetBrains Mono', monospace;
    color: var(--chalk-muted);
    border: 1px solid var(--slate-line2);
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
    color: var(--chalk-muted);
    background: transparent;
    border: 1px solid var(--slate-line2);
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
    color: var(--chalk-muted);
    font: 500 12px/1 'Inter', sans-serif;
    text-decoration: underline;
    padding: 6px 0;
    cursor: pointer;
  }
  .link.danger { color: var(--error); }

  .actions { display: flex; flex-wrap: wrap; gap: 4px 14px; }

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
