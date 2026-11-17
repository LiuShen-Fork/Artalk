import { EmptyState as HeroEmptyState } from '@heroui/react'
import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <HeroEmptyState
      className={cn(
        'flex flex-col items-center gap-3 rounded-xl border border-dashed border-border bg-surface/60 px-6 py-10 text-center',
        className,
      )}
    >
      <span className="grid size-11 place-items-center rounded-full bg-default text-muted">
        <Icon className="size-5" aria-hidden />
      </span>
      <div className="space-y-1">
        <p className="text-sm font-medium">{title}</p>
        {description ? <p className="text-[13px] text-muted">{description}</p> : null}
      </div>
      {action}
    </HeroEmptyState>
  )
}
