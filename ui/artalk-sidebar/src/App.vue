<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useNavStore } from './stores/nav'

const nav = useNavStore()
const route = useRoute()
const { scrollableArea, darkMode } = storeToRefs(nav)

const isLogin = computed(() => String(route.name || '').includes('login'))
const hasTabs = computed(() => Object.keys(nav.tabs || {}).length > 0)
const shellStyle = computed(() => ({
  '--atk-admin-tabs-h': hasTabs.value ? '54px' : '0px',
}))

const query = window.matchMedia('(prefers-color-scheme: dark)')

onMounted(() => query.addEventListener('change', onDarkModeChange))
onUnmounted(() => query.removeEventListener('change', onDarkModeChange))
const onDarkModeChange = (e: MediaQueryListEvent) => {
  // auto switch when localStorage is empty
  if (localStorage.getItem('ATK_SIDEBAR_DARK_MODE') === null) darkMode.value = e.matches
}
</script>

<template>
  <div
    class="app-wrap artalk atk-sidebar"
    :class="{ 'atk-dark-mode': darkMode, 'login-mode': isLogin }"
    :style="shellStyle"
  >
    <AppHeader v-if="!isLogin" />
    <AppNavigation v-if="!isLogin" />

    <div class="main">
      <div ref="scrollableArea" class="atk-sidebar-inner">
        <div class="atk-sidebar-container">
          <router-view />
        </div>
      </div>
      <LoadingLayer v-if="nav.isPageLoading" />
    </div>
  </div>
</template>

<style scoped lang="scss">
$headerHeight: 62px;
$railWidth: 236px;
$maxWidth: 1040px;

.app-wrap {
  min-height: 100vh;
  min-height: 100dvh;
}

.main {
  position: relative;

  .atk-sidebar-inner {
    height: calc(100vh - var(--atk-admin-header-h) - var(--atk-admin-nav-h));
    height: calc(100dvh - var(--atk-admin-header-h) - var(--atk-admin-nav-h));
    padding-bottom: 34px;
    overflow-y: auto;
    overscroll-behavior: contain;
    scrollbar-gutter: stable;

    @media (min-width: 860px) {
      height: calc(100dvh - $headerHeight - var(--atk-admin-tabs-h, 0px));
      padding-bottom: 0;
      margin-left: $railWidth;
    }
  }

  .login-mode & .atk-sidebar-inner {
    height: 100dvh;
    padding-bottom: 0;
    margin-left: 0;
  }

  :deep(.atk-pagination-wrap) {
    position: fixed;
    bottom: 0;
    left: 0;
    z-index: 200;
    width: 100%;
    border-top: 1px solid var(--atk-admin-border);
    background: var(--atk-admin-bg-transl);
    backdrop-filter: blur(16px);
  }
}

:deep(.atk-sidebar-container) {
  position: relative;
}

@media (min-width: 860px) {
  :deep(.atk-sidebar-container) {
    width: 100%;
    max-width: $maxWidth;
    margin: 0 auto;
    padding: 0 28px;
  }

  :deep(.atk-pagination-wrap) {
    left: $railWidth;
    width: calc(100% - $railWidth);
  }
}
</style>
