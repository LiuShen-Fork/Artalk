import { Dropdown } from '@heroui/react'
import { Check, Languages } from 'lucide-react'
import { applyWidgetLocale } from '@/lib/artalk'
import { hoverTitle } from '@/lib/hover-title'
import { chooseLocale, t, useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

/** The languages the console ships, labelled in their own language. */
export const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'ja', label: '日本語' },
  { value: 'ko', label: '한국어' },
  { value: 'fr', label: 'Français' },
  { value: 'pt-PT', label: 'Português' },
  { value: 'ru', label: 'Русский' },
  { value: 'tr', label: 'Türkçe' },
] as const

/**
 * The console's own language switch. The site config only decides which
 * language a visitor starts with; this writes the visitor's pick to
 * localStorage, next to the theme.
 */
export function LanguageMenu({ className }: { className?: string }) {
  const locale = useLocale()
  const current = LANGUAGES.find((item) => item.value === locale)?.label ?? t('language')

  return (
    <Dropdown>
      <Dropdown.Trigger
        className={cn(
          'flex size-9 shrink-0 items-center justify-center rounded-control text-muted transition-colors data-[hovered]:bg-surface-secondary data-[hovered]:text-foreground xl:h-9 xl:w-auto xl:min-w-0 xl:flex-1 xl:gap-1.5 xl:px-2',
          className,
        )}
        aria-label={t('language')}
        {...hoverTitle(t('language'))}
      >
        <Languages className="size-[18px] shrink-0" aria-hidden />
        <span className="hidden truncate text-[12px] xl:inline">{current}</span>
      </Dropdown.Trigger>
      <Dropdown.Popover placement="top end">
        <Dropdown.Menu
          onAction={(key) => {
            const value = String(key)
            // Load the widget's locale plugin first, so the list the pages
            // re-mount already renders in the new language.
            void applyWidgetLocale(value).finally(() => void chooseLocale(value))
          }}
        >
          {LANGUAGES.map((item) => (
            <Dropdown.Item
              key={item.value}
              id={item.value}
              textValue={item.label}
              className="gap-2 text-[13px]"
            >
              <Check
                className={cn('size-4', item.value === locale ? 'opacity-100' : 'opacity-0')}
                aria-hidden
              />
              {item.label}
            </Dropdown.Item>
          ))}
        </Dropdown.Menu>
      </Dropdown.Popover>
    </Dropdown>
  )
}
