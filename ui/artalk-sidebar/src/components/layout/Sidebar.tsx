import { useNavigate, useRouterState } from '@tanstack/react-router'
import { ChevronDown, Globe2, LayoutGrid, UserRound } from 'lucide-react'
import { navGroupsFor, labelKeyFor, type ViewMode } from '@/app/nav'
import { AccountMenu } from '@/components/layout/AccountMenu'
import { LanguageMenu } from '@/components/layout/LanguageMenu'
import { ThemeToggle } from '@/components/layout/ThemeToggle'
import { BrandMark } from '@/components/ui/BrandMark'
import { useBrandName } from '@/lib/console'
import { t } from '@/lib/i18n'
import { session, useSession } from '@/lib/session'
import { cn } from '@/lib/utils'

type Icon = typeof LayoutGrid

/**
 * Admin/user switch. Sits in the wide sidebar (two columns), stacks into a
 * pair of icon buttons on the collapsed rail, and becomes two full rows inside
 * the mobile "more" sheet.
 */
export function ModeSwitch({
  view,
  onChange,
  layout = 'segmented',
}: {
  view: ViewMode
  onChange: (view: ViewMode) => void
  layout?: 'segmented' | 'list'
}) {
  const options: { id: ViewMode; label: string; icon: Icon }[] = [
    { id: 'admin', label: t('viewAdmin'), icon: LayoutGrid },
    { id: 'user', label: t('viewUser'), icon: UserRound },
  ]

  if (layout === 'list') {
    return (
      <div className="space-y-1">
        {options.map(({ id, label, icon: Icon }) => {
          const active = view === id
          return (
            <button
              key={id}
              type="button"
              onClick={() => onChange(id)}
              aria-pressed={active}
              className={cn(
                'flex w-full items-center gap-2.5 rounded-control px-3 py-2.5 text-left text-[13px] font-medium transition-colors',
                active
                  ? 'bg-accent/10 text-accent'
                  : 'text-muted hover:bg-surface-secondary hover:text-foreground',
              )}
            >
              <Icon className="size-4" aria-hidden />
              {label}
            </button>
          )
        })}
      </div>
    )
  }

  return (
    <div
      role="tablist"
      aria-label={t('viewAdmin')}
      className="mx-auto flex w-fit gap-1 rounded-control bg-segment-track p-0.5 xl:mx-0 xl:grid xl:w-full xl:grid-cols-2 xl:gap-1"
    >
      {options.map(({ id, label, icon: Icon }) => {
        const active = view === id
        return (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={active}
            title={label}
            onClick={() => onChange(id)}
            className={cn(
              // Square on the rail, half of a segmented pair on the wide bar: a
              // 28px-tall button stretched across the rail reads as a pill.
              'flex size-8 items-center justify-center gap-2 rounded-[var(--control-radius-inner)] text-[12.5px] font-medium transition-colors xl:h-8 xl:w-full xl:justify-center xl:gap-2 xl:px-2',
              active
                ? 'bg-segment-active text-foreground shadow-sm'
                : 'text-muted hover:text-foreground',
            )}
          >
            <Icon className="size-[15px] shrink-0" aria-hidden />
            <span className="hidden xl:inline">{label}</span>
          </button>
        )
      })}
    </div>
  )
}

function NavLink({
  icon: Icon,
  label,
  active,
  onPress,
}: {
  icon: Icon
  label: string
  active: boolean
  onPress: () => void
}) {
  return (
    <button
      type="button"
      onClick={onPress}
      title={label}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'relative mx-auto flex size-9 items-center justify-center gap-2.5 rounded-control px-2 text-[13px] font-medium transition-colors xl:mx-0 xl:h-9 xl:w-full xl:justify-start xl:px-2.5',
        active
          ? 'bg-accent/10 text-accent'
          : 'text-muted hover:bg-surface-secondary hover:text-foreground',
      )}
    >
      {/* The rail has no room for a section title, so the active row keeps a
          marker of its own on the edge. */}
      <span
        aria-hidden
        className={cn(
          'absolute start-0 top-1/2 h-4 w-0.5 -translate-y-1/2 rounded-full bg-accent transition-opacity xl:hidden',
          active ? 'opacity-100' : 'opacity-0',
        )}
      />
      <Icon className="size-[18px] shrink-0" aria-hidden />
      <span className="hidden truncate xl:inline">{label}</span>
    </button>
  )
}

