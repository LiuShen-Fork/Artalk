import { Moon, Sun } from 'lucide-react'
import { t } from '@/lib/i18n'
import { theme, useDarkMode } from '@/lib/theme'
import { cn } from '@/lib/utils'

/**
 * The light/dark switch. A plain `<button>` rather than a HeroUI `Button` so it
 * keeps exactly the 36px row height used by the nav entries, the site switcher
 * and the account trigger next to it — HeroUI's `sm` size is 32px on desktop.
 */
export function ThemeToggle({ className }: { className?: string }) {
  const dark = useDarkMode()
  const label = dark ? t('lightTheme') : t('darkTheme')

  return (
    <button
      type="button"
      onClick={() => theme.toggle()}
      aria-label={label}
      title={label}
      className={cn(
        'grid size-9 shrink-0 place-items-center rounded-control text-muted transition-colors hover:bg-surface-secondary hover:text-foreground',
        className,
      )}
    >
      {dark ? (
        <Sun className="size-[18px]" aria-hidden />
      ) : (
        <Moon className="size-[18px]" aria-hidden />
      )}
    </button>
  )
}
