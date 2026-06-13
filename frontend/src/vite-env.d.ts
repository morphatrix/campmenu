/// <reference types="vite/client" />

// Injected at build time by vite.config.ts: last git commit date + short hash
// (or the build timestamp when git is unavailable).
declare const __APP_VERSION__: string

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
