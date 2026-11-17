import { Button, ProgressBar } from '@heroui/react'
import { Check, Upload } from 'lucide-react'
import { useRef, useState } from 'react'
import { t } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import { session } from '@/lib/session'

interface FileUploaderProps {
  apiUrl: string
  onDone: (filename: string) => void
  onError?: (message: string) => void
}

/**
 * `.artrans` uploader. XHR instead of fetch purely for upload progress; the
 * server answers with the stored file name, which the import form then sends.
 */
export function FileUploader({ apiUrl, onDone, onError }: FileUploaderProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const xhrRef = useRef<XMLHttpRequest | null>(null)
  const [dragging, setDragging] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [localName, setLocalName] = useState('')
  const [done, setDone] = useState(false)

  const reset = () => {
    setUploading(false)
    setDragging(false)
    setProgress(0)
    setDone(false)
    setLocalName('')
    if (inputRef.current) inputRef.current.value = ''
    onDone('')
  }

  const fail = (message: string) => {
    reset()
    onError?.(message)
  }

  const upload = (file: File) => {
    xhrRef.current?.abort()
    if (!file.name.toLowerCase().endsWith('.artrans')) {
      fail(t('fileFormatHint'))
      return
    }

    onDone('')
    setLocalName(file.name)
    setUploading(true)
    setProgress(0)

    const request = new XMLHttpRequest()
    xhrRef.current = request

    request.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable && event.total > 0) {
        setProgress(Math.min(99, Math.round((event.loaded / event.total) * 100)))
      }
    })

    request.onload = () => {
      if (request.status < 200 || request.status > 299) {
        fail(`HTTP ${request.status}`)
        return
      }
      try {
        const response = JSON.parse(request.response) as { filename?: string }
        if (!response.filename) throw new Error('missing filename')
        setProgress(100)
        setDone(true)
        setUploading(false)
        xhrRef.current = null
        onDone(response.filename)
      } catch {
        fail(t('opFailed'))
      }
    }
    request.onerror = () => fail(t('opFailed'))
    request.ontimeout = () => fail(t('opFailed'))

    const form = new FormData()
    form.append('file', file)
    form.append('token', session.get().user.token ?? '')
    request.open('POST', apiUrl)
    request.timeout = 120_000
    request.send(form)
  }

  const abort = () => {
    xhrRef.current?.abort()
    xhrRef.current = null
    reset()
  }

  return (
    <div className="space-y-1.5">
      {uploading ? (
        <div className="rounded-xl border border-border bg-surface p-3">
          <ProgressBar value={progress} aria-label={t('uploading')} />
          <div className="mt-2 flex items-center justify-between text-[11.5px] text-muted">
            <span>
              {t('uploading')} {progress}%
            </span>
            <Button size="sm" variant="ghost" onPress={abort}>
              {t('cancel')}
            </Button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          onDragOver={(event) => {
            event.preventDefault()
            setDragging(true)
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(event) => {
            event.preventDefault()
            setDragging(false)
            const file = event.dataTransfer.files[0]
            if (file) upload(file)
          }}
          className={cn(
            'flex w-full items-center gap-2.5 rounded-xl border border-dashed px-3 py-3 text-left transition-colors',
            dragging
              ? 'border-accent bg-accent/5'
              : 'border-border bg-surface hover:border-accent/40',
            done && 'border-success/40',
          )}
        >
          <span
            className={cn(
              'grid size-9 shrink-0 place-items-center rounded-lg',
              done ? 'bg-success/15 text-success' : 'bg-default text-muted',
            )}
          >
            {done ? (
              <Check className="size-4" aria-hidden />
            ) : (
              <Upload className="size-4" aria-hidden />
            )}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-[12.5px] font-medium">
              {done ? localName : t('chooseFile')}
            </span>
            <span className="block truncate text-[11px] text-muted">
              {done ? t('uploadReadyToImport') : '.artrans'}
            </span>
          </span>
          <span className="shrink-0 text-[11.5px] text-accent">{done ? t('chooseFile') : ''}</span>
        </button>
      )}
      <input
        ref={inputRef}
        type="file"
        accept=".artrans"
        aria-label={t('chooseFile')}
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0]
          if (file) upload(file)
        }}
      />
    </div>
  )
}
