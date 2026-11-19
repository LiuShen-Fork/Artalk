import { Button, Drawer } from '@heroui/react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { ChevronDown, MoreHorizontal } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import {
  NAV_ITEMS,
  findNavItem,
  labelKeyFor,
  primaryItemsFor,
  secondaryItemsFor,
  type NavItem,
  type ViewMode,
} from '@/app/nav'
import { AccountMenu } from '@/components/layout/AccountMenu'
import { BrandMark } from '@/components/ui/BrandMark'
import { ModeSwitch, Sidebar } from '@/components/layout/Sidebar'
import { SiteSwitcher } from '@/components/layout/SiteSwitcher'
import { ThemeToggle } from '@/components/layout/ThemeToggle'
import { useConfirm } from '@/components/ui/confirm'
import { isEmbedded } from '@/lib/boot'
import { t } from '@/lib/i18n'
import { session, useSession } from '@/lib/session'
import { cn } from '@/lib/utils'

function TabButton({
  icon: Icon,
  label,
  active,
  onPress,
}: {
  icon: NavItem['icon']
  label: string
  active: boolean
  onPress: () => void
}) {
  return (
    <button
      type="button"
      onClick={onPress}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'flex flex-1 flex-col items-center justify-center gap-1 rounded-control py-1.5 transition-colors',
        active ? 'text-accent' : 'text-muted hover:text-foreground',
      )}
    >
      <Icon className="size-[19px]" aria-hidden />
      <span className="text-[10px] font-medium leading-none">{label}</span>
    </button>
  )
}

function MoreSheet({
  items,
  open,
  onOpenChange,
  current,
  onPick,
  isAdmin,
  view,
  onSwitchView,
}: {
  items: NavItem[]
  open: boolean
  onOpenChange: (open: boolean) => void
  current: string
  onPick: (item: NavItem) => void
  isAdmin: boolean
  view: ViewMode
  onSwitchView: (view: ViewMode) => void
}) {
  return (
    <Drawer isOpen={open} onOpenChange={onOpenChange}>
      <Drawer.Backdrop>
        <Drawer.Content placement="bottom">
          <Drawer.Dialog className="gap-3 pb-3">
            <Drawer.Handle />
            <Drawer.Header className="px-4 pt-1 text-start">
              <Drawer.Heading className="text-base">{t('nav')}</Drawer.Heading>
            </Drawer.Header>
            {isAdmin ? (
              <div className="px-3">
                <ModeSwitch view={view} onChange={onSwitchView} layout="list" />
              </div>
            ) : null}
            <div className="grid grid-cols-3 gap-2 px-3 pb-1">
              {items.map((item) => {
                const Icon = item.icon
                const active = item.path === current
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => onPick(item)}
                    className={cn(
                      'flex flex-col items-center gap-2 rounded-xl border px-2 py-3.5 transition-colors',
                      active
                        ? 'border-accent/40 bg-accent/10 text-accent'
                        : 'border-border bg-surface hover:bg-surface-secondary',
                    )}
                  >
                    <Icon className="size-5" aria-hidden />
                    <span className="text-[11px] font-medium leading-none">
                      {t(labelKeyFor(item, view))}
                    </span>
                  </button>
                )
              })}
            </div>
          </Drawer.Dialog>
        </Drawer.Content>
      </Drawer.Backdrop>
    </Drawer>
  )
}

/**
 * The console frame. On a phone (and in the 430px widget iframe) navigation is
 * a bottom tab bar; from 768px up it becomes a sidebar — an icon rail until
 * 1280px, then a full column with section headings.
 */
