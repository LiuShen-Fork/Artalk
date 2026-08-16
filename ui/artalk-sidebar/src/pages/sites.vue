<script setup lang="ts">
import type { ArtalkType } from 'artalk'
import { useNavStore } from '../stores/nav'
import { artalk, bootParams } from '../global'

const nav = useNavStore()
const sites = ref<ArtalkType.SiteData[]>([])
const curtEditSite = ref<ArtalkType.SiteData | null>(null)
const showSiteCreate = ref(false)
const siteCreateInitVal = ref()
const { t } = useI18n()

onMounted(() => {
  nav.updateTabs({}, 'sites')

  nav.setPageLoading(true)
  artalk?.ctx
    .getApi()
    .sites.getSites()
    .then((res) => {
      sites.value = res.data.sites
    })
    .finally(() => {
      nav.setPageLoading(false)
    })

  // Open site create dialog by view params (from URL query)
  const vp = bootParams.viewParams
  if (vp && vp.create_name && vp.create_urls) {
    siteCreateInitVal.value = { name: vp.create_name, urls: vp.create_urls }
    showSiteCreate.value = true
    nextTick(() => {
      siteCreateInitVal.value = null
      bootParams.viewParams = null
    })
  }
})

function create() {
  curtEditSite.value = null
  showSiteCreate.value = true
}

const sitesGrouped = computed(() => {
  if (sites.value.length === 0) return []

  const grp: ArtalkType.SiteData[][] = []
  let j = -1
  for (let i = 0; i < sites.value.length; i++) {
    const item = sites.value[i]
    if (i % 4 === 0) {
      // Each row has 4 items
      grp.push([])
      j++
    }
    grp[j].push(item)
  }
  return grp
})

function edit(site: ArtalkType.SiteData) {
  showSiteCreate.value = false
  curtEditSite.value = site
}

function onNewSiteCreated(siteNew: ArtalkType.SiteData) {
  sites.value.push(siteNew)
  showSiteCreate.value = false
  nav.refreshSites()
}

function onSiteItemUpdate(site: ArtalkType.SiteData) {
  const index = sites.value.findIndex((s) => s.id === site.id)
  if (index != -1) {
    const orgSite = sites.value[index]
    Object.keys(site).forEach((key) => {
      ;(orgSite as any)[key] = (site as any)[key]
    })
  }
  nav.refreshSites()
}

function onSiteItemRemove(id: number) {
  const index = sites.value.findIndex((p) => p.id === id)
  sites.value.splice(index, 1)
  nav.refreshSites()
}
</script>

<template>
  <div class="atk-site-list admin-page">
    <AdminPageHeader eyebrow="Sites" :title="t('siteManage')" :description="t('siteCount', { count: sites.length })">
      <template #actions>
        <button class="admin-button" type="button" @click="create()">
          <i class="atk-icon atk-icon-plus" />
          {{ t('createSite') }}
        </button>
      </template>
    </AdminPageHeader>
    <SiteCreate
      v-if="showSiteCreate"
      :init-val="siteCreateInitVal"
      @close="showSiteCreate = false"
      @done="onNewSiteCreated"
    />
    <div class="atk-site-rows-wrap">
      <template v-for="(ss, i) in sitesGrouped" :key="i">
        <div class="atk-site-row">
          <div
            v-for="site in ss"
            :key="site.id"
            class="atk-site-item"
            :class="{ 'atk-active': curtEditSite === site }"
            @click="edit(site)"
          >
            <div class="atk-site-logo">{{ site.name.substring(0, 1) }}</div>
            <div class="atk-site-name">{{ site.name }}</div>
          </div>
        </div>
        <SiteEditor
          v-if="curtEditSite !== null && ss.includes(curtEditSite)"
          :site="curtEditSite"
          @close="curtEditSite = null"
          @update="onSiteItemUpdate"
          @remove="onSiteItemRemove"
        />
      </template>
    </div>
  </div>
</template>

