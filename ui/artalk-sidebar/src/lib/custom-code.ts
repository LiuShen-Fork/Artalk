import { api } from './api'

/**
 * The console lets a site owner inject their own CSS and JavaScript without a
 * rebuild (see `custom` in the config). Both snippets are fetched once at boot
 * and re-applied when the owner saves new ones from the customize page.
 */
export interface CustomCode {
  css: string
  js: string
}

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

export function applyCustomCode(code: Partial<CustomCode>) {
  applyCustomCss(code.css ?? '')
  applyCustomJs(code.js ?? '')
}

/** Best effort: an unreachable or older server simply means "no custom code". */
export async function loadCustomCode(): Promise<CustomCode> {
  try {
    const data = await api<CustomCode>('/sidebar/custom')
    return { css: data?.css ?? '', js: data?.js ?? '' }
  } catch {
    return { css: '', js: '' }
  }
}

/** Styles first (no flash of default theme), script after the first paint so it can see the DOM. */
export async function installCustomCode() {
  const code = await loadCustomCode()
  applyCustomCss(code.css)
  requestAnimationFrame(() => applyCustomJs(code.js))
}
