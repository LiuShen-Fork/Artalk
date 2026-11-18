import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface PageHeaderProps {
  title: string
  description?: string
  actions?: ReactNode
  className?: string
}

/**
 * The one heading every page starts with. Keeping it to a title, an optional
 * line of context and a slot for the page's primary action is what makes the
 * ten pages feel like one console.
 */
export function PageHeader({ title, description, actions, className }: PageHeaderProps) {
  return (
    <div className={cn('flex items-start justify-between gap-3 md:mb-1 md:items-end', className)}>
      <div className="min-w-0">
        <h1 className="text-[19px] font-semibold leading-tight tracking-[-0.015em] md:text-[23px]">
          {title}
        </h1>
        {description ? (
          <p className="mt-1 text-[13px] leading-snug text-muted md:text-[13.5px]">{description}</p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex shrink-0 items-center gap-1.5 md:gap-2">{actions}</div>
      ) : null}
    </div>
  )
}
