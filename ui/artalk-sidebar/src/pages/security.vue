<route lang="json">
{
  "meta": {
    "title": "Security - Artalk Sidebar"
  }
}
</route>

<script setup lang="ts">
import AdminPageHeader from '../components/AdminPageHeader.vue'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'

type QuotaConf = { minute: number; hour: number; day: number }

type SecurityConfig = {
  enabled: boolean
  action: 'captcha' | 'reject' | 'ban'
  rate_limit: { ip: QuotaConf; global: QuotaConf }
  auto_ban: { enabled: boolean; threshold: number; window: number; duration: number }
  blacklist: string[]
  whitelist: string[]
  event_retention_days: number
  exclude_paths: string[]
}

type CaptchaConfig = {
  enabled: boolean
  always: boolean
  captcha_type: string
  action_reset: number
  action_limit: number
  geetest: { captcha_id: string; captcha_key: string }
}

type RuntimeState = {
  captcha_ready: boolean
  tracked_ips: number
  active_bans: number
  tracked_violators: number
  global: QuotaConf
}

type IPBan = {
  id: number
  ip: string
  ip_region: string
  is_cidr: boolean
  reason: string
  note: string
  source: string
  active: boolean
  expire_at: string
  hit_count: number
  last_hit_at: string
  created_at: string
}

type SecurityEvent = {
  id: number
  ip: string
  ip_region: string
  kind: string
  scope: string
  window: string
  action: string
  path: string
  method: string
  site_name: string
  user_name: string
  ua: string
  detail: string
  blocked: boolean
  date: string
}

type Report = {
  range_days: number
  total: number
  blocked: number
  by_kind: { kind: string; count: number }[]
  by_action: { action: string; count: number }[]
  by_day: { date: string; count: number }[]
  top_ips: { ip: string; ip_region: string; count: number }[]
  top_paths: { path: string; count: number }[]
}

const { t } = useI18n()
const nav = useNavStore()

const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const saveError = ref('')
const saveNotice = ref('')

const config = reactive<SecurityConfig>({
  enabled: false,
  action: 'captcha',
  rate_limit: { ip: { minute: 0, hour: 0, day: 0 }, global: { minute: 0, hour: 0, day: 0 } },
  auto_ban: { enabled: false, threshold: 5, window: 3600, duration: 86400 },
  blacklist: [],
  whitelist: [],
  event_retention_days: 90,
  exclude_paths: [],
})

const captcha = reactive<CaptchaConfig>({
  enabled: false,
  always: false,
  captcha_type: 'image',
  action_reset: 60,
  action_limit: 3,
  geetest: { captcha_id: '', captcha_key: '' },
})

const runtime = reactive<RuntimeState>({
  captcha_ready: false,
  tracked_ips: 0,
  active_bans: 0,
  tracked_violators: 0,
  global: { minute: 0, hour: 0, day: 0 },
})

const blacklistText = ref('')
const whitelistText = ref('')

const captchaTypes = ['image', 'turnstile', 'recaptcha', 'hcaptcha', 'geetest', 'capjs']
const windows = ['minute', 'hour', 'day'] as const
const actionOptions = [
  { value: 'captcha', label: 'captcha' },
  { value: 'reject', label: 'reject' },
  { value: 'ban', label: 'ban' },
] as const

// 封禁名单
const bans = ref<IPBan[]>([])
const bansLoading = ref(false)
const newBan = reactive({ ip: '', reason: '', note: '', duration: 86400 })
const banSubmitting = ref(false)

// 攻击报表
const reportDays = ref(7)
const report = ref<Report | null>(null)
const reportLoading = ref(false)

// 事件记录
const events = ref<SecurityEvent[]>([])
const eventTotal = ref(0)
const eventsLoading = ref(false)
const eventKind = ref('')
const eventDays = ref(7)
const eventIP = ref('')

