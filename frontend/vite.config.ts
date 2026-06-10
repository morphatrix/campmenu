import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// API base is injected at runtime via the <meta name="campmenu-api"> tag in
// index.html (replaced at pod start), so the same build artifact works in any
// environment, or falls back to the VITE_API_URL build-time variable for dev.
export default defineConfig({
  plugins: [react()],
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
