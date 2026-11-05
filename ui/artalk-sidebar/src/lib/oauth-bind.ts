export interface OAuthBindResult {
  ok: boolean
  provider: string
  message: string
}

const BIND_TIMEOUT = 5 * 60 * 1000
const BIND_POLL_INTERVAL = 700

/**
 * Open the provider's OAuth page in a popup and wait for the server's bind
 * callback page to post the result back.
 *
 * The callback page only trusts the origin that sent the first `ATK_BIND`
 * message, so we keep posting it until the popup reports a result.
 */
export function startOAuthBind(url: string): Promise<OAuthBindResult> {
  const target = new URL(url, window.location.href)
  const callbackOrigin = target.origin

  const width = 560
  const height = 680
  const left = Math.max(0, Math.round((window.screen.width - width) / 2))
  const top = Math.max(0, Math.round((window.screen.height - height) / 2))

  const popup = window.open(
    target.toString(),
    'artalk_identity_bind',
    `width=${width},height=${height},left=${left},top=${top},scrollbars=yes,resizable=yes,status=no,location=no,toolbar=no,menubar=no`,
  )

  if (!popup) {
    return Promise.reject(new Error('popup blocked'))
  }

  return new Promise<OAuthBindResult>((resolve, reject) => {
    let settled = false
    let pollTimer: number | undefined
    let timeoutTimer: number | undefined

    const cleanup = () => {
      if (pollTimer !== undefined) window.clearInterval(pollTimer)
      if (timeoutTimer !== undefined) window.clearTimeout(timeoutTimer)
      window.removeEventListener('message', onMessage)
      pollTimer = undefined
      timeoutTimer = undefined
    }

    const finish = (result: OAuthBindResult) => {
      if (settled) return
      settled = true
      cleanup()
      try {
        popup.close()
      } catch {
        // The browser may already have closed it.
      }
      resolve(result)
    }

    const fail = (message: string) => {
      if (settled) return
      settled = true
      cleanup()
      reject(new Error(message))
    }

    function onMessage(event: MessageEvent) {
      if (event.source !== popup || event.origin !== callbackOrigin) return

      const data = event.data as any
      if (!data || data.type !== 'ATK_BIND_CALLBACK') return

      const payload = data.payload || {}
      finish({
        ok: payload.ok === true,
        provider: typeof payload.provider === 'string' ? payload.provider : '',
        message: typeof payload.message === 'string' ? payload.message : '',
      })
    }

    window.addEventListener('message', onMessage)

    pollTimer = window.setInterval(() => {
      if (popup.closed) {
        fail('canceled')
        return
      }
      popup.postMessage({ type: 'ATK_BIND' }, callbackOrigin)
    }, BIND_POLL_INTERVAL)

    timeoutTimer = window.setTimeout(() => fail('timeout'), BIND_TIMEOUT)
  })
}
