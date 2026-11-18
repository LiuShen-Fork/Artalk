import { getArtalk } from './artalk'
import { session } from './session'

export interface AccountProfile {
  id: number
  name: string
  email: string
  link: string
  badge_name: string
  badge_color: string
  is_admin: boolean
  is_ai_assistant?: boolean
  has_password: boolean
  receive_email: boolean
}

export interface AccountProfileResponse {
  user: AccountProfile | null
  is_login: boolean
}

export interface AccountIdentity {
  id: number
  provider: string
  created_at: string
  confirmed_at: string
}

export interface IdentityListResponse {
  identities: AccountIdentity[]
}

export interface AuthProviderInfo {
  name: string
  label: string
  path?: string
  icon: string
}

export interface AuthProviderListResponse {
  providers: AuthProviderInfo[]
  callback_origin?: string
  anonymous: boolean
}

export interface PasswordUpdateResponse {
  token: string
  user: AccountProfile
}

export interface ProfileUpdateInput {
  name: string
  email: string
  link: string
  code?: string
}

export interface PasswordUpdateInput {
  current_password?: string
  new_password: string
  code?: string
}

function serverRoot() {
  const server = (getArtalk().ctx.conf.server || '.').replace(/\/+$/, '')
  return server || '.'
}

function apiURL(path: string) {
  return new URL(
    `${serverRoot()}/api/v2/${path.replace(/^\/+/, '')}`,
    window.location.href,
  ).toString()
}

/**
 * Resolve a path returned by the API (for example `/auth/github?state=...`)
 * against the Artalk server that this sidebar is connected to.
 */
export function resolveServerPath(path: string) {
  return new URL(
    `${serverRoot()}/api/v2${path.startsWith('/') ? '' : '/'}${path}`,
    window.location.href,
  ).toString()
}

/** An API failure that keeps the HTTP status for callers that branch on it. */
export class HttpError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'HttpError'
    this.status = status
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = session.get().user.token
  const headers = new Headers(init.headers)
  if (init.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const resp = await fetch(apiURL(path), {
    ...init,
    headers,
    credentials: 'same-origin',
    referrerPolicy: 'no-referrer',
  })
  const json: any = await resp.json().catch(() => ({}))

  if (!resp.ok) {
    throw new HttpError(
      resp.status,
      json.msg || json.message || resp.statusText || 'Request failed',
    )
  }
  return json as T
}

export const accountApi = {
  getProfile() {
    return request<AccountProfileResponse>('user')
  },

  updateProfile(data: ProfileUpdateInput) {
    return request<{ user: AccountProfile }>('user', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  updatePassword(data: PasswordUpdateInput) {
    return request<PasswordUpdateResponse>('user/password', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  getIdentities() {
    return request<IdentityListResponse>('user/identities')
  },

  startIdentityBind(provider: string) {
    return request<{ path: string }>(`user/identities/${encodeURIComponent(provider)}/bind`, {
      method: 'POST',
    })
  },

  removeIdentity(id: number) {
    return request<{ msg: string }>(`user/identities/${id}`, { method: 'DELETE' })
  },

  /**
   * The OAuth callback page only hands the token back to an origin the server
   * trusts, so the console asks first instead of opening a popup that is bound
   * to fail. A console served from a domain that is not in the site URLs has to
   * add it there before social login can work.
   */
  async isOriginTrusted() {
    try {
      const resp = await fetch(
        apiURL(`conf/domain?url=${encodeURIComponent(window.location.origin)}`),
        { credentials: 'same-origin' },
      )
      if (!resp.ok) return false
      const json = (await resp.json()) as { is_trusted?: boolean }
      return json.is_trusted === true
    } catch {
      return false
    }
  },

  getProviders() {
    // The endpoint is only registered when social login is enabled, so a 404
    // means "no providers" rather than a failure worth surfacing.
    return request<AuthProviderListResponse>('conf/auth/providers').catch((error: unknown) => {
      if (error instanceof HttpError && error.status === 404) return { providers: [] }
      throw error
    })
  },
}

export function errorMessage(error: unknown) {
  if (error instanceof Error && error.message) return error.message
  return String(error || '')
}