const kindOptions = computed(() => [
  { value: '', label: t('securityEventKindAll') },
  { value: 'rate_limit', label: t('securityKindRateLimit') },
  { value: 'captcha_failed', label: t('securityKindCaptchaFailed') },
  { value: 'banned', label: t('securityKindBanned') },
  { value: 'auto_ban', label: t('securityKindAutoBan') },
])

function request<T>(path: string, method: string, data?: any, query?: any) {
  return artalk!.ctx.getApi().request<T>({
    path,
    method,
    body: data,
    query,
    secure: true,
    format: 'json',
  })
}

function errText(e: any) {
  return e?.msg || e?.message || t('opFailed')
}

function applyConfig(res: {
  security: SecurityConfig
  captcha: CaptchaConfig
  runtime: RuntimeState
}) {
  Object.assign(config, res.security)
  config.rate_limit = res.security.rate_limit
  config.auto_ban = res.security.auto_ban
  config.blacklist = res.security.blacklist || []
  config.whitelist = res.security.whitelist || []

  Object.assign(captcha, res.captcha)
  captcha.geetest = res.captcha.geetest || { captcha_id: '', captcha_key: '' }

  Object.assign(runtime, res.runtime)

  blacklistText.value = (config.blacklist || []).join('\n')
  whitelistText.value = (config.whitelist || []).join('\n')
}

async function loadConfig() {
  loading.value = true
  loadError.value = ''

  try {
    const res = await request<{ security: SecurityConfig; captcha: CaptchaConfig; runtime: RuntimeState }>(
      '/security/config',
      'GET',
    )
    applyConfig(res.data)
  } catch (e) {
    loadError.value = errText(e)
  } finally {
    loading.value = false
  }
}

function parseList(text: string) {
  return text
    .split(/[\n,]/)
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
}

async function saveConfig() {
  if (saving.value) return

  saving.value = true
  saveError.value = ''
  saveNotice.value = ''

  try {
    const res = await request<{ security: SecurityConfig; captcha: CaptchaConfig; runtime: RuntimeState }>(
      '/security/config',
      'PUT',
      {
        enabled: config.enabled,
        action: config.action,
        rate_limit: config.rate_limit,
        auto_ban: config.auto_ban,
        blacklist: parseList(blacklistText.value),
        whitelist: parseList(whitelistText.value),
        event_retention_days: config.event_retention_days,
        captcha: {
          enabled: captcha.enabled,
          always: captcha.always,
          captcha_type: captcha.captcha_type,
          action_reset: captcha.action_reset,
          action_limit: captcha.action_limit,
          geetest: captcha.geetest,
        },
      },
    )
    applyConfig(res.data)
    saveNotice.value = t('settingSaved')
  } catch (e) {
    saveError.value = errText(e)
  } finally {
    saving.value = false
  }
}

async function loadBans() {
  bansLoading.value = true
  try {
    const res = await request<{ count: number; bans: IPBan[] }>('/security/bans', 'GET', undefined, {
      limit: 100,
    })
    bans.value = res.data.bans || []
  } catch (e) {
    alert(errText(e))
  } finally {
    bansLoading.value = false
  }
}

async function addBan() {
  if (banSubmitting.value) return
  if (!newBan.ip.trim()) return

  banSubmitting.value = true
  try {
    await request('/security/bans', 'POST', {
      ip: newBan.ip.trim(),
      reason: newBan.reason.trim(),
      note: newBan.note.trim(),
      duration: Number(newBan.duration) || 0,
    })
    newBan.ip = ''
    newBan.reason = ''
    newBan.note = ''
    await Promise.all([loadBans(), loadConfig()])
  } catch (e) {
    alert(errText(e))
  } finally {
    banSubmitting.value = false
  }
}

