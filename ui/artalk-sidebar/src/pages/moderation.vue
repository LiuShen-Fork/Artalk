<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'

type ModerationStatus = 'pass' | 'block' | 'error'

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
const { curtTab } = storeToRefs(nav)
const { site: curtSite } = storeToRefs(user)
const { t } = useI18n()

const logs = ref<ModerationLog[]>([])
const total = ref(0)
const loading = ref(false)

const statusMeta = computed(() => ({
  pass: { label: t('normal'), icon: 'OK' },
  block: { label: t('moderationBlocked'), icon: '!' },
  error: { label: t('opFailed'), icon: 'X' },
}))

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

onMounted(() => {
  nav.updateTabs(
    {
      all: 'all',
      pass: 'normal',
      block: 'moderationBlocked',
      error: 'opFailed',
    },
    'all',
  )
  fetchLogs()
  watch(curtTab, fetchLogs)
  watch(curtSite, fetchLogs)
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
      <button :disabled="loading" @click="fetchLogs">
        {{ loading ? t('refreshing') : t('refresh') }}
      </button>
    </section>

    <section class="log-panel">
      <div class="log-summary">
        {{ t('moderationSummary', { total, count: logs.length }) }}
      </div>
      <div class="log-list">
        <article v-for="item in logs" :key="item.id" class="log-item" :class="item.status">
          <div class="status-icon">{{ statusMeta[item.status].icon }}</div>
          <div class="log-main">
            <div class="log-title">
              <b>{{ statusMeta[item.status].label }}</b>
              <span>{{ item.checker }}</span>
              <em>{{ item.action }}</em>
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

.page-head,
.log-panel {
  border: 1px solid var(--at-color-border);
  background: var(--at-color-bg);
  border-radius: 10px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.05);
}

.page-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 26px;
  margin-bottom: 16px;

  h1 {
    margin: 4px 0 8px;
    font-size: 28px;
  }

  p {
    margin: 0;
    color: var(--at-color-sub);
  }

  button {
    align-self: flex-start;
    border: 0;
    background: #36abcf;
    color: #fff;
    border-radius: 6px;
    padding: 9px 18px;
    cursor: pointer;
  }
}

.eyebrow {
  color: var(--at-color-main);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0;
}

.log-panel {
  overflow: hidden;
}

.log-summary {
  padding: 14px 18px;
  color: var(--at-color-sub);
  border-bottom: 1px solid var(--at-color-border);
}

.log-item {
  display: flex;
  gap: 14px;
  padding: 18px;
  border-bottom: 1px solid var(--at-color-border);

  &.pass .status-icon {
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
  border-radius: 50%;
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

.log-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;

  span,
  em {
    padding: 2px 8px;
    border-radius: 999px;
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
  padding: 36px;
  color: var(--at-color-sub);
  text-align: center;
}

@media (max-width: 850px) {
  .moderation-page {
    padding: 16px 14px 70px;
  }

  .page-head {
    display: block;

    button {
      margin-top: 16px;
    }
  }
}
</style>
