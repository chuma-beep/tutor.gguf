import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  base: './',
  plugins: [svelte()],
  server: {
    // Plain `vite dev` proxies API calls to a local `tutor serve` so the
    // frontend can use relative fetch() URLs that also work when the same
    // build is served from :8082/ or inside Wails (bindings path).
    proxy: {
      '/v1': 'http://127.0.0.1:8082',
      '/health': 'http://127.0.0.1:8082'
    },
    // The offline manual source (docs/manual) lives outside the frontend root.
    fs: { allow: ['..'] }
  }
})
