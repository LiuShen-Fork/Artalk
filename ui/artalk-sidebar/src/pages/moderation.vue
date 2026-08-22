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
  comment_available: boolean
  comment_rid: number
  comment_ua: string
  comment_ip: string
  comment_pending: boolean
  comment_collapsed: boolean
  comment_pinned: boolean
  user_name: string
  user_email: string
  user_link: string
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
const commentAction = ref<string | null>(null)

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

function getCommentActionKey(item: ModerationLog, action: 'approve' | 'delete') {
  return action + ':' + item.comment_id
}

function isCommentActionLoading(item: ModerationLog, action: 'approve' | 'delete') {
  return commentAction.value === getCommentActionKey(item, action)
}

function canOperateComment(item: ModerationLog) {
  return item.comment_available && item.comment_id > 0
}

function buildCommentUpdatePayload(item: ModerationLog, isPending: boolean) {
  return {
    site_name: item.site_name,
    content: item.comment_content,
    page_key: item.page_key,
    nick: item.user_name,
    email: item.user_email,
    link: item.user_link,
    rid: item.comment_rid,
    ua: item.comment_ua,
    ip: item.comment_ip,
    is_collapsed: item.comment_collapsed,
    is_pending: isPending,
    is_pinned: item.comment_pinned,
  }
}

async function approveComment(item: ModerationLog) {
  if (!canOperateComment(item)) return
  if (!window.confirm(t('moderationApproveCommentConfirm'))) return

  commentAction.value = getCommentActionKey(item, 'approve')
  try {
    await artalk!.ctx
      .getApi()
      .comments.updateComment(item.comment_id, buildCommentUpdatePayload(item, false))
    item.comment_pending = false
  } catch (err: any) {
    alert(err?.message || t('opFailed'))
  } finally {
    commentAction.value = null
  }
}

async function deleteComment(item: ModerationLog) {
  if (!canOperateComment(item)) return
  if (!window.confirm(t('moderationDeleteCommentConfirm'))) return

  commentAction.value = getCommentActionKey(item, 'delete')
  try {
    await artalk!.ctx.getApi().comments.deleteComment(item.comment_id)
    item.comment_available = false
    item.comment_content = ''
    item.comment_pending = false
  } catch (err: any) {
    alert(err?.message || t('opFailed'))
  } finally {
    commentAction.value = null
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
  <div class="moderation-page admin-page">
    <AdminPageHeader
      eyebrow="Moderation"
      :title="t('moderation')"
      :description="t('moderationIntro')"
    >
      <template #actions>
        <button class="admin-button danger" :disabled="clearing || !total" @click="clearLogs">
          {{ clearing ? t('refreshing') : t('moderationClear') }}
        </button>
        <button class="admin-button" :disabled="loading" @click="fetchLogs">
          {{ loading ? t('refreshing') : t('refresh') }}
        </button>
      </template>
    </AdminPageHeader>

    <section class="log-panel admin-panel">
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
              <div class="row-actions">
                <button
                  v-if="item.comment_pending"
                  class="admin-button approve-comment-btn"
                  :disabled="!canOperateComment(item) || !!commentAction"
                  @click="approveComment(item)"
                >
                  {{
                    isCommentActionLoading(item, 'approve')
                      ? t('refreshing')
                      : t('moderationApproveComment')
                  }}
                </button>
                <button
                  class="admin-button danger delete-comment-btn"
                  :disabled="!canOperateComment(item) || !!commentAction"
                  @click="deleteComment(item)"
                >
                  {{
                    isCommentActionLoading(item, 'delete')
                      ? t('refreshing')
                      : t('moderationDeleteComment')
                  }}
                </button>
                <button
                  class="admin-button delete-btn"
                  :disabled="deletingID === item.id"
                  @click="deleteLog(item)"
                >
                  {{ t('delete') }}
                </button>
              </div>
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
    border-left-color: var(--atk-admin-sage);
  }

  &.block {
    border-left-color: var(--atk-admin-terracotta);
  }

  &.error {
    border-left-color: var(--atk-admin-danger);
  }

  &.replace .status-icon {
    background: var(--atk-admin-sage-soft);
    color: var(--atk-admin-sage);
  }

  &.block .status-icon {
    background: rgba(168, 95, 72, 0.12);
    color: var(--atk-admin-terracotta);
  }

  &.error .status-icon {
    background: rgba(157, 81, 71, 0.12);
    color: var(--atk-admin-danger);
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

.row-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
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

}

.moderation-page .log-panel,
.moderation-page .log-item {
  border-color: var(--atk-admin-border);
  background: var(--atk-admin-surface);
  box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);
}

.moderation-page .log-panel {
  overflow: hidden;
  border-top: 1px solid var(--atk-admin-border);
  border-radius: var(--atk-admin-radius);
}

.moderation-page .log-summary {
  padding: 16px 20px;
  color: var(--atk-admin-subtle);
  background: var(--atk-admin-surface-muted);
}

.moderation-page .log-list {
  gap: 0;
}

.moderation-page .log-item {
  border: 0;
  border-radius: 0;
  border-bottom: 1px solid var(--atk-admin-border);
  border-left: 0;
  gap: 16px;
  padding: 18px 20px;
}

.moderation-page .log-item:last-child {
  border-bottom: 0;
}

.moderation-page .log-item.replace,
.moderation-page .log-item.block,
.moderation-page .log-item.error {
  border-left: 0;
}

.moderation-page .log-item.replace .status-icon {
  background: var(--atk-admin-sage-soft);
  color: var(--atk-admin-sage);
}

.moderation-page .log-item.block .status-icon {
  background: rgba(168, 95, 72, 0.12);
  color: var(--atk-admin-terracotta);
}

.moderation-page .log-item.error .status-icon {
  background: rgba(157, 81, 71, 0.12);
  color: var(--atk-admin-danger);
}

.moderation-page .status-icon {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  font-size: 15px;
}

.moderation-page .log-title b {
  color: var(--atk-admin-ink);
  font-size: 15px;
}

.moderation-page .log-main p {
  margin: 10px 0;
  line-height: 1.6;
}

.moderation-page .log-main blockquote {
  border-left-width: 2px;
  border-radius: var(--atk-admin-radius-sm);
  line-height: 1.6;
}

.moderation-page .meta span {
  padding: 3px 8px;
  border: 1px solid var(--atk-admin-border);
  border-radius: 999px;
  background: var(--atk-admin-surface-muted);
}

.moderation-page .row-actions .admin-button {
  min-height: 32px;
  padding: 5px 10px;
}

.moderation-page .log-main p,
.moderation-page .meta,
.moderation-page .log-title span,
.moderation-page .log-title em,
.moderation-page .empty {
  color: var(--atk-admin-subtle);
}

.moderation-page .log-main blockquote,
.moderation-page .log-title span,
.moderation-page .log-title em {
  border-color: var(--atk-admin-border);
  background: var(--atk-admin-surface-muted);
}

.moderation-page .approve-comment-btn {
  border-color: var(--atk-admin-sage);
  color: var(--atk-admin-sage);
}

.moderation-page .delete-btn {
  min-height: 32px;
  padding: 5px 10px;
  color: var(--atk-admin-subtle);
}

@media (max-width: 680px) {
  .moderation-page .log-item {
    align-items: flex-start;
    padding: 16px 14px;
  }

  .moderation-page .log-top {
    display: block;
  }

  .moderation-page .row-actions {
    justify-content: flex-start;
    margin-top: 12px;
  }
}
</style>
