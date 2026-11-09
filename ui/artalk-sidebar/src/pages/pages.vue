<script setup lang="ts">
import { storeToRefs } from 'pinia'
import type { ArtalkType } from 'artalk'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'
import Pagination from '../components/Pagination.vue'

const nav = useNavStore()
const { site: curtSite } = storeToRefs(useUserStore())
const pages = ref<ArtalkType.PageData[]>([])
const curtEditPageID = ref<number | null>(null)
const { t } = useI18n()

const pageSize = ref(20)
const pageTotal = ref(0)
const search = ref('')
const pagination = ref<InstanceType<typeof Pagination>>()
const showActBarBorder = ref(false)
const refreshBtn = ref({
  isRun: false,
  statusText: '',
})

onMounted(() => {
  nav.updateTabs(
    {
      all: 'all',
    },
    'all',
  )

  // Users search
  nav.enableSearch(
    (value: string) => {
      search.value = value
      fetchPages(0)
    },
    () => {
      if (search.value === '') return
      search.value = ''
      fetchPages(0)
    },
  )

  fetchPages(0)

  // Refresh task status recovery
  getRefreshTaskStatus().then((d) => {
    if (d.is_progress === true) {
      refreshBtn.value.isRun = true
      refreshBtn.value.statusText = d.msg
      startRefreshTaskWatchdog()
    }
  })
})

watch(curtSite, () => {
  pagination.value?.reset()
  fetchPages(0)
})

onMounted(() => nav.scrollableArea?.addEventListener('scroll', scrollHandler))
onUnmounted(() => nav.scrollableArea?.removeEventListener('scroll', scrollHandler))

function scrollHandler() {
  showActBarBorder.value = nav.scrollableArea!.scrollTop > 10
}

function editPage(page: ArtalkType.PageData) {
  curtEditPageID.value = page.id
}

function fetchPages(offset: number) {
  if (offset === 0) pagination.value?.reset()
  nav.setPageLoading(true)
  artalk?.ctx
    .getApi()
    .pages.getPages({
      site_name: curtSite.value,
      offset: offset,
      limit: pageSize.value,
      search: search.value,
      has_comments: true,
    })
    .then((res) => {
      pageTotal.value = res.data.count || 0
      pages.value = res.data.pages || []
      nav.scrollPageToTop()
    })
    .finally(() => {
      nav.setPageLoading(false)
    })
}

function onChangePage(offset: number) {
  fetchPages(offset)
}

function onPageItemUpdate(page: ArtalkType.PageData) {
  const index = pages.value.findIndex((p) => p.id === page.id)
  if (index != -1) {
    const orgPage = pages.value[index]
    Object.keys(page).forEach((key) => {
      ;(orgPage as any)[key] = (page as any)[key]
    })
  }
}

function onPageItemRemove(id: number) {
  const index = pages.value.findIndex((p) => p.id === id)
  pages.value.splice(index, 1)
}

async function getRefreshTaskStatus() {
  return (await artalk!.ctx.getApi().pages.getPageFetchStatus()).data
}

function startRefreshTaskWatchdog() {
  // TODO: Not perfect polling update status
  const timerID = window.setInterval(async () => {
    const d = await getRefreshTaskStatus()

    if (d.is_progress === false) {
      clearInterval(timerID)
      setRefreshTaskDone()
      return
    }

    refreshBtn.value.statusText = d.msg
  }, 1000)
}

function setRefreshTaskDone() {
  refreshBtn.value.statusText = t('updateComplete')
  window.setTimeout(() => {
    refreshBtn.value.isRun = false
  }, 1500)
}

async function refreshAllPages() {
  if (refreshBtn.value.isRun) return
  refreshBtn.value.isRun = true
  refreshBtn.value.statusText = t('updateReady')

  try {
    await artalk!.ctx.getApi().pages.fetchAllPages({
      site_name: curtSite.value,
    })
  } catch (err: any) {
    alert(err.msg)
    setRefreshTaskDone()
    return
  }

  startRefreshTaskWatchdog()
}

function cacheFlush() {
  artalk!.ctx
    .getApi()
    .cache.flushCache()
    .then((res) => alert(res.data.msg))
    .catch(() => alert(t('opFailed')))
}

function cacheWarm() {
  artalk!.ctx
    .getApi()
    .cache.warmUpCache()
    .then((res) => alert(res.data.msg))
    .catch(() => alert(t('opFailed')))
}

function openPage(url: string) {
  window.open(url)
}
</script>

<template>
  <div class="atk-page-list-wrap admin-page">
    <AdminPageHeader :title="t('page')" description="管理评论页面的标题与缓存。" />
    <div class="atk-header-action-bar" :class="{ bordered: showActBarBorder }">
      <button
        type="button"
        class="atk-update-all-title-btn"
        :disabled="refreshBtn.isRun"
        @click="refreshAllPages()"
      >
        {{ refreshBtn.isRun ? refreshBtn.statusText : t('updateTitle') }}
      </button>
      <button type="button" class="atk-cache-flush-all-btn" @click="cacheFlush()">
        {{ t('cacheClear') }}
      </button>
      <button type="button" class="atk-cache-warm-up-btn" @click="cacheWarm()">
        {{ t('cacheWarm') }}
      </button>
    </div>
    <div class="atk-page-list admin-panel">
      <div v-if="!pages.length && !nav.isPageLoading" class="admin-empty page-empty">
        {{ t('noPages') }}
      </div>
      <div v-for="page in pages" :key="page.id" class="atk-page-item">
        <div class="atk-page-main">
          <div class="atk-title" @click="openPage(page.url)">
            {{ page.title }}
          </div>
          <div class="atk-sub" @click="openPage(page.url)">{{ page.url }}</div>
        </div>
        <div class="atk-page-actions">
          <div class="atk-item atk-edit-btn" @click="editPage(page)">
            <i class="atk-icon atk-icon-edit"></i>
          </div>
        </div>
        <PageEditor
          v-if="curtEditPageID === page.id"
          :page="page"
          @close="curtEditPageID = null"
          @update="onPageItemUpdate"
          @remove="onPageItemRemove"
        />
      </div>
    </div>
    <Pagination
      ref="pagination"
      :page-size="pageSize"
      :total="pageTotal"
      :disabled="nav.isPageLoading"
      @change="onChangePage"
    />
  </div>
