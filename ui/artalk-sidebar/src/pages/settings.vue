<script setup lang="ts">
import YAML from 'yaml'
import { shallowRef } from 'vue'
import { useNavStore } from '../stores/nav'
import { artalk } from '../global'
import settings, { type OptionNode } from '../lib/settings'
import LoadingLayer from '../components/LoadingLayer.vue'

const nav = useNavStore()
const { t } = useI18n()
const isLoading = ref(false)
const tree = shallowRef<OptionNode>()
const selectedRootPath = ref('')
const hiddenRootNodes = new Set(['admin_users'])

const visibleRootNodes = computed(() =>
  (tree.value?.items || []).filter((node) => !hiddenRootNodes.has(node.name)),
)
const basicRootGroup = computed<OptionNode | null>(() => {
  const items = visibleRootNodes.value
    .filter((node) => !node.items)
    .map((node) => ({
      ...node,
      level: 2,
    }))
  if (!items.length) return null

  return {
    name: '__basic__',
    path: '__basic__',
    level: 1,
    type: 'object',
    title: t('config'),
    subTitle: '',
    items,
  }
})
const rootGroups = computed(() =>
  [basicRootGroup.value, ...visibleRootNodes.value.filter((node) => !!node.items)].filter(
    (node): node is OptionNode => !!node,
  ),
)
const activeRoot = computed(
  () =>
    rootGroups.value.find((node) => node.path === selectedRootPath.value) || rootGroups.value[0],
)

watch(rootGroups, (groups) => {
  if (!groups.length) return
  if (!groups.some((node) => node.path === selectedRootPath.value)) {
    selectedRootPath.value = groups[0].path
  }
})

