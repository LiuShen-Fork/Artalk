import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export interface SegmentedItem<T extends string> {
  id: T
  label: string
  icon?: ReactNode
}

/**
 * A tab strip that always sizes itself to its labels. HeroUI's Tabs put the
 * list inside a scroll shadow, which collapses to a single tab plus arrow
 * buttons whenever the strip shares a row with a search field.
 */
export function Segmented<T extends string>({
  items,
  value,
  onChange,
  className,
  ariaLabel,
}: {
  items: SegmentedItem<T>[]
  value: T
  onChange: (value: T) => void
  className?: string
  ariaLabel?: string
}) {
  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      className={cn(
        'grid grid-flow-col auto-cols-fr gap-0.5 rounded-lg bg-default p-0.5 md:inline-grid',
        className,
      )}
    >
      {items.map((item) => {
        const active = item.id === value
        return (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(item.id)}
            className={cn(
              'flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-2.5 py-1.5 text-[12.5px] font-medium transition-colors',
              active
                ? 'bg-surface text-foreground shadow-sm'
                : 'text-muted hover:text-foreground',
            )}
          >
            {item.icon}
            {item.label}
          </button>
        )
      })}
    </div>
  )
}
