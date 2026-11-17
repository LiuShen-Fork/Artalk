import { Button, Input, Spinner, TextArea, toast } from '@heroui/react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw, Save } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Section, Row, ToggleRow } from '@/components/security/Section'
import { BanManager } from '@/components/security/BanManager'
import { AttackReport } from '@/components/security/AttackReport'
import { EventLog } from '@/components/security/EventLog'
import { SelectField } from '@/components/ui/SelectField'
import { PageHeader } from '@/components/ui/PageHeader'
import { SectionNav } from '@/components/ui/SectionNav'
import { errorMessage } from '@/lib/api'
import { t, type MessageKey } from '@/lib/i18n'
import {
  CAPTCHA_TYPES,
  QUOTA_WINDOWS,
  SECURITY_ACTIONS,
  securityApi,
  type CaptchaConfig,
  type SecurityConfig,
  type SecurityRuntime,
} from '@/lib/security'
import { cn } from '@/lib/utils'

type TabId = 'basic' | 'limit' | 'captcha' | 'bans' | 'report' | 'events'

const TABS: { id: TabId; label: MessageKey }[] = [
  { id: 'basic', label: 'securityBasic' },
  { id: 'limit', label: 'securityRateLimit' },
  { id: 'captcha', label: 'captcha' },
  { id: 'bans', label: 'securityBans' },
  { id: 'report', label: 'securityReport' },
  { id: 'events', label: 'securityEvents' },
]

const parseList = (text: string) =>
  text
    .split(/[\n,]/)
    .map((line) => line.trim())
    .filter(Boolean)

