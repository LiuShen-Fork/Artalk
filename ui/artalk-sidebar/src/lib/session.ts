import { useSyncExternalStore } from 'react'
import type { ArtalkType, LocalUser } from 'artalk'
import { bootParams } from './boot'
import { getArtalkOrNull } from './artalk'

export interface SessionState {
  user: LocalUser
  /** The site filter used by every admin request; empty means all sites. */
  site: string
  sites: ArtalkType.SiteData[]
  sitesLoaded: boolean
}

let state: SessionState = {
  user: { ...bootParams.user },
  site: bootParams.site,
  sites: [],
  sitesLoaded: false,
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
    publish({ user: { ...state.user, ...user } })
  },
  setSite(site: string) {
    publish({ site })
  },
  setSites(sites: ArtalkType.SiteData[]) {
    publish({ sites, sitesLoaded: true })
  },
  async refreshSites() {
    const artalk = getArtalkOrNull()
    if (!artalk || !state.user.is_admin) return
    const res = await artalk.ctx.getApi().sites.getSites()
    publish({ sites: res.data.sites ?? [], sitesLoaded: true })
  },
  logout() {
    getArtalkOrNull()?.ctx.getUser().logout()
    publish({ user: { name: '', email: '', link: '', token: '', is_admin: false }, site: '' })
  },
}

export function useSession(): SessionState {
  return useSyncExternalStore(session.subscribe, session.get, session.get)
}
