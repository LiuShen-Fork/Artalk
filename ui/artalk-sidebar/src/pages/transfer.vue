<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'
import { artalk } from '../global'

const nav = useNavStore()
const user = useUserStore()
const { curtTab } = storeToRefs(nav)
const { t } = useI18n()

const importParams = ref({
  siteName: '',
  siteURL: '',
  payload: '',
})

const isLoading = ref(false)

const uploadApiURL = ref('')
const importTaskApiURL = ref('')

const uploadedFilename = ref('')

const importTaskStarted = ref(false)
const importTaskParams = ref<Record<string, string>>({})

const exportTaskStarted = ref(false)

onMounted(() => {
  nav.updateTabs(
    {
      import: 'import',
      export: 'export',
    },
    'import',
  )
  watch(curtTab, (tab) => {
    if (tab === 'export') {
      startExportTask()
      curtTab.value = 'import'
    }
  })

  uploadApiURL.value = `${artalk?.ctx.conf.server}/api/v2/transfer/upload`
  importTaskApiURL.value = `${artalk?.ctx.conf.server}/api/v2/transfer/import`
})

function setError(msg: string) {
  window.alert(msg)
}

function fileUploaded(filename: string) {
  uploadedFilename.value = filename
}

function startImportTask() {
  if (!uploadedFilename.value) {
    setError(`Please upload a data file first`)
    return
  }

  const p = importParams.value
  const siteName = p.siteName.trim()
  const siteURL = p.siteURL.trim()
  const payload = p.payload.trim()

  // Prepare request params
  let rData: any = {}
  if (payload) {
    // Validate payload JSON
    try {
      rData = JSON.parse(payload)
    } catch (err) {
      setError(`Payload JSON invalid: ${err}`)
      return
    }

    if (typeof rData !== 'object' || Array.isArray(rData)) {
      setError(`Payload should be an object`)
      return
    }
  }
  if (siteName) rData.target_site_name = siteName
  if (siteURL) rData.target_site_url = siteURL
  rData.json_file = uploadedFilename.value

  // Create an import task
  importTaskParams.value = {
    ...rData,
    token: user.token,
  }
  importTaskStarted.value = true
}

function importTaskDone() {
  importTaskStarted.value = false
}

async function startExportTask() {
  if (exportTaskStarted.value) return
  exportTaskStarted.value = true
  isLoading.value = true
  try {
    const res = await artalk!.ctx.getApi().transfer.exportArtrans()
    downloadFile(`backup-${getYmdHisFilename()}.artrans`, res.data.artrans)
  } catch (err: any) {
    console.error(err)
    window.alert(`${String(err)}`)
    return
  } finally {
    exportTaskStarted.value = false
    isLoading.value = false
  }
}

function downloadFile(filename: string, text: string) {
  const el = document.createElement('a')
  el.setAttribute('href', `data:text/json;charset=utf-8,${encodeURIComponent(text)}`)
  el.setAttribute('download', filename)
  el.style.display = 'none'
  document.body.appendChild(el)
  el.click()
  document.body.removeChild(el)
}

function getYmdHisFilename() {
  const date = new Date()

  const year = date.getFullYear()
  const month = date.getMonth() + 1
  const day = date.getDate()
  const hours = date.getHours()
  const minutes = date.getMinutes()
  const seconds = date.getSeconds()

  return `${year}${month}${day}-${hours}${padWithZeros(minutes, 2)}${padWithZeros(seconds, 2)}`
}

function padWithZeros(vNumber: number, width: number) {
  let numAsString = vNumber.toString()
  while (numAsString.length < width) {
    numAsString = `0${numAsString}`
  }
  return numAsString
}

const artransferToolHint = computed(() =>
  t('artransferToolHint', { link: '__LINK__' }).replace(
    '__LINK__',
    `<a href="https://artalk.js.org/guide/transfer.html" target="_blank">${t('artransfer')}</a>`,
  ),
)
</script>

<template>
  <LoadingLayer v-if="isLoading" />
  <LogTerminal
    v-if="importTaskStarted"
    :api-url="importTaskApiURL"
    :req-params="importTaskParams"
    @back="importTaskDone()"
  />
  <div v-show="!importTaskStarted" class="transfer-page admin-page">
    <AdminPageHeader :title="t('transfer')" />
    <div class="atk-form admin-panel admin-form">
      <div class="atk-label atk-data-file-label">Artrans {{ t('dataFile') }}</div>
      <FileUploader :api-url="uploadApiURL" @done="fileUploaded">
        <template #tip>
          <!-- eslint-disable-next-line vue/no-v-html -->
          <span v-html="artransferToolHint" />
        </template>
        <template #done-msg>
          {{ t('uploadReadyToImport') }}
        </template>
      </FileUploader>
      <div class="atk-label">{{ t('targetSiteName') }}</div>
      <input
        v-model="importParams.siteName"
        type="text"
        name="AtkSiteName"
        :placeholder="t('inputHint')"
        autocomplete="off"
      />
      <div class="atk-label">{{ t('targetSiteURL') }}</div>
      <input
        v-model="importParams.siteURL"
        type="text"
        name="AtkSiteURL"
        :placeholder="t('inputHint')"
        autocomplete="off"
      />
      <div class="atk-label">{{ t('payload') }} ({{ t('optional') }})</div>
      <textarea
        v-model="importParams.payload"
        name="AtkPayload"
        :placeholder="t('inputHint')"
      ></textarea>
      <div class="transfer-foot">
        <a class="transfer-detail" href="https://artalk.js.org/guide/transfer.html" target="_blank">
          {{ t('moreDetails') }}
        </a>
        <button
          type="button"
          class="admin-button primary transfer-submit"
          name="AtkSubmit"
          @click="startImportTask()"
        >
          {{ t('import') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.transfer-page {
  .admin-form {
    display: flex;
    max-width: 760px;
    flex-direction: column;
    padding: 22px;
  }

  .atk-label {
    margin: 20px 0 8px;
    color: var(--atk-admin-subtle);
    font-size: 12.5px;
    font-weight: 650;
    letter-spacing: 0.01em;

    &:first-child {
      margin-top: 0;
    }
  }

  input[type='text'],
  textarea {
    display: block;
    width: 100%;
    min-height: 42px;
    margin: 0;
    padding: 9px 13px;
    border: 1px solid var(--atk-admin-border);
    line-height: normal;
    border-radius: var(--atk-admin-radius-xs);
    color: var(--atk-admin-ink);
    background: var(--atk-admin-surface-muted);
    font: inherit;
    font-size: 13px;
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;

    &::placeholder {
      color: var(--atk-admin-faint);
    }

    &:focus {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-surface);
      box-shadow: 0 0 0 3px var(--atk-admin-ring);
      outline: none;
    }
  }

  textarea {
    min-height: 118px;
    padding: 11px 13px;
    line-height: 1.6;
    resize: vertical;
  }

  .transfer-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    margin-top: 20px;
    padding-top: 18px;
    border-top: 1px solid var(--atk-admin-border);
  }

  .transfer-detail {
    color: var(--atk-admin-sage);
    font-size: 12.5px;
    font-weight: 550;
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }

  .transfer-submit {
    min-height: 40px;
    padding: 0 22px;
  }
}

@media (max-width: 560px) {
  .transfer-page {
    .admin-form {
      padding: 18px 16px;
    }

    .transfer-foot {
      flex-direction: column-reverse;
      align-items: stretch;
    }

    .transfer-submit {
      width: 100%;
    }

    .transfer-detail {
      align-self: center;
    }
  }
}
</style>
