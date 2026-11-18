import {
  createHashHistory,
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
  Outlet,
  redirect,
  useSearch,
} from '@tanstack/react-router'
import { defaultPathFor } from '@/app/nav'
import { AppShell } from '@/components/layout/AppShell'
import { RoutePending } from '@/components/ui/RoutePending'
import { session } from '@/lib/session'
import type { ModerationTab } from '@/pages/moderation'

// Every page is its own chunk: the console opens inside a 430px iframe, so the
// first paint should not carry the settings form and the transfer uploader.
const LoginPage = lazyRouteComponent(() => import('@/pages/login'), 'LoginPage')
const CommentsPage = lazyRouteComponent(() => import('@/pages/comments'), 'CommentsPage')
const DashboardPage = lazyRouteComponent(() => import('@/pages/dashboard'), 'DashboardPage')
const ModerationPage = lazyRouteComponent(() => import('@/pages/moderation'), 'ModerationPage')
const PagesPage = lazyRouteComponent(() => import('@/pages/pages'), 'PagesPage')
const UsersPage = lazyRouteComponent(() => import('@/pages/users'), 'UsersPage')
const SitesPage = lazyRouteComponent(() => import('@/pages/sites'), 'SitesPage')
const TransferPage = lazyRouteComponent(() => import('@/pages/transfer'), 'TransferPage')
const SettingsPage = lazyRouteComponent(() => import('@/pages/settings'), 'SettingsPage')
const SecurityPage = lazyRouteComponent(() => import('@/pages/security'), 'SecurityPage')
const ConsolePage = lazyRouteComponent(() => import('@/pages/console'), 'ConsolePage')
const ActivityPage = lazyRouteComponent(() => import('@/pages/activity'), 'ActivityPage')
const AccountPage = lazyRouteComponent(() => import('@/pages/account'), 'AccountPage')

const MODERATION_TABS: ModerationTab[] = ['all', 'block', 'replace', 'error']

function requireLogin() {
  const { user } = session.get()
  if (!user?.email) throw redirect({ to: '/login' })
}

function requireAdmin() {
  requireLogin()
  const { user, view } = session.get()
  if (!user.is_admin) throw redirect({ to: '/comments' })
  // An admin browsing the "user" half of the console should see exactly what a
  // visitor sees, so admin-only pages bounce back to the visitor home.
  if (view === 'user') throw redirect({ to: '/activity' })
}

const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: () => {
    throw redirect({
      to: defaultPathFor(!!session.get().user.is_admin, session.get().view),
      replace: true,
    })
  },
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
  beforeLoad: () => {
    if (session.get().user?.email) {
      throw redirect({
        to: defaultPathFor(!!session.get().user.is_admin, session.get().view),
        replace: true,
      })
    }
  },
})

/** Everything below this layout gets the console chrome and requires a login. */
const shellRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'shell',
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
  beforeLoad: requireLogin,
})

const moderationRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/moderation',
  beforeLoad: requireAdmin,
  validateSearch: (search: Record<string, unknown>): { status?: ModerationTab } => {
    const status = search.status
    return MODERATION_TABS.includes(status as ModerationTab)
      ? { status: status as ModerationTab }
      : {}
  },
  component: function ModerationRoute() {
    const { status } = useSearch({ from: '/shell/moderation' })
    return <ModerationPage initialStatus={status} />
  },
})

const routeTree = rootRoute.addChildren([
  loginRoute,
  shellRoute.addChildren([
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/',
      beforeLoad: () => {
        throw redirect({
          to: defaultPathFor(!!session.get().user.is_admin, session.get().view),
          replace: true,
        })
      },
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/dashboard',
      beforeLoad: requireAdmin,
      component: DashboardPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/comments',
      component: CommentsPage,
    }),
    moderationRoute,
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/pages',
      beforeLoad: requireAdmin,
      component: PagesPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/users',
      beforeLoad: requireAdmin,
      component: UsersPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/sites',
      beforeLoad: requireAdmin,
      component: SitesPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/transfer',
      beforeLoad: requireAdmin,
      component: TransferPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/settings',
      beforeLoad: requireAdmin,
      component: SettingsPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/security',
      beforeLoad: requireAdmin,
      component: SecurityPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/console',
      beforeLoad: requireAdmin,
      component: ConsolePage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/activity',
      component: ActivityPage,
    }),
    createRoute({
      getParentRoute: () => shellRoute,
      path: '/account',
      component: AccountPage,
    }),
  ]),
])

export const router = createRouter({
  routeTree,
  history: createHashHistory(),
  defaultPreload: false,
  scrollRestoration: true,
  defaultPendingComponent: RoutePending,
  defaultPendingMs: 120,
  defaultPendingMinMs: 200,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export { requireAdmin }
