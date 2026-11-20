import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/index.css'
import './lib/promise-polyfill'
import { App } from './App'
import { bootArtalk, applyWidgetLocale } from './lib/artalk'
import { installConsole } from './lib/console'
import { browserLocale, hasLocaleChoice, initialLocale, setLocale } from './lib/i18n'
import { session } from './lib/session'
import { theme } from './lib/theme'

async function bootstrap() {
  const artalk = await bootArtalk((locale) => {
    // A language picked in the console outranks the site config from now on.
    if (!locale || hasLocaleChoice()) return
    void setLocale(locale === 'auto' ? browserLocale() : locale)
  })
  // Make sure the very first paint already uses the right language: the
  // visitor's own pick, else the site locale (`auto` = the browser), else the
  // browser.
  const locale = initialLocale(String(artalk.ctx.getConf().locale || ''))
  await setLocale(locale)
  // The embedded comment list speaks the same language as the console.
  await applyWidgetLocale(locale)

  const { syncArtalkUser } = await import('./lib/artalk')
  const verifiedAdmin = await syncArtalkUser(() => session.logout())
  // The URL is written by the host page: only keep its admin claim when the
  // server agrees.
  const data = artalk.ctx.getUser().getData()
  session.setUser(verifiedAdmin === undefined ? data : { ...data, is_admin: verifiedAdmin })
  theme.syncArtalk()

  // Name, icon, styles and scripts the owner saved from the console settings
  // page. Public, tiny, and not worth blocking the first paint on.
  void installConsole()

  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

void bootstrap()
