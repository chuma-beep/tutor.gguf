<script>
  import { onMount } from 'svelte';
  import katex from 'katex';
  import 'katex/dist/katex.min.css';

  let turns = [];
  let input = '';
  let loading = false;
  let streaming = null; // { question, answer }
  let showPromptFor = null; // index of turn showing prompt
  let errorMsg = '';
  let status = null;
  let checking = true;
  let retryLoading = false;
  let paths = null;

  let streamCleanup = null;

  // Streaming backend: wails events when available, else HTTP SSE fallback.
  // Calls onMeta({subdomain,category}), onDelta(content), onDone({content,answer}),
  // onError(err).
  function streamBackend(problem, { onMeta, onDelta, onDone, onError }) {
    // Wails path (events emitted by App.AskStream)
    try {
      // @ts-ignore
      if (window.go && window.go.desktop && window.go.desktop.App) {
        const offs = [];
        // @ts-ignore
        const offMeta = window.runtime.EventsOn('tutor:stream:meta', (e) => onMeta && onMeta(e));
        // @ts-ignore
        const offChunk = window.runtime.EventsOn('tutor:stream:chunk', (e) => onDelta && onDelta(e.content || ''));
        // @ts-ignore
        const offDone = window.runtime.EventsOn('tutor:stream:done', (e) => { cleanup(); onDone && onDone(e); });
        // @ts-ignore
        const offErr = window.runtime.EventsOn('tutor:stream:error', (e) => { cleanup(); onError && onError(e.error || String(e)); });
        const cleanup = () => { [offMeta, offChunk, offDone, offErr].forEach(f => f && f()); };
        streamCleanup = cleanup;
        // @ts-ignore
        window.go.desktop.App.AskStream(problem).then(() => {}).catch((e) => { cleanup(); onError && onError(String(e)); });
        return;
      }
    } catch (e) {
      console.log('wails streaming not available, falling back to HTTP SSE', e);
    }
    // HTTP SSE fallback
    fetch('http://127.0.0.1:8082/v1/complete/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ problem })
    }).then(async (resp) => {
      if (!resp.ok) throw new Error(`server error ${resp.status}`);
      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buf = '';
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });
        let idx;
        while ((idx = buf.indexOf('\n\n')) >= 0) {
          const block = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          for (const line of block.split('\n')) {
            const l = line.trim();
            if (!l.startsWith('data:')) continue;
            const payload = l.slice(5).trim();
            if (payload === '[DONE]') continue;
            let ev;
            try { ev = JSON.parse(payload); } catch { continue; }
            if (ev.error) { onError && onError(ev.error); continue; }
            if (ev.subdomain || ev.category) { onMeta && onMeta({subdomain: ev.subdomain, category: ev.category}); continue; }
            if (ev.done) { onDone && onDone({content: ev.content || '', answer: ev.answer || ''}); continue; }
            if (ev.content) onDelta && onDelta(ev.content);
          }
        }
      }
      // stream closed without done marker
      onDone && onDone({ content: '', answer: '' });
    }).catch((e) => onError && onError(String(e)));
  }

  function saveHistory() {
    try {
      localStorage.setItem('tutor-history', JSON.stringify(turns.slice(-100)));
    } catch {}
  }
  function loadHistory() {
    try {
      const raw = localStorage.getItem('tutor-history');
      if (raw) turns = JSON.parse(raw);
    } catch {}
  }

  onMount(async () => {
    loadHistory();
    await checkStatus();
  });

  async function checkStatus() {
    checking = true;
    try {
      // @ts-ignore
      if (window.go && window.go.desktop && window.go.desktop.App) {
        // @ts-ignore
        status = await window.go.desktop.App.GetStatus();
        // @ts-ignore
        try { paths = await window.go.desktop.App.GetPaths(); } catch {}
      } else {
        // dev without wails: assume ready via HTTP health
        try {
          const r = await fetch('http://127.0.0.1:8082/health');
          status = { ready: r.ok, startErr: r.ok ? '' : 'offline', dbChunks: 0, genModelExists: true, embedExists: true };
        } catch {
          status = { ready: true, startErr: '', dbChunks: 0, genModelExists: true, embedExists: true };
        }
      }
    } catch (e) {
      status = { ready: false, startErr: String(e), dbChunks: 0 };
    } finally {
      checking = false;
    }
  }
  async function retryInit() {
    retryLoading = true;
    try {
      // @ts-ignore
      if (window.go && window.go.desktop && window.go.desktop.App) {
        // @ts-ignore
        status = await window.go.desktop.App.RetryInit();
      } else {
        await checkStatus();
      }
    } catch (e) {
      errorMsg = String(e);
    } finally {
      retryLoading = false;
    }
  }

  function finalizeStream(q, { content, answer, subdomain, category }) {
    const turn = {
      question: q,
      answer: content || streaming?.answer || '',
      boxed: answer || '',
      subdomain: subdomain || streaming?.subdomain || 'other',
      category: category || streaming?.category || 'other',
      chunks: [],
      prompt: ''
    };
    turns = [...turns, turn];
    saveHistory();
    streaming = null;
    loading = false;
  }

  function submit() {
    const q = input.trim();
    if (!q || loading) return;
    input = '';
    loading = true;
    errorMsg = '';
    streaming = { question: q, answer: '', subdomain: 'other', category: 'other' };
    let done = false;
    streamBackend(q, {
      onMeta: (m) => {
        if (streaming) { streaming = { ...streaming, subdomain: m.subdomain || streaming.subdomain, category: m.category || streaming.category }; }
      },
      onDelta: (d) => {
        if (streaming) streaming = { ...streaming, answer: streaming.answer + d };
      },
      onDone: (e) => {
        if (done) return; done = true;
        finalizeStream(q, e);
      },
      onError: (err) => {
        if (done) return; done = true;
        const turn = { question: q, error: String(err) };
        turns = [...turns, turn];
        saveHistory();
        streaming = null;
        loading = false;
      }
    });
  }

  function handleKey(e) {
    if (e.key === 'Enter') submit();
    if (e.key === 'Escape') {
      // clear input
      input = '';
    }
  }
  function clearHistory() {
    turns = [];
    localStorage.removeItem('tutor-history');
    if (streamCleanup) { streamCleanup(); streamCleanup = null; }
  }
  function escapeHtml(s) {
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }

  // Port of internal/renderer/render.go extractSpans + KaTeX
  function renderMath(text) {
    if (!text) return '';
    // split into segments: text, inline \( \), display \[ \], $ $, $$ $$
    const segs = [];
    let i = 0, prev = 0;
    while (i < text.length) {
      let opener = '', closer = '', typ = '';
      if (text[i] === '$' && i+1 < text.length && text[i+1] === '$') {
        opener = '$$'; closer = '$$'; typ = 'display';
      } else if (text[i] === '$') {
        opener = '$'; closer = '$'; typ = 'inline';
      } else if (text[i] === '\\' && i+1 < text.length && text[i+1] === '(') {
        opener = '\\('; closer = '\\)'; typ = 'inline';
      } else if (text[i] === '\\' && i+1 < text.length && text[i+1] === '[') {
        opener = '\\['; closer = '\\]'; typ = 'display';
      }
      if (!opener) { i++; continue; }
      if (i > prev) segs.push({typ:'text', s: text.slice(prev, i)});
      const bodyStart = i + opener.length;
      const j = text.indexOf(closer, bodyStart);
      if (j >= 0) {
        segs.push({typ, s: text.slice(bodyStart, j)});
        i = bodyStart + j - bodyStart + closer.length;
        // Actually computed as bodyStart + j offset
        i = j + closer.length;
      } else {
        segs.push({typ, s: text.slice(bodyStart)});
        i = text.length;
      }
      prev = i;
    }
    if (i > prev) segs.push({typ:'text', s: text.slice(prev, i)});
    // render
    let out = '';
    for (const seg of segs) {
      if (seg.typ === 'text') {
        out += `<span class="prose">${escapeHtml(seg.s)}</span>`;
      } else if (seg.typ === 'inline') {
        try {
          out += katex.renderToString(seg.s, {throwOnError:false, displayMode:false});
        } catch {
          out += `<code>${escapeHtml(seg.s)}</code>`;
        }
      } else {
        try {
          out += `<div class="math-display">${katex.renderToString(seg.s, {throwOnError:false, displayMode:true})}</div>`;
        } catch {
          out += `<div class="math-display"><code>${escapeHtml(seg.s)}</code></div>`;
        }
      }
    }
    return out;
  }
