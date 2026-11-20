import { afterEach, beforeAll, expect, it } from 'vitest'
import { GLOBAL_LOCALES_KEY } from '@/i18n/external'
import { internal, setLocale, t } from '@/i18n'

beforeAll(() => {
  // jsdom has no innerText, which `htmlEncode` relies on.
  Object.defineProperty(HTMLElement.prototype, 'innerText', {
    configurable: true,
    get(this: HTMLElement) {
      return this.textContent
    },
    set(this: HTMLElement, value: string) {
      this.textContent = value
    },
  })
})

afterEach(() => {
  delete (window as unknown as Record<string, unknown>)[GLOBAL_LOCALES_KEY]
  setLocale('en', true)
})

it('re-resolves a locale that arrives after the first lookup', () => {
  // Mount resolves the locale before the external plugin scripts load, so the
  // first lookup can only find what the bundle ships (English here).
  setLocale('ja')
  expect(t('send')).toBe(internal.en.send)

  // The plugin script registers its dictionary when it runs.
  ;(window as unknown as Record<string, unknown>)[GLOBAL_LOCALES_KEY] = {
    ja: { ...internal.en, send: 'コメントを送信' },
  }

  // The same language early-returns, which is why mount has to force it.
  setLocale('ja')
  expect(t('send')).toBe(internal.en.send)

  setLocale('ja', true)
  expect(t('send')).toBe('コメントを送信')
})
