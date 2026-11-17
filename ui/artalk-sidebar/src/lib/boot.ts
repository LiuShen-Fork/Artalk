import type { LocalUser } from 'artalk'

export interface BootParams {
  user: LocalUser
  guestToken: string
  pageKey: string
  site: string
  view: string
  darkMode: boolean
}

function readBootParams(): BootParams {
  const search = new URLSearchParams(document.location.search)
  const fragment = new URLSearchParams(document.location.hash.slice(1))
  const guestToken = fragment.get('guest_token') || fragment.get('atk_guest') || ''

  // Consume credentials before the first render. Old clients may still put the
  // token in the query string; clearing it here cannot undo that initial HTTP
  // request, but it keeps the credential out of history and Referer headers.
  if (
    (search.has('user') || fragment.has('token') || !!guestToken) &&
    window.history.replaceState
  ) {
    window.history.replaceState({}, '', window.location.pathname)
  }

  let profile: Partial<LocalUser> = {}
  try {
    const parsed = JSON.parse(search.get('user') || '{}')
    if (parsed && typeof parsed === 'object') profile = parsed
  } catch {
    // A malformed legacy profile must not prevent credential cleanup or boot.
  }

  const legacyToken = typeof profile.token === 'string' ? profile.token : ''
  const user: LocalUser = {
    name: typeof profile.name === 'string' ? profile.name : '',
    email: typeof profile.email === 'string' ? profile.email : '',
    link: typeof profile.link === 'string' ? profile.link : '',
    token: fragment.get('token') ?? legacyToken,
    is_admin: profile.is_admin === true,
  }

  const stored = localStorage.getItem('ATK_SIDEBAR_DARK_MODE')
  const darkMode =
    search.get('darkMode') != null
      ? search.get('darkMode') === '1'
      : stored != null
        ? stored === '1'
        : window.matchMedia('(prefers-color-scheme: dark)').matches

  return {
    user,
    guestToken,
    pageKey: search.get('pageKey') || '',
    site: search.get('site') || '',
    view: search.get('view') || '',
    darkMode,
  }
}

export const bootParams: BootParams = readBootParams()

/** True when the console is opened inside the host page's sidebar iframe. */
export function isOpenFromSidebar() {
  return !!bootParams.user?.email
}

/** True when embedded in another document (the host reserves its own chrome). */
export const isEmbedded = window.self !== window.top
