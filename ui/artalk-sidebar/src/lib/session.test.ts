import { beforeEach, describe, expect, it, vi } from 'vitest'

// The store only reaches into the widget for a live instance; a stub keeps this
// a pure state test.
vi.mock('./artalk', () => ({ getArtalkOrNull: () => null, avatarUrl: () => '' }))

// jsdom ships no matchMedia, which the boot-parameter reader uses for the theme.
window.matchMedia = ((query: string) => ({
  matches: false,
  media: query,
  onchange: null,
  addEventListener: () => {},
  removeEventListener: () => {},
  addListener: () => {},
  removeListener: () => {},
  dispatchEvent: () => false,
})) as unknown as typeof window.matchMedia

const { session } = await import('./session')

const view = () => session.get().view

describe('session view mode', () => {
  beforeEach(() => {
    localStorage.clear()
    session.logout()
  })

  it('never puts a visitor into the admin view', () => {
    session.setUser({ name: 'guest', is_admin: false })
    expect(view()).toBe('user')
    session.setView('admin')
    expect(view()).toBe('user')
  })

  it('lands a freshly signed-in admin in the console they manage', () => {
    session.setUser({ name: 'admin', is_admin: true })
    expect(view()).toBe('admin')
  })

  it('remembers a deliberate switch to the visitor view', () => {
    session.setUser({ name: 'admin', is_admin: true })
    session.setView('user')
    expect(view()).toBe('user')
    expect(localStorage.getItem('ATK_SIDEBAR_VIEW')).toBe('user')
    // Re-syncing the same admin (e.g. a profile refresh) must not bounce back.
    session.setUser({ name: 'admin', is_admin: true })
    expect(view()).toBe('user')
  })

  it('drops a demoted admin to the visitor view', () => {
    session.setUser({ name: 'admin', is_admin: true })
    session.setUser({ is_admin: false })
    expect(view()).toBe('user')
  })

  it('resets the view on logout', () => {
    session.setUser({ name: 'admin', is_admin: true })
    session.logout()
    expect(view()).toBe('user')
    expect(session.get().user.is_admin).toBe(false)
  })
})
