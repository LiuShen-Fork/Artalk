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
  return (
    <HeroSelect
      selectedKey={String(value)}
      isDisabled={isDisabled}
      aria-label={ariaLabel}
      onSelectionChange={(key) => onChange(String(key))}
    >
      <HeroSelect.Trigger className={cn('w-full', className)}>
        <HeroSelect.Value />
        <HeroSelect.Indicator />
      </HeroSelect.Trigger>
      <HeroSelect.Popover>
        <ListBox>
          {options.map((option) => (
            <ListBox.Item key={option.value} id={option.value} textValue={option.value}>
              {option.label}
            </ListBox.Item>
          ))}
        </ListBox>
      </HeroSelect.Popover>
    </HeroSelect>
  )
}