export function AppShell({ children }: { children: ReactNode }) {
  const { user, site, view } = useSession()
  const isAdmin = !!user?.is_admin
  const navigate = useNavigate()
  const confirm = useConfirm()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const [siteOpen, setSiteOpen] = useState(false)
  const [moreOpen, setMoreOpen] = useState(false)

  const primary = primaryItemsFor(isAdmin, view)
  const secondary = secondaryItemsFor(isAdmin, view)
  const moreActive = secondary.some((item) => item.path === pathname)
  const currentItem =
    findNavItem(pathname.replace('/', '')) ?? NAV_ITEMS.find((item) => item.path === pathname)

  const go = (path: string) => {
    setMoreOpen(false)
    void navigate({ to: path })
  }

  // The profile page belongs to the visitor's own pages, so an admin coming
  // from the account menu switches view first instead of landing outside the
  // menu they are looking at.
  const openAccount = () => {
    setMoreOpen(false)
    if (isAdmin) session.setView('user')
    void navigate({ to: '/account' })
  }

  const switchView = (next: ViewMode) => {
    setMoreOpen(false)
    if (next === view) return
    session.setView(next)
    void navigate({ to: next === 'admin' ? '/dashboard' : '/activity' })
  }

  const logout = async () => {
    const ok = await confirm({
      title: t('logout'),
      description: t('logoutConfirm'),
      confirmText: t('logout'),
      danger: true,
    })
    if (!ok) return
    session.logout()
    void navigate({ to: '/login' })
  }

  return (
    <div className="flex h-full min-h-0 bg-background text-foreground">
      <Sidebar onOpenSites={() => setSiteOpen(true)} onLogout={logout} />

      <div className="flex min-w-0 flex-1 flex-col">
        <header
          className={cn(
            'flex h-12 shrink-0 items-center justify-between gap-2 border-b border-border px-2 md:hidden',
            // The host page floats its own close button over the iframe's top
            // right corner. The account menu only repeats what the account page
            // and the more sheet already offer, so it may sit under that button;
            // the theme toggle just has to stay clear of it.
            isEmbedded && 'pr-3.5',
          )}
        >
          {isAdmin && view === 'admin' ? (
            <Button
              variant="ghost"
              size="sm"
              onPress={() => setSiteOpen(true)}
              className="min-w-0 gap-1.5 px-2 text-[13px] font-medium"
            >
              <span
                className={cn('size-1.5 shrink-0 rounded-full', site ? 'bg-accent' : 'bg-muted')}
              />
              <span className="truncate">{site || t('allSites')}</span>
              <ChevronDown className="size-3.5 shrink-0 opacity-60" aria-hidden />
            </Button>
          ) : (
            <span className="flex min-w-0 items-center gap-2 px-1 text-[13px] font-medium">
              <BrandMark size="sm" />
              <span className="truncate">
                {currentItem ? t(labelKeyFor(currentItem, view)) : t('ctrlCenter')}
              </span>
            </span>
          )}

          <div className="flex shrink-0 items-center gap-1">
            <ThemeToggle />
            <AccountMenu
              name={user?.name ?? ''}
              email={user?.email ?? ''}
              view={isAdmin ? view : undefined}
              onSwitchView={switchView}
              onAccount={openAccount}
              onLogout={logout}
            />
          </div>
        </header>

        <main className="min-h-0 flex-1 overflow-y-auto scrollbar-thin">
          <div className="mx-auto w-full max-w-[600px] px-3.5 py-4 md:max-w-[920px] md:px-8 md:py-7 xl:max-w-[1080px]">
            {children}
          </div>
        </main>

        <nav className="flex shrink-0 items-stretch gap-1 border-t border-border bg-surface/95 px-1.5 pb-[max(env(safe-area-inset-bottom),6px)] pt-1.5 backdrop-blur md:hidden">
          {primary.map((item) => (
            <TabButton
              key={item.id}
              icon={item.icon}
              label={t(labelKeyFor(item, view))}
              active={item.path === pathname}
              onPress={() => go(item.path)}
            />
          ))}
          {/* An admin in "my pages" has no secondary pages left, but still needs
              the sheet for the trip back to the management console. */}
          {secondary.length || isAdmin ? (
            <TabButton
              icon={MoreHorizontal}
              label={t('nav')}
              active={moreActive}
              onPress={() => setMoreOpen(true)}
            />
          ) : null}
        </nav>
      </div>

      <SiteSwitcher open={siteOpen} onOpenChange={setSiteOpen} />
      <MoreSheet
        items={secondary}
        open={moreOpen}
        onOpenChange={setMoreOpen}
        current={pathname}
        onPick={(item) => go(item.path)}
        isAdmin={isAdmin}
        view={view}
        onSwitchView={switchView}
      />
    </div>
  )
}
