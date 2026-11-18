import { consoleInitial, consoleName, useConsoleSettings } from '@/lib/console'
import { cn } from '@/lib/utils'

const SIZES = {
  sm: 'size-7 rounded-lg text-[12px]',
  md: 'size-8 rounded-control text-[13px]',
  lg: 'size-12 rounded-2xl text-lg',
} as const

/**
 * The console logo: the owner's icon when one is configured, otherwise a letter
 * badge built from the console name. Used by the sidebar, the phone header and
 * the sign-in page so all three stay in sync.
 */
export function BrandMark({
  size = 'md',
  className,
}: {
  size?: keyof typeof SIZES
  className?: string
}) {
  const settings = useConsoleSettings()
  const name = consoleName(settings)

  return (
    <span
      title={name}
      className={cn(
        'grid shrink-0 place-items-center overflow-hidden bg-accent font-bold text-accent-foreground',
        SIZES[size],
        className,
      )}
    >
      {settings.icon ? (
        <img src={settings.icon} alt="" className="size-full object-cover" />
      ) : (
        consoleInitial(settings)
      )}
    </span>
  )
}
