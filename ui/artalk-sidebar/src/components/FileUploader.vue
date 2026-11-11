<script setup lang="ts">
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const user = useUserStore()

const props = defineProps<{
  apiUrl: string
}>()

const emit = defineEmits<{
  /** Upload done event */
  (evt: 'done', filename: string): void
}>()

const { apiUrl } = toRefs(props)

let xhr: XMLHttpRequest | null = null
const fileInputEl = ref<HTMLInputElement | null>(null)
const remoteFilename = ref('')
const localFilename = ref('')
const isUploading = ref(false)
const isDragging = ref(false)
const progress = ref(0)
const isDone = ref(false)

function reset() {
  remoteFilename.value = ''
  localFilename.value = ''
  isUploading.value = false
  isDragging.value = false
  progress.value = 0
  isDone.value = false
  if (fileInputEl.value) fileInputEl.value.value = ''
}

function startUploadFile(file: File) {
  remoteFilename.value = ''
  localFilename.value = file.name

  const request = new XMLHttpRequest()
  xhr = request

  const setErr = (message: string) => {
    if (xhr !== request) return
    xhr = null
    reset()
    emit('done', '')
    alert(`File upload failed: ${message}`)
  }

  request.upload.addEventListener('progress', (event) => {
    if (event.lengthComputable && event.total > 0) {
      progress.value = Math.min(99, Math.round((event.loaded / event.total) * 100))
    }
  })

  request.onload = () => {
    if (xhr !== request) return
    if (request.status < 200 || request.status > 299) {
      setErr(`Response HTTP Code: ${request.status}`)
      return
    }

    let response: { filename?: unknown }
    try {
      response = JSON.parse(request.response) as { filename?: unknown }
    } catch {
      setErr('Invalid server response')
      return
    }

    if (typeof response.filename !== 'string' || !response.filename) {
      setErr('The server did not return a file name')
      return
    }

    progress.value = 100
    remoteFilename.value = response.filename
    isDone.value = true
    isUploading.value = false
    xhr = null
    emit('done', remoteFilename.value)
  }

  request.onerror = () => setErr('Network error')
  request.ontimeout = () => setErr('Upload timed out')
  request.onabort = () => {
    if (xhr === request) {
      reset()
      emit('done', '')
    }
  }

  const formData = new FormData()
  formData.append('file', file)
  formData.append('token', user.token)

  request.open('post', apiUrl.value)
  request.timeout = 120_000
  request.send(formData)
}

function startUpload(file: File) {
  const previousRequest = xhr
  xhr = null
  previousRequest?.abort()

  if (!file.name.toLowerCase().endsWith('.artrans')) {
    reset()
    emit('done', '')
    alert('Please choose an .artrans file')
    return
  }

  emit('done', '')
  isUploading.value = true
  startUploadFile(file)
}

function onFileInputChange() {
  const file = fileInputEl.value?.files?.[0]
  if (!file) return
  startUpload(file)
}

function onDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
  isDragging.value = true
}

function onDragLeave(event: DragEvent) {
  if (
    event.currentTarget instanceof HTMLElement &&
    event.currentTarget.contains(event.relatedTarget as Node)
  ) {
    return
  }
  isDragging.value = false
}

function onDrop(event: DragEvent) {
  event.preventDefault()
  isDragging.value = false
  const file = event.dataTransfer?.files[0]
  if (file) startUpload(file)
}

function abortUpload() {
  const request = xhr
  xhr = null
  reset()
  request?.abort()
  emit('done', '')
}
</script>

