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
  Math.max(1, ...(data.value?.trend || []).map((item) => Math.max(item.comments, item.users))),
)
const compactTrend = computed(() => {
  const items = data.value?.trend || []
  if (items.length <= 30) return items
  return items.filter((_, index) => index % 3 === 0 || index === items.length - 1)
})

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
  <div class="dashboard-page">
    <section class="hero-panel">
      <div>
        <div class="eyebrow">Overview</div>
        <h1>总览</h1>
        <p>最近 90 天趋势、评论增长、用户增长和审核概况。</p>
      </div>
      <button class="refresh-btn" :disabled="loading" @click="fetchDashboard">
        {{ loading ? '刷新中' : '刷新' }}
      </button>
    </section>

    <section class="metric-grid">
      <div class="metric-card primary">
        <span>累计 PV</span>
        <strong>{{ formatNumber(data?.pv.total) }}</strong>
        <small>保留页面 PV 的永久累计记录</small>
      </div>
      <div class="metric-card">
        <span>评论</span>
        <strong>{{ formatNumber(data?.comments.total) }}</strong>
        <small>
          今日 +{{ formatNumber(data?.comments.today) }} / 90 天 +{{
            formatNumber(data?.comments.new_90d)
          }}
        </small>
      </div>
      <div class="metric-card">
        <span>用户</span>
        <strong>{{ formatNumber(data?.users.total) }}</strong>
        <small>
          今日 +{{ formatNumber(data?.users.today) }} / 90 天 +{{
            formatNumber(data?.users.new_90d)
          }}
        </small>
      </div>
      <div class="metric-card">
        <span>待审评论</span>
        <strong>{{ formatNumber(data?.pending_comments) }}</strong>
        <small>当前需要人工处理</small>
      </div>
    </section>

    <section class="content-grid">
      <div class="panel trend-panel">
        <div class="panel-head">
          <div>
            <h2>90 天趋势</h2>
            <p>评论与新增用户按天汇总。</p>
          </div>
        </div>
        <div class="trend-chart">
          <div v-for="point in compactTrend" :key="point.date" class="trend-day" :title="point.date">
            <span class="bar comments" :style="{ height: barHeight(point.comments) }" />
            <span class="bar users" :style="{ height: barHeight(point.users) }" />
          </div>
        </div>
        <div class="legend">
          <span><i class="comments" />评论</span>
          <span><i class="users" />用户</span>
        </div>
      </div>

      <div class="panel moderation-panel">
        <div class="panel-head">
          <div>
            <h2>审核概况</h2>
            <p>审核流水仅保留最近 90 天。</p>
          </div>
          <RouterLink to="/moderation">查看</RouterLink>
        </div>
        <div class="review-summary">
          <RouterLink class="ok" :to="{ path: '/moderation', query: { status: 'replace' } }">
            <span class="review-icon">~</span>
            <b>{{ formatNumber(data?.moderation.pass) }}</b>
            <span>已替换</span>
          </RouterLink>
          <RouterLink class="warn" :to="{ path: '/moderation', query: { status: 'block' } }">
            <span class="review-icon">!</span>
            <b>{{ formatNumber(data?.moderation.block) }}</b>
            <span>非正常</span>
          </RouterLink>
          <RouterLink class="bad" :to="{ path: '/moderation', query: { status: 'error' } }">
            <span class="review-icon">X</span>
            <b>{{ formatNumber(data?.moderation.error) }}</b>
            <span>系统失败</span>
          </RouterLink>
        </div>
        <div class="review-list compact">
          <div v-for="item in data?.recent_reviews || []" :key="item.id" class="review-row">
            <span class="status-dot" :class="item.status" />
            <div>
              <b>{{ item.checker }}</b>
              <p>{{ item.message || item.comment_content || '无详情' }}</p>
            </div>
          </div>
          <div v-if="!data?.recent_reviews?.length" class="empty">暂无审核流水</div>
        </div>
      </div>
    </section>

    <section class="panel top-pages">
      <div class="panel-head">
        <div>
          <h2>热门页面</h2>
          <p>按永久 PV 累计排序。</p>
        </div>
      </div>
      <div class="page-table">
        <div v-for="page in data?.top_pages || []" :key="page.id" class="page-row">
          <div>
            <b>{{ page.title || page.key }}</b>
            <span>{{ page.key }}</span>
          </div>
          <strong>{{ formatNumber(page.pv) }} PV</strong>
          <em>{{ formatNumber(page.comment_count) }} 评论</em>
        </div>
        <div v-if="!data?.top_pages?.length" class="empty">暂无页面数据</div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.dashboard-page {
  padding: 24px 28px 70px;
}

