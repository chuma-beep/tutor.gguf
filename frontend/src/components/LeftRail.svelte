<script>
  // Left rail — subdomain history. Groups past questions by classifier tag.
  export let turns = []
  export let onSelectTurn
  export let activeIndex = -1
  export let onRename

  const GROUPS = [
    ['algebra', 'Algebra'],
    ['calculus', 'Calculus'],
    ['discrete_math', 'Discrete'],
    ['geometry', 'Geometry'],
    ['probability', 'Probability'],
    ['number_theory', 'Number theory'],
    ['other', 'Other'],
  ]

  function titleOf(t) {
    if (t.title) return t.title
    return t.question
  }

  function groupOf(t) {
    const g = t.category || t.subdomain || 'other'
    return GROUPS.some(([k]) => k === g) ? g : 'other'
  }

  function groupTurns() {
    const map = new Map()
    for (const [key, label] of GROUPS) map.set(key, { label, items: [] })
    turns.forEach((t, i) => {
      const g = groupOf(t)
      const entry = map.get(g)
      if (entry) entry.items.push({ turn: t, index: i })
    })
    return [...map.values()].filter((g) => g.items.length > 0)
  }

  function handleKey(e, i, t) {
    if (e.key === 'Enter') {
      e.preventDefault()
      e.target.blur()
      onRename && onRename(i, e.target.textContent.trim() || t.question)
    }
  }
</script>

<aside class="rail">
  <h2>History</h2>
  {#each groupTurns() as group (group.label)}
    <div class="group">
      <div class="group-label">{group.label}</div>
      <ul>
        {#each group.items as item (item.index)}
          <li>
            <button
              class="item"
              class:active={item.index === activeIndex}
              on:click={() => onSelectTurn && onSelectTurn(item.index)}
              title={item.turn.question}
            >
              <span
                class="editable"
                contenteditable="true"
                role="textbox"
                aria-label="Rename question"
                on:keydown={(e) => handleKey(e, item.index, item.turn)}
                on:blur={(e) => onRename && onRename(item.index, e.target.textContent.trim() || item.turn.question)}
              >{titleOf(item.turn)}</span>
              <span class="cat">{item.turn.category || item.turn.subdomain || 'other'}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/each}
  {#if turns.length === 0}
    <p class="empty">No questions yet. Ask something in the box.</p>
  {/if}
</aside>

<style>
  .rail {
    background: var(--slate);
    border-right: 1px solid var(--slate-line);
    overflow-y: auto;
    padding: 12px 10px;
  }
  h2 {
    font: 700 11px/1 'JetBrains Mono', monospace;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--chalk-faint);
    margin: 0 0 12px 6px;
  }
  .group { margin-bottom: 14px; }
  .group-label {
    font: 700 10px/1 'Inter', sans-serif;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--green);
    margin: 0 0 6px 6px;
  }
  ul { list-style: none; margin: 0; padding: 0; }
  .item {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 4px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    color: var(--chalk-muted);
    padding: 8px 10px;
    margin-bottom: 4px;
    text-align: left;
    cursor: pointer;
    transition: border-color 160ms ease, background 160ms ease;
  }
  .item:hover { background: var(--slate-elev); }
  .item.active { border-color: var(--green-30); background: var(--green-12); }
  .editable {
    font: 400 12px/1.4 'JetBrains Mono', monospace;
    color: var(--chalk);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }
  .editable:focus { outline: 1px solid var(--green); border-radius: 4px; }
  .cat {
    font: 500 10px/1 'Inter', sans-serif;
    color: var(--chalk-faint);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .empty { font: 400 12px/1.5 'Inter', sans-serif; color: var(--chalk-faint); padding: 0 6px; }
</style>
