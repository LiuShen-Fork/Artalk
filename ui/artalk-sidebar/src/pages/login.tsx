import { Button, Input } from '@heroui/react'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { defaultPathFor } from '@/app/nav'
import { getArtalk } from '@/lib/artalk'
import { t } from '@/lib/i18n'
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

  useEffect(() => {
    getArtalk()
      .ctx.getApi()
      .version.getVersion()
      .then((res) => {
        setVersion(res.data.version)
        setBuildHash(res.data.commit_hash)
      })
      .catch(() => undefined)
  }, [])

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

  const switchMode = (next: Mode) => {
    setMode(next)
    setError('')
    setUserSelector(null)
    setCode('')
    setPassword('')
    setConfirm('')
  }

  return (
    <div className="relative grid min-h-full place-items-center bg-background px-6 py-10">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-56 bg-[radial-gradient(120%_100%_at_50%_0%,color-mix(in_oklab,var(--accent)_18%,transparent),transparent_70%)]"
      />
      <div className="relative w-full max-w-[340px]">
        <div className="mb-6 flex flex-col items-center gap-3 text-center">
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
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              placeholder={mode === 'login' ? t('password') : t('newPassword')}
              fullWidth
              required
            />
          </Field>

          {mode !== 'login' ? (
            <>
              <Field label={t('confirmPassword')}>
                <Input
                  type="password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  placeholder={t('confirmPassword')}
                  fullWidth
                  required
                />
              </Field>
              <Field label={t('verificationCode')}>
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

        <p className="mt-8 text-center text-[11px] text-muted">
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