async function removeBan(item: IPBan) {
  if (item.id <= 0) {
    alert(t('securityConfigBlacklistHint'))
    return
  }
  if (!window.confirm(t('securityUnbanConfirm', { ip: item.ip }))) return

  try {
    await request(`/security/bans/${item.id}`, 'DELETE')
    bans.value = bans.value.filter((ban) => ban.id !== item.id)
    loadConfig()
  } catch (e) {
    alert(errText(e))
  }
}

function banFromIP(ip: string) {
  newBan.ip = ip
  window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' })
}

async function loadReport() {
  reportLoading.value = true
  try {
    const res = await request<Report>('/security/report', 'GET', undefined, { days: reportDays.value })
    report.value = res.data
  } catch (e) {
    alert(errText(e))
  } finally {
    reportLoading.value = false
  }
}

async function loadEvents() {
  eventsLoading.value = true
  try {
    const res = await request<{ count: number; events: SecurityEvent[] }>('/security/events', 'GET', undefined, {
      kind: eventKind.value || undefined,
      ip: eventIP.value.trim() || undefined,
      days: eventDays.value,
      limit: 50,
    })
    events.value = res.data.events || []
    eventTotal.value = res.data.count || 0
  } catch (e) {
    alert(errText(e))
  } finally {
    eventsLoading.value = false
  }
}

async function deleteEvent(item: SecurityEvent) {
  try {
    await request(`/security/events/${item.id}`, 'DELETE')
    events.value = events.value.filter((event) => event.id !== item.id)
    eventTotal.value = Math.max(0, eventTotal.value - 1)
  } catch (e) {
    alert(errText(e))
  }
}

async function clearEvents() {
  if (!window.confirm(t('securityEventClearConfirm'))) return

  try {
    await request('/security/events', 'DELETE', undefined, {
      kind: eventKind.value || undefined,
      ip: eventIP.value.trim() || undefined,
      days: eventDays.value,
    })
    events.value = []
    eventTotal.value = 0
    loadReport()
  } catch (e) {
    alert(errText(e))
  }
}

function formatDuration(seconds: number) {
  if (seconds < 0) return t('securityPermanent')
  if (seconds >= 86400) return `${Math.round(seconds / 86400)} ${t('securityDayUnit')}`
  if (seconds >= 3600) return `${Math.round(seconds / 3600)} ${t('securityHourUnit')}`
  return `${Math.round(seconds / 60)} ${t('securityMinuteUnit')}`
}

function kindLabel(kind: string) {
  switch (kind) {
    case 'rate_limit':
      return t('securityKindRateLimit')
    case 'captcha_failed':
      return t('securityKindCaptchaFailed')
    case 'banned':
      return t('securityKindBanned')
    case 'auto_ban':
      return t('securityKindAutoBan')
    default:
      return kind
  }
}

onMounted(() => {
  nav.updateTabs({})
  nav.scrollPageToTop()
  loadConfig()
  loadBans()
  loadReport()
  loadEvents()
})
</script>

