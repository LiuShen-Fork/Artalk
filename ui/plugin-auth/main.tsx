/* @refresh reload */
import './style.scss'
import type { ArtalkPlugin } from 'artalk'
import { DialogMain } from './DialogMain'
import { createLayer } from './lib/layer'
import { RenderEditorUser } from './EditorUser'
import type { AuthContext } from './types'

export const ArtalkAuthPlugin: ArtalkPlugin = (ctx) => {
  const editor = ctx.inject('editor')
  const api = ctx.inject('api')
  const apiHandlers = ctx.inject('apiHandlers')
  const user = ctx.inject('user')
  const events = ctx.inject('events')
  const config = ctx.inject('config')
  const layers = ctx.inject('layers')

  const authCtx: AuthContext = {
    getEditor: () => editor,
    getApi: () => api,
    getApiHandlers: () => apiHandlers,
    getUser: () => user,
    getEvents: () => events,
    getConf: () => config,
    getLayers: () => layers,
    $t: (key, args) => ctx.$t(key, args),
  }

  apiHandlers.add('need_auth_login', (data) => {
    // Keep the server's explanation. It knows whether the email already
    // belongs to an account, and "Login required" alone leaves the person
    // guessing why their comment was rejected.
    const message = (data as { msg?: string } | null)?.msg
    openAuthDialog(message)
    throw new Error(message || ctx.$t('signIn'))
  })

  let anonymous = false
  const refreshBtn = () => {
    editor.getUI().$submitBtn.innerText =
      user.getData().token || anonymous
        ? authCtx.getConf().get().sendBtn || authCtx.$t('send')
        : authCtx.$t('signIn')
  }

  authCtx.getConf().watchConf(['locale', 'sendBtn'], () => refreshBtn())
  authCtx.getEvents().on('user-changed', () => refreshBtn())

  authCtx.getEvents().on('mounted', () => {
    editor.getUI().$header.style.display = 'none'

    RenderEditorUser(authCtx)
  })

  const onSkip = () => {
    editor.getUI().$header.style.display = ''
    editor.getUI().$name.focus()
    ctx.updateConf({
      beforeSubmit: undefined,
    })

    anonymous = true
    refreshBtn()
  }

  const openAuthDialog = (notice?: string) => {
    createLayer(authCtx).show((layer) => (
      <DialogMain ctx={authCtx} notice={notice} onClose={() => layer.destroy()} onSkip={onSkip} />
    ))
  }

  ctx.updateConf({
    beforeSubmit: (editor, next) => {
      if (!user.getData().token) {
        openAuthDialog()
      } else {
        next()
      }
    },
  })
}