.hero-panel,
.panel,
.metric-card {
  border: 1px solid var(--at-color-border);
  background: var(--at-color-bg);
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.05);
}

.hero-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 28px;
  border-radius: 10px;
  background:
    linear-gradient(135deg, rgba(54, 171, 207, 0.12), transparent 48%),
    var(--at-color-bg);

  h1 {
    margin: 4px 0 8px;
    font-size: 30px;
  }

  p {
    margin: 0;
    color: var(--at-color-sub);
  }
}

.eyebrow {
  color: var(--at-color-main);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

.refresh-btn {
  border: 0;
  background: #36abcf;
  color: #fff;
  border-radius: 6px;
  padding: 9px 18px;
  cursor: pointer;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
  margin: 18px 0;
}

.metric-card {
  border-radius: 8px;
  padding: 18px;

  span,
  small {
    display: block;
    color: var(--at-color-sub);
  }

  strong {
    display: block;
    margin: 10px 0 6px;
    font-size: 28px;
    color: var(--at-color-deep);
  }

  &.primary {
    border-color: rgba(54, 171, 207, 0.35);
  }
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(320px, 0.9fr);
  gap: 16px;
}

.panel {
  border-radius: 10px;
  padding: 20px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h2 {
    margin: 0 0 6px;
    font-size: 18px;
  }

  p {
    margin: 0;
    color: var(--at-color-sub);
    font-size: 13px;
  }

  a {
    color: var(--at-color-main);
    text-decoration: none;
  }
}

.trend-chart {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(5px, 1fr));
  gap: 4px;
  align-items: end;
  height: 210px;
  padding-top: 20px;
  border-bottom: 1px solid var(--at-color-border);
}

.trend-day {
  display: flex;
  align-items: flex-end;
  gap: 1px;
  height: 100%;
}

.bar {
  display: block;
  width: 50%;
  min-height: 2px;
  border-radius: 4px 4px 0 0;

  &.comments {
    background: #36abcf;
  }

  &.users {
    background: #f59f00;
  }
}

.legend {
  display: flex;
  gap: 18px;
  margin-top: 12px;
  color: var(--at-color-sub);
  font-size: 13px;

  i {
    display: inline-block;
    width: 9px;
    height: 9px;
    border-radius: 50%;
    margin-right: 6px;

    &.comments {
      background: #36abcf;
    }

    &.users {
      background: #f59f00;
    }
  }
}

.review-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 12px;

  > div,
  > a {
    border-radius: 8px;
    padding: 12px;
    background: var(--at-color-bg-grey-transl);
  }

  > a {
    color: inherit;
    text-decoration: none;
    transition:
      transform 0.16s ease,
      border-color 0.16s ease,
      background 0.16s ease;
  }

  a:hover {
    transform: translateY(-1px);
    background: rgba(54, 171, 207, 0.08);
  }

  .review-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    margin-bottom: 8px;
    border-radius: 50%;
    font-size: 12px;
    font-weight: 700;
  }

  .ok .review-icon {
    background: rgba(47, 158, 68, 0.12);
    color: #2f9e44;
  }

  .warn .review-icon {
    background: rgba(240, 140, 0, 0.14);
    color: #f08c00;
  }

  .bad .review-icon {
    background: rgba(224, 49, 49, 0.12);
    color: #e03131;
  }

  b,
  span {
    display: block;
  }

  b {
    font-size: 20px;
  }

  span {
    color: var(--at-color-sub);
    font-size: 12px;
  }
}

.review-row,
.page-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-top: 1px solid var(--at-color-border);
}

.review-row p,
.page-row span {
  margin: 3px 0 0;
  color: var(--at-color-sub);
  font-size: 13px;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex: none;

  &.pass {
    background: #2f9e44;
  }

  &.block {
    background: #f08c00;
  }

  &.error {
    background: #e03131;
  }
}

.page-row {
  justify-content: space-between;

  div {
    min-width: 0;
    flex: 1;
  }

  b,
  span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong,
  em {
    white-space: nowrap;
    font-style: normal;
  }

  em {
    color: var(--at-color-sub);
  }
}

.empty {
  padding: 20px;
  color: var(--at-color-sub);
  text-align: center;
}

@media (max-width: 850px) {
  .dashboard-page {
    padding: 16px 14px 70px;
  }

  .hero-panel,
  .content-grid {
    display: block;
  }

  .refresh-btn {
    margin-top: 18px;
  }

  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .panel {
    margin-bottom: 14px;
  }
}
</style>
