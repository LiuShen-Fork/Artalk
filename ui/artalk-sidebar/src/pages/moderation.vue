<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'

type ModerationStatus = 'pass' | 'block' | 'error'
type ModerationTab = 'all' | 'block' | 'replace' | 'error'

type ModerationLog = {
  id: number
  comment_id: number
  site_name: string
  page_key: string
  checker: string
  status: ModerationStatus
  action: string
  message: string
  date: string
  comment_content: string
  comment_pending: boolean
  user_name: string
  user_email: string
}

type ModerationResponse = {
  count: number
  logs: ModerationLog[]
}

const nav = useNavStore()
const user = useUserStore()
const route = useRoute()
const { curtTab } = storeToRefs(nav)
const { site: curtSite } = storeToRefs(user)
const { t } = useI18n()

const logs = ref<ModerationLog[]>([])
const total = ref(0)
const loading = ref(false)
const clearing = ref(false)
const deletingID = ref<number | null>(null)

const statusMeta = computed(() => ({
  replace: { label: t('moderationReplaced'), icon: '~' },
  block: { label: t('moderationBlocked'), icon: '!' },
  error: { label: t('opFailed'), icon: 'X' },
}))

function getRouteStatus(): ModerationTab {
  const status = route.query.status
  return typeof status === 'string' && ['block', 'replace', 'error'].includes(status)
    ? (status as ModerationTab)
    : 'all'
}

function displayStatus(item: ModerationLog): Exclude<ModerationTab, 'all'> {
  if (item.action === 'replace' || item.status === 'pass') return 'replace'
  return item.status
}

async function fetchLogs() {
  loading.value = true
  try {
    const res = await artalk!.ctx.getApi().request<ModerationResponse>({
      path: '/moderation/logs',
      method: 'GET',
      query: {
        site_name: curtSite.value || undefined,
        status: curtTab.value && curtTab.value !== 'all' ? curtTab.value : undefined,
        limit: 80,
      },
      secure: true,
      format: 'json',
    })
    logs.value = res.data.logs || []
    total.value = res.data.count || 0
  } finally {
    loading.value = false
  }
}

async function deleteLog(item: ModerationLog) {
  if (!window.confirm(t('moderationDeleteConfirm'))) return

  deletingID.value = item.id
  try {
    await artalk!.ctx.getApi().request({
      path: `/moderation/logs/${item.id}`,
      method: 'DELETE',
      secure: true,
      format: 'json',
    })
    logs.value = logs.value.filter((log) => log.id !== item.id)
    total.value = Math.max(0, total.value - 1)
  } catch (err: any) {
    alert(err?.message || t('opFailed'))
  } finally {
    deletingID.value = null
  }
}

async function clearLogs() {
  if (!window.confirm(t('moderationClearConfirm'))) return

  clearing.value = true
  try {
    await artalk!.ctx.getApi().request({
      path: '/moderation/logs',
      method: 'DELETE',
      query: { site_name: curtSite.value || undefined },
      secure: true,
      format: 'json',
    })
    logs.value = []
    total.value = 0
  } catch (err: any) {
    alert(err?.message || t('opFailed'))
  } finally {
    clearing.value = false
  }
}

onMounted(() => {
  nav.updateTabs(
    {
      all: 'all',
      block: 'moderationBlocked',
      replace: 'moderationReplaced',
      error: 'opFailed',
    },
    getRouteStatus(),
  )
  fetchLogs()
  watch(curtTab, fetchLogs)
  watch(curtSite, fetchLogs)
  watch(
    () => route.query.status,
    () => {
      curtTab.value = getRouteStatus()
    },
  )
})
</script>

