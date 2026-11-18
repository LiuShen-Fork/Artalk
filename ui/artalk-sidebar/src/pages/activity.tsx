import type { ArtalkType } from 'artalk'
import { Button, Skeleton } from '@heroui/react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ArrowUpRight, CalendarDays, Flame, MessageSquare, RefreshCw, ThumbsUp } from 'lucide-react'
import { EmptyState } from '@/components/ui/EmptyState'
import { PageHeader } from '@/components/ui/PageHeader'
import { UserAvatar } from '@/components/ui/UserAvatar'
import { api } from '@/lib/api'
import { avatarUrl } from '@/lib/artalk'
import { buildWeeks, heatDayCount, isoDay, startOfToday } from '@/lib/heatmap'
import { getLocale, t } from '@/lib/i18n'
import { useSession } from '@/lib/session'
import { cn, formatDate, formatNumber, timeAgo, truncate } from '@/lib/utils'
import { hoverTitle } from '@/lib/hover-title'

/** Whole weeks only: 26 columns × 7 days keeps the grid a tidy rectangle. */
const HEAT_WEEKS = 26
/** The grid starts on a Sunday, which can be up to six days before day 182. */
const HEAT_QUERY_DAYS = heatDayCount(HEAT_WEEKS) + 6
const RECENT_LIMIT = 6

interface ActivityDay {
  date: string
  count: number
}

interface ActivityResponse {
  count: number
  pending: number
  votes: number
  first_date: string
  by_day: ActivityDay[]
  recent: ArtalkType.CommentData[]
}

function currentStreak(byDay: ActivityDay[]): number {
  const counts = new Map(byDay.map((item) => [item.date, item.count]))
  const cursor = startOfToday()
  // A streak stays alive until today ends, so a quiet morning is not a reset.
  if (!counts.get(isoDay(cursor))) cursor.setDate(cursor.getDate() - 1)
  let streak = 0
  while (counts.get(isoDay(cursor))) {
    streak += 1
    cursor.setDate(cursor.getDate() - 1)
  }
  return streak
}

const LEVELS = ['bg-foreground/[0.07]', 'bg-accent/30', 'bg-accent/55', 'bg-accent/80', 'bg-accent']

function level(count: number): number {
  if (count <= 0) return 0
  if (count === 1) return 1
  if (count <= 3) return 2
  if (count <= 6) return 3
  return 4
}

function Stat({
  icon: Icon,
  label,
  value,
  meta,
}: {
  icon: typeof Flame
  label: string
  value: string
  meta?: string
}) {
  return (
    <div className="rounded-xl border border-border bg-surface p-3">
      <p className="flex items-center gap-1.5 text-[11px] font-medium text-muted">
        <Icon className="size-3.5 shrink-0" aria-hidden />
        <span className="truncate">{label}</span>
      </p>
      <p className="mt-1 text-[21px] font-semibold leading-none tabular-nums">{value}</p>
      {meta ? (
        <p className="mt-1.5 truncate text-[10.5px] leading-snug text-muted">{meta}</p>
      ) : null}
    </div>
  )
}

