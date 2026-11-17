import { Button, Input, Spinner, toast } from '@heroui/react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, ShieldBan } from 'lucide-react'
import { useState } from 'react'
import { Row, Section } from '@/components/security/Section'
import { EmptyState } from '@/components/ui/EmptyState'
import { SelectField } from '@/components/ui/SelectField'
import { useConfirm } from '@/components/ui/confirm'
import { errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import { securityApi } from '@/lib/security'
import { formatDateTime } from '@/lib/utils'

const DURATIONS = [
  { value: '3600', key: 'securityHourUnit', units: 1 },
  { value: '86400', key: 'securityDayUnit', units: 1 },
  { value: '604800', key: 'securityDayUnit', units: 7 },
  { value: '-1', key: 'securityPermanent', units: 0 },
]

export function BanManager() {
  const queryClient = useQueryClient()
  const confirm = useConfirm()
  const [ip, setIp] = useState('')
  const [reason, setReason] = useState('')
  const [duration, setDuration] = useState('86400')
  const [submitting, setSubmitting] = useState(false)

  const bansQuery = useQuery({ queryKey: ['security-bans'], queryFn: securityApi.getBans })
  const bans = bansQuery.data?.bans ?? []

  const addBan = async () => {
    if (!ip.trim() || submitting) return
    setSubmitting(true)
    try {
      await securityApi.addBan({
        ip: ip.trim(),
        reason: reason.trim(),
        note: '',
        duration: Number(duration),
      })
      setIp('')
      setReason('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['security-bans'] }),
        queryClient.invalidateQueries({ queryKey: ['security-config'] }),
      ])
      toast.success(t('securityBanAdd'))
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  const removeBan = async (target: string, id: number) => {
    if (id <= 0) {
      toast.info(t('securityConfigBlacklistHint'))
      return
    }
    const ok = await confirm({
      title: t('securityUnbanConfirm', { ip: target }),
      danger: true,
      confirmText: t('securityUnban'),
    })
    if (!ok) return

    try {
      await securityApi.removeBan(id)
      queryClient.setQueryData<{ count: number; bans: typeof bans }>(['security-bans'], (prev) =>
        prev
          ? {
              ...prev,
              count: Math.max(0, prev.count - 1),
              bans: prev.bans.filter((b) => b.id !== id),
            }
          : prev,
      )
      queryClient.invalidateQueries({ queryKey: ['security-config'] })
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    }
  }

  return (
    <Section title={t('securityBans')} description={t('securityBansDesc')}>
      <div className="space-y-1.5">
        <Row label={t('securityBanIP')}>
          <Input
            fullWidth
            value={ip}
            placeholder="203.0.113.7 / 203.0.113.0/24"
            autoComplete="off"
            onChange={(event) => setIp(event.target.value)}
            onKeyDown={(event) => event.key === 'Enter' && void addBan()}
          />
        </Row>
        <Row label={t('securityBanReason')}>
          <Input
            fullWidth
            value={reason}
            placeholder={t('securityBanReasonPlaceholder')}
            autoComplete="off"
            onChange={(event) => setReason(event.target.value)}
            onKeyDown={(event) => event.key === 'Enter' && void addBan()}
          />
        </Row>
        <div className="grid grid-cols-[1fr_auto] items-end gap-1.5">
          <Row label={t('securityBanDuration')}>
            <SelectField
              value={duration}
              onChange={setDuration}
              options={DURATIONS.map((item) => ({
                value: item.value,
                label:
                  item.value === '-1'
                    ? t('securityPermanent')
                    : `${item.units} ${t(item.key as any)}`,
              }))}
            />
          </Row>
          <Button
            size="sm"
            variant="primary"
            className="mb-1 h-9"
            isDisabled={!ip.trim() || submitting}
            onPress={() => void addBan()}
          >
            {submitting ? <Spinner size="sm" /> : <Plus className="size-3.5" aria-hidden />}
            {t('securityBanAdd')}
          </Button>
        </div>
      </div>

      <div className="space-y-1.5 border-t border-border pt-2">
        {bansQuery.isLoading ? (
          <div className="flex justify-center py-4">
            <Spinner size="sm" />
          </div>
        ) : bans.length ? (
          bans.map((ban) => (
            <div
              key={`${ban.id}:${ban.ip}`}
              className="flex items-start justify-between gap-2 rounded-lg border border-border/70 px-2 py-1.5"
            >
              <div className="min-w-0">
                <p className="flex items-center gap-1.5 font-mono text-[12px] font-medium">
                  <ShieldBan className="size-3.5 shrink-0 text-danger" aria-hidden />
                  <span className="truncate">{ban.ip}</span>
                </p>
                <p className="mt-0.5 truncate text-[11px] text-muted">
                  {[ban.reason, ban.note, ban.ip_region].filter(Boolean).join(' · ') ||
                    t('noContent')}
                </p>
                <p className="text-[11px] text-muted/80">
                  {ban.expire_at ? formatDateTime(ban.expire_at) : t('securityPermanent')} ·{' '}
                  {t('securityBanHits')} {ban.hit_count}
                  {ban.id <= 0 ? ` · ${t('securityBanSource')}: config` : ''}
                </p>
              </div>
              <Button
                size="sm"
                variant="ghost"
                className="shrink-0 text-danger"
                onPress={() => void removeBan(ban.ip, ban.id)}
              >
                {t('securityUnban')}
              </Button>
            </div>
          ))
        ) : (
          <EmptyState icon={ShieldBan} title={t('securityNoBans')} />
        )}
      </div>
    </Section>
  )
}
