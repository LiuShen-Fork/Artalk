import { useSyncExternalStore } from 'react'
import type { MessageSchema } from '../i18n'
import { en } from '../i18n-en'

export type MessageKey = keyof MessageSchema
export type TranslateParams = Record<string, string | number>
export type Translate = (key: MessageKey, params?: TranslateParams) => string

const SUPPORTED = ['en', 'zh-CN', 'zh-TW', 'ja', 'ko', 'fr', 'pt-PT', 'ru', 'tr'] as const
type Locale = (typeof SUPPORTED)[number]

let locale: Locale = 'en'
let dictionary: Record<string, string> = en as unknown as Record<string, string>
let loaded: Locale | null = 'en'
let version = 0
const listeners = new Set<() => void>()

function normalize(value: string): Locale | null {
  if (!value || value === 'auto') return null
  const exact = SUPPORTED.find((item) => item.toLowerCase() === value.toLowerCase())
  if (exact) return exact
  const base = value.split('-')[0]?.toLowerCase()
  return SUPPORTED.find((item) => item.toLowerCase().split('-')[0] === base) ?? null
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