</template>

<style scoped lang="scss">
.atk-page-list-wrap {
  .atk-header-action-bar {
    position: sticky;
    top: 0;
    display: flex;
    align-items: center;
    overflow: hidden;
    padding: 10px 15px 0 15px;
    background: var(--at-color-bg);
    z-index: 10;
    border-bottom: 1px solid transparent;
    transition: 0.3s ease-out padding;

    &.bordered {
      padding-bottom: 10px;
      border-color: var(--at-color-border);
    }

    & > button {
      display: inline-flex;
      align-items: center;
      flex-direction: row;
      padding: 2px 10px;
      border: 0;
      cursor: pointer;
      font-size: 13px;

      &:hover {
        background: var(--at-color-bg-grey);
      }
    }
  }
}

.atk-page-list-wrap {
  .atk-header-action-bar {
    position: static;
    gap: 8px;
    margin-bottom: 14px;
    padding: 0;
    background: transparent;
    border: 0;

    &.bordered {
      padding-bottom: 0;
      border: 0;
    }

    & > button {
      height: 36px;
      padding: 0 13px;
      border: 1px solid var(--atk-admin-border);
      border-radius: var(--atk-admin-radius-xs);
      color: var(--atk-admin-subtle);
      background: var(--atk-admin-surface);
      font-size: 12.5px;
      font-weight: 550;
      white-space: nowrap;
      transition:
        color 0.18s ease,
        background-color 0.18s ease,
        border-color 0.18s ease;

      &:hover:not(:disabled) {
        border-color: var(--atk-admin-sage);
        color: var(--atk-admin-ink);
        background: var(--atk-admin-surface-muted);
      }

      &:disabled {
        cursor: default;
        opacity: 0.6;
      }
    }
  }

  .atk-page-list {
    overflow: hidden;
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface);
    border-radius: var(--atk-admin-radius);
    box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);

    .page-empty {
      min-height: 180px;
      justify-content: center;
      font-size: 13px;

      &::before {
        content: '';
        width: 24px;
        height: 24px;
        margin-bottom: 2px;
        background: var(--atk-admin-faint);
        mask: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='1.7' stroke-linecap='round' xmlns='http://www.w3.org/2000/svg'%3E%3Ccircle cx='10.5' cy='10.5' r='6'/%3E%3Cpath d='m15 15 5 5'/%3E%3C/svg%3E")
          center / contain no-repeat;
        -webkit-mask: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='1.7' stroke-linecap='round' xmlns='http://www.w3.org/2000/svg'%3E%3Ccircle cx='10.5' cy='10.5' r='6'/%3E%3Cpath d='m15 15 5 5'/%3E%3C/svg%3E")
          center / contain no-repeat;
      }
    }

    .atk-page-item {
      min-height: 96px;
      border-color: var(--atk-admin-border);
    }

    .atk-page-main {
      padding: 18px 20px;

      .atk-title {
        color: var(--atk-admin-ink);
        font-size: 18px;
      }

      .atk-sub {
        color: var(--atk-admin-subtle);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }
  }
}

@media (max-width: 700px) {
  .atk-page-list-wrap .atk-header-action-bar {
    flex-wrap: wrap;
  }

  .atk-page-list-wrap .atk-header-action-bar > button {
    flex: 1 1 auto;
    justify-content: center;
  }
}

.atk-page-list {
  .atk-page-item {
    display: flex;
    flex-direction: row;
    position: relative;
    min-height: 84px;
    align-items: center;
    transition: background-color 0.18s ease;

    &:not(:last-child) {
      border-bottom: 1px solid var(--atk-admin-border);
    }

    &:hover {
      background: var(--atk-admin-surface-muted);
    }
  }

  .atk-page-main {
    display: flex;
    min-width: 0;
    flex-direction: column;
    flex: auto;
    padding: 15px 18px;

    .atk-title {
      margin-bottom: 5px;
      color: var(--atk-admin-ink);
      font-size: 15.5px;
      font-weight: 600;
      letter-spacing: -0.01em;
      cursor: pointer;
    }

    .atk-sub {
      overflow: hidden;
      color: var(--atk-admin-subtle);
      font-size: 12.5px;
      text-overflow: ellipsis;
      white-space: nowrap;
      cursor: pointer;
    }
  }

  :deep(.atk-page-actions) {
    @extend .atk-list-btn-actions;
    padding-right: 12px;
  }

  :deep(.atk-edit-btn) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border: 1px solid transparent;
    border-radius: 50%;
    color: var(--atk-admin-subtle);
    cursor: pointer;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      border-color 0.18s ease;

    &:hover {
      border-color: var(--atk-admin-border);
      color: var(--atk-admin-ink);
      background: var(--atk-admin-surface);
    }
  }
}
</style>
