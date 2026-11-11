// Theme preference: 'light' (archive paper) | 'dark' (ink on slate) |
// 'system' (follows OS). Persisted in localStorage as 'tutor-theme' and
// applied as data-theme on <html> before first paint (see index.html).
// Mirrors frontend/src/lib/theme.js so the site and the app agree.
export type Theme = "light" | "dark" | "system";

const KEY = "tutor-theme";

export function getTheme(): Theme {
  try {
    const v = localStorage.getItem(KEY);
    if (v === "light" || v === "dark" || v === "system") return v;
  } catch {
    // Storage can be blocked (private mode, embedded webview).
  }
  return "system";
}

export function applyTheme(theme?: Theme): Theme {
  const t = theme ?? getTheme();
  document.documentElement.dataset.theme = t;
  try {
    localStorage.setItem(KEY, t);
  } catch {
    // Non-fatal: the theme still applies for this page view.
  }
  return t;
}

export function cycleTheme(current: Theme): Theme {
  const next: Theme =
    current === "light" ? "dark" : current === "dark" ? "system" : "light";
  return applyTheme(next);
}
