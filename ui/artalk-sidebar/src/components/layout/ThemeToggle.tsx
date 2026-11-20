import { Moon, Sun } from 'lucide-react'
import { t } from '@/lib/i18n'
import { theme, useDarkMode } from '@/lib/theme'
import { cn } from '@/lib/utils'

/**
 * The light/dark switch. A plain `<button>` rather than a HeroUI `Button` so it
 * keeps exactly the 36px row height used by the nav entries, the site switcher
 * and the account trigger next to it — HeroUI's `sm` size is 32px on desktop.
 *
 * On the icon rail it is a square; in the wide sidebar it takes half of the row
 * above the account and shows what it does.
 */
export function ThemeToggle({ className }: { className?: string }) {
  const dark = useDarkMode()
  const label = dark ? t('lightTheme') : t('darkTheme')
  // The visible label is the short form: the two footer buttons share one row,
  // and "Тёмная тема" would be cut off in half of it.
  const short = dark ? t('themeLight') : t('themeDark')

  return (
    <button
      type="button"
      onClick={() => theme.toggle()}
      aria-label={label}
      title={label}
      className={cn(
        'flex size-9 shrink-0 items-center justify-center rounded-control text-muted transition-colors hover:bg-surface-secondary hover:text-foreground xl:h-9 xl:w-auto xl:min-w-0 xl:flex-1 xl:gap-1.5 xl:px-2',
        className,
      )}
    >
      {dark ? (
        <Sun className="size-[18px] shrink-0" aria-hidden />
      ) : (
        <Moon className="size-[18px] shrink-0" aria-hidden />
      )}
      <span className="hidden truncate text-[12px] xl:inline">{short}</span>
    </button>
  )
}