<template>
  <div class="atk-file-upload-group">
    <div
      v-show="!isUploading"
      class="upload-zone atk-fade-in"
      :class="{ done: isDone, dragging: isDragging }"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <input
        ref="fileInputEl"
        class="upload-input"
        type="file"
        name="AtkDataFile"
        :aria-label="t('chooseFile')"
        accept=".artrans"
        @change="onFileInputChange()"
      />
      <span class="upload-icon" aria-hidden="true">
        <svg v-if="!isDone" viewBox="0 0 24 24" fill="none">
          <path
            d="M12 15.5V5.5m0 0 4 4m-4-4-4 4M5 15.5v2.2A2.3 2.3 0 0 0 7.3 20h9.4a2.3 2.3 0 0 0 2.3-2.3v-2.2"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none">
          <path
            d="m6.5 12.5 3.5 3.5 7.5-8"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </span>
      <span class="upload-copy">
        <strong>{{ isDone ? localFilename : t('chooseFile') }}</strong>
        <span v-if="isDone"><slot name="done-msg" /></span>
        <span v-else class="upload-format">.artrans</span>
      </span>
      <span v-if="isDone" class="upload-replace">{{ t('chooseFile') }}</span>
    </div>
    <div v-if="!isDone" v-show="!isUploading" class="upload-tip">
      <slot name="tip" />
    </div>
    <div v-show="isUploading" class="uploading-wrap atk-fade-in">
      <div class="upload-progress">
        <div class="upload-bar" :style="{ width: `${progress}%` }"></div>
      </div>
      <div class="upload-status">
        <span>{{ t('uploading') }} {{ progress }}%</span>
        <button type="button" class="upload-abort" @click="abortUpload()">
          {{ t('cancel') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.atk-file-upload-group {
  display: flex;
  height: auto;
  min-height: 0;
  flex-direction: column;
  gap: 10px;
  overflow: visible;
}

.upload-zone {
  position: relative;
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 84px;
  padding: 16px 18px;
  border: 1px dashed var(--atk-admin-border-strong);
  border-radius: var(--atk-admin-radius-sm);
  background: var(--atk-admin-surface-muted);
  cursor: pointer;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease;

  &:hover,
  &:focus-within,
  &.dragging {
    border-color: var(--atk-admin-sage);
    background: var(--atk-admin-sage-soft);
  }

  &.dragging {
    border-style: solid;
    box-shadow: 0 0 0 3px var(--atk-admin-ring);
  }

  &.done {
    border-style: solid;
    border-color: var(--atk-admin-sage);
    background: var(--atk-admin-sage-soft);
  }
}

.upload-input {
  position: absolute;
  inset: 0;
  z-index: 1;
  width: 100%;
  height: 100%;
  opacity: 0;
  cursor: pointer;
}

.upload-icon {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border-radius: 50%;
  color: var(--atk-admin-sage-strong);
  background: var(--atk-admin-surface);
  box-shadow: 0 0 0 1px var(--atk-admin-border);

  svg {
    width: 21px;
    height: 21px;
  }

  .done & {
    color: var(--atk-admin-sage-strong);
    background: var(--atk-admin-surface);
  }
}

.upload-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 4px;

  strong {
    overflow: hidden;
    color: var(--atk-admin-ink);
    font-size: 13.5px;
    font-weight: 650;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    overflow: hidden;
    color: var(--atk-admin-subtle);
    font-size: 12px;
    line-height: 1.5;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.upload-format {
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.upload-replace {
  flex: none;
  padding-left: 8px;
  color: var(--atk-admin-sage-strong);
  font-size: 12px;
  font-weight: 600;
}

.upload-tip {
  color: var(--atk-admin-faint);
  font-size: 12px;
  line-height: 1.6;

  :deep(a) {
    color: var(--atk-admin-sage);
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}

.uploading-wrap {
  padding: 14px 16px;
  border: 1px solid var(--atk-admin-border);
  border-radius: var(--atk-admin-radius-sm);
  background: var(--atk-admin-surface-muted);
}

.upload-progress {
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--atk-admin-surface-sunken);
}

.upload-bar {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--atk-admin-sage), var(--atk-admin-sage-strong));
  transition: width 0.2s ease;
}

.upload-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 9px;
  color: var(--atk-admin-subtle);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.upload-abort {
  padding: 0;
  border: 0;
  color: var(--atk-admin-danger);
  background: transparent;
  font-size: 12px;
  cursor: pointer;

  &:hover {
    text-decoration: underline;
  }
}
</style>
