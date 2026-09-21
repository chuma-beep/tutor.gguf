import { createRoute, Link, useNavigate, useParams } from "@tanstack/react-router";
import { useEffect } from "react";
import { rootRoute } from "./root";
import { chapters, getChapter } from "@/lib/docs";

const REPO = "https://github.com/chuma-beep/tutor.gguf";

function useInterceptDocsLinks() {
  const navigate = useNavigate();
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      const anchor = (e.target as HTMLElement).closest?.('a[href^="#/docs"]') as HTMLAnchorElement | null;
      if (!anchor) return;
      e.preventDefault();
      const href = anchor.getAttribute("href") ?? "";
      if (href === "#/docs") navigate({ to: "/docs" });
      else {
        const slug = href.replace("#/docs/", "");
        navigate({ to: "/docs/$slug", params: { slug } });
      }
      window.scrollTo({ top: 0 });
    };
    document.addEventListener("click", onClick);
    return () => document.removeEventListener("click", onClick);
  }, [navigate]);
}

function Shell({ slug }: { slug?: string }) {
  useInterceptDocsLinks();
  const chapter = (slug ? getChapter(slug) : chapters[0]) ?? chapters[0];
  const index = chapters.findIndex((c) => c.slug === chapter.slug);
  const prev = chapters[index - 1];
  const next = chapters[index + 1];

  return (
    <div className="min-h-screen bg-muted text-foreground selection:bg-foreground selection:text-background">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-[1440px] items-center justify-between px-5 py-4 sm:px-8 lg:px-12">
          <Link to="/" className="text-sm font-semibold uppercase">tutor.gguf</Link>
          <nav aria-label="Primary navigation" className="flex gap-5 font-mono text-[10px] uppercase text-muted-foreground">
            <Link className="archive-link" to="/docs">Docs</Link>
            <a className="archive-link" href={REPO} target="_blank" rel="noreferrer">Repository ↗</a>
          </nav>
        </div>
      </header>

      <main className="mx-auto max-w-[1440px] px-4 py-4 sm:px-8 sm:py-8 lg:px-12 lg:py-12">
        <article className="border border-border bg-background">
          <header className="grid border-b border-border lg:grid-cols-12">
            <div className="p-4 sm:p-10 lg:col-span-8 lg:p-14">
              <p className="archive-label">Manual / TG-001</p>
              <h1 className="mt-5 wrap-anywhere text-3xl font-medium uppercase leading-[0.9] sm:text-5xl lg:text-7xl">
                {chapter.title}
              </h1>
            </div>
            <dl className="grid min-w-0 grid-cols-2 border-t border-border font-mono text-[10px] uppercase lg:col-span-4 lg:border-l lg:border-t-0">
              {[["Chapter", `${String(index + 1).padStart(2, "0")} / ${String(chapters.length).padStart(2, "0")}`], ["Source", "docs/manual"], ["Format", "Markdown"], ["Status", "Main"]].map(([term, value]) => (
                <div key={term} className="min-w-0 border-b border-r border-border p-4 sm:p-5">
                  <dt className="text-muted-foreground">{term}</dt><dd className="mt-2 wrap-anywhere text-foreground">{value}</dd>
                </div>
              ))}
            </dl>
          </header>

          <div className="grid lg:grid-cols-12">
            <nav aria-label="Chapters" className="min-w-0 border-b border-border p-4 sm:p-10 lg:col-span-4 lg:border-b-0 lg:border-r lg:p-12">
              <p className="archive-label">Chapters</p>
              <ol className="mt-5 flex gap-2 overflow-x-auto pb-2 lg:block lg:space-y-1 lg:overflow-visible lg:pb-0">
                {chapters.map((c, i) => (
                  <li key={c.slug} className="shrink-0">
                    <Link
                      to="/docs/$slug"
                      params={{ slug: c.slug }}
                      className={`flex items-center gap-2 whitespace-nowrap rounded-sm border border-border px-3 py-2.5 text-sm transition-colors lg:items-baseline lg:rounded-none lg:border-0 lg:border-b lg:px-1 lg:py-2 ${c.slug === chapter.slug ? "font-medium text-foreground" : "text-muted-foreground hover:text-foreground"}`}
                    >
                      <span className="font-mono text-[10px]">{String(i + 1).padStart(2, "0")}</span>
                      <span>{c.title}</span>
                    </Link>
                  </li>
                ))}
              </ol>
              {chapter.headings.length > 0 && (
                <>
                  <details className="mt-6 lg:hidden">
                    <summary className="archive-label cursor-pointer">On this page</summary>
                    <ol className="mt-4 space-y-2">
                      {chapter.headings.map((h) => (
                        <li key={h.id}>
                          <a href={`#${h.id}`} className="archive-link font-mono text-[10px] uppercase text-muted-foreground">{h.text}</a>
                        </li>
                      ))}
                    </ol>
                  </details>
                  <div className="hidden lg:block">
                    <p className="archive-label mt-10">On this page</p>
                    <ol className="mt-5 space-y-2">
                      {chapter.headings.map((h) => (
                        <li key={h.id}>
                          <a href={`#${h.id}`} className="archive-link font-mono text-[10px] uppercase text-muted-foreground">{h.text}</a>
                        </li>
                      ))}
                    </ol>
                  </div>
                </>
              )}
            </nav>

            <div className="min-w-0 p-4 sm:p-10 lg:col-span-8 lg:p-12">
              <div className="manual" dangerouslySetInnerHTML={{ __html: chapter.html }} />
              <div className="mt-12 grid gap-px border border-border bg-border sm:grid-cols-2">
                <div className="bg-background p-5">
                  {prev ? (
                    <Link to="/docs/$slug" params={{ slug: prev.slug }} className="archive-action">← {prev.title}</Link>
                  ) : <span className="archive-label text-muted-foreground">Start</span>}
                </div>
                <div className="bg-background p-5 text-left sm:text-right">
                  {next ? (
                    <Link to="/docs/$slug" params={{ slug: next.slug }} className="archive-action">{next.title} →</Link>
                  ) : <span className="archive-label text-muted-foreground">End</span>}
                </div>
              </div>
            </div>
          </div>

          <footer className="flex flex-col justify-between gap-5 border-t border-border bg-muted px-4 py-6 font-mono text-[10px] uppercase text-muted-foreground sm:flex-row sm:items-center sm:px-7">
            <span>Manual / TG-001</span>
            <span>Single source: docs/manual</span>
            <span>GPL-3.0 / Public release</span>
          </footer>
        </article>
      </main>
    </div>
  );
}

export const docsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/docs",
  component: () => <Shell />,
});

export const docsSlugRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/docs/$slug",
  component: DocsSlug,
});

function DocsSlug() {
  const { slug } = useParams({ from: "/docs/$slug" });
  const chapter = getChapter(slug);
  if (!chapter) return <Shell slug={chapters[0]?.slug} />;
  return <Shell slug={slug} />;
}
