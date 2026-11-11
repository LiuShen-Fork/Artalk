<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'
import Pagination from '../components/Pagination.vue'

const wrapEl = ref<HTMLElement>()
const listEl = ref<HTMLElement>()
const user = useUserStore()
const nav = useNavStore()
const { curtTab } = storeToRefs(nav)
const { site: curtSite } = storeToRefs(user)

const search = ref('')
const listLoading = ref(false)
const pageTotal = ref(0)
let onListFetch: (() => void) | undefined
let onListFetched: (() => void) | undefined
let onListFailed: (() => void) | undefined
let onListLoaded: ((comments: unknown[]) => void) | undefined
let onCommentRendered:
  | ((comment: {
      getData: () => { page_url?: string }
      getRender: () => { setOpenURL: (url: string) => void }
      getID: () => number
    }) => void)
  | undefined
const pageSize = ref(20)
const currentPage = ref(1)

/** Mirror the list state into the sidebar's own pager. */
function syncPagination() {
  const data = artalk!.ctx.inject('data')
  const last = data.getListLastFetch()
  if (!last?.data) return

  pageTotal.value = last.params.flatMode ? last.data.count : last.data.roots_count
  pageSize.value = last.params.limit
  currentPage.value = Math.floor(last.params.offset / last.params.limit) + 1
}

function onPageChange(offset: number) {
  artalk!.ctx.editorResetState()
  artalk!.ctx.fetch({
    offset,
    onSuccess: () => artalk!.ctx.listGotoFirst(),
  })
}

const normalizeEmptyCommentState = (comments: unknown[]) => {
  nextTick(() => {
    const commentsWrap = artalk!.ctx.inject('list').getCommentsWrapEl()
    const noComment = commentsWrap.querySelector<HTMLElement>('.atk-list-no-comment')

    if (comments.length > 0 || !noComment) return

    commentsWrap.querySelectorAll('.atk-comment-wrap').forEach((item) => item.remove())
    noComment.textContent = '暂无评论'
  })
}

onUnmounted(() => {
  const ctx = artalk?.ctx
  if (!ctx) return
  if (onCommentRendered) ctx.off('comment-rendered', onCommentRendered)
  if (onListLoaded) ctx.off('list-loaded', onListLoaded)
  if (onListFetch) ctx.off('list-fetch', onListFetch)
  if (onListFetched) ctx.off('list-fetched', onListFetched)
  if (onListFailed) ctx.off('list-failed', onListFailed)
})

onMounted(() => {
  // 初始化导航条
  if (user.is_admin) {
    nav.updateTabs(
      {
        all: 'all',
        pending: 'pending',
        ai: 'aiAssistant',
        personal_all: 'personal',
      },
      'all',
    )
  } else {
    nav.updateTabs(
      {
        all: 'all',
        mentions: 'mentions',
        mine: 'mine',
        pending: 'pending',
      },
      'all',
    )
  }

  watch(curtTab, () => {
    currentPage.value = 1
    artalk!.ctx.fetch({
      offset: 0,
    })
  })

  watch(curtSite, (value) => {
    artalk!.ctx.reload()
  })

  onCommentRendered = (comment) => {
    const pageURL = comment.getData().page_url
    if (!pageURL) return
    comment.getRender().setOpenURL(`${pageURL}#atk-comment-${comment.getID()}`)
  }
  artalk!.ctx.on('comment-rendered', onCommentRendered)

  onListLoaded = (comments) => {
    normalizeEmptyCommentState(comments)
    syncPagination()
  }
  onListFetch = () => {
    listLoading.value = true
  }
  onListFetched = () => {
    listLoading.value = false
  }
  onListFailed = () => {
    listLoading.value = false
  }

  artalk!.ctx.on('list-loaded', onListLoaded)
  artalk!.ctx.on('list-fetch', onListFetch)
  artalk!.ctx.on('list-fetched', onListFetched)
  artalk!.ctx.on('list-failed', onListFailed)

  artalk!.ctx.updateConf({
    noComment: '暂无评论',
    listFetchParamsModifier: (params) => {
      params.site_name = curtSite.value

      let scope = user.is_admin ? 'site' : 'user'
      let type = curtTab.value

      if (curtTab.value === 'personal_all') {
        scope = 'user'
        type = 'all'
      }

      params.scope = scope
      params.type = type
      // The admin list intentionally keeps AI comments in the normal result;
      // the dedicated tab filters them server-side.
      params.split_ai = false

      if (search.value) params.search = search.value
    },
    scrollRelativeTo: () => wrapEl.value!,
  })

  artalk!.reload()

  const $el = artalk!.ctx.inject('list').getEl()

  $el.querySelector<HTMLElement>('.atk-list-header')!.style.display = 'none'
  $el.querySelector<HTMLElement>('.atk-list-footer')!.style.display = 'none'

  listEl.value?.append($el)

  // Comments search
  nav.enableSearch(
    (value: string) => {
      search.value = value
      currentPage.value = 1
      artalk!.reload()
    },
    () => {
      if (search.value === '') return
      search.value = ''
      currentPage.value = 1
      artalk!.reload()
    },
  )
})
</script>

