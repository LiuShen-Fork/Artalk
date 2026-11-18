import { Button, Chip, Skeleton, toast } from '@heroui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Check, RefreshCw, ShieldCheck, Trash2, TriangleAlert, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { EmptyState } from '@/components/ui/EmptyState'
import { Pager } from '@/components/ui/Pager'
import { Segmented } from '@/components/ui/Segmented'
import { PageHeader } from '@/components/ui/PageHeader'
import { useConfirm } from '@/components/ui/confirm'
import { api, siteQuery, errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import { getArtalk } from '@/lib/artalk'
import { useSession } from '@/lib/session'
import { usePageSize } from '@/lib/use-page-size'
import { hoverTitle } from '@/lib/hover-title'

type ModerationStatus = 'pass' | 'block' | 'error'
export type ModerationTab = 'all' | 'block' | 'replace' | 'error'

interface ModerationLog {
  id: number
  comment_id: number
  site_name: string
  page_key: string
  checker: string
  status: ModerationStatus
  action: string
  message: string
  date: string
  comment_content: string
  comment_available: boolean
  comment_rid: number
  comment_ua: string
  comment_ip: string
  comment_pending: boolean
  comment_collapsed: boolean
  comment_pinned: boolean
  user_name: string
  user_email: string
  user_link: string
}

interface ModerationResponse {
  count: number
  logs: ModerationLog[]
}

const displayStatus = (item: ModerationLog): Exclude<ModerationTab, 'all'> =>
  item.action === 'replace' || item.status === 'pass' ? 'replace' : item.status

const STATUS_META = {
  replace: { icon: ShieldCheck, tone: 'text-success', chip: 'success' },
  block: { icon: TriangleAlert, tone: 'text-warning', chip: 'warning' },
  error: { icon: X, tone: 'text-danger', chip: 'danger' },
} as const

export function ModerationPage({ initialStatus }: { initialStatus?: ModerationTab }) {
  const { site } = useSession()
  const confirm = useConfirm()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [tab, setTab] = useState<ModerationTab>(initialStatus ?? 'all')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = usePageSize('moderation', 20)
  const changePageSize = (next: number) => {
    setPageSize(next)
    setPage(1)
  }
  const [busy, setBusy] = useState<string | null>(null)

  // The tab is addressable so the dashboard can deep-link into a status filter.
  useEffect(() => {
    if (!initialStatus) return
    setTab(initialStatus)
    setPage(1)
  }, [initialStatus])

  const queryKey = ['moderation', site, tab, page, pageSize]
  const { data, isPending, isFetching, refetch } = useQuery({
    queryKey,
    queryFn: () =>
      api<ModerationResponse>('/moderation/logs', {
        query: {
          ...siteQuery(),
          status: tab === 'all' ? undefined : tab,
          limit: pageSize,
          offset: (page - 1) * pageSize,
        },
      }),
  })

  const logs = data?.logs ?? []
  const total = data?.count ?? 0

  const refresh = () => void queryClient.invalidateQueries({ queryKey })

  const removeLog = useMutation({
    mutationFn: (id: number) => api(`/moderation/logs/${id}`, { method: 'DELETE' }),
    onSuccess: refresh,
    onError: (error) => toast.danger(errorMessage(error)),
  })

  const clearLogs = useMutation({
    mutationFn: () => api('/moderation/logs', { method: 'DELETE', query: siteQuery() }),
    onSuccess: () => {
      setPage(1)
      refresh()
    },
    onError: (error) => toast.danger(errorMessage(error)),
  })

  const commentPayload = (item: ModerationLog, isPending: boolean) => ({
    site_name: item.site_name,
    content: item.comment_content,
    page_key: item.page_key,
    nick: item.user_name,
    email: item.user_email,
    link: item.user_link,
    rid: item.comment_rid,
    ua: item.comment_ua,
    ip: item.comment_ip,
    is_collapsed: item.comment_collapsed,
    is_pending: isPending,
    is_pinned: item.comment_pinned,
  })

  const runAction = async (key: string, action: () => Promise<unknown>) => {
    setBusy(key)
    try {
      await action()
      refresh()
    } catch (error) {
      toast.danger(errorMessage(error))
    } finally {
      setBusy(null)
    }
  }

  const statusLabel = (status: Exclude<ModerationTab, 'all'>) =>
    status === 'replace'
      ? t('moderationReplaced')
      : status === 'block'
        ? t('moderationBlocked')
        : t('opFailed')

  return (
    <div className="space-y-4 md:space-y-5">
      <PageHeader
        title={t('navModeration')}
        description={t('moderationIntro')}
        actions={
          <>
            <Button
              size="sm"
              variant="ghost"
              isIconOnly
              aria-label={t('refresh')}
              {...hoverTitle(t('refresh'))}
              isDisabled={isFetching}
              onPress={() => void refetch()}
            >
              <RefreshCw className={isFetching ? 'size-4 animate-spin' : 'size-4'} aria-hidden />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              isIconOnly
              aria-label={t('moderationClear')}
              {...hoverTitle(t('moderationClear'))}
              isDisabled={clearLogs.isPending || !total}
              onPress={async () => {
                if (await confirm({ title: t('moderationClearConfirm'), danger: true })) {
                  clearLogs.mutate()
                }
              }}
            >
              <Trash2 className="size-4" aria-hidden />
            </Button>
          </>
        }
      />

      <Segmented
        items={(
          [
            ['all', 'all'],
            ['replace', 'moderationReplaced'],
            ['block', 'moderationBlocked'],
            ['error', 'opFailed'],
          ] as const
        ).map(([id, label]) => ({ id, label: t(label) }))}
        value={tab}
        onChange={(next) => {
          setTab(next)
          setPage(1)
          void navigate({ to: '/moderation', search: { status: next }, replace: true })
        }}
      />

      <p className="px-0.5 text-[11.5px] text-muted">
        {t('moderationSummary', { total, count: logs.length })}
      </p>

      {isPending ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-24 rounded-xl" />
          ))}
        </div>
      ) : logs.length ? (
        <div className="space-y-2">
          {logs.map((item) => {
            const status = displayStatus(item)
            const meta = STATUS_META[status]
            const Icon = meta.icon
            const operable = item.comment_available && item.comment_id > 0
            return (
              <article
                key={item.id}
                className="rounded-xl border border-border bg-surface p-3 md:p-4"
              >
                <div className="flex items-start gap-2">
                  <Icon className={`mt-0.5 size-4 shrink-0 ${meta.tone}`} aria-hidden />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className={`text-[12.5px] font-semibold ${meta.tone}`}>
                        {statusLabel(status)}
                      </span>
                      <span className="text-[11.5px] text-muted">{item.checker}</span>
                      <Chip size="sm" variant="soft">
                        {item.action}
                      </Chip>
                    </div>
                    <p className="mt-1 text-[12px] leading-snug text-muted">
                      {item.message || t('moderationNoMessage')}
                    </p>
                  </div>
                </div>

                <blockquote className="mt-2 rounded-lg border-l-2 border-border bg-surface-secondary/60 px-2.5 py-1.5 text-[12px] leading-relaxed break-words">
                  {item.comment_content || (
                    <span className="text-muted">{t('moderationCommentUnavailable')}</span>
                  )}
                </blockquote>

                <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[10.5px] text-muted">
                  {item.comment_id ? <span>#{item.comment_id}</span> : null}
                  <span>{item.user_name || item.user_email || t('unknownUser')}</span>
                  {item.site_name ? <span>· {item.site_name}</span> : null}
                  <span className="ml-auto tabular-nums">{item.date}</span>
                </div>

                <div className="mt-2 flex flex-wrap justify-end gap-1.5">
                  {item.comment_pending ? (
                    <Button
                      size="sm"
                      variant="secondary"
                      isDisabled={!operable || !!busy}
                      onPress={async () => {
                        if (
                          await confirm({
                            title: t('moderationApproveCommentConfirm'),
                          })
                        ) {
                          await runAction(`approve:${item.id}`, () =>
                            getArtalk()
                              .ctx.getApi()
                              .comments.updateComment(item.comment_id, commentPayload(item, false)),
                          )
                        }
                      }}
                    >
                      <Check className="size-3.5" aria-hidden />
                      {t('moderationApproveComment')}
                    </Button>
                  ) : null}
                  <Button
                    size="sm"
                    variant="ghost"
                    isDisabled={!operable || !!busy}
                    onPress={async () => {
                      if (
                        await confirm({ title: t('moderationDeleteCommentConfirm'), danger: true })
                      ) {
                        await runAction(`comment:${item.id}`, () =>
                          getArtalk().ctx.getApi().comments.deleteComment(item.comment_id),
                        )
                      }
                    }}
                  >
                    {t('moderationDeleteComment')}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    isIconOnly
                    aria-label={t('delete')}
                    {...hoverTitle(t('delete'))}
                    isDisabled={removeLog.isPending || !!busy}
                    onPress={() => void removeLog.mutate(item.id)}
                  >
                    <Trash2 className="size-3.5" aria-hidden />
                  </Button>
                </div>
              </article>
            )
          })}
        </div>
      ) : (
        <EmptyState icon={ShieldCheck} title={t('noModerationLogs')} />
      )}

      <Pager
        page={page}
        pageSize={pageSize}
        total={total}
        disabled={isFetching}
        onChange={setPage}
        onPageSizeChange={changePageSize}
      />
    </div>
  )
}
