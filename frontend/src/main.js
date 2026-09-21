import './style.css'
import { applyTheme } from './lib/theme.js'
import App from './App.svelte'

// Apply persisted theme (or OS preference) before first paint.
applyTheme()

const app = new App({
  target: document.getElementById('app')
})

export default app
