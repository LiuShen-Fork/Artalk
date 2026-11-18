import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { RefreshCw, TrendingUp } from 'lucide-react'
import { PageHeader } from '@/components/ui/PageHeader'
import { EmptyState } from '@/components/ui/EmptyState'
import { api, siteQuery } from '@/lib/api'
import { t } from '@/lib/i18n'
import { formatNumber } from '@/lib/utils'
import { useSession } from '@/lib/session'

interface Metric {
  total: number
  new_90d?: number
  today?: number
}

interface TrendPoint {
  date: string
  comments: number
  users: number
}

interface TopPage {
  id: number
  key: string
  title: string
  site_name: string
  pv: number
  comment_count: number
}

interface Review {
  id: number
  checker: string
  status: 'pass' | 'block' | 'error'
  action: string
  message: string
  date: string
  user_name: string
  comment_content: string
}

interface DashboardData {
  pv: Metric
  comments: Metric
  users: Metric
  pending_comments: number
  pages: number
  trend: TrendPoint[]
  moderation: { pass: number; block: number; error: number }
  top_pages: TopPage[]
  recent_reviews: Review[]
}

function Stat({
  label,
  value,
  meta,
  tone,
}: {
  label: string
  value: string
  meta?: string
  tone?: 'warning'
}) {
  return (
    <div className="rounded-xl border border-border bg-surface p-3 md:p-4">
      <p className="text-[11px] font-medium uppercase tracking-wide text-muted">{label}</p>
      <p
        className={
          tone === 'warning'
            ? 'mt-1 text-[22px] font-semibold leading-none tabular-nums text-warning md:text-[26px]'
            : 'mt-1 text-[22px] font-semibold leading-none tabular-nums md:text-[26px]'
        }
      >
        {value}
      </p>
      {meta ? <p className="mt-1.5 text-[11px] leading-snug text-muted">{meta}</p> : null}
    </div>
  )
}

function Panel({
  title,
  description,
  action,
  children,
}: {
  title: string
  description?: string
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <section className="rounded-xl border border-border bg-surface">
      <header className="flex items-start justify-between gap-2 px-3.5 pb-2 pt-3 md:px-5 md:pb-2.5 md:pt-4">
        <div>
          <h2 className="text-[13.5px] font-semibold md:text-[14.5px]">{title}</h2>
          {description ? <p className="mt-0.5 text-[11.5px] text-muted">{description}</p> : null}
        </div>
        {action}
      </header>
      <div className="px-3.5 pb-3.5 md:px-5 md:pb-4.5">{children}</div>
    </section>
  )
}