/**
 * The wide-screen navigation column: brand, admin/user switch, site scope, the
 * grouped page list and the account row. Below 1280px it collapses to an icon
 * rail so a half-width browser window still leaves room for content; on the
 * rail every row is a centred square so nothing looks half-filled.
 */
export function Sidebar({
  onOpenSites,
  onLogout,
}: {
  onOpenSites: () => void
  onLogout: () => void
}) {
  const { user, site, view } = useSession()
  const isAdmin = !!user?.is_admin
  const brandName = useBrandName()
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const groups = navGroupsFor(isAdmin, view)
  const showGroupTitles = view === 'admin'
  const current = pathname.replace('/', '')

  const switchView = (next: ViewMode) => {
    if (next === view) return
    session.setView(next)
    void navigate({ to: next === 'admin' ? '/dashboard' : '/activity' })
  }

  return (
    <aside className="hidden shrink-0 flex-col border-r border-border bg-surface/50 md:flex md:w-[76px] xl:w-[244px]">
      <div className="flex h-14 shrink-0 items-center justify-center gap-2 px-3 xl:justify-start xl:px-4">
        <BrandMark size="md" />
        <span className="hidden min-w-0 flex-1 truncate text-[13.5px] font-semibold xl:inline">
          {brandName}
          <span className="ml-1 font-normal text-muted">{t('ctrlCenter')}</span>
        </span>
      </div>

      {isAdmin ? (
        <div className="shrink-0 px-2.5 pb-1.5 xl:px-3">
          <ModeSwitch view={view} onChange={switchView} />
        </div>
      ) : null}

      {isAdmin && view === 'admin' ? (
        <div className="shrink-0 px-2.5 pb-1 xl:px-3">
          <button
            type="button"
            onClick={onOpenSites}
            title={t('allSites')}
            className="mx-auto flex size-9 items-center justify-center gap-2 rounded-control px-2 text-[12.5px] text-muted transition-colors hover:bg-surface-secondary hover:text-foreground xl:mx-0 xl:h-9 xl:w-full xl:justify-start"
          >
            <Globe2 className="size-[15px] shrink-0" aria-hidden />
            <span className="hidden min-w-0 flex-1 truncate text-start xl:inline">
              {site || t('allSites')}
            </span>
            <ChevronDown className="hidden size-3.5 shrink-0 opacity-60 xl:block" aria-hidden />
          </button>
        </div>
      ) : null}

      <nav
        aria-label={t('nav')}
        className="scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto px-2.5 py-3 xl:px-3"
      >
        {groups.map((group) => (
          <div key={group.id} className="space-y-1">
            {/* Visitors only have two destinations, so section titles would be
                noise above them. */}
            {showGroupTitles ? (
              <p className="hidden px-2 pb-1.5 pt-0.5 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80 xl:block">
                {t(group.titleKey)}
              </p>
            ) : null}
            {group.items.map((item) => (
              <NavLink
                key={item.id}
                icon={item.icon}
                label={t(labelKeyFor(item, view))}
                active={current === item.id}
                onPress={() => void navigate({ to: item.path })}
              />
            ))}
          </div>
        ))}
      </nav>

      <div className="shrink-0 border-t border-border px-2.5 py-2.5 xl:px-3">
        <div className="flex flex-col items-center gap-1.5">
          {/* Utility icons sit above the profile row, which keeps the full
              sidebar width for the name and the caret. */}
          <div className="flex flex-col items-center gap-1.5 xl:w-full xl:flex-row xl:gap-1 xl:px-1">
            <LanguageMenu />
            <ThemeToggle />
          </div>
          <AccountMenu
            name={user?.name ?? ''}
            email={user?.email ?? ''}
            showText
            view={isAdmin ? view : undefined}
            onSwitchView={switchView}
            onAccount={() => void navigate({ to: '/account' })}
            onLogout={onLogout}
          />
        </div>
      </div>
    </aside>
  )
}
