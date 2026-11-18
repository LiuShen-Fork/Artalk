import { Button, Input, Spinner, toast } from '@heroui/react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { Search, Trash2, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Section } from '@/components/security/Section'
import { EmptyState } from '@/components/ui/EmptyState'
import { Pager } from '@/components/ui/Pager'
import { SelectField } from '@/components/ui/SelectField'
import { useConfirm } from '@/components/ui/confirm'
import { errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import { kindLabel, securityApi } from '@/lib/security'
import { usePageSize } from '@/lib/use-page-size'
import { cn, formatDateTime } from '@/lib/utils'

const KINDS = ['rate_limit', 'captcha_failed', 'banned', 'auto_ban']

export function EventLog() {
  const queryClient = useQueryClient()
  const confirm = useConfirm()
  const [kind, setKind] = useState('')
  const [days, setDays] = useState('7')
  const [ipInput, setIPInput] = useState('')
  const [ip, setIP] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = usePageSize('security-events', 15)
  const changePageSize = (next: number) => {
    setPageSize(next)
    setPage(1)
  }

  useEffect(() => {
    const timer = setTimeout(() => {
      setIP(ipInput.trim())
      setPage(1)
    }, 350)
    return () => clearTimeout(timer)
  }, [ipInput])

  const eventsQuery = useQuery({
    queryKey: ['security-events', kind, days, ip, page, pageSize],
    queryFn: () =>
      securityApi.getEvents({
        kind: kind || undefined,
        ip: ip || undefined,
        days: Number(days),
        limit: pageSize,
        offset: (page - 1) * pageSize,
      }),
    placeholderData: keepPreviousData,
  })

  const events = eventsQuery.data?.events ?? []
  const total = eventsQuery.data?.count ?? 0

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['security-events'] })
    queryClient.invalidateQueries({ queryKey: ['security-report'] })
    queryClient.invalidateQueries({ queryKey: ['security-config'] })
  }

  const removeEvent = async (id: number) => {
    try {
      await securityApi.deleteEvent(id)
      invalidate()
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    }
  }

  const clearEvents = async () => {
    const ok = await confirm({
      title: t('securityEventClearConfirm'),
      danger: true,
      confirmText: t('securityEventClear'),
    })
    if (!ok) return

    try {
      await securityApi.clearEvents({
        kind: kind || undefined,
        ip: ip || undefined,
        days: Number(days),
      })
      setPage(1)
      invalidate()
      toast.success(t('securityEventClear'))
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    }
  }

  return (
    <Section
      title={t('securityEvents')}
      description={t('securityEventsDesc', { total })}
      actions={
        <Button
          size="sm"
          variant="ghost"
          isIconOnly
          aria-label={t('securityEventClear')}
          isDisabled={!total}
          onPress={() => void clearEvents()}
        >
          <Trash2 className="size-4 text-danger" aria-hidden />
        </Button>
      }
    >
      <div className="space-y-1.5">
        <div className="grid grid-cols-2 gap-1.5">
          <SelectField
            value={kind}
            ariaLabel={t('securityEvents')}
            onChange={(value) => {
              setKind(value)
              setPage(1)
            }}
            options={[
              { value: '', label: t('securityEventKindAll') },
              ...KINDS.map((item) => ({ value: item, label: kindLabel(item) })),
            ]}
          />
          <SelectField
            value={days}
            ariaLabel={t('securityEvents')}
            onChange={(value) => {
              setDays(value)
              setPage(1)
            }}
            options={[1, 7, 30].map((range) => ({
              value: String(range),
              label: t(`securityRange_${range}` as any),
            }))}
          />
        </div>
        <div className="relative">
          <Search
            className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted"
            aria-hidden
          />
          <Input
            value={ipInput}
            placeholder="IP"
            autoComplete="off"
            fullWidth
            className="pl-8"
            onChange={(event) => setIPInput(event.target.value)}
          />
          {ipInput ? (
            <button
              type="button"
              aria-label={t('reset')}
              onClick={() => setIPInput('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted hover:text-foreground"
            >
              <X className="size-3" aria-hidden />
            </button>
          ) : null}
        </div>
      </div>

      <div className="space-y-1.5">
        {eventsQuery.isLoading ? (
          <div className="flex justify-center py-6">
            <Spinner size="sm" />
          </div>
        ) : events.length ? (
          events.map((event) => (
            <article
              key={event.id}
              className="space-y-1 rounded-lg border border-border/70 px-2 py-1.5"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="flex min-w-0 flex-wrap items-center gap-1 text-[11px]">
                  <span className="rounded bg-default px-1.5 py-0.5 font-medium">
                    {kindLabel(event.kind)}
                  </span>
                  {event.action ? <span className="text-muted">{event.action}</span> : null}
                  {event.window ? (
                    <span className="text-muted">{t(`securityWindow_${event.window}` as any)}</span>
                  ) : null}
                  {event.blocked ? (
                    <span className="rounded bg-danger/15 px-1.5 py-0.5 text-danger">
                      {t('securityBlocked')}
                    </span>
                  ) : null}
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  isIconOnly
                  aria-label={t('delete')}
                  onPress={() => void removeEvent(event.id)}
                >
                  <X className="size-3.5 text-muted" aria-hidden />
                </Button>
              </div>
              <p className="truncate font-mono text-[11.5px]">
                {event.ip}
                {event.ip_region ? (
                  <span className="ml-1 font-sans text-[10.5px] text-muted">{event.ip_region}</span>
                ) : null}
              </p>
              <p className="truncate text-[11px] text-muted">
                {[
                  event.method && event.path ? `${event.method} ${event.path}` : event.path,
                  event.site_name,
                  event.user_name,
                ]
                  .filter(Boolean)
                  .join(' · ')}
              </p>
              <p className={cn('text-[10.5px] text-muted/80')}>{formatDateTime(event.date)}</p>
              {event.detail ? (
                <code className="block truncate rounded bg-default/60 px-1.5 py-0.5 text-[10.5px] text-muted">
                  {event.detail}
                </code>
              ) : null}
            </article>
          ))
        ) : (
          <EmptyState icon={Search} title={t('securityNoEvents')} />
        )}
      </div>

      <Pager
        page={page}
        pageSize={pageSize}
        total={total}
        onChange={setPage}
        onPageSizeChange={changePageSize}
        disabled={eventsQuery.isFetching}
      />
    </Section>
  )
}
