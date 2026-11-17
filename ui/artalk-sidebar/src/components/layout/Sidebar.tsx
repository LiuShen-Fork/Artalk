import { Button } from '@heroui/react'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { ChevronDown, Globe2, LayoutGrid, Moon, Sun, UserRound } from 'lucide-react'
import { navGroupsFor, labelKeyFor, type ViewMode } from '@/app/nav'
import { AccountMenu } from '@/components/layout/AccountMenu'
import { t } from '@/lib/i18n'
import { session, useSession } from '@/lib/session'
import { theme, useDarkMode } from '@/lib/theme'
import { cn } from '@/lib/utils'

type Icon = typeof LayoutGrid

/**
 * Admin/user switch. A segmented control in the sidebar, two rows inside the
 * mobile "more" sheet.
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
                'flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-[13px] font-medium transition-colors',
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
      className="grid grid-cols-2 gap-0.5 rounded-lg bg-default p-0.5 xl:gap-1 xl:p-1"
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
              'flex items-center justify-center gap-1.5 rounded-md px-1 py-1.5 text-[12.5px] font-medium transition-colors xl:px-2',
              active ? 'bg-surface text-foreground shadow-sm' : 'text-muted hover:text-foreground',
            )}
          >
            <Icon className="size-4 shrink-0" aria-hidden />
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
        'flex w-full items-center justify-center gap-2.5 rounded-lg px-2 py-2 text-[13px] font-medium transition-colors xl:justify-start xl:px-2.5',
        active
          ? 'bg-accent/10 text-accent'
          : 'text-muted hover:bg-surface-secondary hover:text-foreground',
      )}
    >
      <Icon className="size-[18px] shrink-0" aria-hidden />
      <span className="hidden truncate xl:inline">{label}</span>
    </button>
  )
}

/**
 * The wide-screen navigation column: brand, admin/user switch, site scope, the
 * grouped page list and the account row. Below 1280px it collapses to an icon
 * rail so a half-width browser window still leaves room for content.
 */
export function Sidebar({ onOpenSites, onLogout }: { onOpenSites: () => void; onLogout: () => void }) {
  const { user, site, view } = useSession()
  const isAdmin = !!user?.is_admin
  const dark = useDarkMode()
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const groups = navGroupsFor(isAdmin, view)
  const showGroupTitles = view === 'admin'
  const current = pathname.replace('/', '')

  const switchView = (next: ViewMode) => {
    if (next === view) return
    session.setView(next)
    void navigate({ to: next === 'admin' ? '/dashboard' : '/comments' })
  }

  return (
    <aside className="hidden shrink-0 flex-col border-r border-border bg-surface/50 md:flex md:w-[76px] xl:w-[244px]">
      <div className="flex h-14 items-center gap-2 px-3 xl:px-4">
        <span className="grid size-7 shrink-0 place-items-center rounded-lg bg-accent text-[13px] font-bold text-accent-foreground">
          A
        </span>
        <span className="hidden min-w-0 flex-1 truncate text-[13.5px] font-semibold xl:inline">
          Artalk
          <span className="ml-1 font-normal text-muted">{t('ctrlCenter')}</span>
        </span>
      </div>

      {isAdmin ? (
        <div className="px-2 pb-1.5 xl:px-3">
          <ModeSwitch view={view} onChange={switchView} />
        </div>
      ) : null}

      {isAdmin && view === 'admin' ? (
        <div className="px-2 pb-1 xl:px-3">
          <button
            type="button"
            onClick={onOpenSites}
            title={site || t('allSites')}
            className="flex w-full items-center justify-center gap-1.5 rounded-lg px-2 py-1.5 text-[12.5px] text-muted transition-colors hover:bg-surface-secondary hover:text-foreground xl:justify-start"
          >
            <Globe2 className="size-4 shrink-0" aria-hidden />
            <span className="hidden min-w-0 flex-1 truncate text-start xl:inline">
              {site || t('allSites')}
            </span>
            <ChevronDown className="hidden size-3.5 shrink-0 opacity-60 xl:block" aria-hidden />
          </button>
        </div>
      ) : null}

      <nav className="min-h-0 flex-1 space-y-3 overflow-y-auto scrollbar-thin px-2 py-2 xl:px-3">
        {groups.map((group) => (
          <div key={group.id} className="space-y-0.5">
            {/* Visitors only have two destinations, so section titles would be
                noise above them. */}
            {showGroupTitles ? (
              <p className="hidden px-2 pb-0.5 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80 xl:block">
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

      <div className="border-t border-border px-2 py-2 xl:px-3">
        <div className="flex flex-col items-center gap-1 xl:flex-row">
          <Button
            isIconOnly
            size="sm"
            variant="ghost"
            aria-label={dark ? t('lightTheme') : t('darkTheme')}
            onPress={() => theme.toggle()}
          >
            {dark ? <Sun className="size-4" aria-hidden /> : <Moon className="size-4" aria-hidden />}
          </Button>
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
