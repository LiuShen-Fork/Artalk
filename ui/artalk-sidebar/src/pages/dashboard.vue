<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'

type Metric = {
  total: number
  new_90d?: number
  today?: number
}

type TrendPoint = {
  date: string
  comments: number
  users: number
}

type TopPage = {
  id: number
  key: string
  title: string
  site_name: string
  pv: number
  comment_count: number
}

type Review = {
  id: number
  checker: string
  status: 'pass' | 'block' | 'error'
  action: string
  message: string
  date: string
  user_name: string
  comment_content: string
}

type DashboardData = {
  pv: Metric
  comments: Metric
  users: Metric
  pending_comments: number
  pages: number
  trend: TrendPoint[]
  moderation: {
    pass: number
    block: number
    error: number
  }
  top_pages: TopPage[]
  recent_reviews: Review[]
}

const nav = useNavStore()
const user = useUserStore()
const { site: curtSite } = storeToRefs(user)

const data = ref<DashboardData | null>(null)
const loading = ref(false)
const numberFormatter = new Intl.NumberFormat()

const formatNumber = (value?: number) => numberFormatter.format(value || 0)
const trendMax = computed(() =>
  Math.max(1, ...trendWeeks.value.map((item) => Math.max(item.comments, item.users))),
)
/** Group the 90-day series into readable weekly buckets. */
const trendWeeks = computed(() => {
  const items = data.value?.trend || []
  const weeks: { label: string; comments: number; users: number; range: string }[] = []

  for (let end = items.length; end > 0; end -= 7) {
    const start = Math.max(0, end - 7)
    const slice = items.slice(start, end)
    const first = slice[0]
    const last = slice[slice.length - 1]
    weeks.unshift({
      label: first.date.slice(5),
      comments: slice.reduce((sum, item) => sum + item.comments, 0),
      users: slice.reduce((sum, item) => sum + item.users, 0),
      range: `${first.date} — ${last.date}`,
    })
  }

  return weeks
})
const trendRange = computed(() => {
  const items = data.value?.trend || []
  if (!items.length) return ''
  return `${items[0].date} — ${items[items.length - 1].date}`
})
const trendPeak = computed(() =>
  Math.max(0, ...trendWeeks.value.map((item) => Math.max(item.comments, item.users))),
)

function barHeight(value: number) {
  return String(Math.max(2, (value / trendMax.value) * 100)) + '%'
}

async function fetchDashboard() {
  loading.value = true
  try {
    const res = await artalk!.ctx.getApi().request<DashboardData>({
      path: '/dashboard',
      method: 'GET',
      query: {
        site_name: curtSite.value || undefined,
      },
      secure: true,
      format: 'json',
    })
    data.value = res.data
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  nav.updateTabs({})
  fetchDashboard()
  watch(curtSite, fetchDashboard)
})
</script>

