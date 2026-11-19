import { Dropdown } from '@heroui/react'
import { ArrowLeftRight, ChevronDown, CircleUserRound, LogOut } from 'lucide-react'
import type { ViewMode } from '@/app/nav'
import { UserAvatar } from '@/components/ui/UserAvatar'
import { avatarUrl } from '@/lib/artalk'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import { hoverTitle } from '@/lib/hover-title'

interface AccountMenuProps {
  name: string
  email: string
  /** Show the name next to the avatar (desktop sidebar). */
  showText?: boolean
  /** Present only for admins: offer the jump to the other half of the console. */
  view?: ViewMode
  onSwitchView?: (view: ViewMode) => void
  onAccount: () => void
  onLogout: () => void | Promise<void>
}

export function AccountMenu({
  name,
  email,
  showText,
  view,
  onSwitchView,
  onAccount,
  onLogout,
}: AccountMenuProps) {
  return (
    <Dropdown>
      <Dropdown.Trigger
        className={cn(
          'h-9 items-center justify-center gap-1.5 rounded-control bg-transparent data-[hovered]:bg-surface-secondary',
          // The rail keeps a square row like every other nav entry (the avatar
          // is 32px, so horizontal padding would push it past 36); the wide bar
          // turns the same trigger into a full-width profile row.
          showText ? 'w-9 min-w-0 px-0 xl:w-full xl:flex-1 xl:justify-start xl:px-1' : 'w-9 px-0',
        )}
        aria-label={t('account')}
        {...hoverTitle(t('account'))}
      >
        {/* The trigger is a HeroUI button with wrap-friendly content; keeping
            everything in one nowrap flex row stops the caret from dropping
            under the avatar at small widths. */}
        <span className="flex min-w-0 flex-nowrap items-center gap-2 whitespace-nowrap">
          <UserAvatar name={name} src={avatarUrl(email)} size="sm" />
          {/* items-stretch, not items-start: a squeezed flex row would let a
              shrink-to-fit column keep its 132px max-width and paint its
              ellipsis under the caret. */}
          {showText ? (
            <span className="hidden min-w-0 flex-col pr-1.5 text-left leading-tight xl:flex">
              <span className="max-w-[132px] truncate text-[12.5px] font-medium">
                {name || t('unknownUser')}
              </span>
              <span className="max-w-[132px] truncate text-[11px] text-muted">{email}</span>
            </span>
          ) : null}
          {showText ? (
            <ChevronDown className="hidden size-3.5 shrink-0 opacity-50 xl:block" aria-hidden />
          ) : null}
        </span>
      </Dropdown.Trigger>
      <Dropdown.Popover placement="bottom end">
        <div className="border-b border-border px-3 py-2">
          <p className="truncate text-[13px] font-medium">{name || t('unknownUser')}</p>
          {email ? <p className="truncate text-[11px] text-muted">{email}</p> : null}
        </div>
        <Dropdown.Menu
          onAction={(key) => {
            if (key === 'account') onAccount()
            if (key === 'view' && view && onSwitchView) {
              onSwitchView(view === 'admin' ? 'user' : 'admin')
            }
            if (key === 'logout') void onLogout()
          }}
        >
          {view && onSwitchView ? (
            <Dropdown.Item
              id="view"
              textValue={view === 'admin' ? t('viewUser') : t('viewAdmin')}
              className="gap-2 text-[13px]"
            >
              <ArrowLeftRight className="size-4" aria-hidden />
              {view === 'admin' ? t('viewUser') : t('viewAdmin')}
            </Dropdown.Item>
          ) : null}
          <Dropdown.Item id="account" textValue={t('navAccount')} className="gap-2 text-[13px]">
            <CircleUserRound className="size-4" aria-hidden />
            {t('navAccount')}
          </Dropdown.Item>
          <Dropdown.Item
            id="logout"
            textValue={t('logout')}
            variant="danger"
            className="gap-2 text-[13px]"
          >
            <LogOut className="size-4" aria-hidden />
            {t('logout')}
          </Dropdown.Item>
        </Dropdown.Menu>
      </Dropdown.Popover>
    </Dropdown>
  )
}
