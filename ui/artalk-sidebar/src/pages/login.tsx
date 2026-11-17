import { Button, Input, Separator } from '@heroui/react'
import { useNavigate } from '@tanstack/react-router'
import { Eye, EyeOff, Loader2, LogIn } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { defaultPathFor } from '@/app/nav'
import { VerifyCodeButton } from '@/components/VerifyCodeButton'
import { accountApi, errorMessage, resolveServerPath, type AuthProviderInfo } from '@/lib/account-api'
import { getArtalk } from '@/lib/artalk'
import { t } from '@/lib/i18n'
import { startOAuthLogin } from '@/lib/oauth-login'
import { session } from '@/lib/session'
import { cn } from '@/lib/utils'

type Mode = 'login' | 'register' | 'reset'

const MODE_TITLE: Record<Mode, Parameters<typeof t>[0]> = {
  login: 'signIn',
  register: 'signUp',
  reset: 'resetPassword',
}

function Field({
  label,
  children,
  className,
}: {
  label: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <label className={cn('block space-y-1.5', className)}>
      <span className="text-[12px] font-medium text-muted">{label}</span>
      {children}
    </label>
  )
}

function PasswordInput({
  value,
  onChange,
  autoComplete,
  placeholder,
  minLength,
}: {
  value: string
  onChange: (value: string) => void
  autoComplete: string
  placeholder: string
  minLength?: number
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div className="relative">
      <Input
        type={visible ? 'text' : 'password'}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        autoComplete={autoComplete}
        minLength={minLength}
        placeholder={placeholder}
        fullWidth
        className="pr-10"
      />
      <button
        type="button"
        onClick={() => setVisible((prev) => !prev)}
        aria-label={visible ? t('hide') : t('view')}
        className="absolute right-1.5 top-1/2 grid size-7 -translate-y-1/2 place-items-center rounded-md text-muted transition-colors hover:bg-surface-secondary hover:text-foreground"
      >
        {visible ? <EyeOff className="size-4" aria-hidden /> : <Eye className="size-4" aria-hidden />}
      </button>
    </div>
  )
}