<template>
  <div class="security-admin admin-page">
    <AdminPageHeader :title="t('security')" :description="t('securityAdminDesc')">
      <template #actions>
        <button class="admin-button" :disabled="loading" @click="loadConfig">
          {{ loading ? t('refreshing') : t('refresh') }}
        </button>
        <button class="admin-button primary" :disabled="saving || loading" @click="saveConfig">
          {{ saving ? '…' : t('save') }}
        </button>
      </template>
    </AdminPageHeader>

    <div v-if="loading" class="admin-panel state-box">
      <span class="state-spinner" />
      <span>{{ t('refreshing') }}…</span>
    </div>

    <div v-else-if="loadError" class="admin-panel state-box error">
      <p>{{ loadError }}</p>
      <button class="admin-button" type="button" @click="loadConfig">{{ t('refresh') }}</button>
    </div>

    <template v-else>
      <div class="stat-grid">
        <div class="admin-panel stat-card">
          <span>{{ t('securityTrackedIPs') }}</span>
          <strong>{{ runtime.tracked_ips }}</strong>
        </div>
        <div class="admin-panel stat-card">
          <span>{{ t('securityActiveBans') }}</span>
          <strong>{{ runtime.active_bans }}</strong>
        </div>
        <div class="admin-panel stat-card">
          <span>{{ t('securityViolators') }}</span>
          <strong>{{ runtime.tracked_violators }}</strong>
        </div>
        <div class="admin-panel stat-card">
          <span>{{ t('securityGlobalCount') }}</span>
          <strong>
            {{ runtime.global.minute }} / {{ runtime.global.hour }} / {{ runtime.global.day }}
          </strong>
        </div>
      </div>

      <section class="admin-panel card">
        <header class="card-head">
          <div>
            <h2>{{ t('securityBasic') }}</h2>
            <p>{{ t('securityBasicDesc') }}</p>
          </div>
          <span class="admin-chip" :class="{ active: config.enabled }">
            {{ config.enabled ? t('enabled') : t('disabled') }}
          </span>
        </header>

        <div class="card-body">
          <label class="switch-row">
            <input v-model="config.enabled" type="checkbox" />
            <span>{{ t('securityEnable') }}</span>
          </label>

          <div class="field-grid">
            <label class="field">
              <span>{{ t('securityAction') }}</span>
              <select v-model="config.action">
                <option v-for="item in actionOptions" :key="item.value" :value="item.value">
                  {{ t('securityAction_' + item.value) }}
                </option>
              </select>
            </label>

            <label class="field">
              <span>{{ t('securityRetention') }}</span>
              <input v-model.number="config.event_retention_days" type="number" min="1" max="365" />
            </label>
          </div>

          <p v-if="config.action === 'captcha' && !captcha.enabled" class="form-msg warn">
            {{ t('securityCaptchaHint') }}
          </p>

          <div class="field-grid">
            <label class="field">
              <span>{{ t('securityBlacklist') }}</span>
              <textarea v-model="blacklistText" rows="4" :placeholder="t('securityListHint')" />
            </label>

            <label class="field">
              <span>{{ t('securityWhitelist') }}</span>
              <textarea v-model="whitelistText" rows="4" :placeholder="t('securityListHint')" />
            </label>
          </div>
        </div>
      </section>

      <section class="admin-panel card">
        <header class="card-head">
          <div>
            <h2>{{ t('securityRateLimit') }}</h2>
            <p>{{ t('securityRateLimitDesc') }}</p>
          </div>
        </header>

        <div class="card-body">
          <table class="quota-table">
            <thead>
              <tr>
                <th />
                <th>{{ t('securityPerIP') }}</th>
                <th>{{ t('securityGlobal') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="window in windows" :key="window">
                <th>{{ t('securityWindow_' + window) }}</th>
                <td>
                  <input v-model.number="config.rate_limit.ip[window]" type="number" min="0" />
                </td>
                <td>
                  <input v-model.number="config.rate_limit.global[window]" type="number" min="0" />
                </td>
              </tr>
            </tbody>
          </table>
          <p class="hint">{{ t('securityQuotaHint') }}</p>

          <div class="sub-block">
            <label class="switch-row">
              <input v-model="config.auto_ban.enabled" type="checkbox" />
              <span>{{ t('securityAutoBan') }}</span>
            </label>

            <div class="field-grid" v-if="config.auto_ban.enabled">
              <label class="field">
                <span>{{ t('securityAutoBanThreshold') }}</span>
                <input v-model.number="config.auto_ban.threshold" type="number" min="1" />
              </label>
              <label class="field">
                <span>{{ t('securityAutoBanWindow') }}</span>
                <input v-model.number="config.auto_ban.window" type="number" min="60" step="60" />
              </label>
              <label class="field">
                <span>{{ t('securityAutoBanDuration') }}</span>
                <input v-model.number="config.auto_ban.duration" type="number" step="60" />
              </label>
            </div>
            <p class="hint">{{ t('securityAutoBanHint') }}</p>
          </div>
        </div>
      </section>

      <section class="admin-panel card">
        <header class="card-head">
          <div>
            <h2>{{ t('captcha') }}</h2>
            <p>{{ t('securityCaptchaDesc') }}</p>
          </div>
          <span class="admin-chip" :class="{ active: captcha.enabled }">
            {{ captcha.enabled ? t('enabled') : t('disabled') }}
          </span>
        </header>

        <div class="card-body">
          <label class="switch-row">
            <input v-model="captcha.enabled" type="checkbox" />
            <span>{{ t('securityCaptchaEnable') }}</span>
          </label>
          <label class="switch-row">
            <input v-model="captcha.always" type="checkbox" />
            <span>{{ t('securityCaptchaAlways') }}</span>
          </label>

          <div class="field-grid">
            <label class="field">
              <span>{{ t('securityCaptchaType') }}</span>
              <select v-model="captcha.captcha_type">
                <option v-for="type in captchaTypes" :key="type" :value="type">{{ type }}</option>
              </select>
            </label>
            <label class="field">
              <span>{{ t('securityCaptchaLimit') }}</span>
              <input v-model.number="captcha.action_limit" type="number" min="0" />
            </label>
            <label class="field">
              <span>{{ t('securityCaptchaReset') }}</span>
              <input v-model.number="captcha.action_reset" type="number" min="0" />
            </label>
          </div>

          <div v-if="captcha.captcha_type === 'geetest'" class="field-grid">
            <label class="field">
              <span>Geetest Captcha ID</span>
              <input v-model="captcha.geetest.captcha_id" type="text" autocomplete="off" />
            </label>
            <label class="field">
              <span>Geetest Captcha Key</span>
              <input v-model="captcha.geetest.captcha_key" type="password" autocomplete="off" />
            </label>
          </div>
        </div>
      </section>

      <div class="form-foot">
        <span v-if="saveError" class="form-msg error">{{ saveError }}</span>
        <span v-else-if="saveNotice" class="form-msg ok">{{ saveNotice }}</span>
        <span v-else class="form-msg" />
        <button class="admin-button primary" :disabled="saving" @click="saveConfig">
          {{ saving ? '…' : t('save') }}
        </button>
      </div>

      <section class="admin-panel card">
        <header class="card-head">
          <div>
            <h2>{{ t('securityBans') }}</h2>
            <p>{{ t('securityBansDesc') }}</p>
          </div>
          <button class="admin-button" :disabled="bansLoading" @click="loadBans">
            {{ bansLoading ? t('refreshing') : t('refresh') }}
          </button>
        </header>

        <div class="card-body">
          <div class="ban-form">
            <input v-model="newBan.ip" type="text" :placeholder="t('securityBanIP')" />
            <input v-model="newBan.reason" type="text" :placeholder="t('securityBanReason')" />
            <select v-model.number="newBan.duration">
              <option :value="3600">{{ formatDuration(3600) }}</option>
              <option :value="86400">{{ formatDuration(86400) }}</option>
              <option :value="604800">{{ formatDuration(604800) }}</option>
              <option :value="-1">{{ t('securityPermanent') }}</option>
            </select>
            <button class="admin-button primary" :disabled="banSubmitting || !newBan.ip" @click="addBan">
              {{ t('securityBanAdd') }}
            </button>
          </div>

          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>IP</th>
                  <th>{{ t('securityBanReason') }}</th>
                  <th>{{ t('securityBanSource') }}</th>
                  <th>{{ t('securityBanExpire') }}</th>
                  <th>{{ t('securityBanHits') }}</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in bans" :key="item.id + ':' + item.ip">
                  <td>
                    <b>{{ item.ip }}</b>
                    <small v-if="item.ip_region">{{ item.ip_region }}</small>
                  </td>
                  <td>
                    {{ item.reason }}
                    <small v-if="item.note">{{ item.note }}</small>
                  </td>
                  <td>{{ item.source }}</td>
                  <td>{{ item.expire_at || t('securityPermanent') }}</td>
                  <td>{{ item.hit_count }}</td>
                  <td class="cell-actions">
                    <button class="admin-button danger" @click="removeBan(item)">
                      {{ t('securityUnban') }}
                    </button>
                  </td>
                </tr>
                <tr v-if="!bans.length">
                  <td colspan="6" class="empty-cell">{{ t('securityNoBans') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section class="admin-panel card">
        <header class="card-head">
          <div>
            <h2>{{ t('securityReport') }}</h2>
            <p>{{ t('securityReportDesc') }}</p>
          </div>
          <div class="head-tools">
            <select v-model.number="reportDays" @change="loadReport">
              <option :value="1">{{ t('securityRange_1') }}</option>
              <option :value="7">{{ t('securityRange_7') }}</option>
              <option :value="30">{{ t('securityRange_30') }}</option>
            </select>
            <button class="admin-button" :disabled="reportLoading" @click="loadReport">
              {{ reportLoading ? t('refreshing') : t('refresh') }}
            </button>
          </div>
        </header>

        <div class="card-body" v-if="report">
          <div class="stat-grid small">
            <div class="stat-card plain">
              <span>{{ t('securityReportTotal') }}</span>
              <strong>{{ report.total }}</strong>
            </div>
            <div class="stat-card plain">
              <span>{{ t('securityReportBlocked') }}</span>
              <strong>{{ report.blocked }}</strong>
            </div>
          </div>

          <div class="report-grid">
            <div>
              <h3>{{ t('securityByKind') }}</h3>
              <ul class="bars">
                <li v-for="item in report.by_kind" :key="item.kind">
                  <span>{{ kindLabel(item.kind) }}</span>
                  <em>{{ item.count }}</em>
                </li>
                <li v-if="!report.by_kind.length" class="empty-cell">{{ t('noContent') }}</li>
              </ul>
            </div>

            <div>
              <h3>{{ t('securityByDay') }}</h3>
              <ul class="bars">
                <li v-for="item in report.by_day" :key="item.date">
                  <span>{{ item.date }}</span>
                  <em>{{ item.count }}</em>
                </li>
                <li v-if="!report.by_day.length" class="empty-cell">{{ t('noContent') }}</li>
              </ul>
            </div>

            <div>
              <h3>{{ t('securityTopIPs') }}</h3>
              <ul class="bars">
                <li v-for="item in report.top_ips" :key="item.ip">
                  <span>
                    {{ item.ip }}
                    <small v-if="item.ip_region">{{ item.ip_region }}</small>
                  </span>
                  <span class="bar-actions">
                    <em>{{ item.count }}</em>
                    <button class="link-btn" @click="banFromIP(item.ip)">{{ t('securityBan') }}</button>
                  </span>
                </li>
                <li v-if="!report.top_ips.length" class="empty-cell">{{ t('noContent') }}</li>
              </ul>
            </div>

            <div>
              <h3>{{ t('securityTopPaths') }}</h3>
              <ul class="bars">
                <li v-for="item in report.top_paths" :key="item.path">
                  <span>{{ item.path || '-' }}</span>
                  <em>{{ item.count }}</em>
                </li>
                <li v-if="!report.top_paths.length" class="empty-cell">{{ t('noContent') }}</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      <section class="admin-panel card">
        <header class="card-head">
          <div>
            <h2>{{ t('securityEvents') }}</h2>
            <p>{{ t('securityEventsDesc', { total: eventTotal }) }}</p>
          </div>
          <div class="head-tools">
            <button class="admin-button danger" :disabled="!eventTotal" @click="clearEvents">
              {{ t('securityEventClear') }}
            </button>
            <button class="admin-button" :disabled="eventsLoading" @click="loadEvents">
              {{ eventsLoading ? t('refreshing') : t('refresh') }}
            </button>
          </div>
        </header>

        <div class="card-body">
          <div class="filter-row">
            <select v-model="eventKind" @change="loadEvents">
              <option v-for="item in kindOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
            <select v-model.number="eventDays" @change="loadEvents">
              <option :value="1">{{ t('securityRange_1') }}</option>
              <option :value="7">{{ t('securityRange_7') }}</option>
              <option :value="30">{{ t('securityRange_30') }}</option>
            </select>
            <input v-model="eventIP" type="text" placeholder="IP" @keyup.enter="loadEvents" />
            <button class="admin-button" @click="loadEvents">{{ t('search') }}</button>
          </div>

          <div class="event-list">
            <article v-for="item in events" :key="item.id" class="event-item">
              <div class="event-main">
                <div class="event-title">
                  <b>{{ kindLabel(item.kind) }}</b>
                  <span>{{ item.action }}</span>
                  <span v-if="item.window">{{ item.window }}</span>
                  <span v-if="item.blocked" class="blocked">{{ t('securityBlocked') }}</span>
                </div>
                <div class="event-meta">
                  <span>{{ item.ip }}</span>
                  <span v-if="item.ip_region">{{ item.ip_region }}</span>
                  <span>{{ item.method }} {{ item.path }}</span>
                  <span v-if="item.site_name">{{ item.site_name }}</span>
                  <span v-if="item.user_name">{{ item.user_name }}</span>
                  <span>{{ item.date }}</span>
                </div>
                <code v-if="item.detail">{{ item.detail }}</code>
              </div>
              <button class="admin-button danger" @click="deleteEvent(item)">{{ t('delete') }}</button>
            </article>

            <div v-if="!events.length" class="empty-cell">{{ t('securityNoEvents') }}</div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped lang="scss">
.security-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-bottom: 40px;
}

.state-box {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 160px;
  padding: 24px;
  color: var(--atk-admin-subtle);
  font-size: 13px;

  &.error {
    flex-direction: column;
    color: var(--atk-admin-danger);
  }

  p {
    margin: 0;
  }
}

.state-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--atk-admin-border-strong);
  border-top-color: var(--atk-admin-sage);
  border-radius: 50%;
  animation: security-admin-spin 0.8s linear infinite;
}

@keyframes security-admin-spin {
  to {
    transform: rotate(360deg);
  }
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;

  &.small {
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    margin-bottom: 14px;
  }
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;

  span {
    color: var(--atk-admin-subtle);
    font-size: 12px;
  }

  strong {
    color: var(--atk-admin-ink);
    font-size: 20px;
    font-weight: 700;
  }

  &.plain {
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-sm);
    background: var(--atk-admin-surface-muted);
  }
}

