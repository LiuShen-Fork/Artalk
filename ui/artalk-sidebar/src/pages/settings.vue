<script setup lang="ts">
import YAML from 'yaml'
import { shallowRef } from 'vue'
import { storeToRefs } from 'pinia'
import { useNavStore } from '../stores/nav'
import { artalk } from '../global'
import settings, { type OptionNode } from '../lib/settings'
import LoadingLayer from '../components/LoadingLayer.vue'

const nav = useNavStore()
const router = useRouter()
const { t } = useI18n()
const { curtTab } = storeToRefs(nav)
const isLoading = ref(false)
const tree = shallowRef<OptionNode>()
const selectedRootPath = ref('')
const hiddenRootNodes = new Set(['admin_users'])

const rootGroups = computed(() =>
  (tree.value?.items || []).filter((node) => !hiddenRootNodes.has(node.name) && !!node.items),
)
const activeRoot = computed(
  () => rootGroups.value.find((node) => node.path === selectedRootPath.value) || rootGroups.value[0],
)

watch(rootGroups, (groups) => {
  if (!groups.length) return
  if (!groups.some((node) => node.path === selectedRootPath.value)) {
    selectedRootPath.value = groups[0].path
  }
})

onMounted(() => {
  nav.updateTabs({
    sites: 'site',
    transfer: 'transfer',
  })

  watch(curtTab, (tab) => {
    if (tab === 'sites') router.replace('/sites')
    else if (tab === 'transfer') router.replace('/transfer')
  })

  Promise.all([
    artalk!.ctx.getApi().settings.getSettingsTemplate(''),
    artalk!.ctx.getApi().settings.getSettings(),
  ]).then(([template, custom]) => {
    const yamlObj = YAML.parseDocument(template.data.yaml)
    tree.value = settings.init(yamlObj).getTree()
    // console.log(tree.value)
    settings.get().setCustoms(custom.data.yaml)
    settings.get().setEnvs(custom.data.envs)
    if (rootGroups.value.length) selectedRootPath.value = rootGroups.value[0].path
  })
})

function save() {
  let yamlStr: string
  try {
    yamlStr = settings.get().getCustoms().value?.toString() || ''
  } catch (err) {
    alert('YAML export error: ' + err)
    console.error(err)
    return
  }

  // console.log(yamlStr)
  if (!yamlStr) {
    alert('YAML export error: data is empty')
    return
  }

  if (isLoading.value) return
  isLoading.value = true
  artalk!.ctx
    .getApi()
    .settings.applySettings({
      yaml: yamlStr,
    })
    .then(() => {
      alert(t('settingSaved'))
    })
    .catch((err) => {
      console.error(err)
      alert(t('settingSaveFailed') + ': ' + err)
    })
    .finally(() => {
      isLoading.value = false
    })
}
</script>

<template>
  <div class="settings">
    <div class="act-bar">
      <div class="atk-sidebar-container">
        <div class="status-text"></div>
        <button class="save-btn" @click="save()">
          <i class="atk-icon atk-icon-yes" />
          {{ t('apply') }}
        </button>
      </div>
      <LoadingLayer v-if="isLoading" />
    </div>
    <div v-if="tree" class="settings-layout">
      <aside class="settings-index">
        <div class="settings-index-title">{{ t('config') }}</div>
        <button
          v-for="node in rootGroups"
          :key="node.path"
          type="button"
          class="settings-index-item"
          :class="{ active: activeRoot?.path === node.path }"
          @click="selectedRootPath = node.path"
        >
          <span>{{ node.title }}</span>
          <small v-if="node.subTitle">{{ node.subTitle }}</small>
        </button>
      </aside>

      <main class="settings-content">
        <PreferenceGrp
          v-if="activeRoot"
          :key="activeRoot.path"
          :node="activeRoot"
          default-expanded
        />
        <div class="notice">{{ t('settingNotice') }}</div>
      </main>
    </div>
  </div>
</template>

<style scoped lang="scss">
.settings {
  padding: 18px 28px 80px;

  .notice {
    font-size: 13px;
    background: var(--at-color-bg-light);
    color: var(--at-color-light);
    border-radius: 2px;
    text-align: center;
    padding: 8px 10px;
    margin-top: 10px;
    margin-bottom: 20px;
  }

  .act-bar {
    z-index: 999;
    position: fixed;
    height: 55px;
    width: 100%;
    bottom: 0;
    left: 0;
    background: var(--at-color-bg-transl);
    border-top: 1px solid var(--at-color-border);
    padding: 0 20px;

    .atk-sidebar-container {
      display: flex;
      height: 100%;
      align-items: center;
      justify-content: space-between;
      flex-direction: row;
    }

    .status-text {
      padding: 0 5px;
      flex: 1;
    }

    button {
      font-size: 14px;
      display: inline-flex;
      align-items: center;
      padding: 4px 16px;
      cursor: pointer;
      background: transparent;
      border-radius: 2px;
      background: #36abcf;
      color: #fff;
      border: 0;

      &:active {
        opacity: 0.9;
      }

      i {
        margin-right: 8px;

        &::after {
          background-color: #fff;
        }
      }
    }
  }

  .settings-layout {
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr);
    gap: 18px;
    align-items: start;
  }

  .settings-index,
  .settings-content {
    border: 1px solid var(--at-color-border);
    background: var(--at-color-bg);
    border-radius: 8px;
    box-shadow: 0 10px 30px rgba(15, 23, 42, 0.05);
  }

  .settings-index {
    position: sticky;
    top: 16px;
    padding: 10px;
  }

  .settings-index-title {
    padding: 8px 10px 10px;
    color: var(--at-color-sub);
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0;
  }

  .settings-index-item {
    width: 100%;
    display: block;
    text-align: left;
    border: 0;
    background: transparent;
    color: var(--at-color-font);
    border-radius: 6px;
    padding: 10px;
    margin-bottom: 4px;
    cursor: pointer;

    span,
    small {
      display: block;
    }

    span {
      font-weight: 600;
    }

    small {
      margin-top: 4px;
      color: var(--at-color-sub);
      font-size: 12px;
      line-height: 1.35;
    }

    &.active,
    &:hover {
      background: var(--at-color-bg-grey-transl);
    }

    &.active {
      color: var(--at-color-main);
    }
  }

  .settings-content {
    min-width: 0;
    padding: 6px 24px 20px;

    :deep(.pf-grp.level-1 > .pf-head) {
      margin-top: 18px;
    }
  }

  :deep(input[type='text']),
  :deep(input[type='password']),
  :deep(textarea),
  :deep(select) {
    font-size: 17px;
    width: 100%;
    min-height: 35px;
    padding: 3px 5px;
    border: 0;
    border-bottom: 1px solid var(--at-color-border);
    outline: none;
    background: transparent;
    -webkit-appearance: none;
    border-radius: 0;

    &:focus {
      border-bottom-color: var(--at-color-main);
    }
  }
  :deep(textarea) {
    resize: vertical;
    line-height: 1.5;
    min-height: 140px;
  }
}

@media (max-width: 900px) {
  .settings {
    padding: 12px 14px 80px;

    .settings-layout {
      display: block;
    }

    .settings-index {
      position: static;
      margin-bottom: 12px;
    }

    .settings-index-item {
      margin-bottom: 6px;
    }
  }
}
</style>