export function LoginPage() {
  const navigate = useNavigate()
  const [mode, setMode] = useState<Mode>('login')
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [userSelector, setUserSelector] = useState<string[] | null>(null)
  const [version, setVersion] = useState('')
  const [buildHash, setBuildHash] = useState('')
  const [providers, setProviders] = useState<AuthProviderInfo[]>([])
  const [trustedOrigin, setTrustedOrigin] = useState<boolean | null>(null)
  const [socialBusy, setSocialBusy] = useState('')

  useEffect(() => {
    getArtalk()
      .ctx.getApi()
      .version.getVersion()
      .then((res) => {
        setVersion(res.data.version)
        setBuildHash(res.data.commit_hash)
      })
      .catch(() => undefined)

    let alive = true
    void accountApi
      .getProviders()
      .then((res) => {
        if (alive) setProviders((res.providers ?? []).filter((item) => item.name !== 'email'))
      })
      .catch(() => undefined)
    void accountApi.isOriginTrusted().then((trusted) => {
      if (alive) setTrustedOrigin(trusted)
    })
    return () => {
      alive = false
    }
  }, [])

  const socialProviders = useMemo(() => providers, [providers])

  const finishLogin = (data: { user: any; token: string }) => {
    const artalk = getArtalk()
    artalk.ctx.getUser().update({ ...(data.user as any), token: data.token })
    session.setUser(artalk.ctx.getUser().getData())
    void navigate({ to: defaultPathFor(!!artalk.ctx.getUser().getData().is_admin), replace: true })
  }

  const doLogin = async (username?: string) => {
    setError('')
    setSubmitting(true)
    try {
      const res = await getArtalk()
        .ctx.getApi()
        .user.login({ name: username || '', email, password })
      finishLogin(res.data)
    } catch (e: any) {
      if (e?.data?.need_name_select) setUserSelector(e.data.need_name_select as string[])
      else setError(e?.message || t('loginFailure'))
    } finally {
      setSubmitting(false)
    }
  }

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    if (mode === 'login') {
      await doLogin()
      return
    }
    if (password !== confirm) {
      setError(t('passwordMismatch'))
      return
    }
    setSubmitting(true)
    try {
      const auth = getArtalk().ctx.getApi().auth
      const res =
        mode === 'register'
          ? await auth.registerByEmail({
              name: name.trim(),
              email: email.trim(),
              password,
              code: code.trim(),
            })
          : await auth.registerByEmail({
              email: email.trim(),
              password,
              code: code.trim(),
            })
      finishLogin(res.data)
    } catch (e: any) {
      setError(e?.message || t('loginFailure'))
    } finally {
      setSubmitting(false)
    }
  }

  const socialLogin = async (provider: AuthProviderInfo) => {
    if (socialBusy) return
    setError('')
    setSocialBusy(provider.name)
    try {
      const path = provider.path || `/auth/${provider.name}`
      const token = await startOAuthLogin(resolveServerPath(path))
      const artalk = getArtalk()
      artalk.ctx.getUser().update({ token })
      session.setUser({ token })
      const profile = await accountApi.getProfile()
      finishLogin({ user: profile.user, token })
    } catch (e) {
      const message = errorMessage(e)
      setError(message === 'canceled' ? '' : message || t('loginFailure'))
    } finally {
      setSocialBusy('')
    }
  }

  const switchMode = (next: Mode) => {
    setMode(next)
    setError('')
    setUserSelector(null)
    setCode('')
    setPassword('')
    setConfirm('')
  }

  return (
    <div className="flex min-h-full bg-background">
      {/* Wide screens get a brand panel; the iframe and phones go straight to the form. */}
      <aside className="relative hidden flex-1 overflow-hidden border-r border-border bg-surface/60 lg:flex lg:flex-col lg:justify-between lg:p-10">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(90%_70%_at_20%_0%,color-mix(in_oklab,var(--accent)_22%,transparent),transparent_65%)]"
        />
        <div className="relative flex items-center gap-2.5">
          <span className="grid size-8 place-items-center rounded-xl bg-accent text-[14px] font-bold text-accent-foreground">
            A
          </span>
          <span className="text-[15px] font-semibold">Artalk</span>
        </div>
        <div className="relative max-w-[320px] space-y-3">
          <h1 className="text-[26px] font-semibold leading-tight tracking-[-0.02em]">
            {t('ctrlCenter')}
          </h1>
          <p className="text-[13.5px] leading-relaxed text-muted">{t('loginPanelDesc')}</p>
        </div>
        <p className="relative text-[11px] text-muted">
          {version ? (
            <span className="tabular-nums">
              v{version}
              {buildHash ? ` / ${buildHash.slice(0, 7)}` : ''}
            </span>
          ) : null}
        </p>
      </aside>

      <div className="flex min-w-0 flex-1 items-center justify-center px-5 py-8 lg:px-8">
        <div className="w-full max-w-[340px] lg:max-w-[360px]">
          <div className="mb-6 flex flex-col items-center gap-3 text-center lg:hidden">
            <span className="grid size-12 place-items-center rounded-2xl bg-accent text-lg font-bold text-accent-foreground shadow-lg shadow-accent/25">
              A
            </span>
            <div>
              <h1 className="text-[17px] font-semibold tracking-tight">Artalk</h1>
              <p className="mt-0.5 text-[13px] text-muted">
                {mode === 'login' ? t('ctrlCenter') : t(MODE_TITLE[mode])}
              </p>
            </div>
          </div>
          <h2 className="mb-4 hidden text-[17px] font-semibold tracking-tight lg:block">
            {mode === 'login' ? t('ctrlCenter') : t(MODE_TITLE[mode])}
          </h2>

          <form onSubmit={submit} className="space-y-3.5">
            {mode === 'register' ? (
              <Field label={t('username')}>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoComplete="nickname"
                  placeholder={t('username')}
                  fullWidth
                  required
                />
              </Field>
            ) : null}

            <Field label={t('email')}>
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                autoComplete="username"
                placeholder={t('email')}
                fullWidth
                required
              />
            </Field>

            <Field label={mode === 'login' ? t('password') : t('newPassword')}>
              <PasswordInput
                value={password}
                onChange={setPassword}
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                placeholder={mode === 'login' ? t('password') : t('newPassword')}
                minLength={mode === 'login' ? undefined : 8}
              />
            </Field>

            {mode !== 'login' ? (
              <>
                <Field label={t('confirmPassword')}>
                  <PasswordInput
                    value={confirm}
                    onChange={setConfirm}
                    autoComplete="new-password"
                    minLength={8}
                    placeholder={t('confirmPassword')}
                  />
                </Field>
                <Field label={t('verificationCode')}>
                  <div className="flex items-center gap-1.5">
                    <Input
                      value={code}
                      onChange={(e) => setCode(e.target.value)}
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                      placeholder={t('verificationCode')}
                      fullWidth
                      required
                    />
                    <VerifyCodeButton email={email} disabled={!email.trim()} onError={setError} />
                  </div>
                </Field>
              </>
            ) : null}

            {error ? (
              <p
                role="alert"
                className="rounded-lg bg-danger/10 px-3 py-2 text-[12px] leading-relaxed text-danger"
              >
                {error}
              </p>
            ) : null}

            <Button type="submit" variant="primary" fullWidth isDisabled={submitting}>
              {submitting ? '…' : t(MODE_TITLE[mode])}
            </Button>
          </form>

          {socialProviders.length ? (
            <div className="mt-5 space-y-3">
              <div className="flex items-center gap-2">
                <Separator className="flex-1" />
                <span className="text-[11px] text-muted">{t('socialLoginHint')}</span>
                <Separator className="flex-1" />
              </div>
              <div className="grid gap-1.5">
                {socialProviders.map((provider) => (
                  <Button
                    key={provider.name}
                    variant="outline"
                    fullWidth
                    isDisabled={!!socialBusy || trustedOrigin === false}
                    onPress={() => void socialLogin(provider)}
                    className="justify-center gap-2 text-[13px]"
                  >
                    {socialBusy === provider.name ? (
                      <Loader2 className="size-4 animate-spin" aria-hidden />
                    ) : provider.icon ? (
                      <img src={provider.icon} alt="" className="size-4" />
                    ) : (
                      <LogIn className="size-4" aria-hidden />
                    )}
                    <span className="truncate">{provider.label || provider.name}</span>
                  </Button>
                ))}
              </div>
              {trustedOrigin === false ? (
                <p className="text-center text-[11px] leading-relaxed text-warning">
                  {t('socialLoginUntrusted')}
                </p>
              ) : null}
            </div>
          ) : null}

          <div className="mt-4 flex flex-col items-center gap-2 text-[12px] text-muted">
            {mode === 'login' ? (
              <>
                <span>
                  {t('noAccountPrompt')}{' '}
                  <button
                    type="button"
                    className="font-medium text-accent"
                    onClick={() => switchMode('register')}
                  >
                    {t('signUp')}
                  </button>
                </span>
                <button
                  type="button"
                  className="font-medium text-accent"
                  onClick={() => switchMode('reset')}
                >
                  {t('forgetPassword')}
                </button>
              </>
            ) : (
              <span>
                {t('haveAccountPrompt')}{' '}
                <button
                  type="button"
                  className="font-medium text-accent"
                  onClick={() => switchMode('login')}
                >
                  {t('signIn')}
                </button>
              </span>
            )}
          </div>

          <p className="mt-8 text-center text-[11px] text-muted lg:hidden">
            Powered by{' '}
            <a href="https://artalk.js.org" target="_blank" rel="noreferrer" className="text-accent">
              Artalk
            </a>
            {version ? (
              <span className="ml-1 tabular-nums">
                v{version}
                {buildHash ? ` / ${buildHash.slice(0, 7)}` : ''}
              </span>
            ) : null}
          </p>
        </div>
      </div>

      {userSelector ? (
        <div className="absolute inset-0 z-10 grid place-items-center bg-overlay/60 px-6 backdrop-blur-sm">
          <div className="w-full max-w-[300px] rounded-xl border border-border bg-surface p-4 shadow-xl">
            <p className="mb-2 text-[13px] font-medium">{t('loginSelectHint')}</p>
            <div className="space-y-1">
              {userSelector.map((username) => (
                <button
                  key={username}
                  type="button"
                  className="w-full rounded-lg px-3 py-2 text-left text-[13px] hover:bg-surface-secondary"
                  onClick={() => {
                    setUserSelector(null)
                    void doLogin(username)
                  }}
                >
                  {username}
                </button>
              ))}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