.card {
  overflow: hidden;
}

.card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 22px 0;

  h2 {
    margin: 0;
    color: var(--atk-admin-ink);
    font-size: 16px;
    font-weight: 700;
  }

  p {
    margin: 6px 0 0;
    color: var(--atk-admin-subtle);
    font-size: 12.5px;
    line-height: 1.55;
  }
}

.head-tools {
  display: flex;
  flex: none;
  gap: 8px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px 22px 22px;
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--atk-admin-ink);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;

  input {
    width: 16px;
    height: 16px;
    accent-color: var(--atk-admin-sage);
  }
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}

.field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 7px;

  > span {
    color: var(--atk-admin-subtle);
    font-size: 12px;
    font-weight: 650;
  }
}

input,
select,
textarea {
  width: 100%;
  padding: 0 12px;
  border: 1px solid var(--atk-admin-border);
  border-radius: var(--atk-admin-radius-xs);
  color: var(--atk-admin-ink);
  background: var(--atk-admin-surface-muted);
  font-size: 13px;
  font-family: inherit;

  &:focus {
    border-color: var(--atk-admin-sage);
    background: var(--atk-admin-surface);
    box-shadow: 0 0 0 3px var(--atk-admin-ring);
    outline: none;
  }
}

input,
select {
  height: 40px;
}

