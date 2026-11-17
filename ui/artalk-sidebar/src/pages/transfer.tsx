import { Button, Input, Label, Spinner, TextArea, TextField, toast } from '@heroui/react'
import { ArrowLeft, ArrowRight, Download, ExternalLink } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { FileUploader } from '@/components/FileUploader'
import { errorMessage } from '@/lib/api'
import { t } from '@/lib/i18n'
import { getArtalk } from '@/lib/artalk'
import { session } from '@/lib/session'

const serverBase = () => `${getArtalk().ctx.conf.server}/api/v2`

export function TransferPage() {
  const [filename, setFilename] = useState('')
  const [siteName, setSiteName] = useState('')
  const [siteURL, setSiteURL] = useState('')
  const [payload, setPayload] = useState('')
  const [task, setTask] = useState<Record<string, string> | null>(null)
  const [exporting, setExporting] = useState(false)

  const startImport = () => {
    if (!filename) {
      toast.danger(t('uploadReadyToImport'))
      return
    }

    let data: Record<string, unknown> = {}
    if (payload.trim()) {
      try {
        data = JSON.parse(payload)
      } catch (error) {
        toast.danger(t('payloadInvalid', { error: String(error) }))
        return
      }
      if (typeof data !== 'object' || Array.isArray(data) || data === null) {
        toast.danger(t('payloadInvalid', { error: 'object expected' }))
        return
      }
    }
    if (siteName.trim()) data.target_site_name = siteName.trim()
    if (siteURL.trim()) data.target_site_url = siteURL.trim()
    data.json_file = filename

    const params: Record<string, string> = { token: session.get().user.token ?? '' }
    Object.entries(data).forEach(([key, value]) => {
      params[key] = String(value)
    })
    setTask(params)
  }

  const exportData = async () => {
    setExporting(true)
    try {
      const res = await getArtalk().ctx.getApi().transfer.exportArtrans()
      const stamp = new Date().toISOString().replace(/[-:T]/g, '').slice(0, 14)
      const link = document.createElement('a')
      link.href = `data:text/json;charset=utf-8,${encodeURIComponent(res.data.artrans)}`
      link.download = `backup-${stamp}.artrans`
      document.body.appendChild(link)
      link.click()
      link.remove()
    } catch (error) {
      toast.danger(errorMessage(error) || t('opFailed'))
    } finally {
      setExporting(false)
    }
  }

  if (task) {
    return <ImportTerminal params={task} onBack={() => setTask(null)} />
  }

  return (
    <div className="space-y-3">
      <PageHeader title={t('transfer')} description={t('transferDesc')} />

      <section className="space-y-3 rounded-xl border border-border bg-surface p-3">
        <div>
          <h2 className="text-[13px] font-semibold">{t('import')}</h2>
          <p className="mt-0.5 text-[11.5px] text-muted">{t('importDesc')}</p>
        </div>

        <FileUploader
          apiUrl={`${serverBase()}/transfer/upload`}
          onDone={setFilename}
          onError={(message) => toast.danger(message)}
        />

        <TextField className="w-full">
          <Label className="text-[12px] text-muted">{t('targetSiteName')}</Label>
          <Input
            value={siteName}
            onChange={(event) => setSiteName(event.target.value)}
            placeholder={t('inputHint')}
            autoComplete="off"
          />
        </TextField>
        <TextField className="w-full">
          <Label className="text-[12px] text-muted">{t('targetSiteURL')}</Label>
          <Input
            value={siteURL}
            onChange={(event) => setSiteURL(event.target.value)}
            placeholder={t('inputHint')}
            autoComplete="off"
          />
        </TextField>
        <TextField className="w-full">
          <Label className="text-[12px] text-muted">
            {t('payload')} ({t('optional')})
          </Label>
          <TextArea
            value={payload}
            onChange={(event) => setPayload(event.target.value)}
            rows={3}
            placeholder="{}"
            className="font-mono text-[12px]"
          />
        </TextField>

        <div className="flex items-center justify-between gap-2 border-t border-border pt-3">
          <a
            href="https://artalk.js.org/guide/transfer.html"
            target="_blank"
            rel="noreferrer noopener"
            className="flex items-center gap-1 text-[11.5px] text-accent hover:underline"
          >
            {t('moreDetails')}
            <ExternalLink className="size-3" aria-hidden />
          </a>
          <Button size="sm" variant="primary" onPress={startImport}>
            {t('import')}
            <ArrowRight className="size-3.5" aria-hidden />
          </Button>
        </div>
      </section>

      <section className="flex items-center justify-between gap-3 rounded-xl border border-border bg-surface p-3">
        <div className="min-w-0">
          <h2 className="text-[13px] font-semibold">{t('export')}</h2>
          <p className="mt-0.5 text-[11.5px] text-muted">{t('exportDesc')}</p>
        </div>
        <Button
          size="sm"
          variant="secondary"
          isDisabled={exporting}
          onPress={() => void exportData()}
        >
          {exporting ? <Spinner size="sm" /> : <Download className="size-3.5" aria-hidden />}
          {t('export')}
        </Button>
      </section>
    </div>
  )
}

/**
 * The import endpoint streams a plain-text progress page. Posting it into a
 * hidden iframe keeps that stream out of the SPA's own request pipeline while
 * still rendering the log the server emits.
 */
function ImportTerminal({
  params,
  onBack,
}: {
  params: Record<string, string>
  onBack: () => void
}) {
  const wrapRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const host = wrapRef.current
    if (!host) return

    const frameName = `f_${Date.now()}`
    const frame = document.createElement('iframe')
    frame.name = frameName
    frame.className = 'h-full w-full border-0 bg-white'
    frame.title = t('import')
    host.appendChild(frame)

    const form = document.createElement('form')
    form.method = 'POST'
    form.action = `${serverBase()}/transfer/import`
    form.target = frameName
    form.style.display = 'none'
    Object.entries(params).forEach(([key, value]) => {
      const input = document.createElement('input')
      input.type = 'hidden'
      input.name = key
      input.value = value
      form.appendChild(input)
    })

    host.appendChild(form)
    form.submit()
    form.remove()

    return () => frame.remove()
  }, [params])

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <button
        type="button"
        onClick={onBack}
        className="flex items-center gap-1 self-start rounded-lg px-1.5 py-1 text-[12px] text-muted transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        {t('back')}
      </button>
      <div
        ref={wrapRef}
        className="min-h-0 flex-1 overflow-hidden rounded-xl border border-border bg-white"
      />
    </div>
  )
}
