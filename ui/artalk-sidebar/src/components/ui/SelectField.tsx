import { ListBox, Select as HeroSelect } from '@heroui/react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export interface SelectOption {
  value: string
  label: ReactNode
}

interface SelectFieldProps {
  /** Current value; numbers are matched through their string form. */
  value: string | number
  options: SelectOption[]
  onChange: (value: string) => void
  isDisabled?: boolean
  className?: string
  ariaLabel?: string
}

/**
 * HeroUI's `Select` needs a trigger, a value slot, a popover and a list box to
 * be usable; every call site in this app wants exactly that, so it lives here.
 */
export function SelectField({
  value,
  options,
  onChange,
  isDisabled,
  className,
  ariaLabel,
}: SelectFieldProps) {
  // A value that is not among the options would leave the trigger showing the
  // "select an item" placeholder — and the placeholder is long enough to wrap
  // the control onto several lines. Keep the real value visible instead.
  const current = String(value)
  const list = options.some((option) => option.value === current)
    ? options
    : [{ value: current, label: current }, ...options]

  return (
    <HeroSelect
      selectedKey={current}
      isDisabled={isDisabled}
      aria-label={ariaLabel}
      onSelectionChange={(key) => onChange(String(key))}
    >
      <HeroSelect.Trigger className={cn('w-full whitespace-nowrap', className)}>
        <HeroSelect.Value />
        <HeroSelect.Indicator />
      </HeroSelect.Trigger>
      <HeroSelect.Popover>
        <ListBox>
          {list.map((option) => (
            <ListBox.Item key={option.value} id={option.value} textValue={option.value}>
              {option.label}
            </ListBox.Item>
          ))}
        </ListBox>
      </HeroSelect.Popover>
    </HeroSelect>
  )
}
