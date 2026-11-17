import { useSyncExternalStore } from 'react'
import type { ArtalkType, LocalUser } from 'artalk'
import { getArtalkOrNull } from './artalk'
import { bootParams } from './boot'
import type { ViewMode } from '@/app/nav'

const VIEW_STORAGE_KEY = 'ATK_SIDEBAR_VIEW'

function readStoredView(): ViewMode {
  try {
    return localStorage.getItem(VIEW_STORAGE_KEY) === 'user' ? 'user' : 'admin'
  } catch {
    return 'admin'
  }
}

function storeView(view: ViewMode) {
  try {
    localStorage.setItem(VIEW_STORAGE_KEY, view)
  } catch {
    // Private mode: the preference simply does not survive the session.
  }
}

export interface SessionState {
  user: LocalUser
  /** The site filter used by every admin request; empty means all sites. */
  site: string
  sites: ArtalkType.SiteData[]
  sitesLoaded: boolean
  /** Which half of the console the sidebar is showing. Visitors are pinned to `user`. */
  view: ViewMode
}

let state: SessionState = {
  user: { ...bootParams.user },
  site: bootParams.site,
  sites: [],
  sitesLoaded: false,
  view: bootParams.user?.is_admin ? readStoredView() : 'user',
}

const listeners = new Set<() => void>()

function publish(patch: Partial<SessionState>) {
  state = { ...state, ...patch }
  listeners.forEach((listener) => listener())
}

export const session = {
  get: () => state,
  subscribe(listener: () => void) {
    listeners.add(listener)
    return () => listeners.delete(listener)
  },
  setUser(user: Partial<LocalUser>) {
    const next = { ...state.user, ...user }
    // A fresh login (or a boot without a signed-in user) has no view yet, so an
    // admin lands in the console they manage rather than in "my pages".
    const becameAdmin = next.is_admin && !state.user.is_admin
    publish({ user: next, view: next.is_admin ? (becameAdmin ? readStoredView() : state.view) : 'user' })
  },
  setSite(site: string) {
    publish({ site })
  },
  setSites(sites: ArtalkType.SiteData[]) {
    publish({ sites, sitesLoaded: true })
  },
  setView(view: ViewMode) {
    if (!state.user.is_admin && view === 'admin') return
    storeView(view)
    publish({ view })
  },
  async refreshSites() {
    const artalk = getArtalkOrNull()
    if (!artalk || !state.user.is_admin) return
    const res = await artalk.ctx.getApi().sites.getSites()
    publish({ sites: res.data.sites ?? [], sitesLoaded: true })
  },
  logout() {
    getArtalkOrNull()?.ctx.getUser().logout()
    publish({
      user: { name: '', email: '', link: '', token: '', is_admin: false },
      site: '',
      view: 'user',
    })
  },
}

export function useSession(): SessionState {
  return useSyncExternalStore(session.subscribe, session.get, session.get)
}
