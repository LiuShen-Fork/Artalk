import { useSyncExternalStore } from 'react'
import { api } from './api'

/**
 * Site identity and custom code of this console, editable from the console's
 * own "site settings" page (`console` in the config file).
 *
 * The whole payload is public: the sign-in page has to render the same name,
 * icon and description before anyone authenticates.
 */
export interface ConsoleSettings {
  name: string
  icon: string
  description: string
  css: string
  js: string
}

export const DEFAULT_CONSOLE_NAME = 'Artalk'

const EMPTY: ConsoleSettings = { name: '', icon: '', description: '', css: '', js: '' }

const STYLE_ID = 'artk-custom-css'
const SCRIPT_ID = 'artk-custom-js'

export function applyCustomCss(css: string) {
  const existing = document.getElementById(STYLE_ID) as HTMLStyleElement | null
  if (!css) {
    existing?.remove()
    return
  }
  const style = existing ?? document.createElement('style')
  if (!existing) {
    style.id = STYLE_ID
    // Appended last, so equal-specificity rules win over the bundled styles.
    document.head.append(style)
  }
  style.textContent = css
}

export function applyCustomJs(js: string) {
  // Re-inserting the element is the only way to re-run a script.
  document.getElementById(SCRIPT_ID)?.remove()
  if (!js) return
  const script = document.createElement('script')
  script.id = SCRIPT_ID
  script.textContent = js
  document.head.append(script)
}

export function applyConsole(settings: Partial<ConsoleSettings>) {
  applyCustomCss(settings.css ?? '')
  applyCustomJs(settings.js ?? '')
}

/** Name to print: the configured one, or the product name. */
export function consoleName(settings: ConsoleSettings): string {
  return settings.name.trim() || DEFAULT_CONSOLE_NAME
}

/** Letter badge used when no icon is configured. */
export function consoleInitial(settings: ConsoleSettings): string {
  return consoleName(settings).trim().charAt(0).toUpperCase() || 'A'
}

let state: ConsoleSettings = EMPTY
let version = 0
const listeners = new Set<() => void>()

function emit() {
  version += 1
  listeners.forEach((listener) => listener())
}

export const consoleStore = {
  get(): ConsoleSettings {
    return state
  },
  subscribe(listener: () => void) {
    listeners.add(listener)
    return () => listeners.delete(listener)
  },
  version(): number {
    return version
  },
  /** Merge a server payload in; styles and scripts follow immediately. */
  set(next: Partial<ConsoleSettings>) {
    state = { ...state, ...next }
    applyConsole(state)
    emit()
  },
  /**
   * The console body can carry a copy of the settings (admin only). Anyone else
   * asks the public endpoint, which is also how the sign-in page gets its look.
   */
  hydrate(next: Partial<ConsoleSettings>) {
    state = { ...state, ...next }
    emit()
  },
}

export function useConsoleSettings(): ConsoleSettings {
  useSyncExternalStore(consoleStore.subscribe, consoleStore.version, consoleStore.version)
  return state
}

/** Name line for the sidebar and sign-in page. */
export function useBrandName(): string {
  return consoleName(useConsoleSettings())
}

/** Best effort: an unreachable or older server simply means "default look". */
export async function loadConsoleSettings(): Promise<ConsoleSettings> {
  try {
    const data = await api<ConsoleSettings>('/sidebar/console')
    return { ...EMPTY, ...data }
  } catch {
    return EMPTY
  }
}

/** Styles first (no flash of the default theme), script after the first paint so it can see the DOM. */
export async function installConsole() {
  const settings = await loadConsoleSettings()
  consoleStore.hydrate(settings)
  applyCustomCss(settings.css)
  requestAnimationFrame(() => applyCustomJs(settings.js))
}
