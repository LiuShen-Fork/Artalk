import { UserAvatar } from '@/components/ui/UserAvatar'
import { avatarUrl } from '@/lib/artalk'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'

interface AccountButtonProps {
  name: string
  email: string
  /** Show the name and address next to the avatar (desktop sidebar). */
  showText?: boolean
  onPress: () => void
}

/**
 * The avatar is a direct shortcut to the account page, not a menu: profile,
 * the sign-in secret, linked identities and sign-out all live on that page, so
 * there is exactly one place where account actions can be found.
 */
export function AccountButton({ name, email, showText, onPress }: AccountButtonProps) {
  return (
    <button
      type="button"
      onClick={onPress}
      aria-label={t('account')}
      className={cn(
        'flex h-9 min-w-0 items-center gap-2 rounded-control px-1 transition-colors hover:bg-surface-secondary',
        showText ? 'xl:flex-1 xl:justify-start' : 'w-9 justify-center',
      )}
    >
      <UserAvatar name={name} src={avatarUrl(email)} size="sm" />
      {showText ? (
        <span className="hidden min-w-0 flex-col items-start leading-tight xl:flex">
          <span className="max-w-[132px] truncate text-[12.5px] font-medium">
            {name || t('unknownUser')}
          </span>
          <span className="max-w-[132px] truncate text-[11px] text-muted">{email}</span>
        </span>
      ) : null}
    </button>
  )
}
