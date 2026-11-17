import { Button } from '@heroui/react'
import { useEffect, useRef, useState } from 'react'
import { getArtalk } from '@/lib/artalk'
import { t } from '@/lib/i18n'

interface VerifyCodeButtonProps {
  email: string
  disabled?: boolean
  onError: (message: string) => void
}

/** Sends a login-code email and locks itself for a minute while it is valid. */
export function VerifyCodeButton({ email, disabled, onError }: VerifyCodeButtonProps) {
  const [sending, setSending] = useState(false)
  const [remaining, setRemaining] = useState(0)
  const timer = useRef<number | undefined>(undefined)

  useEffect(() => {
    return () => {
      if (timer.current !== undefined) window.clearInterval(timer.current)
    }
  }, [])

  const send = async () => {
    if (sending || remaining > 0) return

    const target = email.trim()
    if (!target) {
      onError(t('email'))
      return
    }

    setSending(true)
    try {
      await getArtalk().ctx.getApi().auth.sendVerifyEmail({ email: target })
      setRemaining(60)
      timer.current = window.setInterval(() => {
        setRemaining((prev) => {
          if (prev <= 1 && timer.current !== undefined) {
            window.clearInterval(timer.current)
            timer.current = undefined
            return 0
          }
          return prev - 1
        })
      }, 1000)
    } catch (error) {
      onError((error as Error)?.message || t('opFailed'))
    } finally {
      setSending(false)
    }
  }

  return (
    <Button
      size="sm"
      variant="secondary"
      isDisabled={disabled || sending || remaining > 0}
      className="shrink-0 tabular-nums"
      onPress={() => void send()}
    >
      {sending
        ? '…'
        : remaining > 0
          ? t('waitSeconds', { seconds: String(remaining) })
          : t('verifySend')}
    </Button>
  )
}
