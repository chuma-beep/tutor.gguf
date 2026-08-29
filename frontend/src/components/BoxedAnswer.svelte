<script>
  import katex from 'katex'

  export let value = ''

  function renderValue(v) {
    if (!v) return ''
    try {
      return katex.renderToString(v, { throwOnError: false, displayMode: false })
    } catch {
      return v
    }
  }
</script>

<div class="boxed-wrap" role="status" aria-label="Final answer: {value}">
  <span class="corner tl" aria-hidden="true"></span>
  <span class="corner tr" aria-hidden="true"></span>
  <span class="corner bl" aria-hidden="true"></span>
  <span class="corner br" aria-hidden="true"></span>
  <span class="label">Final</span>
  <span class="value">{@html renderValue(value)}</span>
</div>

<style>
  .boxed-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 10px;
    padding: 6px 16px 6px 14px;
    border: 1px solid var(--amber);
    background: var(--amber-12);
    border-radius: 10px 6px;
    box-shadow: inset 0 0 0 1px var(--amber), 0 0 0 4px var(--amber-12);
    margin: 4px 0;
    animation: settle 150ms ease-out;
  }

  .label {
    font: 700 10px/1 'JetBrains Mono', ui-monospace, monospace;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--amber-tick);
  }

  .value {
    font: 700 14px/1.2 'JetBrains Mono', ui-monospace, monospace;
    color: var(--chalk-bright);
  }

  .value :global(.katex) {
    font-size: 1.15em;
  }

  /* Corner ticks — the TUI's ┌┐└┘ box-drawing, at 2px weight */
  .corner {
    position: absolute;
    width: 10px;
    height: 10px;
    border-color: var(--amber-tick);
    pointer-events: none;
  }
  .corner.tl { top: -1px; left: -1px; border-top: 2px solid; border-left: 2px solid; border-radius: 6px 0 0 0; }
  .corner.tr { top: -1px; right: -1px; border-top: 2px solid; border-right: 2px solid; border-radius: 0 6px 0 0; }
  .corner.bl { bottom: -1px; left: -1px; border-bottom: 2px solid; border-left: 2px solid; border-radius: 0 0 0 6px; }
  .corner.br { bottom: -1px; right: -1px; border-bottom: 2px solid; border-right: 2px solid; border-radius: 0 0 6px 0; }

  @keyframes settle {
    from { opacity: 0; transform: scale(0.96); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
