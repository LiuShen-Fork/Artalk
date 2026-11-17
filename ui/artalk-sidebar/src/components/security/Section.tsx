import { Switch } from '@heroui/react'
import type { ReactNode } from 'react'

export function Section({
  title,
  description,
  actions,
  children,
}: {
  title: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="space-y-3 rounded-xl border border-border bg-surface p-3">
      <header className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h2 className="text-[13px] font-semibold">{title}</h2>
          {description ? (
            <p className="mt-0.5 text-[11.5px] leading-snug text-muted">{description}</p>
          ) : null}
        </div>
        {actions ? <div className="flex shrink-0 items-center gap-1.5">{actions}</div> : null}
      </header>
      {children}
    </section>
  )
}

export function Row({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div className="space-y-1 py-1">
      <p className="text-[12px] font-medium text-muted">{label}</p>
      {children}
      {hint ? <p className="text-[11px] leading-snug text-muted/80">{hint}</p> : null}
    </div>
  )
}

export function ToggleRow({
  label,
  description,
  isSelected,
  onChange,
}: {
  label: string
  description?: string
  isSelected: boolean
  onChange: (value: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between gap-3 py-1">
      <div className="min-w-0">
        <p className="text-[12.5px] font-medium">{label}</p>
        {description ? (
          <p className="mt-0.5 text-[11px] leading-snug text-muted">{description}</p>
        ) : null}
      </div>
      <Switch isSelected={isSelected} onChange={onChange} aria-label={label} className="shrink-0">
        <Switch.Content>
          <Switch.Control>
            <Switch.Thumb />
          </Switch.Control>
        </Switch.Content>
      </Switch>
    </div>
  )
}