onMounted(() => {
  nav.updateTabs({})

  Promise.all([
    artalk!.ctx.getApi().settings.getSettingsTemplate('zh-CN'),
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
  <div class="settings admin-page">
    <div v-if="tree" class="settings-layout">
      <AdminPageHeader :title="t('settings')" :description="t('settingNotice')" />

      <div class="settings-mobile-index">
        <select v-model="selectedRootPath" :aria-label="t('config')">
          <option v-for="node in rootGroups" :key="node.path" :value="node.path">
            {{ node.title }}
          </option>
        </select>
      </div>

      <aside class="settings-index admin-panel">
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

      <main class="settings-content admin-panel">
        <PreferenceGrp
          v-if="activeRoot"
          :key="activeRoot.path"
          :node="activeRoot"
          default-expanded
        />
      </main>
    </div>

    <div v-if="tree" class="act-bar">
      <div class="act-dock">
        <div class="act-status">
          <span class="act-dot" aria-hidden="true" />
          <span class="act-title">{{ activeRoot?.title }}</span>
        </div>
        <button class="save-btn" type="button" :disabled="isLoading" @click="save()">
          <i class="atk-icon atk-icon-yes" />
          {{ isLoading ? '…' : t('apply') }}
        </button>
      </div>
      <LoadingLayer v-if="isLoading" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.settings {
  padding: 18px 28px 0;

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
    position: sticky;
    bottom: 0;
    z-index: 30;
    padding: 34px 0 18px;
    background: linear-gradient(
      to top,
      var(--atk-admin-canvas) 0%,
      var(--atk-admin-canvas) 46%,
      transparent 100%
    );
    pointer-events: none;
  }

  .act-dock {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    max-width: 760px;
    margin: 0 auto;
    padding: 9px 10px 9px 18px;
    border: 1px solid var(--atk-admin-border);
    border-radius: 16px;
    background: var(--atk-admin-bg-transl, rgba(255, 255, 255, 0.86));
    box-shadow: var(--atk-admin-shadow-sm);
    backdrop-filter: blur(18px) saturate(1.15);
    pointer-events: auto;
  }

  .act-status {
    display: flex;
    min-width: 0;
    flex: 1;
    align-items: center;
    gap: 10px;
  }

  .act-dot {
    flex: none;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--atk-admin-sage);
    box-shadow: 0 0 0 4px var(--atk-admin-sage-soft);
  }

  .act-title {
    flex: none;
    color: var(--atk-admin-ink);
    font-size: 13px;
    font-weight: 650;
    white-space: nowrap;
  }

  .save-btn {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    gap: 7px;
    min-width: 104px;
    min-height: 40px;
    padding: 0 16px;
    border: 1px solid var(--atk-admin-ink);
    border-radius: 12px;
    color: var(--atk-admin-surface);
    background: var(--atk-admin-ink);
    font-size: 13.5px;
    font-weight: 650;
    cursor: pointer;
    transition:
      background-color 0.18s ease,
      border-color 0.18s ease,
      opacity 0.18s ease;

    .atk-icon {
      display: block;
    }

    &:hover:not(:disabled) {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-sage);
    }

    &:active:not(:disabled) {
      opacity: 0.88;
    }

    &:disabled {
      cursor: wait;
      opacity: 0.65;
    }
  }

  .settings-layout {
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr);
    gap: 20px;
    align-items: start;
  }

  :deep(.admin-page-head) {
    grid-column: 1 / -1;
  }

  .settings-index,
  .settings-content {
    border: 1px solid var(--at-color-border);
    background: var(--at-color-bg);
    border-radius: 8px;
    box-shadow: 0 10px 30px rgba(15, 23, 42, 0.05);
  }

  .settings-mobile-index {
    display: none;
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
    border-radius: 8px;
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
    padding: 10px 26px 26px;

    :deep(.pf-grp.level-1 > .pf-head) {
      margin-top: 18px;
    }

    :deep(.pf-grp),
    :deep(.pf-body) {
      background: transparent;
    }

    :deep(.pf-body) {
      display: block !important;
    }

    :deep(.pf-item) {
      padding: 12px 0;
      margin-bottom: 0;
      border-bottom: 1px solid var(--atk-admin-border);
    }

    :deep(.pf-item:last-child) {
      border-bottom: 0;
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
    padding: 12px 14px 0;

    .settings-layout {
      grid-template-columns: 190px minmax(0, 1fr);
      gap: 14px;
    }

    .settings-index {
      top: 12px;
    }

    .settings-index-item {
      margin-bottom: 6px;
    }
  }
}

@media (max-width: 720px) {
  .settings {
    .settings-layout {
      display: block;
    }

    .settings-index {
      display: none;
    }

    .settings-mobile-index {
      display: block;
      margin-bottom: 12px;

      select {
        width: 100%;
        min-height: 42px;
        padding: 8px 12px;
        border: 1px solid var(--atk-admin-border);
        border-radius: var(--atk-admin-radius-sm);
        color: var(--atk-admin-ink);
        background: var(--atk-admin-surface);
      }
    }

    .settings-content {
      padding: 8px 16px 20px;
    }
  }
}

.settings {
  .settings-index,
  .settings-content {
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface);
    box-shadow: 0 2px 8px rgba(72, 60, 46, 0.04);
  }

  .settings-index,
  .settings-content {
    border-radius: var(--atk-admin-radius);
  }

  .settings-index-item {
    border-radius: 8px;
    color: var(--atk-admin-ink);

    small {
      color: var(--atk-admin-subtle);
    }

    &.active,
    &:hover {
      background: var(--atk-admin-surface-muted);
    }

    &.active {
      color: var(--atk-admin-sage);
    }
  }

  .settings-index-title {
    color: var(--atk-admin-subtle);
    letter-spacing: 0;
  }

  :deep(input[type='text']),
  :deep(input[type='password']),
  :deep(textarea),
  :deep(select) {
    min-height: 40px;
    padding: 8px 12px;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-xs);
    background: var(--atk-admin-surface-muted);
    color: var(--atk-admin-ink);
    font-size: 14px;
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;

    &:focus {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-surface);
      box-shadow: 0 0 0 3px var(--atk-admin-ring);
      outline: none;
    }
  }

  :deep(.pf-head .title) {
    color: var(--atk-admin-ink);
    font-size: 17px;
    font-weight: 650;
    letter-spacing: -0.01em;
  }

  :deep(.pf-grp.level-1 > .pf-head .title::before) {
    height: 18px;
    width: 4px;
    left: -10px;
    border-radius: 2px;
  }

  :deep(.pf-grp.level-1.expand > .pf-head .title::before) {
    height: 22px;
  }

  :deep(.pf-item .sub-title) {
    color: var(--atk-admin-faint);
    font-size: 12.5px;
  }

  :deep(select) {
    appearance: none;
    -webkit-appearance: none;
    padding-right: 34px;
    background-image: url("data:image/svg+xml,%3Csvg width='12' height='8' viewBox='0 0 12 8' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1.5 6 6.5l5-5' stroke='%238a9a93' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    background-size: 12px;
    cursor: pointer;
  }

  .settings-mobile-index select {
    appearance: none;
    -webkit-appearance: none;
    padding-right: 34px;
    background-image: url("data:image/svg+xml,%3Csvg width='12' height='8' viewBox='0 0 12 8' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M1 1.5 6 6.5l5-5' stroke='%238a9a93' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 14px center;
    background-size: 12px;
    cursor: pointer;
  }

  .settings-content {
    background: var(--atk-admin-surface);
  }
}

.settings :deep(.pf-item input[type='checkbox']) {
  appearance: none;
  -webkit-appearance: none;
  position: relative;
  width: 42px;
  height: 24px;
  min-height: 24px;
  padding: 0;
  border: 1px solid var(--atk-admin-border-strong);
  border-radius: 999px;
  background: var(--atk-admin-surface-sunken);
  cursor: pointer;
  transition:
    background-color 0.2s ease,
    border-color 0.2s ease;

  &::after {
    content: '';
    position: absolute;
    top: 3px;
    left: 3px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 3px rgba(10, 20, 16, 0.28);
    transition: transform 0.2s var(--atk-admin-ease);
  }

  &:checked {
    border-color: var(--atk-admin-sage);
    background: var(--atk-admin-sage);

    &::after {
      transform: translateX(18px);
    }
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}
</style>
