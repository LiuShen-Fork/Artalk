import type { ArtalkType } from 'artalk'
import { getArtalk } from './artalk'
import { session } from './session'

export interface ApiOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  query?: Record<string, unknown>
  body?: unknown
  signal?: AbortSignal
}

export interface ApiResponse<T> {
  data: T
  msg?: string
}

/**
 * The widget's request client defaults to `text/plain`, which Fiber's body
 * parser rejects with 422. Every write from the console is JSON.
 */
const JSON_CONTENT_TYPE = 'application/json' as never

function bodyParams(body: unknown) {
  return body === undefined ? {} : { body, type: JSON_CONTENT_TYPE }
}

function compact(query?: Record<string, unknown>) {
  if (!query) return undefined
  const entries = Object.entries(query).filter(
    ([, value]) => value !== undefined && value !== null && value !== '',
  )
  return entries.length ? Object.fromEntries(entries) : undefined
}

/**
 * Call `/api/v2/*` through the Artalk client so the console inherits its base
 * URL, Authorization header, guest credential and error handling.
 */
export async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const res = await getArtalk()
    .ctx.getApi()
    .request<T>({
      path,
      method: options.method ?? 'GET',
      query: compact(options.query),
      ...bodyParams(options.body),
      secure: true,
      format: 'json',
      signal: options.signal,
    })
  return res.data
}

/** Same as {@link api}, but keeps the server's `msg` for confirmation copy. */
export async function apiWithMessage<T>(path: string, options: ApiOptions = {}) {
  const res = await getArtalk()
    .ctx.getApi()
    .request<T & { msg?: string }>({
      path,
      method: options.method ?? 'GET',
      query: compact(options.query),
      ...bodyParams(options.body),
      secure: true,
      format: 'json',
      signal: options.signal,
    })
  return res.data
}

/** The signed-in user's site filter, spread into admin query strings. */
export function siteQuery(): { site_name?: string } {
  const site = session.get().site
  return site ? { site_name: site } : {}
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  if (typeof error === 'string') return error
  const data = (error as any)?.data
  if (data && typeof data === 'object' && typeof data.msg === 'string') return data.msg
  return ''
}

/**
 * The sidebar's admin endpoints return a few counters the widget's public types
 * do not declare, so they are widened here once instead of at every call site.
 */
export type PageData = ArtalkType.PageData & { comment_count?: number; pv?: number }
export type SiteData = ArtalkType.SiteData & { comment_count?: number }
