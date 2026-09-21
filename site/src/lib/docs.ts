import { marked } from "marked";

// Single source of truth for the manual: docs/manual/*.md at the repo root,
// shared by the public site and the offline serve UI. Vite inlines each file
// (?raw) at build time, so the site stays fully static.
const sources = import.meta.glob("../../../docs/manual/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

export interface Chapter {
  slug: string;
  title: string;
  order: number;
  html: string;
  headings: { id: string; text: string }[];
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .trim()
    .replace(/[\s_]+/g, "-");
}

function parseFrontmatter(raw: string): { title: string; order: number; body: string } {
  const match = raw.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
  let title = "Untitled";
  let order = 999;
  let body = raw;
  if (match) {
    body = match[2];
    for (const line of match[1].split("\n")) {
      const [key, ...rest] = line.split(":");
      const value = rest.join(":").trim();
      if (key.trim() === "title") title = value;
      if (key.trim() === "order") order = Number(value) || 999;
    }
  }
  return { title, order, body };
}

// Rewrite [text](slug) cross-links between chapters to the docs route, and
// external repo links stay absolute (they are already absolute in source).
function rewriteLinks(html: string): string {
  return html.replace(/href="([a-z0-9-]+)"/g, 'href="#/docs/$1"');
}

// Wrap rendered tables in a scroll container so wide tables (config,
// benchmark) scroll inside their own box instead of blowing out the page
// grid on narrow screens. Applied here so the public site and the offline
// serve UI stay identical.
function wrapTables(html: string): string {
  return html
    .replace(/<table>/g, '<div class="table-scroll"><table>')
    .replace(/<\/table>/g, "</table></div>");
}

function addHeadingIds(html: string): { html: string; headings: { id: string; text: string }[] } {
  const headings: { id: string; text: string }[] = [];
  const seen = new Map<string, number>();
  const out = html.replace(/<h([23])>([\s\S]*?)<\/h\1>/g, (_m, level, inner) => {
    const text = inner.replace(/<[^>]+>/g, "");
    let id = slugify(text);
    const n = seen.get(id) ?? 0;
    seen.set(id, n + 1);
    if (n > 0) id = `${id}-${n}`;
    if (level === "2") headings.push({ id, text });
    return `<h${level} id="${id}">${inner}</h${level}>`;
  });
  return { html: out, headings };
}

marked.setOptions({ breaks: false });

function loadChapters(): Chapter[] {
  const chapters: Chapter[] = [];
  for (const [path, raw] of Object.entries(sources)) {
    const file = path.split("/").pop() ?? path;
    const slug = file.replace(/^\d+-/, "").replace(/\.md$/, "");
    const { title, order, body } = parseFrontmatter(raw);
    const rendered = wrapTables(rewriteLinks(marked.parse(body) as string));
    const { html, headings } = addHeadingIds(rendered);
    chapters.push({ slug, title, order, html, headings });
  }
  chapters.sort((a, b) => a.order - b.order);
  return chapters;
}

export const chapters: Chapter[] = loadChapters();

export function getChapter(slug: string): Chapter | undefined {
  return chapters.find((c) => c.slug === slug);
}
