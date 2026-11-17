import { Dropdown } from '@heroui/react'
import { ChevronDown, CircleUserRound, LogOut } from 'lucide-react'
import { UserAvatar } from '@/components/ui/UserAvatar'
import { avatarUrl } from '@/lib/artalk'
import { t } from '@/lib/i18n'

interface AccountMenuProps {
  name: string
  email: string
  onAccount: () => void
  onLogout: () => void | Promise<void>
}

export function AccountMenu({ name, email, onAccount, onLogout }: AccountMenuProps) {
  return (
    <Dropdown>
      <Dropdown.Trigger
        className="h-9 gap-1 rounded-lg bg-transparent px-1 data-[hovered]:bg-surface-secondary"
        aria-label={t('account')}
      >
        <UserAvatar name={name} src={avatarUrl(email)} size="sm" />
        <ChevronDown className="size-3.5 opacity-50" aria-hidden />
      </Dropdown.Trigger>
      <Dropdown.Popover placement="bottom end">
        <div className="border-b border-border px-3 py-2">
          <p className="truncate text-[13px] font-medium">{name || t('unknownUser')}</p>
          {email ? <p className="truncate text-[11px] text-muted">{email}</p> : null}
        </div>
        <Dropdown.Menu
          onAction={(key) => {
            if (key === 'account') onAccount()
            if (key === 'logout') void onLogout()
          }}
        >
          <Dropdown.Item id="account" textValue={t('account')} className="gap-2 text-[13px]">
            <CircleUserRound className="size-4" aria-hidden />
            {t('account')}
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