function Heatmap({ byDay }: { byDay: ActivityDay[] }) {
  const weeks = buildWeeks(byDay, HEAT_WEEKS)
  const weekdayFormat = new Intl.DateTimeFormat(getLocale(), { weekday: 'narrow' })
  const monthFormat = new Intl.DateTimeFormat(getLocale(), { month: 'short' })

  // Row labels sit next to Monday, Wednesday and Friday only: enough to read
  // the grid, not enough to crowd it. 2024-01-07 is a Sunday, so 7 + row lands
  // on the right weekday every time.
  const weekdayLabels = Array.from({ length: 7 }, (_, row) =>
    row === 1 || row === 3 || row === 5 ? weekdayFormat.format(new Date(2024, 0, 7 + row)) : '',
  )

  const months: { index: number; label: string }[] = []
  weeks.forEach((week, index) => {
    const first = new Date(`${week[0].date}T00:00:00`)
    const previous = index > 0 ? new Date(`${weeks[index - 1][0].date}T00:00:00`) : null
    if (!previous || previous.getMonth() !== first.getMonth()) {
      if (months.length === 0 || index - months[months.length - 1].index >= 2) {
        months.push({ index, label: monthFormat.format(first) })
      }
    }
  })

  const total = byDay.reduce((sum, item) => sum + item.count, 0)
  const activeDays = byDay.filter((item) => item.count > 0).length

  return (
    <section className="rounded-xl border border-border bg-surface">
      <header className="flex items-start justify-between gap-2 px-3.5 pb-2.5 pt-3">
        <div className="min-w-0">
          <h2 className="text-[13.5px] font-semibold">{t('activityHeatmap')}</h2>
          <p className="mt-0.5 text-[11.5px] leading-snug text-muted">{t('activityHeatmapDesc')}</p>
        </div>
        <span className="shrink-0 text-[11px] tabular-nums text-muted">
          {t('activityActiveDays', { count: activeDays })}
        </span>
      </header>

      {/* The grid is fully fluid: every week is an equal fraction of the card,
          so the squares stay square, the columns stretch to whatever width the
          card has, and the grid can never spill out of it. */}
      <div className="px-3.5 pb-3">
        <div>
          <div
            className="mb-1 grid gap-[3px]"
            style={{ gridTemplateColumns: `12px repeat(${weeks.length}, minmax(0, 1fr))` }}
          >
            <span aria-hidden />
            {months.map((month) => (
              <span
                key={`${month.index}-${month.label}`}
                className={cn(
                  'whitespace-nowrap text-[9px] leading-[11px] text-muted',
                  // The last label grows leftwards from the edge instead of
                  // running past the card.
                  month.index > weeks.length - 3 && 'justify-self-end',
                )}
                style={{ gridColumnStart: month.index + 2 }}
              >
                {month.label}
              </span>
            ))}
          </div>

          <div
            className="grid grid-flow-col gap-[3px]"
            style={{
              gridTemplateColumns: `12px repeat(${weeks.length}, minmax(0, 1fr))`,
              gridTemplateRows: 'repeat(7, auto)',
            }}
          >
            {weekdayLabels.map((label, row) => (
              <span
                key={`label-${row}`}
                className="flex items-center text-[9px] leading-none text-muted"
              >
                {label}
              </span>
            ))}

            {weeks.map((week) =>
              week.map((cell) => (
                <span
                  key={cell.date}
                  title={
                    cell.future
                      ? undefined
                      : `${cell.date} · ${t('activityComments', { count: cell.count })}`
                  }
                  className={cn(
                    'aspect-square w-full rounded-[2px]',
                    cell.future
                      ? // Days that have not happened yet keep the rectangle
                        // whole: a dashed outline reads as "not yet", a blank
                        // hole would make the last column look broken.
                        'border border-dashed border-border/70'
                      : LEVELS[level(cell.count)],
                    cell.date === isoDay(startOfToday()) && 'ring-1 ring-inset ring-accent/60',
                  )}
                />
              )),
            )}
          </div>
        </div>

        <div className="mt-2 flex items-center justify-between text-[10px] text-muted">
          <span className="truncate">
            {formatDate(weeks[0]?.[0]?.date)} — {t('activityToday')}
          </span>
          <span className="flex shrink-0 items-center gap-1">
            {t('activityLess')}
            {LEVELS.map((tone) => (
              <i key={tone} className={cn('size-2.5 rounded-[2px]', tone)} />
            ))}
            {t('activityMore')}
          </span>
        </div>

        <p className="mt-1.5 text-[10.5px] text-muted">
          {t('activityRangeTotal', { count: total, days: heatDayCount(HEAT_WEEKS) })}
        </p>
      </div>
    </section>
  )
}

function WeekdayBars({ byDay }: { byDay: ActivityDay[] }) {
  const totals = new Array<number>(7).fill(0)
  for (const item of byDay) {
    const date = new Date(`${item.date}T00:00:00`)
    totals[date.getDay()] += item.count
  }

  const busiest = Math.max(1, ...totals)
  const weekdays = [1, 2, 3, 4, 5, 6, 0]
  const format = new Intl.DateTimeFormat(getLocale(), { weekday: 'short' })

  return (
    <section className="rounded-xl border border-border bg-surface">
      <header className="px-3.5 pb-1.5 pt-3">
        <h2 className="text-[13.5px] font-semibold">{t('activityWeekday')}</h2>
        <p className="mt-0.5 text-[11.5px] leading-snug text-muted">{t('activityWeekdayDesc')}</p>
      </header>

      <div className="flex items-end gap-1.5 px-3.5 pb-3">
        {weekdays.map((weekday) => {
          const value = totals[weekday]
          const label = format.format(new Date(2024, 0, 7 + weekday))
          return (
            <div key={weekday} className="flex min-w-0 flex-1 flex-col items-center gap-1">
              <span className="text-[10px] leading-none tabular-nums text-muted">
                {value || ''}
              </span>
              <div className="flex h-14 w-full items-end">
                <div
                  className={cn(
                    'w-full rounded-[4px] transition-[height]',
                    value ? 'bg-accent/70' : 'bg-foreground/[0.07]',
                  )}
                  style={{ height: Math.max(4, Math.round((value / busiest) * 56)) }}
                  title={`${label} · ${t('activityComments', { count: value })}`}
                />
              </div>
              <span className="text-[10px] leading-none text-muted">{label}</span>
            </div>
          )
        })}
      </div>
    </section>
  )
}

