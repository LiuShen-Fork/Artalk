import { Button, Input, Label, Modal, Spinner, Switch, TextField, toast } from '@heroui/react'
import { ExternalLink, RotateCw, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useConfirm } from '@/components/ui/confirm'
import { errorMessage, type PageData } from '@/lib/api'
import { t } from '@/lib/i18n'
import { getArtalk } from '@/lib/artalk'

interface PageEditorDialogProps {
  page?: PageData
  onClose: () => void
  onSaved: () => void
}

/**
 * Edit a single page: its title, its key (the identifier comments are stored
 * under) and whether only admins may comment on it. The title can also be
 * re-scraped from the live URL, which is what the "sync" action does.
 */
export function PageEditorDialog({ page, onClose, onSaved }: PageEditorDialogProps) {
  return (
    <Modal isOpen={!!page} onOpenChange={(open) => !open && onClose()}>
      <Modal.Backdrop>
        <Modal.Container>
          <Modal.Dialog className="gap-3">
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t('editTitle')}</Modal.Heading>
            </Modal.Header>
            {page ? (
              <PageForm key={page.id} page={page} onClose={onClose} onSaved={onSaved} />
            ) : null}
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}

function PageForm({
  page,
  onClose,
  onSaved,
}: {
  page: PageData
  onClose: () => void
  onSaved: () => void
}) {
  const confirm = useConfirm()
  const [title, setTitle] = useState(page.title || '')
  const [key, setKey] = useState(page.key || '')
  const [adminOnly, setAdminOnly] = useState(!!page.admin_only)
  const [busy, setBusy] = useState<string | null>(null)

  const handleError = (error: unknown) => toast.danger(errorMessage(error))

  const save = async () => {
    setBusy('save')
    try {
      await getArtalk()
        .ctx.getApi()
        .pages.updatePage(page.id, { ...page, title, key, admin_only: adminOnly })
      toast.success(t('updateComplete'))
      onSaved()
      onClose()
    } catch (error) {
      handleError(error)
    } finally {
      setBusy(null)
    }
  }

  const sync = async () => {
    setBusy('sync')
    try {
      await getArtalk().ctx.getApi().pages.fetchPage(page.id)
      toast.success(t('updateComplete'))
      onSaved()
      onClose()
    } catch (error) {
      handleError(error)
    } finally {
      setBusy(null)
    }
  }

  const remove = async () => {
    if (
      !(await confirm({
        title: t('pageDeleteConfirm', { title: page.title || page.key }),
        danger: true,
      }))
    ) {
      return
    }
    setBusy('delete')
    try {
      await getArtalk().ctx.getApi().pages.deletePage(page.id)
      onSaved()
      onClose()
    } catch (error) {
      handleError(error)
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="min-h-0 space-y-3 overflow-y-auto px-1 pb-1">
      {page.url ? (
        <a
          href={page.url}
          target="_blank"
          rel="noreferrer noopener"
          className="flex items-center gap-1 break-all rounded-lg bg-surface-secondary/60 px-2.5 py-2 text-[11.5px] text-muted hover:text-accent"
        >
          <ExternalLink className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">{page.url}</span>
        </a>
      ) : null}

      <TextField className="w-full">
        <Label className="text-[12px] text-muted">{t('editTitle')}</Label>
        <Input value={title} onChange={(event) => setTitle(event.target.value)} />
      </TextField>
      <TextField className="w-full">
        <Label className="text-[12px] text-muted">{t('switchKey')}</Label>
        <Input
          value={key}
          onChange={(event) => setKey(event.target.value)}
          className="font-mono text-[12px]"
        />
      </TextField>
      <Switch isSelected={adminOnly} onChange={setAdminOnly}>
        <Switch.Content>
          <Switch.Control>
            <Switch.Thumb />
          </Switch.Control>
          {adminOnly ? t('commentOnlyAdmin') : t('commentAllowAll')}
        </Switch.Content>
      </Switch>

      <p className="text-[11px] text-muted">
        {t('comments')} · {page.comment_count ?? 0} · PV · {page.pv ?? 0}
      </p>

      <div className="flex items-center justify-between gap-2 pt-1">
        <div className="flex gap-1.5">
          <Button
            size="sm"
            variant="ghost"
            isDisabled={!!busy}
            onPress={() => void sync()}
            aria-label={t('refresh')}
          >
            {busy === 'sync' ? (
              <Spinner size="sm" />
            ) : (
              <RotateCw className="size-3.5" aria-hidden />
            )}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            isDisabled={!!busy}
            onPress={() => void remove()}
            aria-label={t('delete')}
          >
            {busy === 'delete' ? (
              <Spinner size="sm" />
            ) : (
              <Trash2 className="size-3.5" aria-hidden />
            )}
          </Button>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" onPress={onClose}>
            {t('cancel')}
          </Button>
          <Button variant="primary" size="sm" isDisabled={!!busy} onPress={() => void save()}>
            {busy === 'save' ? <Spinner size="sm" /> : null}
            {t('save')}
          </Button>
        </div>
      </div>
    </div>
  )
}
