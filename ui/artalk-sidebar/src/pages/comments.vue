<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { artalk } from '../global'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'

const wrapEl = ref<HTMLElement>()
const listEl = ref<HTMLElement>()
const user = useUserStore()
const nav = useNavStore()
const { curtTab } = storeToRefs(nav)
const { site: curtSite } = storeToRefs(user)

const search = ref('')

const normalizeEmptyCommentState = (comments: unknown[]) => {
  nextTick(() => {
    const commentsWrap = artalk!.ctx.inject('list').getCommentsWrapEl()
    const noComment = commentsWrap.querySelector<HTMLElement>('.atk-list-no-comment')

    if (comments.length > 0 || !noComment) return

    commentsWrap.querySelectorAll('.atk-comment-wrap').forEach((item) => item.remove())
    noComment.textContent = '暂无评论'
  })
}

onMounted(() => {
  // 初始化导航条
  if (user.is_admin) {
    nav.updateTabs(
      {
        all: 'all',
        pending: 'pending',
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

  watch(curtTab, (curtTab) => {
    artalk!.ctx.fetch({
      offset: 0,
    })
  })

  watch(curtSite, (value) => {
    artalk!.ctx.reload()
  })

  artalk!.ctx.on('comment-rendered', (comment) => {
    const pageURL = comment.getData().page_url
    comment.getRender().setOpenURL(`${pageURL}#atk-comment-${comment.getID()}`)
  })

  artalk!.ctx.on('list-loaded', normalizeEmptyCommentState)

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
      artalk!.reload()
    },
    () => {
      if (search.value === '') return
      search.value = ''
      artalk!.reload()
    },
  )
})
</script>

<template>
  <div ref="wrapEl" class="comments-wrap admin-page">
    <div ref="listEl" />
  </div>
</template>

<style scoped lang="scss">
.comments-wrap {
  :deep(.atk-list) {
    overflow: visible;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius);
    background: var(--atk-admin-surface);
    box-shadow: var(--atk-admin-shadow-sm);
  }

  :deep(.atk-list-body) {
    min-height: 190px;
  }

  :deep(.atk-list-comments-wrap) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 14px;
  }

  :deep(.atk-list-comments-wrap > .atk-comment-wrap) {
    margin: 0;
    border: 1px solid var(--atk-admin-border);
    border-radius: 16px;
    background: var(--atk-admin-card);
    box-shadow: 0 12px 28px rgba(29, 36, 51, 0.06);
    transition:
      background-color 0.18s ease,
      border-color 0.18s ease,
      box-shadow 0.18s ease,
      transform 0.18s ease;

    &:hover {
      border-color: var(--atk-admin-border-strong);
      background: var(--atk-admin-card-hover);
      box-shadow: 0 16px 34px rgba(29, 36, 51, 0.1);
      transform: translateY(-1px);
    }

    & > .atk-comment {
      padding: 18px;
    }
  }

  :deep(.atk-list-no-comment) {
    min-height: 210px;
    height: auto;
    border: 1px dashed var(--atk-admin-border-strong);
    border-radius: 18px;
    color: var(--atk-admin-subtle);
    background:
      linear-gradient(135deg, rgba(99, 102, 241, 0.08), transparent 34%),
      linear-gradient(315deg, rgba(20, 184, 166, 0.1), transparent 36%),
      var(--atk-admin-card);
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

  :deep(.atk-comment > .atk-main > .atk-body > .atk-content) {
    color: var(--atk-admin-ink);
    line-height: 1.7;
  }

  :deep(.atk-comment > .atk-main > .atk-footer) {
    margin-top: 10px;
  }

  @media (max-width: 560px) {
    :deep(.atk-list-comments-wrap) {
      gap: 10px;
      padding: 10px;
    }

    :deep(.atk-list-comments-wrap > .atk-comment-wrap) {
      & > .atk-comment {
        padding: 15px 12px;
      }
    }
  }
}
</style>
