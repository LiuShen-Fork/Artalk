import { Button, Input, Label, Modal, Spinner, Switch, TextField, toast } from '@heroui/react'
import { useState } from 'react'
import { Shuffle } from 'lucide-react'
import { t } from '@/lib/i18n'
import { hoverTitle } from '@/lib/hover-title'
import { getArtalk } from '@/lib/artalk'
import { errorMessage } from '@/lib/api'
import { formatNumber } from '@/lib/utils'

export interface EditableUser {
  id: number
  name: string
  email: string
  link: string
  badge_name: string
  badge_color: string
  is_admin: boolean
  receive_email: boolean
  comment_count?: number
  last_ip?: string
  last_ua?: string
}

export interface UserEditorDialogProps {
  isOpen: boolean
  user?: EditableUser
  onClose: () => void
  onSaved: () => void
}

const DEFAULT_BADGE_COLOR = '#a85f48'

/**
 * Create/edit form for a single user. Mirrors the fields the public widget's
 * admin API accepts, so nothing here can put a user into a state the regular
 * endpoints could not.
 */
export function UserEditorDialog({ isOpen, user, onClose, onSaved }: UserEditorDialogProps) {
  return (
    <Modal isOpen={isOpen} onOpenChange={(open) => !open && onClose()}>
      <Modal.Backdrop>
        <Modal.Container>
          <Modal.Dialog className="gap-3">
            <Modal.CloseTrigger aria-label={t('close')} {...hoverTitle(t('close'))} />
            <Modal.Header>
              <Modal.Heading>{user ? t('userEdit') : t('userCreate')}</Modal.Heading>
            </Modal.Header>
            {isOpen ? (
              <UserForm key={user?.id ?? 'new'} user={user} onClose={onClose} onSaved={onSaved} />
            ) : null}
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}

function UserForm({
  user,
  onClose,
  onSaved,
}: {
  user?: EditableUser
  onClose: () => void
  onSaved: () => void
}) {
  const isCreate = !user
  const [form, setForm] = useState({
    name: user?.name ?? '',
    email: user?.email ?? '',
    link: user?.link ?? '',
    badge_name: user?.badge_name ?? '',
    badge_color: user?.badge_color ?? '',
    is_admin: user?.is_admin ?? false,
    receive_email: user?.receive_email ?? true,
    password: '',
  })
  const [saving, setSaving] = useState(false)
  const [showUA, setShowUA] = useState(false)

  const set = <K extends keyof typeof form>(key: K, value: (typeof form)[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  const submit = async () => {
    if (!form.name.trim() || !form.email.trim()) {
      toast.danger(t('moderationNoMessage'))
      return
    }
    setSaving(true)
    try {
      const payload = {
        ...form,
        email: form.email.trim(),
        name: form.name.trim(),
        badge_color:
          form.badge_name !== '' && form.badge_color === ''
            ? DEFAULT_BADGE_COLOR
            : form.badge_color,
      }
      const api = getArtalk().ctx.getApi()
      if (isCreate) {
        await api.users.createUser(payload)
      } else {
        await api.users.updateUser(user.id, payload)
      }
      onSaved()
      onClose()
    } catch (error) {
      toast.danger(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form
      className="min-h-0 space-y-3 overflow-y-auto px-1 pb-1"
      onSubmit={(event) => {
        event.preventDefault()
        void submit()
      }}
    >
      {!isCreate ? (
        <div className="grid grid-cols-2 gap-2 rounded-lg bg-surface-secondary/60 p-2.5 text-[11px]">
          <div>
            <span className="text-muted">{t('comments')}</span>
            <p className="mt-0.5 text-[12.5px] font-medium tabular-nums">
              {formatNumber(user.comment_count)}
            </p>
          </div>
          <div className="min-w-0">
            <span className="text-muted">{t('securityLastIp')}</span>
            <p className="mt-0.5 truncate font-mono text-[12px]">{user.last_ip || '—'}</p>
          </div>
          <div className="col-span-2 min-w-0">
            <span className="text-muted">{t('securityLastUA')}</span>
            <p className="mt-0.5 break-words text-[11.5px] leading-snug">
              {showUA || !user.last_ua ? (
                user.last_ua || '—'
              ) : (
                <button
                  type="button"
                  className="text-accent underline-offset-2 hover:underline"
                  onClick={() => setShowUA(true)}
                >
                  {t('show')}
                </button>
              )}
            </p>
          </div>
        </div>
      ) : null}

      <div className="grid grid-cols-2 gap-2">
        <TextField isRequired className="col-span-2">
          <Label className="text-[12px] text-muted">{t('username')}</Label>
          <Input
            value={form.name}
            onChange={(event) => set('name', event.target.value)}
            autoComplete="off"
          />
        </TextField>
        <TextField isRequired className="col-span-2">
          <Label className="text-[12px] text-muted">{t('email')}</Label>
          <Input
            type="email"
            value={form.email}
            onChange={(event) => set('email', event.target.value)}
            autoComplete="off"
          />
        </TextField>
        <TextField className="col-span-2">
          <Label className="text-[12px] text-muted">{t('link')}</Label>
          <Input
            value={form.link}
            onChange={(event) => set('link', event.target.value)}
            autoComplete="off"
            placeholder="https://"
          />
        </TextField>
        <TextField>
          <Label className="text-[12px] text-muted">{t('badgeText')}</Label>
          <Input
            value={form.badge_name}
            onChange={(event) => set('badge_name', event.target.value)}
            autoComplete="off"
          />
        </TextField>
        <div>
          <Label className="text-[12px] text-muted">{t('badgeColor')}</Label>
          <div className="mt-1 flex items-center gap-1.5">
            <span className="relative size-9 shrink-0 overflow-hidden rounded-lg border border-border">
              <input
                type="color"
                aria-label={t('badgeColor')}
                value={form.badge_color || DEFAULT_BADGE_COLOR}
                onChange={(event) => set('badge_color', event.target.value)}
                className="absolute -inset-2 size-[calc(100%+16px)] cursor-pointer border-0 bg-transparent p-0"
              />
            </span>
            <Input
              value={form.badge_color}
              onChange={(event) => set('badge_color', event.target.value)}
              placeholder={DEFAULT_BADGE_COLOR}
              className="font-mono text-[12px]"
            />
            <Button
              size="sm"
              variant="ghost"
              isIconOnly
              aria-label={t('badgeColor')}
              {...hoverTitle(t('badgeColor'))}
              onPress={() => set('badge_color', DEFAULT_BADGE_COLOR)}
            >
              <Shuffle className="size-3.5" aria-hidden />
            </Button>
          </div>
        </div>
      </div>

      <div className="space-y-2 rounded-lg border border-border p-2.5">
        <Switch isSelected={form.is_admin} onChange={(value) => set('is_admin', value)}>
          <Switch.Content>
            <Switch.Control>
              <Switch.Thumb />
            </Switch.Control>
            {t('admin')}
          </Switch.Content>
        </Switch>
        {/* A local password is what lets the account sign in without social
            login, so it is offered for ordinary users too — not just admins. */}
        <TextField>
          <Label className="text-[12px] text-muted">{t('password')}</Label>
          <Input
            type="text"
            value={form.password}
            onChange={(event) => set('password', event.target.value)}
            autoComplete="new-password"
            placeholder={isCreate ? '' : `(${t('passwordEmptyHint')})`}
          />
        </TextField>
        <Switch isSelected={form.receive_email} onChange={(value) => set('receive_email', value)}>
          <Switch.Content>
            <Switch.Control>
              <Switch.Thumb />
            </Switch.Control>
            {t('emailNotify')}
          </Switch.Content>
        </Switch>
      </div>

      <div className="flex justify-end gap-2 pt-1">
        <Button variant="ghost" size="sm" type="button" onPress={onClose}>
          {t('cancel')}
        </Button>
        <Button variant="primary" size="sm" type="submit" isDisabled={saving}>
          {saving ? <Spinner size="sm" /> : null}
          {t('save')}
        </Button>
      </div>
    </form>
  )
}
