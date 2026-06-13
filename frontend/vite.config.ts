import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { execSync } from 'node:child_process'

// buildVersion resolves the "last commit date · short hash" shown discreetly in
// the UI so we know which version is deployed. Prefers an explicit override,
// then live git, then the build timestamp when git/.git is unavailable.
function buildVersion(): string {
  if (process.env.VITE_APP_VERSION) return process.env.VITE_APP_VERSION
  try {
    return execSync("git log -1 --date=format:'%Y-%m-%d %H:%M' --format='%cd · %h'", {
      stdio: ['ignore', 'pipe', 'ignore'],
    }).toString().trim()
  } catch {
    return `build ${new Date().toISOString().slice(0, 16).replace('T', ' ')}`
  }
}

// API base is injected at runtime via the <meta name="campmenu-api"> tag in
// index.html (replaced at pod start), so the same build artifact works in any
// environment, or falls back to the VITE_API_URL build-time variable for dev.
export default defineConfig({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(buildVersion()),
  },
  // Disable the module-preload polyfill so the built index.html contains no
  // inline script, keeping the Content-Security-Policy free of 'unsafe-inline'.
  build: {
    modulePreload: { polyfill: false },
  },
  server: {
    host: true,
    port: 5173,
  },
})