export function DashboardPage() {
  const { site } = useSession()
  const navigate = useNavigate()
  const { data, isPending, isFetching, refetch } = useQuery({
    queryKey: ['dashboard', site],
    queryFn: () => api<DashboardData>('/dashboard', { query: siteQuery() }),
  })

  const trend = data?.trend ?? []
  const weeks: { label: string; comments: number; users: number; range: string }[] = []
  for (let end = trend.length; end > 0; end -= 7) {
    const slice = trend.slice(Math.max(0, end - 7), end)
    weeks.unshift({
      label: slice[0].date.slice(5),
      comments: slice.reduce((sum, item) => sum + item.comments, 0),
      users: slice.reduce((sum, item) => sum + item.users, 0),
      range: `${slice[0].date} — ${slice[slice.length - 1].date}`,
    })
  }
  const peak = Math.max(1, ...weeks.map((week) => Math.max(week.comments, week.users)))
  const bar = (value: number) => (value > 0 ? `${Math.max(4, (value / peak) * 100)}%` : '2px')

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('navDashboard')}
        description={t('dashDesc')}
        actions={
          <Button
            size="sm"
            variant="ghost"
            isIconOnly
            aria-label={t('refresh')}
            isDisabled={isFetching}
            onPress={() => void refetch()}
          >
            <RefreshCw className={isFetching ? 'size-4 animate-spin' : 'size-4'} aria-hidden />
          </Button>
        }
      />

      {isPending ? (
        <div className="grid grid-cols-2 gap-2.5 md:grid-cols-4 md:gap-3">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-[74px] rounded-xl" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-2.5 md:grid-cols-4 md:gap-3">
          <Stat label={t('dashPv')} value={formatNumber(data?.pv.total)} meta={t('dashPvMeta')} />
          <Stat
            label={t('comment')}
            value={formatNumber(data?.comments.total)}
            meta={t('dashDeltaMeta', {
              today: formatNumber(data?.comments.today),
              recent: formatNumber(data?.comments.new_90d),
            })}
          />
          <Stat
            label={t('user')}
            value={formatNumber(data?.users.total)}
            meta={t('dashDeltaMeta', {
              today: formatNumber(data?.users.today),
              recent: formatNumber(data?.users.new_90d),
            })}
          />
          <Stat
            label={t('pending')}
            value={formatNumber(data?.pending_comments)}
            meta={(data?.pending_comments ?? 0) > 0 ? t('dashPendingAlert') : t('dashNoBacklog')}
            tone={(data?.pending_comments ?? 0) > 0 ? 'warning' : undefined}
          />
        </div>
      )}

      <div className="space-y-3 md:grid md:grid-cols-[1.35fr_1fr] md:items-start md:gap-3 md:space-y-0">
        <Panel
          title={t('dashTrend')}
          description={t('dashTrendDesc')}
          action={<TrendingUp className="size-4 text-muted" aria-hidden />}
        >
          {weeks.length ? (
            <>
              <div className="flex h-28 items-end gap-1 md:h-36 md:gap-1.5">
                {weeks.map((week) => (
                  <div
                    key={week.range}
                    title={`${week.range} · ${t('comment')} ${week.comments} · ${t('user')} ${week.users}`}
                    className="flex h-full flex-1 items-end justify-center gap-0.5"
                  >
                    <span
                      className="w-1/3 rounded-t-[3px] bg-accent"
                      style={{ height: bar(week.comments), opacity: week.comments ? 1 : 0.15 }}
                    />
                    <span
                      className="w-1/3 rounded-t-[3px] bg-accent/30"
                      style={{ height: bar(week.users), opacity: week.users ? 1 : 0.15 }}
                    />
                  </div>
                ))}
              </div>
              {/* First, middle and last week: a legible axis at 430px without
                crowding the bars. */}
              <div className="mt-1.5 flex justify-between text-[10px] text-muted">
                {[weeks[0], weeks[Math.floor((weeks.length - 1) / 2)], weeks[weeks.length - 1]].map(
                  (week, index) => (
                    <span key={`${week.range}-${index}`}>{week.label}</span>
                  ),
                )}
              </div>
              <div className="mt-2 flex items-center gap-3 text-[11px] text-muted">
                <span className="flex items-center gap-1">
                  <i className="size-2 rounded-sm bg-accent" /> {t('comment')}
                </span>
                <span className="flex items-center gap-1">
                  <i className="size-2 rounded-sm bg-accent/30" /> {t('user')}
                </span>
                <span className="ml-auto tabular-nums">{t('dashPeak', { value: peak })}</span>
              </div>
            </>
          ) : (
            <p className="py-6 text-center text-[12px] text-muted">{t('noContent')}</p>
          )}
        </Panel>

        <Panel
          title={t('moderation')}
          description={t('moderationIntro')}
          action={
            <Button size="sm" variant="ghost" onPress={() => void navigate({ to: '/moderation' })}>
              {t('all')}
            </Button>
          }
        >
          <div className="grid grid-cols-3 gap-2">
            {(
              [
                ['pass', data?.moderation.pass, 'text-success', 'replace'],
                ['block', data?.moderation.block, 'text-warning', 'block'],
                ['error', data?.moderation.error, 'text-danger', 'error'],
              ] as const
            ).map(([id, value, color, status]) => (
              <button
                key={id}
                type="button"
                onClick={() => void navigate({ to: '/moderation', search: { status } as never })}
                className="rounded-lg border border-border bg-surface-secondary/60 px-2 py-2 text-center transition-colors hover:bg-surface-secondary"
              >
                <b className={`block text-base tabular-nums ${color}`}>{formatNumber(value)}</b>
                <span className="text-[10.5px] text-muted">
                  {t(
                    id === 'pass'
                      ? 'moderationReplaced'
                      : id === 'block'
                        ? 'moderationBlocked'
                        : 'opFailed',
                  )}
                </span>
              </button>
            ))}
          </div>
          <div className="mt-2.5 space-y-1.5">
            {(data?.recent_reviews ?? []).slice(0, 4).map((item) => (
              <div key={item.id} className="flex gap-2 rounded-lg px-1 py-1">
                <span
                  className={`mt-1 size-1.5 shrink-0 rounded-full ${
                    item.status === 'pass'
                      ? 'bg-success'
                      : item.status === 'block'
                        ? 'bg-warning'
                        : 'bg-danger'
                  }`}
                />
                <div className="min-w-0">
                  <p className="truncate text-[12px] font-medium">{item.checker}</p>
                  <p className="line-clamp-2 text-[11.5px] leading-snug text-muted">
                    {item.message || item.comment_content || t('moreDetails')}
                  </p>
                </div>
              </div>
            ))}
            {!data?.recent_reviews?.length ? (
              <p className="py-3 text-center text-[12px] text-muted">{t('noModerationLogs')}</p>
            ) : null}
          </div>
        </Panel>
      </div>

      <Panel title={t('dashTopPages')} description={t('dashTopPagesDesc')}>
        {(data?.top_pages ?? []).length ? (
          <div className="divide-y divide-border">
            {data!.top_pages.map((page, index) => (
              <div key={page.id} className="flex items-center gap-2.5 py-2">
                <span className="w-5 text-[11px] tabular-nums text-muted">
                  {String(index + 1).padStart(2, '0')}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[12.5px] font-medium">{page.title || page.key}</p>
                  <p className="truncate text-[11px] text-muted">{page.key}</p>
                </div>
                <div className="shrink-0 text-right">
                  <p className="text-[12px] font-semibold tabular-nums">
                    {formatNumber(page.pv)}
                    <span className="ml-0.5 text-[10px] font-normal text-muted">PV</span>
                  </p>
                  <p className="text-[10.5px] text-muted tabular-nums">
                    {page.comment_count} {t('comment')}
                  </p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState icon={TrendingUp} title={t('dashNoPages')} />
        )}
      </Panel>
    </div>
  )
}
