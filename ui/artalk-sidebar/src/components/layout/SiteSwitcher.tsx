import { Button, Drawer, Spinner } from '@heroui/react'
import { Check, Globe2, Settings2 } from 'lucide-react'
import { useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useSession, session } from '@/lib/session'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import { useMediaQuery } from '@/lib/use-media'

interface SiteSwitcherProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * A bottom sheet listing every site plus the "all sites" view. The choice is
 * stored in the session store, so every page's next query is scoped to it
 * without any page needing its own site picker.
 */
export function SiteSwitcher({ open, onOpenChange }: SiteSwitcherProps) {
  const { site, sites, sitesLoaded } = useSession()
  const navigate = useNavigate()
  // A bottom sheet suits a phone; on a desktop window the same list reads
  // better sliding in from the edge it was opened from.
  const wide = useMediaQuery('(min-width: 768px)')

  useEffect(() => {
    if (open && !sitesLoaded) void session.refreshSites()
  }, [open, sitesLoaded])

  return (
    <Drawer isOpen={open} onOpenChange={onOpenChange}>
      <Drawer.Backdrop>
        <Drawer.Content placement={wide ? 'right' : 'bottom'}>
          <Drawer.Dialog className={wide ? 'h-full gap-0 p-0' : 'max-h-[72vh] gap-0 p-0'}>
            {wide ? null : <Drawer.Handle />}
            <Drawer.Header className="px-4 pb-2 pt-1 text-start">
              <Drawer.Heading className="text-base">{t('siteManage')}</Drawer.Heading>
            </Drawer.Header>
            <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-2">
              {!sitesLoaded ? (
                <div className="flex justify-center py-6">
                  <Spinner size="sm" />
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => {
                    session.setSite('')
                    onOpenChange(false)
                  }}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors',
                    !site ? 'bg-accent/10 text-accent' : 'hover:bg-surface-secondary',
                  )}
                >
                  <span className="grid size-8 place-items-center rounded-lg bg-default">
                    <Globe2 className="size-4" aria-hidden />
                  </span>
                  <span className="flex-1 text-sm font-medium">{t('allSites')}</span>
                  {!site ? <Check className="size-4" aria-hidden /> : null}
                </button>
              )}
              {sites.map((item) => {
                const active = item.name === site
                return (
                  <button
                    key={item.name}
                    type="button"
                    onClick={() => {
                      session.setSite(item.name)
                      onOpenChange(false)
                    }}
                    className={cn(
                      'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors',
                      active ? 'bg-accent/10 text-accent' : 'hover:bg-surface-secondary',
                    )}
                  >
                    <span className="grid size-8 place-items-center rounded-lg bg-default text-xs font-semibold uppercase">
                      {item.name.slice(0, 1)}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium">{item.name}</span>
                      {item.urls ? (
                        <span className="block truncate text-[11px] text-muted">{item.urls}</span>
                      ) : null}
                    </span>
                    {active ? <Check className="size-4" aria-hidden /> : null}
                  </button>
                )
              })}
            </div>
            <Drawer.Footer className="border-t border-border p-2">
              <Button
                variant="ghost"
                fullWidth
                className="justify-start gap-2"
                onPress={() => {
                  onOpenChange(false)
                  void navigate({ to: '/sites' })
                }}
              >
                <Settings2 className="size-4" aria-hidden />
                {t('siteManage')}
              </Button>
            </Drawer.Footer>
          </Drawer.Dialog>
        </Drawer.Content>
      </Drawer.Backdrop>
    </Drawer>
  )
}
