import { Button, Skeleton, TextArea, toast } from '@heroui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, RotateCcw, Save } from 'lucide-react'
import { useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { useConfirm } from '@/components/ui/confirm'
import { api, errorMessage } from '@/lib/api'
import { applyCustomCode, type CustomCode } from '@/lib/custom-code'
import { t } from '@/lib/i18n'

const CODE_CLASS = 'font-mono text-[12px] leading-relaxed'

function CodeField({
  label,
  hint,
  value,
  placeholder,
  onChange,
}: {
  label: string
  hint: string
  value: string
  placeholder: string
  onChange: (next: string) => void
}) {
  return (
    <section className="rounded-xl border border-border bg-surface">
      <header className="flex items-start justify-between gap-2 px-3.5 pb-2 pt-3">
        <div className="min-w-0">
          <h2 className="text-[13.5px] font-semibold">{label}</h2>
          <p className="mt-0.5 text-[11.5px] leading-snug text-muted">{hint}</p>
        </div>
        <span className="shrink-0 pt-0.5 text-[10.5px] tabular-nums text-muted">
          {value.length.toLocaleString()}
        </span>
      </header>
      <div className="px-3.5 pb-3">
        <TextArea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          rows={8}
          fullWidth
          spellCheck={false}
          autoCapitalize="off"
          autoCorrect="off"
          className={CODE_CLASS}
        />
      </div>
    </section>
  )
}

export function CustomizePage() {
  const confirm = useConfirm()
  const queryClient = useQueryClient()

  const queryKey = ['custom-code']
  const { data, isPending } = useQuery({
    queryKey,
    queryFn: () => api<CustomCode>('/sidebar/custom'),
  })
  const css = useDraft(data?.css)
  const js = useDraft(data?.js)

  const dirty = css.value !== (data?.css ?? '') || js.value !== (data?.js ?? '')

  const mutation = useMutation({
    mutationFn: (payload: CustomCode) =>
      api<CustomCode>('/sidebar/custom', { method: 'PUT', body: payload }),
    onSuccess: (_result, payload) => {
      applyCustomCode(payload)
      void queryClient.invalidateQueries({ queryKey })
      css.commit(payload.css)
      js.commit(payload.js)
      toast.success(t('customSaved'))
    },
    onError: (error) => toast.danger(errorMessage(error) || t('opFailed')),
  })

  const reset = async () => {
    const ok = await confirm({
      title: t('customResetConfirm'),
      description: t('customResetConfirmDesc'),
      confirmText: t('reset'),
      danger: true,
    })
    if (!ok) return
    css.edit('')
    js.edit('')
    mutation.mutate({ css: '', js: '' })
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('customize')}
        description={t('customizeDesc')}
        actions={
          <>
            <Button
              size="sm"
              variant="ghost"
              isIconOnly
              aria-label={t('reset')}
              isDisabled={mutation.isPending || !(css.value || js.value)}
              onPress={() => void reset()}
            >
              <RotateCcw className="size-4" aria-hidden />
            </Button>
            <Button
              size="sm"
              variant="primary"
              isDisabled={!dirty || mutation.isPending}
              onPress={() => mutation.mutate({ css: css.value, js: js.value })}
            >
              <Save className="size-4" aria-hidden />
              {t('apply')}
            </Button>
          </>
        }
      />

      <div className="flex items-start gap-2 rounded-xl border border-warning/30 bg-warning/10 px-3.5 py-2.5 text-[11.5px] leading-snug text-warning">
        <AlertTriangle className="mt-px size-3.5 shrink-0" aria-hidden />
        <p>{t('customTrustHint')}</p>
      </div>

      {isPending ? (
        <div className="space-y-3">
          <Skeleton className="h-[220px] rounded-xl" />
          <Skeleton className="h-[220px] rounded-xl" />
        </div>
      ) : (
        <>
          <CodeField
            label={t('customCss')}
            hint={t('customCssHint')}
            value={css.value}
            placeholder="/* :root { --accent: #0f766e; --control-radius: 4px; } */"
            onChange={css.edit}
          />
          <CodeField
            label={t('customJs')}
            hint={t('customJsHint')}
            value={js.value}
            placeholder="// console.log('hello from the console')"
            onChange={js.edit}
          />
        </>
      )}
    </div>
  )
}

/**
 * The textarea keeps local edits until the server acknowledges them; `null`
 * means "untouched, follow the query data".
 */
function useDraft(initial: string | undefined) {
  const [edited, setEdited] = useState<string | null>(null)
  const value = edited ?? initial ?? ''
  return {
    value,
    edited,
    edit: setEdited,
    commit: (next: string) => setEdited(next || null),
  }
}
