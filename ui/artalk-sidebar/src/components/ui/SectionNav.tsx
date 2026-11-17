import { cn } from '@/lib/utils'

export interface SectionNavItem<T extends string> {
  id: T
  label: string
}

interface SectionNavProps<T extends string> {
  items: SectionNavItem<T>[]
  value: T
  onChange: (id: T) => void
}

/**
 * Section switching for the long pages. Phones and the narrow iframe get a
 * wrapping chip row; from md up the same list becomes a vertical rail beside
 * the content, which is what keeps a settings page from scrolling forever on a
 * desktop.
 */
export function SectionNav<T extends string>({ items, value, onChange }: SectionNavProps<T>) {
  return (
    <>
      <div className="flex flex-wrap gap-1.5 md:hidden">
        {items.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => onChange(item.id)}
            aria-current={item.id === value ? 'true' : undefined}
            className={cn(
              'rounded-full border px-2.5 py-1 text-[12px] font-medium transition-colors',
              item.id === value
                ? 'border-accent/40 bg-accent/10 text-accent'
                : 'border-border bg-surface text-muted hover:text-foreground',
            )}
          >
            {item.label}
          </button>
        ))}
      </div>

      <nav className="hidden shrink-0 md:flex md:w-[176px] md:flex-col md:gap-0.5">
        {items.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => onChange(item.id)}
            aria-current={item.id === value ? 'page' : undefined}
            className={cn(
              'rounded-lg px-2.5 py-1.5 text-left text-[13px] transition-colors',
              item.id === value
                ? 'bg-accent/10 font-medium text-accent'
                : 'text-muted hover:bg-surface-secondary hover:text-foreground',
            )}
          >
            {item.label}
          </button>
        ))}
      </nav>
    </>
  )
}
