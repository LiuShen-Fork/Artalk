import Artalk from 'artalk'
import md5 from 'crypto-js/md5'
import sha256 from 'crypto-js/sha256'
import { bootParams } from './boot'
import { restoreGuestCredential } from './guest-handoff'
import { session } from './session'

let instance: Artalk | null = null

/**
 * Create the headless Artalk instance that powers every API call in the
 * console. It is attached to a hidden element: the public widget is only
 * rendered explicitly by the comments page.
 */
export function bootArtalk(onLocale: (locale?: string) => void): Promise<Artalk> {
  const el = document.createElement('div')
  el.style.display = 'none'
  document.body.append(el)

  let ready = false
  let resolveReady: (value: Artalk) => void = () => undefined
  const whenReady = new Promise<Artalk>((resolve) => {
    resolveReady = resolve
  })

  // `Artalk.use` is the widget's plugin registrar, not a React hook.
  // eslint-disable-next-line react-hooks/rules-of-hooks
  Artalk.use((ctx) => {
    ctx.watchConf(['locale'], (conf) => {
      onLocale(typeof conf.locale === 'string' ? conf.locale : undefined)
      if (!ready) {
        ready = true
        resolveReady(instance as Artalk)
      }
    })
  })

  instance = Artalk.init({
    el,
    server: '../',
    pageKey: bootParams.pageKey,
    site: bootParams.site,
    darkMode: bootParams.darkMode,
    pvAdd: false,
    noComment: '<div></div>',
    flatMode: true,
    pagination: { pageSize: 20, readMore: false, autoLoad: false },
    listUnreadHighlight: true,
    fetchCommentsOnInit: false,
  })

  // `watchConf` fires as soon as the config request settles, but a slow or
  // failing backend must not leave the console on a blank screen forever.
  return Promise.race([
    whenReady,
    new Promise<Artalk>((resolve) => setTimeout(() => resolve(instance as Artalk), 4000)),
  ])
}

export function getArtalk(): Artalk {
  if (!instance) throw new Error('Artalk is not initialized')
  return instance
}

export function getArtalkOrNull(): Artalk | null {
  return instance
}

/**
 * Reconcile the credential handed over by the host page with the one stored in
 * this browser, then verify it against the server.
 */
export async function syncArtalkUser(onLogout: () => void) {
  const ctx = getArtalk().ctx
  await restoreGuestCredential(ctx, bootParams.guestToken)

  if (bootParams.user?.email) {
    ctx.getUser().update(bootParams.user)
  } else if (ctx.getUser().checkHasBasicUserInfo()) {
    session.setUser(ctx.getUser().getData())
  } else {
    session.logout()
    return
  }

  ctx.getApiHandlers().remove('need_login')
  ctx.getApiHandlers().add('need_login', async () => {
    onLogout()
    throw new Error('Need login')
  })

  const { name, email } = ctx.getUser().getData()
  try {
    const res = await ctx.getApi().user.getUserStatus({ email, name })
    if (res.data.is_admin && !res.data.is_login) {
      onLogout()
      return
    }
    if (ctx.getUser().getData().token || ctx.getUser().getGuestToken?.(ctx.getConf().server)) {
      ctx
        .getApi()
        .notifies.markAllNotifyRead({ email, name })
        .catch(() => undefined)
    }
  } catch {
    // The session stays usable offline; individual requests surface errors.
  }
}

/** Display avatar for the signed-in user, following the server's Gravatar config. */
export function avatarUrl(email: string): string {
  if (!email) return ''
  const conf = getArtalkOrNull()?.ctx.conf?.gravatar
  if (!conf?.mirror) return ''

  const params = typeof conf.params === 'string' ? conf.params : ''
  const normalized = email.trim().toLowerCase()
  const hash = params.includes('sha256=1')
    ? sha256(normalized).toString()
    : md5(normalized).toString()

  return `${conf.mirror.replace(/\/$/, '')}/${hash}?${params.replace(/^\?/, '')}`
}