<template>
  <div ref="wrapEl" class="comments-wrap admin-page">
    <div ref="listEl" />
    <Pagination
      :total="pageTotal"
      :page-size="pageSize"
      :page="currentPage"
      :disabled="listLoading"
      @change="onPageChange"
    />
  </div>
</template>

<style scoped lang="scss">
.comments-wrap {
  :deep(.atk-list) {
    overflow: visible;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius);
    background: var(--atk-admin-surface);
    box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);
  }

  :deep(.atk-list .atk-pagination-wrap) {
    display: none;
  }

  :deep(.atk-list-body) {
    min-height: 190px;
    padding: 16px;
  }

  :deep(.atk-list-comments-wrap) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 0;
  }

  :deep(.atk-list-comments-wrap > .atk-comment-wrap) {
    margin: 0;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-sm);
    background: var(--atk-admin-surface-muted);
    box-shadow: 0 12px 28px rgba(29, 36, 51, 0.06);
    transition:
      background-color 0.18s ease,
      border-color 0.18s ease,
      box-shadow 0.18s ease,
      transform 0.18s ease;

    &:hover {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-sage-soft);
      box-shadow: 0 16px 34px rgba(29, 36, 51, 0.1);
      transform: translateY(-1px);
    }

    & > .atk-comment {
      padding: 16px;
    }
  }

  :deep(.atk-list-no-comment) {
    min-height: 158px;
    height: auto;
    color: var(--atk-admin-subtle);
    border: 0;
    border-radius: 0;
    background: transparent;
    font-size: 15px;
    font-weight: 650;
    letter-spacing: 0;
  }

  :deep(.atk-comment > .atk-avatar img) {
    border-radius: 50%;
  }

  :deep(.atk-comment > .atk-main > .atk-header .atk-nick),
  :deep(.atk-comment > .atk-main > .atk-header .atk-nick a) {
    color: var(--atk-admin-sage);
    font-weight: 650;
  }

  :deep(.atk-icon-verified) {
    background-color: var(--atk-admin-sage);
    background-image: none;
    mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 22 22' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M20.396 11c-.018-.646-.215-1.275-.57-1.816-.354-.54-.852-.972-1.438-1.246.223-.607.27-1.264.14-1.897-.131-.634-.437-1.218-.882-1.687-.47-.445-1.053-.75-1.687-.882-.633-.13-1.29-.083-1.897.14-.273-.587-.704-1.086-1.245-1.44S11.647 1.62 11 1.604c-.646.017-1.273.213-1.813.568s-.969.854-1.24 1.44c-.608-.223-1.267-.272-1.902-.14-.635.13-1.22.436-1.69.882-.445.47-.749 1.055-.878 1.688-.13.633-.08 1.29.144 1.896-.587.274-1.087.705-1.443 1.245-.356.54-.555 1.17-.574 1.817.02.647.218 1.276.574 1.817.356.54.856.972 1.443 1.245-.224.606-.274 1.263-.144 1.896.13.634.433 1.218.877 1.688.47.443 1.054.747 1.687.878.633.132 1.29.084 1.897-.136.274.586.705 1.084 1.246 1.439.54.354 1.17.551 1.816.569.647-.016 1.276-.213 1.817-.567s.972-.854 1.245-1.44c.604.239 1.266.296 1.903.164.636-.132 1.22-.447 1.68-.907.46-.46.776-1.044.908-1.681s.075-1.299-.165-1.903c.586-.274 1.084-.705 1.439-1.246.354-.54.551-1.17.569-1.816zM9.662 14.85l-3.429-3.428 1.293-1.302 2.072 2.072 4.4-4.794 1.347 1.246z'/%3E%3C/svg%3E");
    mask-repeat: no-repeat;
    mask-position: center;
    mask-size: contain;
    -webkit-mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 22 22' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M20.396 11c-.018-.646-.215-1.275-.57-1.816-.354-.54-.852-.972-1.438-1.246.223-.607.27-1.264.14-1.897-.131-.634-.437-1.218-.882-1.687-.47-.445-1.053-.75-1.687-.882-.633-.13-1.29-.083-1.897.14-.273-.587-.704-1.086-1.245-1.44S11.647 1.62 11 1.604c-.646.017-1.273.213-1.813.568s-.969.854-1.24 1.44c-.608-.223-1.267-.272-1.902-.14-.635.13-1.22.436-1.69.882-.445.47-.749 1.055-.878 1.688-.13.633-.08 1.29.144 1.896-.587.274-1.087.705-1.443 1.245-.356.54-.555 1.17-.574 1.817.02.647.218 1.276.574 1.817.356.54.856.972 1.443 1.245-.224.606-.274 1.263-.144 1.896.13.634.433 1.218.877 1.688.47.443 1.054.747 1.687.878.633.132 1.29.084 1.897-.136.274.586.705 1.084 1.246 1.439.54.354 1.17.551 1.816.569.647-.016 1.276-.213 1.817-.567s.972-.854 1.245-1.44c.604.239 1.266.296 1.903.164.636-.132 1.22-.447 1.68-.907.46-.46.776-1.044.908-1.681s.075-1.299-.165-1.903c.586-.274 1.084-.705 1.439-1.246.354-.54.551-1.17.569-1.816zM9.662 14.85l-3.429-3.428 1.293-1.302 2.072 2.072 4.4-4.794 1.347 1.246z'/%3E%3C/svg%3E");
    -webkit-mask-repeat: no-repeat;
    -webkit-mask-position: center;
    -webkit-mask-size: contain;
  }

  :deep(.atk-comment > .atk-main > .atk-body > .atk-content) {
    color: var(--atk-admin-ink);
    line-height: 1.7;
  }

  :deep(.atk-comment > .atk-main > .atk-footer) {
    margin-top: 10px;
  }

  :deep(.atk-comment > .atk-main > .atk-footer .atk-actions) {
    row-gap: 4px;
    color: var(--atk-admin-subtle);
    font-size: 12px;
  }

  :deep(
    .atk-comment > .atk-main > .atk-footer .atk-actions > span:not(:last-child):not(.atk-hide)
  ) {
    margin-right: 13px;
  }

  @media (max-width: 560px) {
    :deep(.atk-list-body) {
      padding: 12px;
    }

    :deep(.atk-list-comments-wrap) {
      gap: 10px;
    }

    :deep(.atk-list-comments-wrap > .atk-comment-wrap) {
      & > .atk-comment {
        padding: 14px 12px;
      }
    }
  }
}
</style>