textarea {
  padding: 10px 12px;
  line-height: 1.6;
  resize: vertical;
}

.form-msg {
  color: var(--atk-admin-subtle);
  font-size: 12.5px;
  line-height: 1.5;
  margin: 0;

  &.error {
    color: var(--atk-admin-danger);
  }

  &.ok {
    color: var(--atk-admin-sage-strong);
  }

  &.warn {
    color: var(--atk-admin-terracotta);
  }
}

.hint {
  margin: 0;
  color: var(--atk-admin-faint);
  font-size: 12px;
  line-height: 1.6;
}

.form-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 0 4px;
}

.sub-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--atk-admin-border);
}

.quota-table {
  width: 100%;
  border-collapse: collapse;

  th,
  td {
    padding: 8px 6px;
    text-align: left;
    font-size: 12.5px;
    color: var(--atk-admin-subtle);
  }

  td input {
    max-width: 140px;
  }
}

.ban-form,
.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;

  input,
  select {
    flex: 1;
    min-width: 120px;
    width: auto;
  }

  .admin-button {
    flex: none;
  }
}

.table-wrap {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;

  th,
  td {
    padding: 10px 8px;
    border-bottom: 1px solid var(--atk-admin-border);
    text-align: left;
    vertical-align: top;
    color: var(--atk-admin-ink);
  }

  th {
    color: var(--atk-admin-subtle);
    font-weight: 650;
    white-space: nowrap;
  }

  small {
    display: block;
    margin-top: 3px;
    color: var(--atk-admin-faint);
  }
}

