<script>
  import { chapters, getChapter, DOCS_URL, REPO_URL } from '../lib/docs.js'

  export let slug = null
  export let onNavigate = () => {}

  let root = null

  $: chapter = (slug && getChapter(slug)) || chapters[0]
  $: index = chapters.findIndex((c) => c.slug === chapter.slug)
  $: prev = chapters[index - 1]
  $: next = chapters[index + 1]

  function go(s) {
    onNavigate(s)
    if (root) root.scrollTop = 0
  }

  function handleActivate(e) {
    const a = e.target && e.target.closest ? e.target.closest('a[data-docs]') : null
    if (!a) return
    e.preventDefault()
    go(a.getAttribute('data-docs'))
  }

  function handleKey(e) {
    // Keyboard activation of in-app links (click covers pointer + Enter on
    // native anchors; this covers Space and other key-driven travellers).
    if (e.key !== 'Enter' && e.key !== ' ') return
    handleActivate(e)
  }
</script>

<section class="transcript docs" aria-label="Offline manual" bind:this={root} on:click={handleActivate} on:keydown={handleKey}>
  <div class="record-head">
    <div>
      <p class="archive-label">Manual / TG-001 · Offline copy</p>
      <h2 class="record-title">{chapter.title}</h2>
    </div>
    <dl class="facts">
      <div><dt>Chapter</dt><dd>{String(index + 1).padStart(2, '0')} / {String(chapters.length).padStart(2, '0')}</dd></div>
      <div><dt>Source</dt><dd>docs/manual</dd></div>
      <div><dt>Format</dt><dd>Markdown</dd></div>
      <div><dt>Status</dt><dd>Local</dd></div>
    </dl>
  </div>

  <div class="chapters">
    {#each chapters as c, i}
      <button class:active={c.slug === chapter.slug} on:click={() => go(c.slug)}>
        <span class="num">{String(i + 1).padStart(2, '0')}</span> {c.title}
      </button>
    {/each}
  </div>

  <div class="manual">{@html chapter.html}</div>

  {#if chapter.headings.length > 0}
    <div class="onpage">
      <p class="archive-label">On this page</p>
      {#each chapter.headings as h}
        <a href="#{h.id}" on:click|preventDefault={() => { const el = root && root.querySelector('#' + CSS.escape(h.id)); if (el) el.scrollIntoView() }}>{h.text}</a>
      {/each}
    </div>
  {/if}

  <div class="pager">
    <div>{#if prev}<button class="link" on:click={() => go(prev.slug)}>← {prev.title}</button>{/if}</div>
    <div>{#if next}<button class="link" on:click={() => go(next.slug)}>{next.title} →</button>{/if}</div>
  </div>

  <footer class="record-foot">
    <span>Manual / TG-001 local</span>
    <span><a href={DOCS_URL} target="_blank" rel="noreferrer">View online ↗</a> · <a href={REPO_URL} target="_blank" rel="noreferrer">Repository ↗</a></span>
    <span>100% offline</span>
  </footer>
</section>

<style>
  .docs { position: relative; }
  .chapters {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 16px 0 8px;
  }
  .chapters button {
    background: transparent;
    border: 1px solid var(--slate-line2);
    color: var(--chalk-muted);
    border-radius: var(--radius-sm);
    padding: 10px 14px;
    font: 400 12px/1.4 'Inter', sans-serif;
    cursor: pointer;
    transition: border-color 160ms ease, color 160ms ease;
  }
  .chapters button:hover { border-color: var(--slate-line2); color: var(--chalk-bright); }
  .chapters button.active { border-color: var(--slate-line2); color: var(--chalk-bright); background: var(--slate); }
  .chapters .num { font: 400 10px/1 'JetBrains Mono', monospace; color: var(--chalk-faint); margin-right: 6px; }
  .onpage { display: flex; flex-wrap: wrap; gap: 4px 14px; align-items: baseline; margin: 20px 0 0; }
  .onpage a { font: 400 11px/1.6 'JetBrains Mono', monospace; color: var(--chalk-muted); }
  .pager { display: flex; justify-content: space-between; gap: 8px; margin-top: 24px; }
  .link {
    background: none;
    border: none;
    color: var(--chalk-muted);
    font: 500 12px/1 'Inter', sans-serif;
    text-decoration: underline;
    padding: 6px 0;
    cursor: pointer;
  }
  .record-foot a { color: var(--chalk-faint); }
</style>