<template>
  <div class="dashboard-page admin-page">
    <AdminPageHeader
      title="总览"
      description="最近 90 天的评论与用户趋势，以及需要你处理的审核事项。"
    >
      <template #actions>
        <button class="admin-button" :disabled="loading" @click="fetchDashboard">
          {{ loading ? '刷新中…' : '刷新数据' }}
        </button>
      </template>
    </AdminPageHeader>

    <section class="stat-panel admin-panel">
      <div class="stat">
        <span class="stat-label">累计 PV</span>
        <strong class="stat-value">{{ formatNumber(data?.pv.total) }}</strong>
        <span class="stat-meta">永久累计记录</span>
      </div>
      <div class="stat">
        <span class="stat-label">评论总数</span>
        <strong class="stat-value">{{ formatNumber(data?.comments.total) }}</strong>
        <span class="stat-meta">
          今日 +{{ formatNumber(data?.comments.today) }} · 90 天 +{{
            formatNumber(data?.comments.new_90d)
          }}
        </span>
      </div>
      <div class="stat">
        <span class="stat-label">用户总数</span>
        <strong class="stat-value">{{ formatNumber(data?.users.total) }}</strong>
        <span class="stat-meta">
          今日 +{{ formatNumber(data?.users.today) }} · 90 天 +{{
            formatNumber(data?.users.new_90d)
          }}
        </span>
      </div>
      <div class="stat" :class="{ alert: (data?.pending_comments || 0) > 0 }">
        <span class="stat-label">待审评论</span>
        <strong class="stat-value">{{ formatNumber(data?.pending_comments) }}</strong>
        <span class="stat-meta">
          {{ (data?.pending_comments || 0) > 0 ? '需要人工处理' : '当前没有积压' }}
        </span>
      </div>
    </section>

    <section class="content-grid">
      <div class="dashboard-panel admin-panel trend-panel">
        <div class="panel-head">
          <div>
            <h2>90 天趋势</h2>
            <p>评论与新增用户按周汇总</p>
          </div>
          <span class="panel-note">
            {{ trendRange }}<template v-if="trendPeak"> · 峰值 {{ trendPeak }}</template>
          </span>
        </div>
        <div class="trend-chart">
          <div
            v-for="week in trendWeeks"
            :key="week.range"
            class="trend-week"
            :title="`${week.range} · 评论 ${week.comments} · 用户 ${week.users}`"
          >
            <span class="bar comments" :style="{ height: barHeight(week.comments) }" />
            <span class="bar users" :style="{ height: barHeight(week.users) }" />
          </div>
        </div>
        <div class="trend-axis">
          <span v-for="(week, index) in trendWeeks" :key="week.range">
            {{ index % 2 === 0 ? week.label : '' }}
          </span>
        </div>
        <div class="legend">
          <span><i class="comments" />评论</span>
          <span><i class="users" />用户</span>
        </div>
      </div>

      <div class="dashboard-panel admin-panel moderation-panel">
        <div class="panel-head">
          <div>
            <h2>审核概况</h2>
            <p>审核流水仅保留最近 90 天</p>
          </div>
          <RouterLink class="panel-link" to="/moderation">查看全部</RouterLink>
        </div>
        <div class="review-summary">
          <RouterLink class="ok" :to="{ path: '/moderation', query: { status: 'replace' } }">
            <b>{{ formatNumber(data?.moderation.pass) }}</b>
            <span>已替换</span>
          </RouterLink>
          <RouterLink class="warn" :to="{ path: '/moderation', query: { status: 'block' } }">
            <b>{{ formatNumber(data?.moderation.block) }}</b>
            <span>非正常</span>
          </RouterLink>
          <RouterLink class="bad" :to="{ path: '/moderation', query: { status: 'error' } }">
            <b>{{ formatNumber(data?.moderation.error) }}</b>
            <span>系统失败</span>
          </RouterLink>
        </div>
        <div class="review-list">
          <div v-for="item in data?.recent_reviews || []" :key="item.id" class="review-row">
            <span class="status-dot" :class="item.status" />
            <div class="review-copy">
              <b>{{ item.checker }}</b>
              <p>{{ item.message || item.comment_content || '无详情' }}</p>
            </div>
          </div>
          <div v-if="!data?.recent_reviews?.length" class="empty">暂无审核流水</div>
        </div>
      </div>
    </section>

    <section class="dashboard-panel admin-panel top-pages">
      <div class="panel-head">
        <div>
          <h2>热门页面</h2>
          <p>按永久 PV 累计排序</p>
        </div>
      </div>
      <div class="page-table">
        <div v-for="(page, index) in data?.top_pages || []" :key="page.id" class="page-row">
          <span class="page-rank">{{ String(index + 1).padStart(2, '0') }}</span>
          <div class="page-copy">
            <b>{{ page.title || page.key }}</b>
            <span>{{ page.key }}</span>
          </div>
          <strong>{{ formatNumber(page.pv) }} <small>PV</small></strong>
          <em>{{ formatNumber(page.comment_count) }} 评论</em>
        </div>
        <div v-if="!data?.top_pages?.length" class="empty">暂无页面数据</div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.stat-panel {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 18px;
  overflow: hidden;
}

.stat {
  position: relative;
  min-height: 128px;
  padding: 20px 22px;
  border-right: 1px solid var(--atk-admin-border);

  &:last-child {
    border-right: 0;
  }

  &.alert::after {
    content: '';
    position: absolute;
    top: 18px;
    right: 18px;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--atk-admin-terracotta);
    box-shadow: 0 0 0 4px var(--atk-admin-terracotta-soft);
  }

  &.alert .stat-value {
    color: var(--atk-admin-terracotta);
  }
}

.stat-label {
  display: block;
  color: var(--atk-admin-subtle);
  font-size: 12.5px;
  font-weight: 550;
}

.stat-value {
  display: block;
  margin: 8px 0 4px;
  color: var(--atk-admin-ink);
  font-size: 30px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
  line-height: 1.15;
}

.stat-meta {
  display: block;
  color: var(--atk-admin-faint);
  font-size: 12px;
  line-height: 1.5;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(300px, 0.95fr);
  gap: 18px;
}

.dashboard-panel {
  padding: 20px 22px;
}

.top-pages {
  margin-top: 18px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h2 {
    margin: 0 0 4px;
    color: var(--atk-admin-ink);
    font-size: 16px;
    font-weight: 650;
    letter-spacing: -0.01em;
  }

  p {
    margin: 0;
    color: var(--atk-admin-faint);
    font-size: 12.5px;
  }
}

