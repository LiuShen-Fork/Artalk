import { Button } from '@heroui/react'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react'
import { SelectField } from '@/components/ui/SelectField'
import { t } from '@/lib/i18n'
import { PAGE_SIZE_OPTIONS, pageWindow } from '@/lib/pager'
import { cn } from '@/lib/utils'

interface PagerProps {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
  /** Pass a handler to let the reader pick how many rows they want; omit to hide the picker. */
  onPageSizeChange?: (size: number) => void
  pageSizeOptions?: number[]
  disabled?: boolean
  className?: string
}

/**
 * The console's only pager. Every list uses it so paging looks and behaves the
 * same everywhere: first/last jumps, a numbered window with gaps, a live total
 * and an optional page-size picker. HeroUI's Pagination renders one item per
 * page, which stops fitting as soon as a site has more than a handful.
 *
 * The cluster is centred: the total, the controls and the picker read as one
 * group under the list instead of being spread against the card edges. Each
 * piece keeps its own width, so on a narrow screen the group wraps as whole
 * items and stays centred on however many lines it needs.
 */
export function Pager({
  page,
  pageSize,
  total,
  onChange,
  onPageSizeChange,
  pageSizeOptions,
  disabled,
  className,
}: PagerProps) {
  const pages = Math.max(1, Math.ceil(total / Math.max(pageSize, 1)))
  // One page means nothing to page through: a lone "1" next to a "next" button
  // that cannot move is noise, so the pager stays out of the way.
  if (pages <= 1) return null

  const go = (next: number) => onChange(Math.min(Math.max(next, 1), pages))

  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-center gap-x-2.5 gap-y-2 sm:gap-x-3',
        className,
      )}
    >
      <span className="whitespace-nowrap text-xs leading-none tabular-nums text-muted">
        {t('pagerTotal', { total })}
      </span>

      <div className="flex shrink-0 items-center gap-0.5">
        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          className="hidden sm:inline-flex"
          isDisabled={disabled || page <= 1}
          aria-label={t('pagerFirst')}
          onPress={() => go(1)}
        >
          <ChevronsLeft className="size-4" aria-hidden />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          isDisabled={disabled || page <= 1}
          aria-label={t('back')}
          onPress={() => go(page - 1)}
        >
          <ChevronLeft className="size-4" aria-hidden />
        </Button>

        <div className="hidden items-center gap-0.5 sm:flex">
          {pageWindow(page, pages).map((item, index) =>
            item === null ? (
              <span key={`gap-${index}`} className="px-1 text-xs text-muted" aria-hidden>
                …
              </span>
            ) : (
              <Button
                key={item}
                size="sm"
                variant={item === page ? 'primary' : 'ghost'}
                className="min-w-8 px-0 tabular-nums"
                isDisabled={disabled}
                aria-current={item === page ? 'page' : undefined}
                aria-label={t('pagerGoTo', { page: item })}
                onPress={() => go(item)}
              >
                {item}
              </Button>
            ),
          )}
        </div>
        <span className="px-1 text-xs tabular-nums text-muted sm:hidden">
          {page} / {pages}
        </span>

        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          isDisabled={disabled || page >= pages}
          aria-label={t('nextStep')}
          onPress={() => go(page + 1)}
        >
          <ChevronRight className="size-4" aria-hidden />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          className="hidden sm:inline-flex"
          isDisabled={disabled || page >= pages}
          aria-label={t('pagerLast')}
          onPress={() => go(pages)}
        >
          <ChevronsRight className="size-4" aria-hidden />
        </Button>
      </div>

      {onPageSizeChange && (
        <div className="flex shrink-0 items-center gap-1.5 text-xs text-muted">
          <span className="whitespace-nowrap">{t('perPage')}</span>
          <SelectField
            className="h-9 w-16 sm:w-[68px]"
            ariaLabel={t('perPage')}
            value={String(pageSize)}
            options={(pageSizeOptions ?? PAGE_SIZE_OPTIONS).map((size) => ({
              value: String(size),
              label: String(size),
            }))}
            onChange={(next) => onPageSizeChange(Number(next))}
            isDisabled={disabled}
          />
        </div>
      )}
    </div>
  )
}
