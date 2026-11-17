import { api } from './api'
import { t } from './i18n'

export interface QuotaConf {
  minute: number
  hour: number
  day: number
}

export interface SecurityConfig {
  enabled: boolean
  action: 'captcha' | 'reject' | 'ban'
  rate_limit: { ip: QuotaConf; global: QuotaConf }
  auto_ban: { enabled: boolean; threshold: number; window: number; duration: number }
  blacklist: string[]
  whitelist: string[]
  event_retention_days: number
  exclude_paths: string[]
}

export interface CaptchaConfig {
  enabled: boolean
  always: boolean
  captcha_type: string
  action_reset: number
  action_limit: number
  geetest: { captcha_id: string; captcha_key: string }
}

export interface SecurityRuntime {
  captcha_ready: boolean
  tracked_ips: number
  active_bans: number
  tracked_violators: number
  global: QuotaConf
}

export interface SecurityConfigPayload {
  security: SecurityConfig
  captcha: CaptchaConfig
  runtime: SecurityRuntime
}

export interface IPBan {
  id: number
  ip: string
  ip_region: string
  is_cidr: boolean
  reason: string
  note: string
  source: string
  active: boolean
  expire_at: string
  hit_count: number
  last_hit_at: string
  created_at: string
}

export interface SecurityEvent {
  id: number
  ip: string
  ip_region: string
  kind: string
  scope: string
  window: string
  action: string
  path: string
  method: string
  site_name: string
  user_name: string
  ua: string
  detail: string
  blocked: boolean
  date: string
}

export interface SecurityReport {
  range_days: number
  total: number
  blocked: number
  by_kind: { kind: string; count: number }[]
  by_action: { action: string; count: number }[]
  by_day: { date: string; count: number }[]
  top_ips: { ip: string; ip_region: string; count: number }[]
  top_paths: { path: string; count: number }[]
}

export const QUOTA_WINDOWS = ['minute', 'hour', 'day'] as const
export const CAPTCHA_TYPES = ['image', 'turnstile', 'recaptcha', 'hcaptcha', 'geetest', 'capjs']
export const SECURITY_ACTIONS = ['captcha', 'reject', 'ban'] as const
export const EVENT_KINDS = ['rate_limit', 'captcha_failed', 'banned', 'auto_ban'] as const

/** Event kinds map to dedicated labels; unknown kinds fall back to the raw value. */
export function kindLabel(kind: string): string {
  switch (kind) {
    case 'rate_limit':
      return t('securityKindRateLimit')
    case 'captcha_failed':
      return t('securityKindCaptchaFailed')
    case 'banned':
      return t('securityKindBanned')
    case 'auto_ban':
      return t('securityKindAutoBan')
    default:
      return kind
  }
}

export const securityApi = {
  getConfig: () => api<SecurityConfigPayload>('/security/config'),
  saveConfig: (body: Record<string, unknown>) =>
    api<SecurityConfigPayload>('/security/config', { method: 'PUT', body }),
  getBans: () => api<{ count: number; bans: IPBan[] }>('/security/bans', { query: { limit: 200 } }),
  addBan: (body: { ip: string; reason: string; note: string; duration: number }) =>
    api<unknown>('/security/bans', { method: 'POST', body }),
  removeBan: (id: number) => api<unknown>(`/security/bans/${id}`, { method: 'DELETE' }),
  getReport: (days: number) => api<SecurityReport>('/security/report', { query: { days } }),
  getEvents: (query: { kind?: string; ip?: string; days: number; limit: number; offset: number }) =>
    api<{ count: number; events: SecurityEvent[] }>('/security/events', { query }),
  deleteEvent: (id: number) => api<unknown>(`/security/events/${id}`, { method: 'DELETE' }),
  clearEvents: (query: { kind?: string; ip?: string; days: number }) =>
    api<unknown>('/security/events', { method: 'DELETE', query }),
}
