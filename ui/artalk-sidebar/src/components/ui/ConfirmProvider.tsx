import { AlertDialog, Button } from '@heroui/react'
import { useCallback, useRef, useState, type ReactNode } from 'react'
import { ConfirmContext, type ConfirmFn, type ConfirmOptions } from '@/components/ui/confirm'
import { t } from '@/lib/i18n'

/**
 * Promise-based confirmation backed by HeroUI's AlertDialog. Call sites read
 * `if (!(await confirm({...}))) return`, which keeps destructive flows in one
 * line instead of an extra piece of local state per page.
 */
export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [dialog, setDialog] = useState<ConfirmOptions | null>(null)
  const resolver = useRef<((value: boolean) => void) | null>(null)

  const confirm = useCallback<ConfirmFn>((options) => {
    return new Promise<boolean>((resolve) => {
      resolver.current = resolve
      setDialog(options)
    })
  }, [])

  const settle = (value: boolean) => {
    resolver.current?.(value)
    resolver.current = null
    setDialog(null)
  }

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      <AlertDialog isOpen={!!dialog} onOpenChange={(open) => !open && settle(false)}>
        <AlertDialog.Backdrop>
          <AlertDialog.Container>
            <AlertDialog.Dialog className="gap-4">
              <AlertDialog.Header className="text-start">
                <AlertDialog.Heading>{dialog?.title}</AlertDialog.Heading>
              </AlertDialog.Header>
              {dialog?.description ? (
                <div className="text-[13px] leading-relaxed text-muted">{dialog.description}</div>
              ) : null}
              <AlertDialog.Footer className="justify-end gap-2">
                <Button variant="ghost" size="sm" onPress={() => settle(false)}>
                  {dialog?.cancelText ?? t('cancel')}
                </Button>
                <Button
                  variant={dialog?.danger ? 'danger' : 'primary'}
                  size="sm"
                  onPress={() => settle(true)}
                >
                  {dialog?.confirmText ?? t('apply')}
                </Button>
              </AlertDialog.Footer>
            </AlertDialog.Dialog>
          </AlertDialog.Container>
        </AlertDialog.Backdrop>
      </AlertDialog>
    </ConfirmContext.Provider>
  )
}
