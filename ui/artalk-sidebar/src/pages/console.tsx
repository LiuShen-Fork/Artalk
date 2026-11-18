import { Button, Input, Skeleton, TextArea, toast } from '@heroui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Save } from 'lucide-react'
import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { api, errorMessage } from '@/lib/api'
import {
  consoleInitial,
  consoleName,
  consoleStore,
  DEFAULT_CONSOLE_NAME,
  type ConsoleSettings,
} from '@/lib/console'
import { t } from '@/lib/i18n'

const CODE_CLASS = 'font-mono text-[12px] leading-relaxed'
const QUERY_KEY = ['console-settings']

export function ConsolePage() {
  const queryClient = useQueryClient()

  const { data, isPending } = useQuery({
    queryKey: QUERY_KEY,
    queryFn: () => api<ConsoleSettings>('/sidebar/console'),
  })

  const [draft, setDraft] = useState<ConsoleSettings | null>(null)
  useEffect(() => {
    if (data) setDraft(data)
  }, [data])

  const value = draft ?? data ?? null
  const patch = (next: Partial<ConsoleSettings>) =>
    setDraft((prev) => ({ ...(prev ?? data ?? emptyConsole()), ...next }))

  const dirty =
    !!value &&
    !!data &&
    Object.keys(value).some((key) => {
      const field = key as keyof ConsoleSettings
      return (value[field] ?? '') !== (data[field] ?? '')
    })

  const mutation = useMutation({
    mutationFn: (payload: ConsoleSettings) =>
      api<ConsoleSettings>('/sidebar/console', { method: 'PUT', body: payload }),
    onSuccess: (result, payload) => {
      consoleStore.set({ ...payload, ...result })
      void queryClient.invalidateQueries({ queryKey: QUERY_KEY })
      toast.success(t('customSaved'))
    },
    onError: (error) => toast.danger(errorMessage(error) || t('opFailed')),
  })

  return (
    <div className="space-y-4 md:space-y-5">
      <PageHeader
        title={t('navConsole')}
        description={t('consoleDesc')}
        actions={
          <Button
            size="sm"
            variant="primary"
            isDisabled={!dirty || mutation.isPending}
            onPress={() => value && mutation.mutate(value)}
          >
            <Save className="size-4" aria-hidden />
            {t('apply')}
          </Button>
        }
      />

      {isPending || !value ? (
        <div className="space-y-3">
          <Skeleton className="h-[188px] rounded-xl" />
          <Skeleton className="h-[220px] rounded-xl" />
        </div>
      ) : (
        <>
          <Card title={t('consoleBrand')} hint={t('consoleBrandDesc')}>
            <Preview name={value.name} icon={value.icon} description={value.description} />
            <Field label={t('consoleName')} hint={t('consoleNameHint')}>
              <Input
                value={value.name}
                onChange={(event) => patch({ name: event.target.value })}
                placeholder={DEFAULT_CONSOLE_NAME}
                maxLength={40}
                fullWidth
              />
            </Field>
            <Field label={t('consoleIcon')} hint={t('consoleIconHint')}>
              <Input
                value={value.icon}
                onChange={(event) => patch({ icon: event.target.value })}
                placeholder="https://example.com/logo.svg"
                fullWidth
                spellCheck={false}
              />
            </Field>
            <Field label={t('consoleDescription')} hint={t('consoleDescriptionHint')}>
              <TextArea
                value={value.description}
                onChange={(event) => patch({ description: event.target.value })}
                placeholder={t('loginPanelDesc')}
                rows={2}
                maxLength={300}
                fullWidth
              />
            </Field>
          </Card>

          <div className="flex items-start gap-2 rounded-xl border border-warning/30 bg-warning/10 px-3.5 py-2.5 text-[11.5px] leading-snug text-warning">
            <AlertTriangle className="mt-px size-3.5 shrink-0" aria-hidden />
            <p>{t('customTrustHint')}</p>
          </div>

          <Card title={t('customCss')} hint={t('customCssHint')} counter={value.css.length}>
            <TextArea
              value={value.css}
              onChange={(event) => patch({ css: event.target.value })}
              placeholder="/* :root { --accent: #0f766e; --control-radius: 4px; } */"
              rows={8}
              fullWidth
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              className={CODE_CLASS}
            />
          </Card>

          <Card title={t('customJs')} hint={t('customJsHint')} counter={value.js.length}>
            <TextArea
              value={value.js}
              onChange={(event) => patch({ js: event.target.value })}
              placeholder="// console.log('hello from the console')"
              rows={6}
              fullWidth
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              className={CODE_CLASS}
            />
          </Card>
        </>
      )}
    </div>
  )
}

function emptyConsole(): ConsoleSettings {
  return { name: '', icon: '', description: '', css: '', js: '' }
}

function Card({
  title,
  hint,
  counter,
  children,
}: {
  title: string
  hint: string
  counter?: number
  children: React.ReactNode
}) {
  return (
    <section className="rounded-xl border border-border bg-surface">
      <header className="flex items-start justify-between gap-2 px-3.5 pb-2 pt-3">
        <div className="min-w-0">
          <h2 className="text-[13.5px] font-semibold">{title}</h2>
          <p className="mt-0.5 text-[11.5px] leading-snug text-muted">{hint}</p>
        </div>
        {counter !== undefined && (
          <span className="shrink-0 pt-0.5 text-[10.5px] tabular-nums text-muted">
            {counter.toLocaleString()}
          </span>
        )}
      </header>
      <div className="space-y-3 px-3.5 pb-3.5">{children}</div>
    </section>
  )
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <label className="block space-y-1.5">
      <span className="block text-[12px] font-medium">{label}</span>
      {children}
      {hint && <span className="block text-[11px] leading-snug text-muted">{hint}</span>}
    </label>
  )
}

/** WYSIWYG of the sign-in page header, so the owner sees the effect before saving. */
function Preview({ name, icon, description }: Partial<ConsoleSettings>) {
  const settings = {
    name: name ?? '',
    icon: icon ?? '',
    description: description ?? '',
    css: '',
    js: '',
  }
  return (
    <div className="flex items-start gap-3 rounded-lg bg-default/60 p-3">
      <span className="grid size-10 shrink-0 place-items-center overflow-hidden rounded-xl bg-accent text-[15px] font-bold text-accent-foreground">
        {settings.icon ? (
          <img src={settings.icon} alt="" className="size-full object-cover" />
        ) : (
          consoleInitial(settings)
        )}
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-semibold">{consoleName(settings)}</p>
        <p className="mt-0.5 line-clamp-2 text-[11.5px] leading-snug text-muted">
          {settings.description.trim() || t('loginPanelDesc')}
        </p>
      </div>
    </div>
  )
}