</script>

<main>
  <header>
    <h1>tutor.gguf — on-device math tutor</h1>
    <div class="sub">Svelte desktop · Wails v2 · KaTeX · <button class="link" on:click={clearHistory}>Ctrl+L clear</button> · Esc clear input</div>
  </header>

  {#if checking}
    <div class="setup">Checking setup…</div>
  {:else if status && !status.ready}
    <div class="setup">
      <h3>Setup required — Tutor not ready</h3>
      <p class="muted">{status.startErr || 'Models or index missing. Run setup.'}</p>
      <ul class="status">
        <li>Llama: {status.llamaExists ? 'found ' + (status.llamaServer || '') : 'missing — run tutor setup'}</li>
        <li>Gen model: {status.genModelExists ? (status.genModelBytes/1e9).toFixed(2)+' GB' : 'missing'}</li>
        <li>Embed: {status.embedExists ? (status.embedBytes/1e6).toFixed(0)+' MB' : 'missing'}</li>
        <li>DB: {status.dbChunks} chunks @ {status.dbPath}</li>
      </ul>
      <div class="setup-row">
        <button on:click={retryInit} disabled={retryLoading}>{retryLoading ? 'Retrying…' : 'Retry'}</button>
        <button class="ghost" on:click={checkStatus}>Refresh</button>
        <span class="muted">Background setup: run <code>tutor setup</code> in terminal, then Retry. Chat is enabled when DB has chunks.</span>
      </div>
      {#if paths}<details><summary>Paths</summary><pre class="prompt">{JSON.stringify(paths, null, 2)}</pre></details>{/if}
    </div>
  {/if}

  <div class="transcript">
    {#each turns as t, idx}
      <div class="turn">
        <div class="q">Q: {t.question} {#if t.subdomain}<span class="pill">{t.category || t.subdomain}</span>{/if}</div>
        {#if t.error}
          <div class="error">error: {t.error}</div>
        {:else}
          <div class="answer">{@html renderMath(t.answer)}</div>
          {#if t.boxed}
            <div class="boxed">Final: <span class="badge">{t.boxed}</span></div>
          {/if}
          {#if t.chunks && t.chunks.length}
            <details><summary>Sources [{t.chunks.length}]</summary>
              <ul class="sources">
                {#each t.chunks as c, i}
                  <li><b>[{i+1}] ({c.Subdomain || c.subdomain})</b> {c.Text || c.text || ''} <span class="sim">{(c.Similarity||c.similarity||0).toFixed(3)}</span></li>
                {/each}
              </ul>
            </details>
          {/if}
          {#if t.prompt}
            <button class="link" on:click={()=> showPromptFor = showPromptFor===idx ? null : idx}>{showPromptFor===idx ? 'Hide' : 'View'} prompt sent to Qwen</button>
            {#if showPromptFor===idx}
              <pre class="prompt">{t.prompt}</pre>
            {/if}
          {/if}
        {/if}
      </div>
    {/each}
    {#if streaming}
      <div class="turn streaming">
        <div class="q">Q: {streaming.question}</div>
        <div class="answer">{@html renderMath(streaming.answer)} <span class="cursor">▍</span></div>
      </div>
    {/if}
    {#if turns.length===0 && !streaming}
      <div class="empty">Ask a math question — e.g. "Find the derivative of x^2" or "A JAMB candidate: the sum of the first n terms is 3n²+n. Find the common difference."</div>
    {/if}
  </div>

  <div class="input-row">
    {#if loading}<span class="spinner">● thinking</span>{/if}
    <input bind:value={input} on:keydown={handleKey} placeholder="Ask a math question (Enter to send)" maxlength="512" disabled={loading} />
    <button on:click={submit} disabled={loading || !input.trim()}>Send</button>
    <button class="ghost" on:click={clearHistory}>Clear</button>
  </div>
  {#if errorMsg}<div class="error">{errorMsg}</div>{/if}
</main>

<style>
  main { max-width: 900px; margin: 0 auto; padding: 16px; text-align: left; }
  header h1 { color: #7c5cff; margin: 0; font-size: 20px; }
  .sub { color: #888; font-size: 12px; margin-bottom: 12px; }
  .transcript { border: 1px solid #4b3b8b; border-radius: 12px; padding: 12px; min-height: 300px; max-height: 60vh; overflow-y: auto; background: rgba(255,255,255,0.03); }
  .turn { margin-bottom: 16px; padding-bottom: 12px; border-bottom: 1px solid rgba(255,255,255,0.08); }
  .q { font-weight: 700; color: #ffb86c; margin-bottom: 6px; }
  .pill { background: #2a2a4a; color: #a8aaff; padding: 2px 6px; border-radius: 99px; font-size: 11px; margin-left: 8px; }
  .answer :global(.katex) { font-size: 1.05em; }
  .answer :global(.math-display) { margin: 8px 0; }
  .boxed { margin-top: 6px; }
  .badge { background: #1a3a2a; border: 1px solid #2a6a4a; padding: 2px 8px; border-radius: 6px; }
  .error { color: #ff6b6b; }
  .sources { font-size: 12px; color: #bbb; }
  .sim { color: #777; }
  .prompt { white-space: pre-wrap; background: #111; padding: 8px; border-radius: 6px; font-size: 11px; overflow-x: auto; }
  .input-row { display: flex; gap: 8px; margin-top: 12px; align-items: center; }
  .input-row input { flex: 1; padding: 10px; border-radius: 8px; border: none; background: rgba(255,255,255,0.9); color: #111; }
  .input-row button { padding: 10px 14px; border-radius: 8px; border: none; cursor: pointer; background: #7c5cff; color: white; }
  .input-row button:disabled { opacity: 0.5; cursor: not-allowed; }
  .ghost { background: transparent !important; color: #aaa !important; border: 1px solid #444 !important; }
  .link { background: none; border: none; color: #8ab4ff; cursor: pointer; text-decoration: underline; padding: 0; }
  .spinner { color: #7c5cff; font-size: 12px; margin-right: 8px; }
  .cursor { animation: blink 1s infinite; }
  @keyframes blink { 50% { opacity: 0; } }
  .empty { color: #aaa; font-style: italic; padding: 20px; text-align: center; }
  .setup { border: 1px solid #6a4; background: rgba(100,170,80,0.08); border-radius: 12px; padding: 12px; margin-bottom: 12px; }
  .setup h3 { margin: 0 0 6px; color: #b6f0a0; }
  .muted { color: #aaa; font-size: 12px; }
  .status { font-size: 12px; color: #bbb; }
  .setup-row { display: flex; gap: 8px; align-items: center; margin-top: 8px; flex-wrap: wrap; }
</style>
