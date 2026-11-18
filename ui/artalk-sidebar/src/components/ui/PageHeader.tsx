import { useRouterState } from '@tanstack/react-router'
import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { groupTitleKeyFor, navItemForPath } from '@/app/nav'
import { isEmbedded } from '@/lib/boot'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'

interface PageHeaderProps {
  title: string
  description?: string
  actions?: ReactNode
  /**
   * The small line above the title. Defaults to the nav group the page lives
   * in, so every page is placed in the console without repeating the lookup.
   */
  eyebrow?: string
  /** Defaults to the page's own nav icon; pass `null` for a plain header. */
  icon?: LucideIcon | null
  className?: string
}

/**
 * The one heading every page starts with: the section it belongs to, an icon,
 * the title, a line of context and a slot for the page's primary action. The
 * rule underneath separates the heading from the page body — without it the
 * first card reads as part of the title block.
 */
export function PageHeader({
  title,
  description,
  actions,
  eyebrow,
  icon,
  className,
}: PageHeaderProps) {
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const item = navItemForPath(pathname)
  const Icon = icon === undefined ? item?.icon : (icon ?? undefined)
  const groupKey = groupTitleKeyFor(pathname)
  const eyebrowText = eyebrow ?? (groupKey ? t(groupKey) : undefined)

  return (
    <div
      className={cn(
        'flex items-start justify-between gap-3 border-b border-border/70 pb-4 md:items-end md:gap-6 md:pb-5',
        className,
      )}
    >
      <div className="flex min-w-0 flex-1 items-start gap-2.5 md:gap-3">
        {Icon ? (
          <span className="mt-0.5 grid size-9 shrink-0 place-items-center rounded-[10px] bg-accent/10 text-accent md:size-10">
            <Icon className="size-[18px] md:size-5" aria-hidden />
          </span>
        ) : null}
        <div className="min-w-0">
          {eyebrowText ? (
            <p className="mb-0.5 text-[10.5px] font-medium uppercase tracking-[0.12em] text-accent/90">
              {eyebrowText}
            </p>
          ) : null}
          <h1 className="text-[20px] font-semibold leading-tight tracking-[-0.015em] md:text-[24px]">
            {title}
          </h1>
          {description ? (
            <p className="mt-1 max-w-[72ch] text-[13px] leading-snug text-muted md:text-[13.5px]">
              {description}
            </p>
          ) : null}
        </div>
      </div>
      {actions ? (
        <div
          className={cn(
            'flex shrink-0 items-center gap-1.5 md:gap-2',
            // Wide embeds only: the host floats a close button over the frame's
            // top right corner, right where a page puts its primary action.
            isEmbedded && 'md:pr-[52px]',
          )}
        >
          {actions}
        </div>
      ) : null}
    </div>
  )
}
