import { Link } from "@tanstack/react-router";
import { ThemeToggle } from "./theme-toggle";

const REPO = "https://github.com/chuma-beep/tutor.gguf";

// Shared masthead for the landing page and the manual. The only difference
// is the landing page's in-page anchors and the docs wordmark, which returns
// to the index via the router.
export function ArchiveHeader({ variant }: { variant: "landing" | "docs" }) {
  const landing = variant === "landing";

  return (
    <header className="border-b border-border bg-background">
      <div className="mx-auto flex max-w-[1440px] items-center justify-between px-5 py-4 sm:px-8 lg:px-12">
        {landing ? (
          <a href="#top" className="text-sm font-semibold uppercase">tutor.gguf</a>
        ) : (
          <Link to="/" className="text-sm font-semibold uppercase">tutor.gguf</Link>
        )}
        <nav
          aria-label="Primary navigation"
          className="flex gap-3 font-mono text-[10px] uppercase text-muted-foreground sm:gap-5"
        >
          {landing && (
            <>
              <a className="archive-link hidden sm:inline" href="#archive">Archive</a>
              <a className="archive-link hidden sm:inline" href="#specification">Specification</a>
            </>
          )}
          <Link className="archive-link" to="/docs">Docs</Link>
          <a className="archive-link" href={REPO} target="_blank" rel="noreferrer">
            <span className="hidden sm:inline">Repository</span>
            <span className="sm:hidden">Repo</span>
            <span className="hidden sm:inline"> ↗</span>
          </a>
          <ThemeToggle />
        </nav>
      </div>
    </header>
  );
}
