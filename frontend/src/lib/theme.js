// Theme preference: 'light' (default, archive paper) | 'dark' (chalkboard) |
// 'system' (follows OS). Persisted in localStorage as 'tutor-theme' and
// applied as data-theme on <html> before first paint (see main.js).
const KEY = 'tutor-theme'

export function getTheme() {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {}
  return 'light'
}

export function applyTheme(theme) {
  const t = theme || getTheme()
  document.documentElement.setAttribute('data-theme', t)
  try { localStorage.setItem(KEY, t) } catch {}
  return t
}

export function cycleTheme(current) {
  const next = current === 'light' ? 'dark' : current === 'dark' ? 'system' : 'light'
  return applyTheme(next)
}
