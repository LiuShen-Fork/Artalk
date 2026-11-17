import { Button, Input, Label, Modal, Spinner, TextField, TextArea, toast } from '@heroui/react'
import type { ArtalkType } from 'artalk'
import { ExternalLink, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useConfirm } from '@/components/ui/confirm'
import { errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import { getArtalk } from '@/lib/artalk'

const parseURLs = (value: string) =>
  value
    .split(/[\s,]+/)
    .map((item) => item.trim())
    .filter(Boolean)

/** Sites group comments by domain; a site can own several URLs. */
export function SiteCreateDialog({
  isOpen,
  onClose,
  onCreated,
}: {
  isOpen: boolean
  onClose: () => void
  onCreated: () => void
}) {
  const [name, setName] = useState('')
  const [urls, setURLs] = useState('')
  const [saving, setSaving] = useState(false)

  const submit = async () => {
    if (!name.trim()) {
      toast.danger(t('siteNameInputHint'))
      return
    }
    setSaving(true)
    try {
      await getArtalk()
        .ctx.getApi()
        .sites.createSite({ name: name.trim(), urls: parseURLs(urls) })
      onCreated()
      onClose()
    } catch (error) {
      toast.danger(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal isOpen={isOpen} onOpenChange={(open) => !open && onClose()}>
      <Modal.Backdrop>
        <Modal.Container>
          <Modal.Dialog className="gap-3">
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{t('createSite')}</Modal.Heading>
            </Modal.Header>
            <form
              className="space-y-3 px-1 pb-1"
              onSubmit={(event) => {
                event.preventDefault()
                void submit()
              }}
            >
              <TextField isRequired>
                <Label className="text-[12px] text-muted">{t('siteName')}</Label>
                <Input
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  autoComplete="off"
                />
              </TextField>
              <TextField>
                <Label className="text-[12px] text-muted">{t('siteUrls')}</Label>
                <TextArea
                  value={urls}
                  onChange={(event) => setURLs(event.target.value)}
                  rows={3}
                  placeholder={`https://a.com\nhttps://b.com`}
                  className="font-mono text-[12px]"
                />
              </TextField>
              <p className="text-[11px] text-muted">{t('multiSepHint')}</p>
              <div className="flex justify-end gap-2 pt-1">
                <Button variant="ghost" size="sm" type="button" onPress={onClose}>
                  {t('cancel')}
                </Button>
                <Button variant="primary" size="sm" type="submit" isDisabled={saving}>
                  {saving ? <Spinner size="sm" /> : null}
                  {t('add')}
                </Button>
              </div>
            </form>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}

export function SiteEditorDialog({
  site,
  onClose,
  onSaved,
}: {
  site?: ArtalkType.SiteData
  onClose: () => void
  onSaved: () => void
}) {
  return (
    <Modal isOpen={!!site} onOpenChange={(open) => !open && onClose()}>
      <Modal.Backdrop>
        <Modal.Container>
          <Modal.Dialog className="gap-3">
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>{site?.name}</Modal.Heading>
            </Modal.Header>
            {site ? (
              <SiteForm key={site.id} site={site} onClose={onClose} onSaved={onSaved} />
            ) : null}
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}

function SiteForm({
  site,
  onClose,
  onSaved,
}: {
  site: ArtalkType.SiteData
  onClose: () => void
  onSaved: () => void
}) {
  const confirm = useConfirm()
  const [name, setName] = useState(site.name || '')
  const [urls, setURLs] = useState(site.urls_raw || (site.urls ?? []).join('\n'))
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setSaving(true)
    try {
      await getArtalk()
        .ctx.getApi()
        .sites.updateSite(site.id, { ...site, name: name.trim(), urls: parseURLs(urls) })
      toast.success(t('updateComplete'))
      onSaved()
      onClose()
    } catch (error) {
      toast.danger(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!(await confirm({ title: t('siteDeleteConfirm', { name: site.name }), danger: true }))) {
      return
    }
    setSaving(true)
    try {
      await getArtalk().ctx.getApi().sites.deleteSite(site.id)
      onSaved()
      onClose()
    } catch (error) {
      toast.danger(errorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3 px-1 pb-1">
      <TextField isRequired>
        <Label className="text-[12px] text-muted">{t('siteName')}</Label>
        <Input value={name} onChange={(event) => setName(event.target.value)} />
      </TextField>
      <TextField>
        <Label className="text-[12px] text-muted">{t('siteUrls')}</Label>
        <TextArea
          value={urls}
          onChange={(event) => setURLs(event.target.value)}
          rows={4}
          className="font-mono text-[12px]"
        />
      </TextField>
      <p className="text-[11px] text-muted">{t('multiSepHint')}</p>

      <div className="flex items-center justify-between gap-2 pt-1">
        <div className="flex gap-1.5">
          <Button
            size="sm"
            variant="ghost"
            isDisabled={saving || !site.urls?.length}
            aria-label={t('view')}
            onPress={() => site.urls.length && window.open(site.urls[0])}
          >
            <ExternalLink className="size-3.5" aria-hidden />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            isDisabled={saving}
            aria-label={t('delete')}
            onPress={() => void remove()}
          >
            <Trash2 className="size-3.5" aria-hidden />
          </Button>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" onPress={onClose}>
            {t('cancel')}
          </Button>
          <Button variant="primary" size="sm" isDisabled={saving} onPress={() => void save()}>
            {saving ? <Spinner size="sm" /> : null}
            {t('save')}
          </Button>
        </div>
      </div>
    </div>
  )
}
