import { Button, Input, Skeleton, Spinner, toast } from '@heroui/react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ExternalLink, FileText, Flame, RefreshCw, Search, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { EmptyState } from '@/components/ui/EmptyState'
import { Pager } from '@/components/ui/Pager'
import { PageHeader } from '@/components/ui/PageHeader'
import { PageEditorDialog } from '@/components/PageEditorDialog'
import { errorMessage, type PageData } from '@/lib/api'
import { t } from '@/lib/i18n'
import { debounce, formatNumber } from '@/lib/utils'
import { getArtalk } from '@/lib/artalk'
import { useSession } from '@/lib/session'

const PAGE_SIZE = 20

interface PagesResponse {
  count: number
  pages: PageData[]
}

export function PagesPage() {
  const { site } = useSession()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [editing, setEditing] = useState<PageData>()
  const [task, setTask] = useState<{ running: boolean; text: string }>({
    running: false,
    text: '',
  })
  const timer = useRef<number | undefined>(undefined)
  const cancelled = useRef(false)

  const { data, isPending, isFetching } = useQuery({
    queryKey: ['pages', site, page, search],
    queryFn: async ({ signal }) => {
      const res = await getArtalk()
        .ctx.getApi()
        .pages.getPages(
          {
            site_name: site || undefined,
            offset: (page - 1) * PAGE_SIZE,
            limit: PAGE_SIZE,
            search,
            has_comments: true,
          },
          { signal },
        )
      return (res.data ?? { count: 0, pages: [] }) as PagesResponse
    },
  })

  const pages = data?.pages ?? []
  const refresh = () => void queryClient.invalidateQueries({ queryKey: ['pages'] })

  useEffect(() => {
    cancelled.current = false
    return () => {
      cancelled.current = true
      window.clearTimeout(timer.current)
    }
  }, [])

  /** The backend scrapes every page's <title> in the background; poll until it finishes. */
  const watchTask = () => {
    timer.current = window.setTimeout(async () => {
      if (cancelled.current) return
      try {
        const res = await getArtalk().ctx.getApi().pages.getPageFetchStatus()
        if (cancelled.current) return
        if (res.data.is_progress) {
          setTask({ running: true, text: res.data.msg })
          watchTask()
        } else {
          setTask({ running: false, text: t('updateComplete') })
          refresh()
        }
      } catch {
        setTask({ running: false, text: t('opFailed') })
      }
    }, 1000)
  }

  const refreshAllTitles = async () => {
    if (task.running) return
    setTask({ running: true, text: t('updateReady') })
    try {
      await getArtalk()
        .ctx.getApi()
        .pages.fetchAllPages({ site_name: site || undefined })
      watchTask()
    } catch (error) {
      toast.danger(errorMessage(error))
      setTask({ running: false, text: t('opFailed') })
    }
  }

  const cacheAction = async (kind: 'flush' | 'warm') => {
    setTask({ running: true, text: kind === 'flush' ? t('cacheClear') : t('cacheWarm') })
    try {
      const api = getArtalk().ctx.getApi()
      const res = kind === 'flush' ? await api.cache.flushCache() : await api.cache.warmUpCache()
      toast.success(res.data.msg || t('updateComplete'))
    } catch (error) {
      toast.danger(errorMessage(error))
    } finally {
      setTask({ running: false, text: '' })
    }
  }

  const runSearch = debounce((value: string) => {
    setSearch(value)
    setPage(1)
  }, 350)

  return (
    <div className="space-y-3">
      <PageHeader title={t('page')} description={t('pageManageDesc')} />

      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted"
          aria-hidden
        />
        <Input
          placeholder={t('search')}
          aria-label={t('search')}
          fullWidth
          className="bg-surface pl-9"
          onChange={(event) => runSearch(event.target.value)}
        />
      </div>

      <div className="flex items-center gap-1.5">
        <Button
          size="sm"
          variant="secondary"
          className="flex-1"
          isDisabled={task.running}
          onPress={() => void refreshAllTitles()}
        >
          {task.running ? <Spinner size="sm" /> : <RefreshCw className="size-3.5" aria-hidden />}
          <span className="truncate">{task.running ? task.text : t('updateTitle')}</span>
        </Button>
        <Button
          size="sm"
          variant="ghost"
          isDisabled={task.running}
          aria-label={t('cacheClear')}
          onPress={() => void cacheAction('flush')}
        >
          <Trash2 className="size-3.5" aria-hidden />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          isDisabled={task.running}
          aria-label={t('cacheWarm')}
          onPress={() => void cacheAction('warm')}
        >
          <Flame className="size-3.5" aria-hidden />
        </Button>
      </div>

      {isPending ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-14 rounded-xl" />
          ))}
        </div>
      ) : pages.length ? (
        <div className="overflow-hidden rounded-xl border border-border bg-surface">
          {pages.map((item, index) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setEditing(item)}
              className={`flex w-full items-center gap-2.5 px-3 py-2.5 text-left transition-colors hover:bg-surface-secondary/60 ${
                index ? 'border-t border-border' : ''
              }`}
            >
              <FileText className="size-4 shrink-0 text-muted" aria-hidden />
              <div className="min-w-0 flex-1">
                <p className="truncate text-[13px] font-medium">{item.title || item.key}</p>
                <p className="truncate text-[11px] text-muted">{item.url || item.key}</p>
              </div>
              <div className="shrink-0 text-right text-[11px] tabular-nums text-muted">
                <p>{formatNumber(item.comment_count)}</p>
                <p className="text-[10px]">PV {formatNumber(item.pv)}</p>
              </div>
              {item.url ? (
                <a
                  href={item.url}
                  target="_blank"
                  rel="noreferrer noopener"
                  onClick={(event) => event.stopPropagation()}
                  className="shrink-0 p-1 text-muted hover:text-accent"
                  aria-label={t('page')}
                >
                  <ExternalLink className="size-3.5" aria-hidden />
                </a>
              ) : null}
            </button>
          ))}
        </div>
      ) : (
        <EmptyState icon={FileText} title={t('noPages')} />
      )}

      <Pager
        page={page}
        pageSize={PAGE_SIZE}
        total={data?.count ?? 0}
        disabled={isFetching}
        onChange={setPage}
      />

      <PageEditorDialog page={editing} onClose={() => setEditing(undefined)} onSaved={refresh} />
    </div>
  )
}
