import { marked } from 'marked'

export const DOCS_URL = 'https://chuma-beep.github.io/tutor.gguf/docs/'
export const REPO_URL = 'https://github.com/chuma-beep/tutor.gguf'

// Offline manual: docs/manual/*.md at the repo root, shared with the public
// site. Vite inlines each file (?raw) at build time, so the bundle — and the
// Go embed of frontend/dist — carries the docs with zero network need.
const sources = import.meta.glob('../../../docs/manual/*.md', {
  query: '?raw',
  import: 'default',
  eager: true
})

function slugify(text) {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .trim()
    .replace(/[\s_]+/g, '-')
}

function parseFrontmatter(raw) {
  const match = raw.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/)
  let title = 'Untitled'
  let order = 999
  let body = raw
  if (match) {
    body = match[2]
    for (const line of match[1].split('\n')) {
      const idx = line.indexOf(':')
      if (idx < 0) continue
      const key = line.slice(0, idx).trim()
      const value = line.slice(idx + 1).trim()
      if (key === 'title') title = value
      if (key === 'order') order = Number(value) || 999
    }
  }
  return { title, order, body }
}

// Cross-chapter links [text](slug) become in-app hooks (data-docs); absolute
// URLs pass through untouched for the online footer.
function rewriteLinks(html) {
  return html.replace(/href="([a-z0-9-]+)"/g, 'href="#" data-docs="$1"')
}

// Wrap rendered tables in a scroll container so wide tables (config,
// benchmark) scroll inside their own box instead of blowing out the
// transcript on narrow screens. Mirrors site/src/lib/docs.ts.
function wrapTables(html) {
  return html
    .replace(/<table>/g, '<div class="table-scroll"><table>')
    .replace(/<\/table>/g, '</table></div>')
}

function addHeadingIds(html) {
  const headings = []
  const seen = new Map()
  const out = html.replace(/<h([23])>([\s\S]*?)<\/h\1>/g, (_m, level, inner) => {
    const text = inner.replace(/<[^>]+>/g, '')
    let id = slugify(text)
    const n = seen.get(id) ?? 0
    seen.set(id, n + 1)
    if (n > 0) id = `${id}-${n}`
    if (level === '2') headings.push({ id, text })
    return `<h${level} id="${id}">${inner}</h${level}>`
  })
  return { html: out, headings }
}

marked.setOptions({ breaks: false })

function loadChapters() {
  const chapters = []
  for (const [path, raw] of Object.entries(sources)) {
    const file = path.split('/').pop() ?? path
    const slug = file.replace(/^\d+-/, '').replace(/\.md$/, '')
    const { title, order, body } = parseFrontmatter(raw)
    const rendered = wrapTables(rewriteLinks(marked.parse(body)))
    const { html, headings } = addHeadingIds(rendered)
    chapters.push({ slug, title, order, html, headings })
  }
  chapters.sort((a, b) => a.order - b.order)
  return chapters
}

export const chapters = loadChapters()

export function getChapter(slug) {
  return chapters.find((c) => c.slug === slug)
}
