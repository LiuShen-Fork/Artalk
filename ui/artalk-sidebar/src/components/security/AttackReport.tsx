import { Button, Spinner, toast } from '@heroui/react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ShieldBan } from 'lucide-react'
import { useState } from 'react'
import { Section } from '@/components/security/Section'
import { EmptyState } from '@/components/ui/EmptyState'
import { errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import { kindLabel, securityApi } from '@/lib/security'
import { cn, formatNumber } from '@/lib/utils'

const RANGES = [1, 7, 30]

export function AttackReport() {
  const queryClient = useQueryClient()
  const [days, setDays] = useState(7)
  const reportQuery = useQuery({
    queryKey: ['security-report', days],
    queryFn: () => securityApi.getReport(days),
  })
  const report = reportQuery.data

  const banIP = async (ip: string) => {
    try {
      await securityApi.addBan({ ip, reason: 'report', note: '', duration: 86400 })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['security-bans'] }),
        queryClient.invalidateQueries({ queryKey: ['security-config'] }),
      ])
      toast.success(t('securityBanAdd'))
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    }
  }

  const maxCount = (items: { count: number }[]) =>
    items.reduce((max, item) => Math.max(max, item.count), 0) || 1

  return (
    <Section
      title={t('securityReport')}
      description={t('securityReportDesc')}
      actions={
        <div className="flex items-center gap-1">
          {RANGES.map((range) => (
            <button
              key={range}
              type="button"
              onClick={() => setDays(range)}
              className={cn(
                'h-6 rounded-control border px-2 text-[11px] transition-colors',
                days === range
                  ? 'border-accent/40 bg-accent/10 text-accent'
                  : 'border-border text-muted hover:text-foreground',
              )}
            >
              {t(`securityRange_${range}` as any)}
            </button>
          ))}
        </div>
      }
    >
      {reportQuery.isLoading ? (
        <div className="flex justify-center py-6">
          <Spinner size="sm" />
        </div>
      ) : report ? (
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-2">
            <div className="rounded-lg bg-default/50 px-2.5 py-2">
              <p className="text-[11px] text-muted">{t('securityReportTotal')}</p>
              <p className="text-[17px] font-semibold tabular-nums">{formatNumber(report.total)}</p>
            </div>
            <div className="rounded-lg bg-danger/10 px-2.5 py-2">
              <p className="text-[11px] text-muted">{t('securityReportBlocked')}</p>
              <p className="text-[17px] font-semibold tabular-nums text-danger">
                {formatNumber(report.blocked)}
              </p>
            </div>
          </div>

          <ReportList
            title={t('securityByKind')}
            items={report.by_kind}
            max={maxCount(report.by_kind)}
            label={(item) => kindLabel(item.kind)}
          />
          <ReportList
            title={t('securityByDay')}
            items={report.by_day}
            max={maxCount(report.by_day)}
            label={(item) => item.date}
          />

          <div className="space-y-1">
            <h3 className="text-[12px] font-medium text-muted">{t('securityTopIPs')}</h3>
            {report.top_ips.length ? (
              report.top_ips.map((item) => (
                <div
                  key={item.ip}
                  className="flex items-center justify-between gap-2 rounded-lg border border-border/70 px-2 py-1"
                >
                  <span className="min-w-0 truncate font-mono text-[11.5px]">
                    {item.ip}
                    {item.ip_region ? (
                      <span className="ml-1 font-sans text-[10.5px] text-muted">
                        {item.ip_region}
                      </span>
                    ) : null}
                  </span>
                  <span className="flex shrink-0 items-center gap-1.5">
                    <span className="text-[11px] tabular-nums text-muted">{item.count}</span>
                    <Button
                      size="sm"
                      variant="ghost"
                      isIconOnly
                      aria-label={t('securityBan')}
                      onPress={() => void banIP(item.ip)}
                    >
                      <ShieldBan className="size-3.5 text-danger" aria-hidden />
                    </Button>
                  </span>
                </div>
              ))
            ) : (
              <EmptyState icon={ShieldBan} title={t('noContent')} />
            )}
          </div>

          <ReportList
            title={t('securityTopPaths')}
            items={report.top_paths.map((item) => ({ ...item, path: item.path || '-' }))}
            max={maxCount(report.top_paths)}
            label={(item) => item.path}
          />
        </div>
      ) : (
        <p className="text-[12px] text-danger">{errorMessage(reportQuery.error)}</p>
      )}
    </Section>
  )
}

function ReportList<T extends { count: number }>({
  title,
  items,
  max,
  label,
}: {
  title: string
  items: T[]
  max: number
  label: (item: T) => string
}) {
  return (
    <div className="space-y-1">
      <h3 className="text-[12px] font-medium text-muted">{title}</h3>
      {items.length ? (
        items.slice(0, 8).map((item, index) => (
          <div key={index} className="relative overflow-hidden rounded-md bg-default/40">
            <div
              className="absolute inset-y-0 left-0 bg-accent/15"
              style={{ width: `${Math.max(4, (item.count / max) * 100)}%` }}
              aria-hidden
            />
            <div className="relative flex items-center justify-between gap-2 px-2 py-1">
              <span className="min-w-0 truncate text-[11.5px]">{label(item)}</span>
              <span className="shrink-0 text-[11px] tabular-nums text-muted">{item.count}</span>
            </div>
          </div>
        ))
      ) : (
        <p className="text-[11.5px] text-muted">{t('noContent')}</p>
      )}
    </div>
  )
}
