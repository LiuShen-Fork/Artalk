import { useSyncExternalStore } from 'react'
import { bootParams } from './boot'

let dark = bootParams.darkMode
const listeners = new Set<() => void>()

function apply() {
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.dataset.theme = dark ? 'dark' : 'light'
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
}

// Apply before the first paint so the console never flashes the wrong theme.
apply()

if (typeof window !== 'undefined' && window.matchMedia) {
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  media.addEventListener?.('change', (event) => {
    if (localStorage.getItem('ATK_SIDEBAR_DARK_MODE') == null) {
      dark = event.matches
      apply()
      listeners.forEach((listener) => listener())
    }
  })
}

export const theme = {
  get: () => dark,
  subscribe(listener: () => void) {
    listeners.add(listener)
    return () => listeners.delete(listener)
  },
  set(value: boolean) {
    dark = value
    localStorage.setItem('ATK_SIDEBAR_DARK_MODE', value ? '1' : '0')
    apply()
    listeners.forEach((listener) => listener())
  },
  toggle() {
    theme.set(!dark)
  },
  /** Called after Artalk boots so the embedded widget follows the console. */
  syncArtalk() {
    // Imported lazily to avoid a cycle with lib/artalk.
    import('./artalk').then(({ getArtalkOrNull }) => getArtalkOrNull()?.setDarkMode(dark))
  },
}

export function useDarkMode(): boolean {
  return useSyncExternalStore(theme.subscribe, theme.get, theme.get)
}
