<script lang="ts" setup>
/**
 * The part of the navigation bar that is displayed on mobile devices and the sidebar
 */
import type { PageItem } from './AppNavigationMenu'
import { useNavigationMenu } from './AppNavigationMenu'

const { t } = useI18n()

const indicatorState = ref<'pages' | 'tabs'>('tabs')
const tabListEl = ref<HTMLElement | null>(null)
const nav = useNavigationMenu({
  onGoPage: () => {
    indicatorState.value = 'tabs'
  },
})

const currentPage = computed<PageItem | undefined>(() => nav.pages[nav.curtPage])

const iconStyle = (icon: string) => {
  const url = `url('${icon.replace(/'/g, "\\'")}')`
  return {
    maskImage: url,
    WebkitMaskImage: url,
  }
}

/**
 * For tab list for horizontal scroll
 */
const wheelHandler = (evt: WheelEvent) => {
  evt.preventDefault()
  if (tabListEl.value) tabListEl.value.scrollLeft += evt.deltaY
}
onMounted(() => tabListEl.value?.addEventListener('wheel', wheelHandler))
onUnmounted(() => tabListEl.value?.removeEventListener('wheel', wheelHandler))

/**
 * Toggle indicator
 */
function toggleIndicator() {
  indicatorState.value = indicatorState.value !== 'tabs' ? 'tabs' : 'pages'
}
</script>

<template>
  <div class="top-navigation" :class="{ 'pages-open': indicatorState === 'pages' }">
    <div class="nav-row">
      <button
        type="button"
        class="page-trigger"
        :aria-expanded="indicatorState === 'pages'"
        @click="toggleIndicator()"
      >
        <span
          v-if="currentPage"
          class="page-icon"
          :style="iconStyle(currentPage.icon)"
          aria-hidden="true"
        />
        <span v-if="currentPage" class="page-label">{{ t(currentPage.label) }}</span>
        <span class="chevron" :class="{ open: indicatorState === 'pages' }" />
      </button>

      <div v-if="Object.keys(nav.tabs).length" ref="tabListEl" class="tab-list" role="tablist">
        <button
          v-for="(tabLabel, tabName) in nav.tabs"
          :key="tabName"
          type="button"
          class="tab"
          role="tab"
          :aria-selected="nav.curtTab === tabName"
          :class="{ active: nav.curtTab === tabName }"
          @click="nav.goTab(tabName as string)"
        >
          {{ t(tabLabel) }}
        </button>
      </div>

      <button
        v-if="nav.isSearchEnabled"
        type="button"
        class="search-btn"
        :aria-label="t('searchHint')"
        @click="nav.showSearch()"
      >
        <span class="search-icon" />
      </button>
    </div>

    <Transition name="sheet">
      <nav v-if="indicatorState === 'pages'" class="page-sheet" aria-label="Pages">
        <button
          v-for="(page, pageName) in nav.pages"
          :key="pageName"
          type="button"
          class="page-item"
          :class="{ active: pageName === nav.curtPage }"
          @click="nav.goPage(pageName as string)"
        >
          <span class="page-icon" :style="iconStyle(page.icon)" aria-hidden="true" />
          <span>{{ t(page.label) }}</span>
        </button>
      </nav>
    </Transition>

    <AppNavigationSearch v-if="nav.isMobile" :state="nav.searchState" />
  </div>
</template>

<style lang="scss" scoped>
.top-navigation {
  position: relative;
  z-index: 25;
  height: var(--atk-admin-nav-h);
  border-bottom: 1px solid var(--atk-admin-border);
  background: var(--atk-admin-bg-transl);
  backdrop-filter: blur(18px);

  @media (min-width: 860px) {
    display: none;
  }

  &.pages-open {
    z-index: 40;
  }
}

.nav-row {
  display: flex;
  align-items: center;
  height: 100%;
  padding: 0 8px 0 10px;
  gap: 6px;
}

.page-trigger {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 7px;
  max-width: 42%;
  height: 36px;
  padding: 0 9px;
  border: 1px solid transparent;
  border-radius: var(--atk-admin-radius-xs);
  background: transparent;
  color: var(--atk-admin-ink);
  cursor: pointer;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease;

  &:hover,
  &[aria-expanded='true'] {
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface-muted);
  }
}

.page-icon {
  flex: none;
  width: 17px;
  height: 17px;
  background-color: var(--atk-admin-sage);
  mask-repeat: no-repeat;
  mask-position: center;
  mask-size: contain;
  -webkit-mask-repeat: no-repeat;
  -webkit-mask-position: center;
  -webkit-mask-size: contain;
}

