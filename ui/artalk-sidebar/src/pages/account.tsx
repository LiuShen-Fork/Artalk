import { Button, Input, Label, Spinner, TextField, toast } from '@heroui/react'
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Link2, Link2Off, ShieldCheck, UserRound } from 'lucide-react'
import { Section, Row } from '@/components/security/Section'
import { PageHeader } from '@/components/ui/PageHeader'
import { VerifyCodeButton } from '@/components/VerifyCodeButton'
import { useConfirm } from '@/components/ui/confirm'
import { getArtalk } from '@/lib/artalk'
import {
  accountApi,
  errorMessage,
  resolveServerPath,
  type AccountIdentity,
  type AccountProfile,
  type AuthProviderInfo,
} from '@/lib/account-api'
import { startOAuthBind } from '@/lib/oauth-bind'
import { t } from '@/lib/i18n'
import { session } from '@/lib/session'

/**
 * Account self-service: profile, password and linked social identities. This is
 * the page ordinary users land on, so it never assumes admin rights.
 */
export function AccountPage() {
  const navigate = useNavigate()
  const confirm = useConfirm()

  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [link, setLink] = useState('')
  const [initialEmail, setInitialEmail] = useState('')
  const [emailCode, setEmailCode] = useState('')
  const [hasPassword, setHasPassword] = useState(false)
  const [profileSaving, setProfileSaving] = useState(false)

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordCode, setPasswordCode] = useState('')
  const [passwordSaving, setPasswordSaving] = useState(false)

  const [identities, setIdentities] = useState<AccountIdentity[]>([])
  const [providers, setProviders] = useState<AuthProviderInfo[]>([])
  const [binding, setBinding] = useState('')

  const emailChanged = email.trim().toLowerCase() !== initialEmail.toLowerCase()
  const oauthProviders = useMemo(
    () => providers.filter((provider) => provider.name !== 'email' && provider.name !== 'skip'),
    [providers],
  )
  const passwordMismatch = !!confirmPassword && newPassword !== confirmPassword
  const canSubmitPassword =
    newPassword.length >= 8 &&
    !passwordMismatch &&
    (hasPassword ? !!currentPassword : !!passwordCode.trim())

  const identityFor = (provider: string) =>
    identities.find((item) => item.provider === provider) ?? null

  const applyProfile = (user: AccountProfile) => {
    setName(user.name)
    setEmail(user.email)
    setLink(user.link)
    setInitialEmail(user.email)
    setHasPassword(user.has_password)

    getArtalk().ctx.getUser().update({ name: user.name, email: user.email, link: user.link })
    session.setUser({ name: user.name, email: user.email, link: user.link })
  }

  useEffect(() => {
    let alive = true

    const load = async () => {
      setLoading(true)
      setLoadError('')
      try {
        const [profileRes, identityRes] = await Promise.all([
          accountApi.getProfile(),
          accountApi.getIdentities(),
        ])
        if (!alive) return
        if (!profileRes.is_login || !profileRes.user) {
          void navigate({ to: '/login', replace: true })
          return
        }

        applyProfile(profileRes.user)
        setIdentities(identityRes.identities ?? [])

        // The provider list only exists when social login is configured; the
        // profile and password forms must keep working without it.
        try {
          const providerRes = await accountApi.getProviders()
          if (alive) setProviders(providerRes.providers ?? [])
        } catch {
          if (alive) setProviders([])
        }
      } catch (error) {
        if (alive) setLoadError(errorMessage(error) || t('opFailed'))
      } finally {
        if (alive) setLoading(false)
      }
    }

    void load()
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const saveProfile = async () => {
    if (profileSaving) return

    if (emailChanged && !emailCode.trim()) {
      toast.danger(t('emailChangeHint'))
      return
    }

    setProfileSaving(true)
    try {
      const res = await accountApi.updateProfile({
        name: name.trim(),
        email: email.trim(),
        link: link.trim(),
        code: emailCode.trim() || undefined,
      })
      applyProfile(res.user)
      setEmailCode('')
      toast.success(t('settingSaved'))
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    } finally {
      setProfileSaving(false)
    }
  }

  const changePassword = async () => {
    if (passwordSaving || !canSubmitPassword) return

    setPasswordSaving(true)
    try {
      const res = await accountApi.updatePassword({
        current_password: hasPassword ? currentPassword : undefined,
        new_password: newPassword,
        code: hasPassword ? undefined : passwordCode.trim(),
      })

      getArtalk().ctx.getUser().update({ token: res.token })
      session.setUser({ token: res.token })
      setHasPassword(res.user.has_password)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setPasswordCode('')
      toast.success(t('passwordUpdated'))
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    } finally {
      setPasswordSaving(false)
    }
  }

  const refreshIdentities = async () => {
    const res = await accountApi.getIdentities()
    setIdentities(res.identities ?? [])
  }

  const bind = async (provider: AuthProviderInfo) => {
    if (binding) return
    setBinding(provider.name)

    try {
      const { path } = await accountApi.startIdentityBind(provider.name)
      const result = await startOAuthBind(resolveServerPath(path))
      if (!result.ok) throw new Error(result.message || t('bindFailed'))

      await refreshIdentities()
      toast.success(t('bindSuccess'))
    } catch (error) {
      // The popup can close before its result reaches us, so trust the identity
      // list over the thrown error when a binding already exists.
      await refreshIdentities()
      if (!identityFor(provider.name)) {
        toast.danger(errorMessage(error) || t('bindFailed'))
      }
    } finally {
      setBinding('')
    }
  }

  const unbind = async (identity: AccountIdentity) => {
    const ok = await confirm({
      title: t('unbind'),
      description: identity.provider,
      danger: true,
      confirmText: t('unbind'),
    })
    if (!ok) return

    try {
      await accountApi.removeIdentity(identity.id)
      await refreshIdentities()
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    }
  }

  return (
    <div className="space-y-3">
      <PageHeader title={t('account')} description={t('securityDesc')} />

      {loading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-[12.5px] text-muted">
          <Spinner size="sm" />
          {t('refreshing')}
        </div>
      ) : loadError ? (
        <Section title={t('account')}>
          <p className="text-[12.5px] text-danger">{loadError}</p>
        </Section>
      ) : (
        <>
          <Section title={t('profileSection')} description={t('emailChangeHint')}>
            <div className="space-y-1.5">
              <TextField className="w-full">
                <Label className="text-[12px] text-muted">{t('username')}</Label>
                <Input
                  value={name}
                  autoComplete="nickname"
                  onChange={(event) => setName(event.target.value)}
                />
              </TextField>
              <TextField className="w-full">
                <Label className="text-[12px] text-muted">{t('link')}</Label>
                <Input
                  value={link}
                  placeholder="https://"
                  autoComplete="url"
                  onChange={(event) => setLink(event.target.value)}
                />
              </TextField>
              <TextField className="w-full">
                <Label className="text-[12px] text-muted">{t('email')}</Label>
                <Input
                  type="email"
                  value={email}
                  autoComplete="email"
                  onChange={(event) => setEmail(event.target.value)}
                />
              </TextField>
              {emailChanged ? (
                <Row label={t('verificationCode')} hint={t('emailChangeHint')}>
                  <div className="flex items-center gap-1.5">
                    <Input
                      value={emailCode}
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                      placeholder={t('verificationCode')}
                      onChange={(event) => setEmailCode(event.target.value)}
                    />
                    <VerifyCodeButton email={email} onError={toast.danger} />
                  </div>
                </Row>
              ) : null}
              <Button
                size="sm"
                variant="primary"
                fullWidth
                isDisabled={profileSaving}
                onPress={() => void saveProfile()}
              >
                {profileSaving ? (
                  <Spinner size="sm" />
                ) : (
                  <UserRound className="size-3.5" aria-hidden />
                )}
                {t('save')}
              </Button>
            </div>
          </Section>

          <Section
            title={t('password')}
            description={hasPassword ? t('changePassword') : t('passwordUnset')}
          >
            <div className="space-y-1.5">
              {hasPassword ? (
                <TextField className="w-full">
                  <Label className="text-[12px] text-muted">{t('currentPassword')}</Label>
                  <Input
                    type="password"
                    value={currentPassword}
                    autoComplete="current-password"
                    placeholder={t('inputHint')}
                    onChange={(event) => setCurrentPassword(event.target.value)}
                  />
                </TextField>
              ) : (
                <Row label={t('verificationCode')}>
                  <div className="flex items-center gap-1.5">
                    <Input
                      value={passwordCode}
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                      placeholder={t('verificationCode')}
                      onChange={(event) => setPasswordCode(event.target.value)}
                    />
                    <VerifyCodeButton email={email} onError={toast.danger} />
                  </div>
                </Row>
              )}
              <div className="grid grid-cols-2 gap-1.5">
                <TextField className="w-full">
                  <Label className="text-[12px] text-muted">{t('newPassword')}</Label>
                  <Input
                    type="password"
                    value={newPassword}
                    autoComplete="new-password"
                    placeholder={t('inputHint')}
                    onChange={(event) => setNewPassword(event.target.value)}
                  />
                </TextField>
                <TextField className="w-full">
                  <Label className="text-[12px] text-muted">{t('confirmPassword')}</Label>
                  <Input
                    type="password"
                    value={confirmPassword}
                    autoComplete="new-password"
                    placeholder={t('inputHint')}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                  />
                </TextField>
              </div>
              {passwordMismatch ? (
                <p className="text-[11.5px] text-danger">{t('passwordMismatch')}</p>
              ) : null}
              <Button
                size="sm"
                variant="primary"
                fullWidth
                isDisabled={!canSubmitPassword || passwordSaving}
                onPress={() => void changePassword()}
              >
                {passwordSaving ? (
                  <Spinner size="sm" />
                ) : (
                  <ShieldCheck className="size-3.5" aria-hidden />
                )}
                {t('save')}
              </Button>
            </div>
          </Section>

          <Section title={t('bindingsSection')} description={t('bindingsDesc')}>
            {oauthProviders.length ? (
              <div className="space-y-1.5">
                {oauthProviders.map((provider) => {
                  const identity = identityFor(provider.name)
                  return (
                    <div
                      key={provider.name}
                      className="flex items-center justify-between gap-2 rounded-lg border border-border/70 px-2 py-1.5"
                    >
                      <div className="flex min-w-0 items-center gap-2">
                        {provider.icon ? (
                          <img
                            src={provider.icon}
                            alt=""
                            className="size-5 shrink-0 rounded"
                            loading="lazy"
                          />
                        ) : (
                          <span className="grid size-5 shrink-0 place-items-center rounded bg-default text-muted">
                            <Link2 className="size-3" aria-hidden />
                          </span>
                        )}
                        <div className="min-w-0">
                          <p className="truncate text-[12.5px] font-medium">{provider.label}</p>
                          <p className={`text-[11px] ${identity ? 'text-success' : 'text-muted'}`}>
                            {identity ? t('bound') : t('notBound')}
                          </p>
                        </div>
                      </div>
                      {identity ? (
                        <Button
                          size="sm"
                          variant="ghost"
                          className="shrink-0 text-danger"
                          onPress={() => void unbind(identity)}
                        >
                          <Link2Off className="size-3.5" aria-hidden />
                          {t('unbind')}
                        </Button>
                      ) : (
                        <Button
                          size="sm"
                          variant="secondary"
                          className="shrink-0"
                          isDisabled={!!binding}
                          onPress={() => void bind(provider)}
                        >
                          {binding === provider.name ? <Spinner size="sm" /> : t('bind')}
                        </Button>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="text-[12px] text-muted">{t('bindingsEmpty')}</p>
            )}
          </Section>
        </>
      )}
    </div>
  )
}