function RecentList({ items }: { items: ArtalkType.CommentData[] }) {
  const navigate = useNavigate()

  return (
    <section className="rounded-xl border border-border bg-surface">
      <header className="flex items-center justify-between gap-2 px-3.5 pb-2 pt-3">
        <div className="min-w-0">
          <h2 className="text-[13.5px] font-semibold">{t('activityRecent')}</h2>
          <p className="mt-0.5 text-[11.5px] text-muted">{t('activityRecentDesc')}</p>
        </div>
        <Button size="sm" variant="ghost" onPress={() => void navigate({ to: '/comments' })}>
          {t('all')}
        </Button>
      </header>

      {items.length ? (
        <ul className="divide-y divide-border px-3.5 pb-2">
          {items.map((item) => (
            <li key={item.id} className="py-2.5">
              <div className="flex items-baseline justify-between gap-2">
                <p className="min-w-0 truncate text-[12px] font-medium">
                  {truncate(item.page_key || '—', 48)}
                </p>
                <span className="shrink-0 text-[10.5px] tabular-nums text-muted">
                  {timeAgo(item.date)}
                </span>
              </div>
              <p className="mt-1 line-clamp-2 text-[12px] leading-snug text-muted">
                {item.content}
              </p>
              <div className="mt-1.5 flex items-center gap-2 text-[10.5px] text-muted">
                {item.site_name ? (
                  <span className="max-w-[45%] truncate rounded-[4px] bg-surface-secondary px-1.5 py-0.5">
                    {item.site_name}
                  </span>
                ) : null}
                {item.is_pending ? (
                  <span className="rounded-[4px] bg-warning/15 px-1.5 py-0.5 text-warning">
                    {t('pending')}
                  </span>
                ) : null}
                {item.page_url ? (
                  <a
                    href={item.page_url}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="ms-auto inline-flex shrink-0 items-center gap-0.5 text-accent hover:underline"
                  >
                    {t('openPage')}
                    <ArrowUpRight className="size-3" aria-hidden />
                  </a>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <div className="px-3.5 pb-3.5">
          <EmptyState
            icon={MessageSquare}
            title={t('activityEmptyTitle')}
            description={t('activityEmptyDesc')}
          />
        </div>
      )}
    </section>
  )
}

export function ActivityPage() {
  const { user } = useSession()
  const { data, isPending, isFetching, refetch } = useQuery({
    queryKey: ['activity', user?.email],
    queryFn: () =>
      api<ActivityResponse>('/user/activity', {
        query: { days: HEAT_QUERY_DAYS, limit: RECENT_LIMIT },
      }),
  })

  const byDay = data?.by_day ?? []
  const cutoff = startOfToday()
  cutoff.setDate(cutoff.getDate() - 29)
  const recent30 = byDay
    .filter((item) => item.date >= isoDay(cutoff))
    .reduce((sum, item) => sum + item.count, 0)
  const streak = currentStreak(byDay)

  return (
    <div className="space-y-4 md:space-y-5">
      <PageHeader
        title={t('navMyActivity')}
        description={t('myActivityDesc')}
        actions={
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
        }
      />

      <div className="flex items-center gap-3 rounded-xl border border-border bg-surface p-3">
        <UserAvatar name={user?.name} src={avatarUrl(user?.email ?? '')} size="lg" />
        <div className="min-w-0 flex-1">
          <p className="truncate text-[14px] font-semibold">{user?.name || t('unknownUser')}</p>
          <p className="truncate text-[11.5px] text-muted">{user?.email}</p>
        </div>
        {data?.first_date ? (
          <span className="flex shrink-0 items-center gap-1 text-[10.5px] text-muted">
            <CalendarDays className="size-3.5" aria-hidden />
            {t('activitySince', { date: data.first_date })}
          </span>
        ) : null}
      </div>

      <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_minmax(0,380px)] xl:items-start">
        <div className="space-y-3">
          {isPending ? (
            <div className="grid grid-cols-2 gap-2.5 md:grid-cols-4">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-[76px] rounded-xl" />
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2.5 md:grid-cols-4">
              <Stat
                icon={MessageSquare}
                label={t('activityTotal')}
                value={formatNumber(data?.count)}
                meta={
                  data?.pending ? t('activityPendingCount', { count: data.pending }) : undefined
                }
              />
              <Stat
                icon={CalendarDays}
                label={t('activityRecent30')}
                value={formatNumber(recent30)}
              />
              <Stat icon={ThumbsUp} label={t('activityVotes')} value={formatNumber(data?.votes)} />
              <Stat
                icon={Flame}
                label={t('activityStreak')}
                value={t('activityDays', { count: streak })}
              />
            </div>
          )}

          <Heatmap byDay={byDay} />
          <WeekdayBars byDay={byDay} />
        </div>

        <RecentList items={data?.recent ?? []} />
      </div>
    </div>
  )
}