.page-label {
  overflow: hidden;
  font-size: 13px;
  font-weight: 650;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.chevron {
  flex: none;
  width: 6px;
  height: 6px;
  margin-left: 1px;
  border-right: 1.5px solid var(--atk-admin-faint);
  border-bottom: 1.5px solid var(--atk-admin-faint);
  transform: rotate(45deg) translateY(-1px);
  transition: transform 0.18s ease;

  &.open {
    transform: rotate(225deg) translateY(-1px);
  }
}

.tab-list {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 2px;
  height: 100%;
  overflow-x: auto;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
}

.tab {
  flex: none;
  height: 32px;
  padding: 0 11px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--atk-admin-subtle);
  font-size: 13px;
  font-weight: 550;
  white-space: nowrap;
  cursor: pointer;
  transition:
    color 0.18s ease,
    background-color 0.18s ease;

  &:hover {
    color: var(--atk-admin-ink);
    background: var(--atk-admin-surface-muted);
  }

  &.active {
    color: var(--atk-admin-sage-strong);
    background: var(--atk-admin-sage-soft);
    font-weight: 650;
  }
}

.search-btn {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  margin-left: auto;
  border: 1px solid transparent;
  border-radius: var(--atk-admin-radius-xs);
  background: transparent;
  cursor: pointer;

  &:hover {
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface-muted);
  }

  .search-icon {
    width: 17px;
    height: 17px;
    background-color: var(--atk-admin-subtle);
    mask: url("data:image/svg+xml,%3Csvg width='15' height='15' viewBox='0 0 15 15' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill-rule='evenodd' clip-rule='evenodd' d='M9.89266 3.70572C8.18412 1.99718 5.41402 1.99718 3.70548 3.70572C1.99694 5.41427 1.99694 8.18436 3.70548 9.89291C5.41402 11.6015 8.18412 11.6015 9.89266 9.89291C11.6012 8.18436 11.6012 5.41427 9.89266 3.70572ZM2.8216 2.82184C5.0183 0.625142 8.57985 0.625142 10.7765 2.82184C12.8239 4.86916 12.9631 8.10202 11.1942 10.3106L13.4282 12.5446L12.5443 13.4284L10.3103 11.1945C8.10177 12.9633 4.86892 12.8241 2.8216 10.7768C0.624897 8.58009 0.624897 5.01854 2.8216 2.82184Z' fill='black'/%3E%3C/svg%3E")
      center / contain no-repeat;
    -webkit-mask: url("data:image/svg+xml,%3Csvg width='15' height='15' viewBox='0 0 15 15' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath fill-rule='evenodd' clip-rule='evenodd' d='M9.89266 3.70572C8.18412 1.99718 5.41402 1.99718 3.70548 3.70572C1.99694 5.41427 1.99694 8.18436 3.70548 9.89291C5.41402 11.6015 8.18412 11.6015 9.89266 9.89291C11.6012 8.18436 11.6012 5.41427 9.89266 3.70572ZM2.8216 2.82184C5.0183 0.625142 8.57985 0.625142 10.7765 2.82184C12.8239 4.86916 12.9631 8.10202 11.1942 10.3106L13.4282 12.5446L12.5443 13.4284L10.3103 11.1945C8.10177 12.9633 4.86892 12.8241 2.8216 10.7768C0.624897 8.58009 0.624897 5.01854 2.8216 2.82184Z' fill='black'/%3E%3C/svg%3E")
      center / contain no-repeat;
  }
}

.page-sheet {
  position: absolute;
  top: calc(100% + 6px);
  right: 8px;
  left: 8px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px;
  padding: 8px;
  border: 1px solid var(--atk-admin-border);
  border-radius: var(--atk-admin-radius);
  background: var(--atk-admin-surface);
  box-shadow: var(--atk-admin-shadow);
  transform-origin: top center;
}

.page-item {
  display: flex;
  align-items: center;
  gap: 9px;
  height: 42px;
  padding: 0 11px;
  border: 0;
  border-radius: var(--atk-admin-radius-xs);
  background: transparent;
  color: var(--atk-admin-subtle);
  font-size: 13px;
  font-weight: 550;
  text-align: left;
  cursor: pointer;
  transition:
    color 0.18s ease,
    background-color 0.18s ease;

  &:hover {
    color: var(--atk-admin-ink);
    background: var(--atk-admin-surface-muted);
  }

  &.active {
    color: var(--atk-admin-sage-strong);
    background: var(--atk-admin-sage-soft);
    font-weight: 650;
  }

  .page-icon {
    width: 18px;
    height: 18px;
  }
}

.sheet-enter-active,
.sheet-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s var(--atk-admin-ease);
}

.sheet-enter-from,
.sheet-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}
</style>
