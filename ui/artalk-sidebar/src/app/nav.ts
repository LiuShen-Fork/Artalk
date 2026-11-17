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

export interface NavItem {
  id: PageId
  path: `/${string}`
  titleKey: MessageKey
  descriptionKey?: MessageKey
  icon: LucideIcon
  adminOnly?: boolean
  /** Shown in the primary bar instead of the "more" sheet. */
  primary?: boolean
}

export const NAV_ITEMS: NavItem[] = [
  {
    id: 'dashboard',
    path: '/dashboard',
    titleKey: 'dashboard',
    icon: LayoutDashboard,
    adminOnly: true,
    primary: true,
  },
  { id: 'comments', path: '/comments', titleKey: 'comments', icon: MessagesSquare, primary: true },
  {
    id: 'moderation',
    path: '/moderation',
    titleKey: 'moderation',
    icon: ShieldCheck,
    adminOnly: true,
    primary: true,
  },
  { id: 'users', path: '/users', titleKey: 'user', icon: Users, adminOnly: true, primary: true },
  { id: 'pages', path: '/pages', titleKey: 'page', icon: FileText, adminOnly: true },
  { id: 'sites', path: '/sites', titleKey: 'site', icon: Globe2, adminOnly: true },
  { id: 'security', path: '/security', titleKey: 'security', icon: ShieldAlert, adminOnly: true },
  {
    id: 'settings',
    path: '/settings',
    titleKey: 'settings',
    icon: SlidersHorizontal,
    adminOnly: true,
  },
  { id: 'transfer', path: '/transfer', titleKey: 'transfer', icon: ArrowDownUp, adminOnly: true },
  { id: 'account', path: '/account', titleKey: 'account', icon: CircleUserRound },
]

export function navItemsFor(isAdmin: boolean): NavItem[] {
  return NAV_ITEMS.filter((item) => isAdmin || !item.adminOnly)
}

export function primaryItemsFor(isAdmin: boolean): NavItem[] {
  if (!isAdmin) {
    // A signed-in visitor only has two destinations, so both deserve a tab.
    return navItemsFor(false)
  }
  return navItemsFor(true).filter((item) => item.primary)
}

export function secondaryItemsFor(isAdmin: boolean): NavItem[] {
  const primary = new Set(primaryItemsFor(isAdmin).map((item) => item.id))
  return navItemsFor(isAdmin).filter((item) => !primary.has(item.id))
}

export function findNavItem(id: string): NavItem | undefined {
  return NAV_ITEMS.find((item) => item.id === id)
}

export function defaultPathFor(isAdmin: boolean): string {
  return isAdmin ? '/dashboard' : '/comments'
}

export function isPathAllowed(path: string, isAdmin: boolean): boolean {
  if (path === '/login') return true
  const item = NAV_ITEMS.find((entry) => entry.path === path)
  if (!item) return false
  return isAdmin || !item.adminOnly
}
