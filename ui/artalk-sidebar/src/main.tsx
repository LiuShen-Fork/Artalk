import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/index.css'
import './lib/promise-polyfill'
import { App } from './App'
import { bootArtalk } from './lib/artalk'
import { installConsole } from './lib/console'
import { setLocale } from './lib/i18n'
import { session } from './lib/session'
import { theme } from './lib/theme'

async function bootstrap() {
  const artalk = await bootArtalk((locale) => {
    if (locale) void setLocale(locale)
  })
  // Make sure the very first paint already uses the server locale.
  await setLocale(String(artalk.ctx.getConf().locale || 'en'))

  const { syncArtalkUser } = await import('./lib/artalk')
  await syncArtalkUser(() => session.logout())
  session.setUser(artalk.ctx.getUser().getData())
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