export function SecurityPage() {
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<TabId>('basic')
  const [draft, setDraft] = useState<SecurityConfig | null>(null)
  const [captcha, setCaptcha] = useState<CaptchaConfig | null>(null)
  const [runtime, setRuntime] = useState<SecurityRuntime | null>(null)
  const [blacklistText, setBlacklistText] = useState('')
  const [whitelistText, setWhitelistText] = useState('')
  const [saving, setSaving] = useState(false)

  const configQuery = useQuery({ queryKey: ['security-config'], queryFn: securityApi.getConfig })

  useEffect(() => {
    const data = configQuery.data
    if (!data) return
    setDraft(data.security)
    setCaptcha({
      ...data.captcha,
      geetest: data.captcha.geetest ?? { captcha_id: '', captcha_key: '' },
    })
    setRuntime(data.runtime)
    setBlacklistText((data.security.blacklist ?? []).join('\n'))
    setWhitelistText((data.security.whitelist ?? []).join('\n'))
  }, [configQuery.data])

  const patch = (values: Partial<SecurityConfig>) =>
    setDraft((prev) => (prev ? { ...prev, ...values } : prev))
  const patchCaptcha = (values: Partial<CaptchaConfig>) =>
    setCaptcha((prev) => (prev ? { ...prev, ...values } : prev))

  const save = async () => {
    if (!draft || !captcha || saving) return
    setSaving(true)
    try {
      const res = await securityApi.saveConfig({
        enabled: draft.enabled,
        action: draft.action,
        rate_limit: draft.rate_limit,
        auto_ban: draft.auto_ban,
        blacklist: parseList(blacklistText),
        whitelist: parseList(whitelistText),
        event_retention_days: draft.event_retention_days,
        captcha: {
          enabled: captcha.enabled,
          always: captcha.always,
          captcha_type: captcha.captcha_type,
          action_reset: captcha.action_reset,
          action_limit: captcha.action_limit,
          geetest: captcha.geetest,
        },
      })
      queryClient.setQueryData(['security-config'], res)
      toast.success(t('settingSaved'))
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    } finally {
      setSaving(false)
    }
  }

  const loading = configQuery.isLoading && !draft

  return (
    <div className="space-y-3 md:space-y-4">
      <PageHeader
        title={t('security')}
        description={t('securityAdminDesc')}
        actions={
          <>
            <Button
              size="sm"
              variant="ghost"
              isIconOnly
              aria-label={t('refresh')}
              isDisabled={configQuery.isFetching}
              onPress={() => void configQuery.refetch()}
            >
              <RefreshCw
                className={cn('size-4', configQuery.isFetching && 'animate-spin')}
                aria-hidden
              />
            </Button>
            <Button
              size="sm"
              variant="primary"
              isDisabled={!draft || saving}
              onPress={() => void save()}
            >
              {saving ? <Spinner size="sm" /> : <Save className="size-3.5" aria-hidden />}
              {t('save')}
            </Button>
          </>
        }
      />

      {loading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-[12.5px] text-muted">
          <Spinner size="sm" />
          {t('refreshing')}
        </div>
      ) : configQuery.isError ? (
        <Section title={t('security')}>
          <p className="text-[12.5px] text-danger">{errorMessage(configQuery.error)}</p>
          <Button size="sm" variant="secondary" onPress={() => void configQuery.refetch()}>
            {t('refresh')}
          </Button>
        </Section>
      ) : draft && captcha && runtime ? (
        <>
          <div className="grid grid-cols-2 gap-2 md:grid-cols-4 md:gap-3">
            <Stat label={t('securityTrackedIPs')} value={runtime.tracked_ips} />
            <Stat label={t('securityActiveBans')} value={runtime.active_bans} />
            <Stat label={t('securityViolators')} value={runtime.tracked_violators} />
            <Stat
              label={t('securityGlobalCount')}
              value={`${runtime.global.minute}/${runtime.global.hour}/${runtime.global.day}`}
              small
            />
          </div>

          <div className="md:flex md:items-start md:gap-6">
            <SectionNav
              items={TABS.map((item) => ({ id: item.id, label: t(item.label) }))}
              value={tab}
              onChange={setTab}
            />

            <div className="min-w-0 flex-1 space-y-3 md:max-w-[820px] md:space-y-4">

          {tab === 'basic' ? (
            <Section title={t('securityBasic')} description={t('securityBasicDesc')}>
              <ToggleRow
                label={t('securityEnable')}
                isSelected={draft.enabled}
                onChange={(enabled) => patch({ enabled })}
              />
              <Row label={t('securityAction')}>
                <SelectField
                  value={draft.action}
                  onChange={(action) => patch({ action: action as SecurityConfig['action'] })}
                  options={SECURITY_ACTIONS.map((action) => ({
                    value: action,
                    label: t(`securityAction_${action}` as any),
                  }))}
                />
              </Row>
              {draft.action === 'captcha' && !captcha.enabled ? (
                <p className="rounded-md bg-warning/10 px-2 py-1.5 text-[11.5px] text-warning">
                  {t('securityCaptchaHint')}
                </p>
              ) : null}
              <Row label={t('securityRetention')}>
                <Input
                  type="number"
                  min={1}
                  max={365}
                  value={String(draft.event_retention_days)}
                  onChange={(event) =>
                    patch({ event_retention_days: Number(event.target.value) || 0 })
                  }
                />
              </Row>
              <Row label={t('securityBlacklist')}>
                <TextArea
                  value={blacklistText}
                  rows={4}
                  placeholder={t('securityListHint')}
                  className="font-mono text-[12px]"
                  onChange={(event) => setBlacklistText(event.target.value)}
                />
              </Row>
              <Row label={t('securityWhitelist')}>
                <TextArea
                  value={whitelistText}
                  rows={4}
                  placeholder={t('securityListHint')}
                  className="font-mono text-[12px]"
                  onChange={(event) => setWhitelistText(event.target.value)}
                />
              </Row>
            </Section>
          ) : null}

          {tab === 'limit' ? (
            <Section title={t('securityRateLimit')} description={t('securityRateLimitDesc')}>
              <div className="space-y-2">
                <div className="grid grid-cols-[56px_1fr_1fr] items-center gap-1.5 text-[11px] text-muted">
                  <span />
                  <span>{t('securityPerIP')}</span>
                  <span>{t('securityGlobal')}</span>
                </div>
                {QUOTA_WINDOWS.map((window) => (
                  <div key={window} className="grid grid-cols-[56px_1fr_1fr] items-center gap-1.5">
                    <span className="text-[11.5px] text-muted">
                      {t(`securityWindow_${window}` as any)}
                    </span>
                    {(['ip', 'global'] as const).map((scope) => (
                      <Input
                        key={scope}
                        type="number"
                        min={0}
                        aria-label={`${scope} ${window}`}
                        value={String(draft.rate_limit[scope][window])}
                        onChange={(event) =>
                          patch({
                            rate_limit: {
                              ...draft.rate_limit,
                              [scope]: {
                                ...draft.rate_limit[scope],
                                [window]: Number(event.target.value) || 0,
                              },
                            },
                          })
                        }
                      />
                    ))}
                  </div>
                ))}
                <p className="text-[11px] leading-snug text-muted/80">{t('securityQuotaHint')}</p>
              </div>

              <div className="space-y-2 border-t border-border pt-2">
                <ToggleRow
                  label={t('securityAutoBan')}
                  description={t('securityAutoBanHint')}
                  isSelected={draft.auto_ban.enabled}
                  onChange={(enabled) => patch({ auto_ban: { ...draft.auto_ban, enabled } })}
                />
                {draft.auto_ban.enabled ? (
                  <div className="grid grid-cols-3 gap-1.5">
                    <Row label={t('securityAutoBanThreshold')}>
                      <Input
                        type="number"
                        min={1}
                        value={String(draft.auto_ban.threshold)}
                        onChange={(event) =>
                          patch({
                            auto_ban: {
                              ...draft.auto_ban,
                              threshold: Number(event.target.value) || 0,
                            },
                          })
                        }
                      />
                    </Row>
                    <Row label={t('securityAutoBanWindow')}>
                      <Input
                        type="number"
                        min={60}
                        step={60}
                        value={String(draft.auto_ban.window)}
                        onChange={(event) =>
                          patch({
                            auto_ban: {
                              ...draft.auto_ban,
                              window: Number(event.target.value) || 0,
                            },
                          })
                        }
                      />
                    </Row>
                    <Row label={t('securityAutoBanDuration')}>
                      <Input
                        type="number"
                        step={60}
                        value={String(draft.auto_ban.duration)}
                        onChange={(event) =>
                          patch({
                            auto_ban: {
                              ...draft.auto_ban,
                              duration: Number(event.target.value) || 0,
                            },
                          })
                        }
                      />
                    </Row>
                  </div>
                ) : null}
              </div>
            </Section>
          ) : null}

          {tab === 'captcha' ? (
            <Section title={t('captcha')} description={t('securityCaptchaDesc')}>
              <ToggleRow
                label={t('securityCaptchaEnable')}
                isSelected={captcha.enabled}
                onChange={(enabled) => patchCaptcha({ enabled })}
              />
              <ToggleRow
                label={t('securityCaptchaAlways')}
                isSelected={captcha.always}
                onChange={(always) => patchCaptcha({ always })}
              />
              <Row label={t('securityCaptchaType')}>
                <SelectField
                  value={captcha.captcha_type}
                  onChange={(captcha_type) => patchCaptcha({ captcha_type })}
                  options={CAPTCHA_TYPES.map((type) => ({ value: type, label: type }))}
                />
              </Row>
              <div className="grid grid-cols-2 gap-1.5">
                <Row label={t('securityCaptchaLimit')}>
                  <Input
                    type="number"
                    min={0}
                    value={String(captcha.action_limit)}
                    onChange={(event) =>
                      patchCaptcha({ action_limit: Number(event.target.value) || 0 })
                    }
                  />
                </Row>
                <Row label={t('securityCaptchaReset')}>
                  <Input
                    type="number"
                    min={0}
                    value={String(captcha.action_reset)}
                    onChange={(event) =>
                      patchCaptcha({ action_reset: Number(event.target.value) || 0 })
                    }
                  />
                </Row>
              </div>
              {captcha.captcha_type === 'geetest' ? (
                <div className="space-y-1.5 border-t border-border pt-2">
                  <Row label="Geetest Captcha ID">
                    <Input
                      fullWidth
                      value={captcha.geetest.captcha_id}
                      autoComplete="off"
                      onChange={(event) =>
                        patchCaptcha({
                          geetest: { ...captcha.geetest, captcha_id: event.target.value },
                        })
                      }
                    />
                  </Row>
                  <Row label="Geetest Captcha Key">
                    <Input
                      fullWidth
                      type="password"
                      autoComplete="off"
                      value={captcha.geetest.captcha_key}
                      onChange={(event) =>
                        patchCaptcha({
                          geetest: { ...captcha.geetest, captcha_key: event.target.value },
                        })
                      }
                    />
                  </Row>
                </div>
              ) : null}
            </Section>
          ) : null}

          {tab === 'bans' ? <BanManager /> : null}
          {tab === 'report' ? <AttackReport /> : null}
          {tab === 'events' ? <EventLog /> : null}
            </div>
          </div>
        </>
      ) : null}
    </div>
  )
}

function Stat({ label, value, small }: { label: string; value: number | string; small?: boolean }) {
  return (
    <div className="rounded-xl border border-border bg-surface px-3 py-2">
      <p className="text-[11px] text-muted">{label}</p>
      <p className={cn('mt-0.5 font-semibold tabular-nums', small ? 'text-[14px]' : 'text-[19px]')}>
        {value}
      </p>
    </div>
  )
}
