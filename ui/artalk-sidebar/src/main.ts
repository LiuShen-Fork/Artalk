import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Artalk from 'artalk'
import { createRouter, createWebHashHistory } from 'vue-router'
import { routes } from 'vue-router/auto-routes'
import { setupI18n } from './i18n'
import 'artalk/Artalk.css'
import './style.scss'
import App from './App.vue'
import { setArtalk } from './global'
import { setupArtalk, syncArtalkUser } from './artalk'
import { AdminPages, UserPages } from './components/AppNavigationMenu'
import { useUserStore } from './stores/user'
import './lib/promise-polyfill'

// I18n
// @see https://vue-i18n.intlify.dev
const { i18n, setLocale } = setupI18n()

// Router
// @see https://router.vuejs.org/
const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// Pinia
// @see https://pinia.vuejs.org
const pinia = createPinia()

// Artalk
// @see https://artalk.js.org
const artalkLoader = () =>
  new Promise<Artalk>((notifyArtalkLoaded) => {
    let artalkLoaded = false
    let artalk: Artalk | null = null

    Artalk.use((ctx) => {
      // When artalk is ready, notify the loader and load the locale
      ctx.watchConf(['locale'], async (conf) => {
        if (typeof conf.locale === 'string' && conf.locale !== 'auto') await setLocale(conf.locale) // update i18n locale

        if (!artalkLoaded) {
          artalkLoaded = true
          notifyArtalkLoaded(artalk!)
        }
      })
    })

    artalk = setupArtalk()
  })

// Mount Vue app
;(async () => {
  const artalk = await artalkLoader()
  setArtalk(artalk)

  const app = createApp(App)
  app.use(i18n)
  app.use(pinia)

  // Keep regular users inside their own console. Admin APIs are guarded on
  // the server too; this only prevents pointless page loads and confusing
  // error states from direct hash navigation.
  router.beforeEach((to) => {
    const name = String(to.name || '').replace(/^\//, '')
    if (!name || name === 'login') return true

    const user = useUserStore()
    const allowed = user.is_admin ? AdminPages : UserPages
    if (allowed[name]) return true
    return user.is_admin ? '/dashboard' : '/comments'
  })

  app.use(router)

  // user sync from artalk to sidebar
  await syncArtalkUser(artalk.ctx, router)

  app.mount('#app')
})()