<style scoped lang="scss">
.atk-site-list {
  & > .atk-header {
    display: flex;
    flex-direction: row;
    padding: 15px 30px;
    align-items: center;

    .atk-title {
      flex: auto;
      padding-right: 10px;
    }

    .atk-actions {
      display: flex;
      flex-direction: row;

      .atk-item {
        display: flex;
        height: 30px;
        width: 30px;
        justify-content: center;
        align-items: center;
        user-select: none;
        cursor: pointer;
        border-radius: 2px;

        &:hover {
          background: var(--at-color-bg-grey);
        }
      }
    }
  }

  .atk-site-rows-wrap {
    position: relative;

    .atk-site-row {
      display: flex;
      flex-direction: row;
      padding: 10px 20px;
      padding-bottom: 0;
    }

    .atk-site-item {
      display: flex;
      flex-basis: 25%;
      flex-direction: column;
      align-items: center;
      padding-bottom: 5px;
      user-select: none;
      cursor: pointer;
      border: 1px solid transparent;

      .atk-site-logo {
        margin: 15px;
        text-align: center;
        font-size: 20px;
        height: 65px;
        width: 65px;
        line-height: 65px;
        background: var(--atk-admin-sage);
        color: #fff;
        border-radius: 4px;
      }

      .atk-site-name {
        text-align: center;
        font-size: 15px;
        color: var(--at-color-sub);
        padding: 0 17px;
        word-break: break-word;
      }

      &.atk-active {
        background-color: var(--at-color-bg-grey);
        border: 1px solid var(--at-color-border);
        margin-top: -1px;
        border-radius: 0 0 4px 4px;

        .atk-site-name {
          color: var(--at-color-deep);
        }
      }

      &:hover {
        .atk-site-name {
          color: var(--at-color-font);
        }
      }
    }
  }

  :deep(.atk-site-edit),
  :deep(.atk-site-add) {
    position: relative;
    min-height: 120px;
    width: 100%;
    border-top: 1px solid var(--at-color-border);
    border-bottom: 1px solid var(--at-color-border);
    margin-bottom: -10px;

    @media (min-width: 1024px) {
      border-left: 1px solid var(--at-color-border);
      border-right: 1px solid var(--at-color-border);
      border-radius: 4px;
      padding-top: 10px;
    }

    .atk-header {
      display: flex;
      flex-direction: row;
      align-items: center;
      padding: 10px 30px 0 35px;
      justify-content: space-between;

      .atk-site-info {
        .atk-site-name {
          cursor: pointer;
          display: inline-block;
          font-size: 23px;
          position: relative;
          line-height: 1.6em;

          &:after {
            content: ' ';
            position: absolute;
            width: 100%;
            height: 6px;
            background: var(--at-color-main);
            opacity: 0.4;
            left: 0;
            bottom: 6px;
          }
        }

        .atk-site-urls {
          display: flex;
          width: 100%;
          margin-top: 6px;
          flex-wrap: wrap;
          min-height: 23px;
          margin-bottom: 15px;

          .atk-url-item {
            background: var(--at-color-bg-grey);
            color: var(--at-color-font);
            border-radius: 2px;
            padding: 0 8px;
            font-size: 13px;
            margin-bottom: 3px;
            margin-right: 3px;
            cursor: pointer;

            &:hover {
            }
          }
        }
      }

      .atk-close-btn {
        width: 50px;
        height: 50px;
        display: flex;
        justify-content: center;
        align-items: center;
        cursor: pointer;

        &:hover i::after {
          background-color: var(--at-color-red);
        }
      }
    }

    .atk-main {
      position: relative;
      display: flex;
      flex-direction: row;
      padding: 0 30px 6px 35px;
      padding-bottom: 10px;

      .atk-site-text-actions {
        @extend .atk-list-text-actions;
        height: 90px;
        padding: 0;
        padding-left: 10px;

        .atk-item {
          margin-bottom: 17px;
          margin-right: 25px;
        }
      }

      .atk-site-btn-actions {
        @extend .atk-list-btn-actions;

        padding-right: 9px;
      }

      .atk-item-text-editor-layer {
        padding: 10px 20px;
      }
    }
  }

  :deep(.atk-site-add) {
    position: relative;

    .atk-header {
      .atk-title {
        font-size: 20px;
      }
    }

    .atk-form {
      padding: 20px 40px;
    }
  }
}

.atk-site-list {
  & > .atk-header {
    margin-bottom: 14px;
    padding: 16px 18px;
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface);
    border-radius: var(--atk-admin-radius);
    box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);

    .atk-title {
      color: var(--atk-admin-ink);
      font-family: Georgia, 'Songti SC', 'STSong', serif;
      font-size: 19px;
    }

    .atk-actions .atk-item {
      width: 34px;
      height: 34px;
      border: 1px solid var(--atk-admin-border);
      border-radius: 50%;

      &:hover {
        background: var(--atk-admin-sage-soft);
      }
    }
  }

  .atk-site-rows-wrap {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 12px;

    .atk-site-row {
      display: contents;
      padding: 0;
    }

    .atk-site-item {
      min-height: 152px;
      justify-content: center;
      padding: 14px;
      border: 1px solid var(--atk-admin-border);
      border-radius: var(--atk-admin-radius);
      background: var(--atk-admin-surface);
      transition: background-color 0.2s ease, border-color 0.2s ease;

      .atk-site-logo {
        margin: 0 0 12px;
        background: var(--atk-admin-sage);
        border-radius: 50%;
      }

      .atk-site-name {
        color: var(--atk-admin-subtle);
      }

      &.atk-active,
      &:hover {
        margin-top: 0;
        background: var(--atk-admin-surface-muted);
        border-color: var(--atk-admin-sage);
        border-radius: var(--atk-admin-radius);

        .atk-site-name {
          color: var(--atk-admin-ink);
        }
      }
    }
  }

  :deep(.atk-site-edit) {
    grid-column: 1 / -1;
    min-width: 0;
    margin: 0;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius);
    background: var(--atk-admin-surface);
    overflow: hidden;

    .atk-header {
      padding: 20px 24px 10px;
      border-bottom: 0;
      background: transparent;
    }

    .atk-site-info .atk-site-name {
      font-size: 21px;
      font-weight: 600;

      &::after {
        display: none;
      }
    }

    .atk-site-info .atk-site-urls {
      margin-top: 8px;
      margin-bottom: 0;

      .atk-url-item {
        margin-right: 6px;
        padding: 4px 9px;
        border: 1px solid var(--atk-admin-border);
        border-radius: 999px;
        background: var(--atk-admin-surface-muted);
        color: var(--atk-admin-subtle);
      }
    }

    .atk-close-btn {
      width: 36px;
      height: 36px;
      border-radius: 50%;

      &:hover {
        background: var(--atk-admin-surface-muted);
      }
    }

    .atk-main {
      align-items: center;
      padding: 12px 24px 22px;
    }

    .atk-site-text-actions {
      display: flex;
      align-items: center;
      gap: 8px;
      height: auto;
      padding: 0;

      .atk-item {
        margin: 0;
        padding: 7px 12px;
        border: 1px solid var(--atk-admin-border);
        border-radius: 999px;
        color: var(--atk-admin-ink);

        &:hover {
          background: var(--atk-admin-surface-muted);
        }
      }
    }

    .atk-site-btn-actions {
      margin-left: auto;
      padding: 0;

      .atk-item {
        width: 36px;
        height: 36px;
        border-radius: 50%;
      }
    }
  }
}

@media (max-width: 560px) {
  .atk-site-list .atk-site-rows-wrap {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
