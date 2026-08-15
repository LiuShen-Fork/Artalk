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

  artalk!.ctx.updateConf({
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
    overflow: hidden;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius);
    background: var(--atk-admin-surface);
    box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);
  }

  :deep(.atk-comment-wrap) {
    border-bottom: 1px solid var(--atk-admin-border);
  }
}
</style>
