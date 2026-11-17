const LOGIN_TIMEOUT = 5 * 60 * 1000
const LOGIN_POLL_INTERVAL = 700

/**
 * Open a provider's OAuth page in a popup and wait for the server callback to
 * hand back a token.
 *
 * The callback page only answers the window that first sent `ATK_LOGIN`, so we
 * keep announcing ourselves until it posts `ATK_AUTH_CALLBACK`. The server
 * still validates the opener origin against the trusted domain list, which is
 * why the console checks that list before offering these buttons.
 */
export function startOAuthLogin(url: string): Promise<string> {
  const target = new URL(url, window.location.href)
  const callbackOrigin = target.origin

  const width = 560
  const height = 680
  const left = Math.max(0, Math.round((window.screen.width - width) / 2))
  const top = Math.max(0, Math.round((window.screen.height - height) / 2))

  const popup = window.open(
    target.toString(),
    'artalk_social_login',
    `width=${width},height=${height},left=${left},top=${top},scrollbars=yes,resizable=yes,status=no,location=no,toolbar=no,menubar=no`,
  )

  if (!popup) {
    return Promise.reject(new Error('popup blocked'))
  }

  return new Promise<string>((resolve, reject) => {
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

    const finish = (token: string) => {
      if (settled) return
      settled = true
      cleanup()
      try {
        popup.close()
      } catch {
        // The browser may already have closed it.
      }
      resolve(token)
    }

    const fail = (message: string) => {
      if (settled) return
      settled = true
      cleanup()
      reject(new Error(message))
    }

    function onMessage(event: MessageEvent) {
      if (event.source !== popup || event.origin !== callbackOrigin) return

      const data = event.data as { type?: string; payload?: unknown }
      if (!data || data.type !== 'ATK_AUTH_CALLBACK') return
      if (typeof data.payload !== 'string' || !data.payload) {
        fail('empty token')
        return
      }
      finish(data.payload)
    }

    window.addEventListener('message', onMessage)

    pollTimer = window.setInterval(() => {
      if (popup.closed) {
        fail('canceled')
        return
      }
      popup.postMessage({ type: 'ATK_LOGIN' }, callbackOrigin)
    }, LOGIN_POLL_INTERVAL)

    timeoutTimer = window.setTimeout(() => fail('timeout'), LOGIN_TIMEOUT)
  })
}