.panel-note {
  color: var(--atk-admin-faint);
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.panel-link {
  color: var(--atk-admin-sage);
  font-size: 12.5px;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;

  &:hover {
    text-decoration: underline;
  }
}

.trend-chart {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(4px, 1fr));
  gap: 5px;
  align-items: end;
  height: 156px;
  padding: 16px 4px 0;
  border-bottom: 1px solid var(--atk-admin-border);
  background: repeating-linear-gradient(
    to top,
    transparent,
    transparent calc(25% - 1px),
    var(--atk-admin-border) calc(25% - 1px),
    var(--atk-admin-border) 25%
  );
  background-size: 100% 100%;
}

.trend-week {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 100%;
}

.trend-axis {
  display: flex;
  margin: 8px 4px 0;
  color: var(--atk-admin-faint);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  white-space: nowrap;

  span {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.bar {
  display: block;
  width: 50%;
  min-height: 2px;
  border-radius: 3px 3px 1px 1px;
  transition:
    filter 0.18s ease,
    opacity 0.18s ease;

  &.comments {
    background: linear-gradient(180deg, var(--atk-admin-sage), var(--atk-admin-sage-strong));
    opacity: 0.9;
  }

  &.users {
    background: linear-gradient(180deg, var(--atk-admin-terracotta), #9c452c);
    opacity: 0.8;
  }

  .trend-week:hover & {
    opacity: 1;
    filter: brightness(1.08);
  }
}

.legend {
  display: flex;
  gap: 18px;
  margin-top: 12px;
  color: var(--atk-admin-subtle);
  font-size: 12.5px;

  i {
    display: inline-block;
    width: 8px;
    height: 8px;
    margin-right: 6px;
    border-radius: 50%;

    &.comments {
      background: var(--atk-admin-sage);
    }

    &.users {
      background: var(--atk-admin-terracotta);
    }
  }
}

.review-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 10px;

  a {
    padding: 10px 12px;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-sm);
    color: inherit;
    background: var(--atk-admin-surface-muted);
    text-decoration: none;
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-sage-soft);
    }
  }

  b,
  span {
    display: block;
  }

  b {
    color: var(--atk-admin-ink);
    font-size: 19px;
    font-variant-numeric: tabular-nums;
  }

  span {
    color: var(--atk-admin-subtle);
    font-size: 11.5px;
  }

  .ok b {
    color: var(--atk-admin-sage);
  }

  .warn b {
    color: var(--atk-admin-terracotta);
  }

  .bad b {
    color: var(--atk-admin-danger);
  }
}

.review-row,
.page-row {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 11px 0;
  border-top: 1px solid var(--atk-admin-border);
}

.review-row:first-child {
  border-top: 0;
}

.review-copy {
  min-width: 0;
}

.review-row p {
  margin: 3px 0 0;
  overflow: hidden;
  color: var(--atk-admin-subtle);
  font-size: 12.5px;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.status-dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;

  &.pass {
    background: var(--atk-admin-sage);
  }

  &.block {
    background: var(--atk-admin-terracotta);
  }

  &.error {
    background: var(--atk-admin-danger);
  }
}

.page-row {
  justify-content: space-between;
}

.page-rank {
  flex: none;
  width: 24px;
  color: var(--atk-admin-faint);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

.page-copy {
  flex: 1;
  min-width: 0;

  b,
  span {
    display: block;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  b {
    color: var(--atk-admin-ink);
    font-size: 13.5px;
    font-weight: 600;
  }

  span {
    margin-top: 2px;
    color: var(--atk-admin-faint);
    font-size: 11.5px;
  }
}

.page-row > strong,
.page-row > em {
  flex: none;
  color: var(--atk-admin-subtle);
  font-size: 12.5px;
  font-style: normal;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.page-row > strong {
  color: var(--atk-admin-ink);
  font-weight: 650;

  small {
    color: var(--atk-admin-faint);
    font-size: 10.5px;
    font-weight: 600;
  }
}

.empty {
  padding: 24px 8px;
  color: var(--atk-admin-faint);
  font-size: 13px;
  text-align: center;
}

@media (max-width: 1180px) {
  .stat-panel {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .stat {
    border-bottom: 1px solid var(--atk-admin-border);

    &:nth-child(2n) {
      border-right: 0;
    }

    &:nth-last-child(-n + 2) {
      border-bottom: 0;
    }
  }

  .content-grid {
    display: block;
  }

  .dashboard-panel {
    margin-bottom: 16px;
  }

  .top-pages {
    margin-top: 0;
  }
}

@media (max-width: 560px) {
  .stat {
    min-height: 112px;
    padding: 16px;
  }

  .stat-value {
    font-size: 25px;
  }

  .dashboard-panel {
    padding: 16px;
  }

  .panel-note {
    display: none;
  }

  .page-row > em {
    display: none;
  }
}
</style>
