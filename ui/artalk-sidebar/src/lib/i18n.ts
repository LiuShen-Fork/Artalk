import { useSyncExternalStore } from 'react'
import type { MessageSchema } from '../i18n'
import { en } from '../i18n-en'

export type MessageKey = keyof MessageSchema
export type TranslateParams = Record<string, string | number>
export type Translate = (key: MessageKey, params?: TranslateParams) => string

const SUPPORTED = ['en', 'zh-CN', 'zh-TW', 'ja', 'ko', 'fr', 'pt-PT', 'ru', 'tr'] as const
type Locale = (typeof SUPPORTED)[number]

/** The console remembers the language its visitor picked, like the theme. */
export const LOCALE_STORAGE_KEY = 'ATK_SIDEBAR_LOCALE'

let locale: Locale = 'en'
let dictionary: Record<string, string> = en as unknown as Record<string, string>
let loaded: Locale | null = 'en'
let version = 0
const listeners = new Set<() => void>()

// English is the built-in locale; mark the document even when `setLocale('en')`
// is a no-op, so the page always advertises a language.
if (typeof document !== 'undefined') document.documentElement.lang = locale

function normalize(value: string): Locale | null {
  if (!value || value === 'auto') return null
  const exact = SUPPORTED.find((item) => item.toLowerCase() === value.toLowerCase())
  if (exact) return exact
  const base = value.split('-')[0]?.toLowerCase()
  return SUPPORTED.find((item) => item.toLowerCase().split('-')[0] === base) ?? null
}

/** The best shipped locale for the browser's language list. */
export function browserLocale(languages?: readonly string[]): Locale {
  const list =
    languages ?? (typeof navigator === 'undefined' ? [] : (navigator.languages ?? [navigator.language]))
  for (const value of list) {
    const match = normalize(value)
    if (match) return match
  }
  return 'en'
}

/**
 * Which locale the console should start with: the visitor's saved choice
 * first, then the site's `locale` (an `auto` value means "the browser"), and
 * finally the browser itself. The site no longer pins the console to a single
 * language — that is only the default.
 */
export function pickLocale(
  saved: string | null | undefined,
  siteLocale: string | null | undefined,
  languages?: readonly string[],
): Locale {
  return normalize(saved ?? '') ?? normalize(siteLocale ?? '') ?? browserLocale(languages)
}

function readSavedLocale(): string | null {
  try {
    return localStorage.getItem(LOCALE_STORAGE_KEY)
  } catch {
    return null
  }
}

/** True once the visitor picked a language in the console. */
export function hasLocaleChoice(): boolean {
  return normalize(readSavedLocale() ?? '') != null
}

/** The locale to load on boot, before any server round-trip. */
export function initialLocale(siteLocale?: string): Locale {
  return pickLocale(readSavedLocale(), siteLocale)
}

/** Switch the console language and remember the choice. */
export function chooseLocale(value: string): Promise<void> {
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, value)
  } catch {
    // Private mode: the language still applies to this session.
  }
  return setLocale(value)
}

/** Translate with the active locale, interpolating `{name}` placeholders. */
export function t(key: MessageKey, params?: TranslateParams): string {
  let text = dictionary[key] ?? (en as Record<string, string>)[key] ?? key
  if (params) {
    text = text.replace(/\{(\w+)\}/g, (match, name: string) =>
      params[name] != null ? String(params[name]) : match,
    )
  }
  return text
}

export function getLocale(): Locale {
  return locale
}

export function subscribeI18n(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function i18nVersion() {
  return version
}

/**
 * Re-render the tree whenever the active locale changes (the Artalk conf can
 * switch it at runtime, e.g. when the host admin panel changes language).
 */
export function useLocaleVersion(): number {
  return useSyncExternalStore(subscribeI18n, i18nVersion, i18nVersion)
}

/** The active locale, re-rendering the caller when it changes. */
export function useLocale(): string {
  return useSyncExternalStore(subscribeI18n, getLocale, getLocale)
}

export async function setLocale(value: string): Promise<void> {
  const next = normalize(value)
  if (!next || (next === locale && loaded === next)) return

  try {
    if (next === 'en') {
      dictionary = en as unknown as Record<string, string>
    } else {
      const module: any = await import(`../i18n/${next}.ts`)
      dictionary = (module.default || module[Object.keys(module)[0]]) as Record<string, string>
    }
    locale = next
    loaded = next
    document.documentElement.lang = next
    version += 1
    listeners.forEach((listener) => listener())
  } catch (error) {
    console.error(`Failed to load locale "${next}"`, error)
  }
}