.cell-actions {
  text-align: right;
  white-space: nowrap;
}

.empty-cell {
  padding: 22px 8px;
  color: var(--atk-admin-faint);
  font-size: 12.5px;
  text-align: center;
}

.report-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 18px;

  h3 {
    margin: 0 0 8px;
    color: var(--atk-admin-ink);
    font-size: 13px;
    font-weight: 700;
  }
}

.bars {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;

  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 6px 10px;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-xs);
    background: var(--atk-admin-surface-muted);
    color: var(--atk-admin-ink);
    font-size: 12px;
  }

  em {
    color: var(--atk-admin-subtle);
    font-style: normal;
  }

  small {
    margin-left: 6px;
    color: var(--atk-admin-faint);
  }
}

.bar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.link-btn {
  padding: 0;
  border: 0;
  background: none;
  color: var(--atk-admin-sage-strong);
  font-size: 12px;
  cursor: pointer;

  &:hover {
    text-decoration: underline;
  }
}

.event-list {
  display: flex;
  flex-direction: column;
}

.event-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--atk-admin-border);

  &:last-child {
    border-bottom: 0;
  }
}

.event-main {
  min-width: 0;
  flex: 1;
}

.event-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;

  b {
    color: var(--atk-admin-ink);
    font-size: 13.5px;
  }

  span {
    padding: 2px 8px;
    border-radius: 5px;
    background: var(--atk-admin-surface-muted);
    color: var(--atk-admin-subtle);
    font-size: 11.5px;
  }

  .blocked {
    color: var(--atk-admin-terracotta);
  }
}

.event-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 6px;
  color: var(--atk-admin-subtle);
  font-size: 12px;
}

.event-main code {
  display: block;
  margin-top: 6px;
  color: var(--atk-admin-faint);
  font-size: 11.5px;
  word-break: break-all;
}

@media (max-width: 680px) {
  .security-admin {
    padding: 16px 14px 70px;
  }

  .card-head {
    flex-direction: column;
    padding-right: 16px;
    padding-left: 16px;
  }

  .card-body {
    padding-right: 16px;
    padding-left: 16px;
  }

  .form-foot {
    align-items: stretch;
    flex-direction: column;

    .admin-button {
      width: 100%;
    }
  }

  .event-item {
    flex-direction: column;
  }
}
</style>
