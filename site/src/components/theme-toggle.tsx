import { useEffect, useState } from "react";
import { applyTheme, cycleTheme, getTheme, type Theme } from "@/lib/theme";

const LABELS: Record<Theme, string> = {
  light: "○ Light",
  dark: "● Dark",
  system: "◐ System",
};

// On narrow screens the label collapses to its glyph so the masthead keeps
// the wordmark, both links and the toggle on one line at 320px.
const GLYPHS: Record<Theme, string> = {
  light: "○",
  dark: "●",
  system: "◐",
};

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(getTheme);

  // Follow changes made in another tab (docs and landing share the key).
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key !== "tutor-theme" && e.key !== null) return;
      const next = getTheme();
      applyTheme(next);
      setTheme(next);
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  return (
    <button
      type="button"
      className="archive-link -m-1 p-1"
      onClick={() => setTheme(cycleTheme(theme))}
      title="Colour theme: light archive, dark slate, or follow system"
      aria-label={`Colour theme: ${theme}. Activate to switch.`}
    >
      <span className="sm:hidden" aria-hidden="true">{GLYPHS[theme]}</span>
      <span className="hidden sm:inline">{LABELS[theme]}</span>
    </button>
  );
}
