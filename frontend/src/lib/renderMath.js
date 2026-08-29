import katex from 'katex'

export function escapeHtml(s) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

// Port of internal/renderer/render.go extractSpans + KaTeX.
// Precedence: $$ display > $ inline > \( inline > \[ display.
// An unclosed span renders as math through EOF (streaming tolerance).
export function renderMath(text) {
  if (!text) return ''
  const segs = []
  let i = 0
  let prev = 0
  while (i < text.length) {
    let opener = ''
    let closer = ''
    let typ = ''
    if (text[i] === '$' && i + 1 < text.length && text[i + 1] === '$') {
      opener = '$$'; closer = '$$'; typ = 'display'
    } else if (text[i] === '$') {
      opener = '$'; closer = '$'; typ = 'inline'
    } else if (text[i] === '\\' && i + 1 < text.length && text[i + 1] === '(') {
      opener = '\\('; closer = '\\)'; typ = 'inline'
    } else if (text[i] === '\\' && i + 1 < text.length && text[i + 1] === '[') {
      opener = '\\['; closer = '\\]'; typ = 'display'
    }
    if (!opener) { i++; continue }
    if (i > prev) segs.push({ typ: 'text', s: text.slice(prev, i) })
    const bodyStart = i + opener.length
    const j = text.indexOf(closer, bodyStart)
    if (j >= 0) {
      segs.push({ typ, s: text.slice(bodyStart, j) })
      i = j + closer.length
    } else {
      segs.push({ typ, s: text.slice(bodyStart) })
      i = text.length
    }
    prev = i
  }
  if (i > prev) segs.push({ typ: 'text', s: text.slice(prev, i) })

  let out = ''
  for (const seg of segs) {
    if (seg.typ === 'text') {
      out += `<span class="prose">${escapeHtml(seg.s)}</span>`
    } else if (seg.typ === 'inline') {
      try {
        out += katex.renderToString(seg.s, { throwOnError: false, displayMode: false })
      } catch {
        out += `<code>${escapeHtml(seg.s)}</code>`
      }
    } else {
      try {
        out += `<div class="math-display">${katex.renderToString(seg.s, { throwOnError: false, displayMode: true })}</div>`
      } catch {
        out += `<div class="math-display"><code>${escapeHtml(seg.s)}</code></div>`
      }
    }
  }
  return out
}