<template>
  <div class="moderation-page">
    <section class="page-head">
      <div>
        <div class="eyebrow">Moderation</div>
        <h1>{{ t('moderation') }}</h1>
        <p>{{ t('moderationIntro') }}</p>
      </div>
      <div class="head-actions">
        <button class="clear-btn" :disabled="clearing || !total" @click="clearLogs">
          {{ clearing ? t('refreshing') : t('moderationClear') }}
        </button>
        <button class="refresh-btn" :disabled="loading" @click="fetchLogs">
          {{ loading ? t('refreshing') : t('refresh') }}
        </button>
      </div>
    </section>

    <section class="log-panel">
      <div class="log-summary">
        {{ t('moderationSummary', { total, count: logs.length }) }}
      </div>
      <div class="log-list">
        <article
          v-for="item in logs"
          :key="item.id"
          class="log-item"
          :class="displayStatus(item)"
        >
          <div class="status-icon">{{ statusMeta[displayStatus(item)].icon }}</div>
          <div class="log-main">
            <div class="log-top">
              <div class="log-title">
                <b>{{ statusMeta[displayStatus(item)].label }}</b>
                <span>{{ item.checker }}</span>
                <em>{{ item.action }}</em>
              </div>
              <button
                class="delete-btn"
                :disabled="deletingID === item.id"
                @click="deleteLog(item)"
              >
                {{ t('delete') }}
              </button>
            </div>
            <p>{{ item.message || t('moderationNoMessage') }}</p>
            <blockquote>{{ item.comment_content || t('moderationCommentUnavailable') }}</blockquote>
            <div class="meta">
              <span>#{{ item.comment_id }}</span>
              <span>{{ item.user_name || item.user_email || t('unknownUser') }}</span>
              <span>{{ item.site_name }}</span>
              <span>{{ item.date }}</span>
            </div>
          </div>
        </article>
        <div v-if="!logs.length" class="empty">{{ t('noModerationLogs') }}</div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.moderation-page {
  padding: 24px 28px 70px;
}

.page-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin: 2px 0 22px;

  h1 {
    margin: 4px 0 8px;
    font-size: 28px;
  }

  p {
    margin: 0;
    color: var(--at-color-sub);
  }

}

.head-actions {
  display: flex;
  gap: 8px;
}

button {
  border: 1px solid transparent;
  border-radius: 6px;
  padding: 8px 12px;
  cursor: pointer;
  font: inherit;

  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
}

.refresh-btn {
  background: #168aac;
  color: #fff;
}

.clear-btn,
.delete-btn {
  border-color: rgba(211, 47, 47, 0.32);
  background: transparent;
  color: #c92a2a;
}

.eyebrow {
  color: var(--at-color-main);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0;
}

.log-panel {
  border-top: 1px solid var(--at-color-border);
}

.log-summary {
  padding: 13px 2px;
  color: var(--at-color-sub);
  font-size: 13px;
}

.log-list {
  display: grid;
  gap: 10px;
}

.log-item {
  display: flex;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--at-color-border);
  border-left-width: 3px;
  border-radius: 8px;
  background: var(--at-color-bg);

  &.replace {
    border-left-color: #2f9e44;
  }

  &.block {
    border-left-color: #f08c00;
  }

  &.error {
    border-left-color: #e03131;
  }

  &.replace .status-icon {
    background: rgba(47, 158, 68, 0.12);
    color: #2f9e44;
  }

  &.block .status-icon {
    background: rgba(240, 140, 0, 0.14);
    color: #f08c00;
  }

  &.error .status-icon {
    background: rgba(224, 49, 49, 0.12);
    color: #e03131;
  }
}

.status-icon {
  width: 34px;
  height: 34px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  flex: none;
}

.log-main {
  min-width: 0;
  flex: 1;

  p {
    margin: 8px 0;
    color: var(--at-color-font);
  }

  blockquote {
    margin: 0;
    padding: 10px 12px;
    border-left: 3px solid var(--at-color-border);
    background: var(--at-color-bg-grey-transl);
    color: var(--at-color-sub);
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.log-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.log-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;

  span,
  em {
    padding: 2px 8px;
    border-radius: 5px;
    background: var(--at-color-bg-grey-transl);
    color: var(--at-color-sub);
    font-size: 12px;
    font-style: normal;
  }
}

.meta {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 10px;
  color: var(--at-color-sub);
  font-size: 12px;
}

.empty {
  padding: 42px 16px;
  color: var(--at-color-sub);
  text-align: center;
}

@media (max-width: 850px) {
  .moderation-page {
    padding: 16px 14px 70px;
  }

  .page-head {
    display: block;
  }

  .head-actions {
    margin-top: 16px;
  }
}
</style>
