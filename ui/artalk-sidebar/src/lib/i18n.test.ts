import { describe, expect, it } from 'vitest'
import { browserLocale, pickLocale } from './i18n'

describe('pickLocale', () => {
  it('lets the visitor’s saved choice win over everything else', () => {
    expect(pickLocale('ja', 'zh-CN', ['fr'])).toBe('ja')
  })

  it('falls back to the site locale when nothing was saved', () => {
    expect(pickLocale(null, 'zh-TW', ['fr'])).toBe('zh-TW')
  })

  it('treats a site locale of "auto" as the browser language', () => {
    expect(pickLocale(null, 'auto', ['fr-FR'])).toBe('fr')
  })

  it('ignores a saved value that is no longer shipped', () => {
    expect(pickLocale('de', 'zh-CN', ['fr'])).toBe('zh-CN')
  })

  it('ends on English when neither the site nor the browser match', () => {
    expect(pickLocale(null, '', ['de-DE'])).toBe('en')
  })
})

describe('browserLocale', () => {
  it('takes the first language it ships', () => {
    expect(browserLocale(['de-DE', 'zh-TW'])).toBe('zh-TW')
  })

  it('matches a language by its base tag', () => {
    expect(browserLocale(['pt-BR'])).toBe('pt-PT')
    expect(browserLocale(['en-GB'])).toBe('en')
  })
})
