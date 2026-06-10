import type { User } from './types'

// applyAppearance reflects a user's preferences onto <html>: theme (light/dark
// /auto), color palette and colorblind mode. Called whenever the user changes.
export function applyAppearance(user: User | null) {
  const root = document.documentElement
  const theme = user?.theme ?? 'auto'
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  const dark = theme === 'dark' || (theme === 'auto' && prefersDark)
  root.classList.toggle('dark', dark)
  root.setAttribute('data-palette', user?.colorPalette ?? 'default')
  root.setAttribute('data-colorblind', String(user?.colorblindMode ?? false))
}

export const PALETTES = ['default', 'sunset', 'forest', 'ocean'] as const
export const THEMES = ['light', 'dark', 'auto'] as const
