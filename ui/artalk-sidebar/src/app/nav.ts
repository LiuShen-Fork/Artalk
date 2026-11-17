import type { LucideIcon } from 'lucide-react'
import {
  ArrowDownUp,
  CircleUserRound,
  FileText,
  Globe2,
  LayoutDashboard,
  MessagesSquare,
  ShieldAlert,
  ShieldCheck,
  SlidersHorizontal,
  Users,
} from 'lucide-react'
import type { MessageKey } from '@/lib/i18n'

export type PageId =
  | 'dashboard'
  | 'comments'
  | 'moderation'
  | 'pages'
  | 'users'
  | 'sites'
  | 'transfer'
  | 'settings'
  | 'security'
  | 'account'

/**
 * The console has two faces. Admins get the management pages; everybody,
 * admins included, can drop into "my pages" to read their own comments and
 * edit their own account. A plain visitor only ever sees the second one.
 */
export type ViewMode = 'admin' | 'user'

export type NavGroupId = 'overview' | 'content' | 'community' | 'system' | 'mine'

export interface NavItem {
  id: PageId
  path: `/${string}`
  titleKey: MessageKey
  /** Label used in "my pages" mode, where the same route means something else. */
  userTitleKey?: MessageKey
  icon: LucideIcon
  adminOnly?: boolean
  group: NavGroupId
  /** Sits in the mobile tab bar instead of the "more" sheet. */
  primary?: boolean
}

export const NAV_GROUPS: { id: NavGroupId; titleKey: MessageKey }[] = [
  { id: 'overview', titleKey: 'groupOverview' },
  { id: 'content', titleKey: 'groupContent' },
  { id: 'community', titleKey: 'groupCommunity' },
  { id: 'system', titleKey: 'groupSystem' },
  { id: 'mine', titleKey: 'groupMine' },
]

export const NAV_ITEMS: NavItem[] = [
  {
    id: 'dashboard',
    path: '/dashboard',
    titleKey: 'dashboard',
    icon: LayoutDashboard,
    adminOnly: true,
    group: 'overview',
    primary: true,
  },
  {
    id: 'comments',
    path: '/comments',
    titleKey: 'comments',
    userTitleKey: 'myComments',
    icon: MessagesSquare,
    group: 'content',
    primary: true,
  },
  {
    id: 'moderation',
    path: '/moderation',
    titleKey: 'moderation',
    icon: ShieldCheck,
    adminOnly: true,
    group: 'content',
    primary: true,
  },
  {
    id: 'pages',
    path: '/pages',
    titleKey: 'page',
    icon: FileText,
    adminOnly: true,
    group: 'content',
  },
  {
    id: 'users',
    path: '/users',
    titleKey: 'user',
    icon: Users,
    adminOnly: true,
    group: 'community',
    primary: true,
  },
  {
    id: 'sites',
    path: '/sites',
    titleKey: 'site',
    icon: Globe2,
    adminOnly: true,
    group: 'community',
  },
  {
    id: 'security',
    path: '/security',
    titleKey: 'security',
    icon: ShieldAlert,
    adminOnly: true,
    group: 'system',
  },
  {
    id: 'settings',
    path: '/settings',
    titleKey: 'settings',
    icon: SlidersHorizontal,
    adminOnly: true,
    group: 'system',
  },
  {
    id: 'transfer',
    path: '/transfer',
    titleKey: 'transfer',
    icon: ArrowDownUp,
    adminOnly: true,
    group: 'system',
  },
  {
    id: 'account',
    path: '/account',
    titleKey: 'account',
    icon: CircleUserRound,
    group: 'mine',
  },
]

/** The pages a visitor owns: their comments and their account. */
const VISITOR_PAGES: PageId[] = ['comments', 'account']

export function isAdminOnly(id: PageId): boolean {
  return !!NAV_ITEMS.find((item) => item.id === id)?.adminOnly
}

export function navItemsFor(isAdmin: boolean, view: ViewMode = isAdmin ? 'admin' : 'user') {
  if (!isAdmin || view === 'user') {
    return NAV_ITEMS.filter((item) => !item.adminOnly && VISITOR_PAGES.includes(item.id))
  }
  return NAV_ITEMS
}

export function navGroupsFor(isAdmin: boolean, view: ViewMode = isAdmin ? 'admin' : 'user') {
  const items = navItemsFor(isAdmin, view)
  return NAV_GROUPS.map((group) => ({
    ...group,
    items: items.filter((item) => item.group === group.id),
  })).filter((group) => group.items.length > 0)
}

export function labelKeyFor(item: NavItem, view: ViewMode): MessageKey {
  return view === 'user' && item.userTitleKey ? item.userTitleKey : item.titleKey
}

export function primaryItemsFor(isAdmin: boolean, view: ViewMode = isAdmin ? 'admin' : 'user') {
  const items = navItemsFor(isAdmin, view)
  if (!isAdmin || view === 'user') return items
  return items.filter((item) => item.primary)
}

export function secondaryItemsFor(isAdmin: boolean, view: ViewMode = isAdmin ? 'admin' : 'user') {
  const primary = new Set(primaryItemsFor(isAdmin, view).map((item) => item.id))
  return navItemsFor(isAdmin, view).filter((item) => !primary.has(item.id))
}

export function findNavItem(id: string): NavItem | undefined {
  return NAV_ITEMS.find((item) => item.id === id)
}

export function defaultPathFor(isAdmin: boolean, view: ViewMode = isAdmin ? 'admin' : 'user'): string {
  if (!isAdmin || view === 'user') return '/comments'
  return '/dashboard'
}

export function isPathAllowed(path: string, isAdmin: boolean): boolean {
  if (path === '/login') return true
  const item = NAV_ITEMS.find((entry) => entry.path === path)
  if (!item) return false
  return isAdmin || !item.adminOnly
}
