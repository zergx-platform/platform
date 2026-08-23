import { type ShadcnTheme, themeData } from './themes-data'

const STORAGE_KEY_THEME = 'opencode-theme-id'
const STORAGE_KEY_APPEARANCE = 'oc-appearance'

const STYLE_ID = 'oc-theme-dynamic'

export const ALL_THEMES = Object.keys(themeData).sort()

function getStyleEl(): HTMLStyleElement {
  let el = document.getElementById(STYLE_ID) as HTMLStyleElement | null
  if (!el) {
    el = document.createElement('style')
    el.id = STYLE_ID
    document.head.appendChild(el)
  }
  return el
}

function themeToCSS(colors: ShadcnTheme, radius: string = '0.625rem'): string {
  const cssVars = Object.entries(colors).map(([k, v]) => `--${k}:${v}`)
  cssVars.push(`--radius:${radius}`)
  return `:root{${cssVars.join(';')}}`
}

const darkOnlyCache = new Map<string, boolean>()

export function isDarkOnlyTheme(themeId: string): boolean {
  const cached = darkOnlyCache.get(themeId)
  if (cached !== undefined) return cached
  const entry = themeData[themeId]
  if (!entry) return false
  const is =
    entry.dark &&
    entry.light &&
    entry.dark.background === entry.light.background
  darkOnlyCache.set(themeId, is)
  return is
}

export function applyTheme(themeId: string, mode: 'dark' | 'light') {
  const entry = themeData[themeId]
  if (!entry) return

  const effectiveMode = isDarkOnlyTheme(themeId) ? 'dark' : mode
  const colors = effectiveMode === 'dark' ? entry.dark : entry.light
  const css = themeToCSS(colors)

  const el = getStyleEl()
  el.textContent = css

  document.documentElement.setAttribute('data-theme', effectiveMode)
  localStorage.setItem(STORAGE_KEY_THEME, themeId)
  localStorage.setItem(STORAGE_KEY_APPEARANCE, mode)
}

export function getSavedTheme(): string {
  return localStorage.getItem(STORAGE_KEY_THEME) || 'opencode'
}

export function getSavedMode(): 'dark' | 'light' {
  const saved = localStorage.getItem(STORAGE_KEY_APPEARANCE)
  return saved === 'light' ? 'light' : 'dark'
}

export function initTheme() {
  const theme = getSavedTheme()
  const mode = getSavedMode()
  applyTheme(theme, mode)
}
