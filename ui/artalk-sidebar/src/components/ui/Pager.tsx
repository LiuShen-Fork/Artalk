import { Button } from '@heroui/react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'

interface PagerProps {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
  disabled?: boolean
  className?: string
}

/**
 * A compact pager for a 430px panel: arrows, the current page and the total.
 * HeroUI's Pagination renders one item per page, which stops fitting as soon as
 * a site has more than a handful of pages.
 */
export function Pager({ page, pageSize, total, onChange, disabled, className }: PagerProps) {
  const pages = Math.max(1, Math.ceil(total / Math.max(pageSize, 1)))
  if (total <= pageSize) return null

  return (
    <div className={cn('flex items-center justify-between gap-2 pt-1', className)}>
      <span className="text-xs tabular-nums text-muted">
        {page} / {pages}
        <span className="mx-1.5 text-border">·</span>
        {total}
      </span>
      <div className="flex items-center gap-1">
        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          isDisabled={disabled || page <= 1}
          aria-label={t('back')}
          onPress={() => onChange(page - 1)}
        >
          <ChevronLeft className="size-4" aria-hidden />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          isDisabled={disabled || page >= pages}
          aria-label={t('nextStep')}
          onPress={() => onChange(page + 1)}
        >
          <ChevronRight className="size-4" aria-hidden />
        </Button>
      </div>
    </div>
  )
}
