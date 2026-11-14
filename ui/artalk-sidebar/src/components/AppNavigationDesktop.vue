<script lang="ts" setup>
/**
 * The part of the navigation bar that is displayed on desktop
 */
import { useNavigationMenu } from './AppNavigationMenu'

const nav = useNavigationMenu()
const { t } = useI18n()

const iconStyle = (icon: string) => {
  const url = `url('${icon.replace(/'/g, "\\'")}')`
  return {
    maskImage: url,
    WebkitMaskImage: url,
  }
}
</script>

<template>
  <aside class="sidebar-navigation">
    <nav class="rail-nav">
      <button
        v-for="(page, pageName) in nav.pages"
        :key="pageName"
        type="button"
        class="rail-item"
        :class="{ active: pageName === nav.curtPage }"
        :aria-current="pageName === nav.curtPage ? 'page' : undefined"
        @click="nav.goPage(pageName as string)"
      >
        <span class="rail-icon" :style="iconStyle(page.icon)" aria-hidden="true" />
        <span class="rail-label">{{ t(page.label) }}</span>
      </button>
    </nav>
  </aside>

  <div v-if="Object.keys(nav.tabs).length || nav.isSearchEnabled" class="top-tabbar-wrap">
    <div class="top-tabbar">
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

    <AppNavigationSearch v-if="!nav.isMobile" :state="nav.searchState" />
  </div>
</template>

<style lang="scss" scoped>
$railWidth: 236px;

.sidebar-navigation {
  position: fixed;
  top: var(--atk-admin-header-h);
  left: 0;
  z-index: 20;
  width: $railWidth;
  height: calc(100vh - var(--atk-admin-header-h));
  height: calc(100dvh - var(--atk-admin-header-h));
  padding: 18px 14px;
  border-right: 1px solid var(--atk-admin-border);
  background: var(--atk-admin-bg-transl);
  backdrop-filter: blur(18px);

  @media (max-width: 859px) {
    display: none;
  }
}

.rail-nav {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.rail-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 11px;
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

    .rail-icon {
      background-color: var(--atk-admin-sage);
    }
  }
}

.rail-icon {
  flex: none;
  width: 18px;
  height: 18px;
  background-color: currentColor;
  mask-repeat: no-repeat;
  mask-position: center;
  mask-size: contain;
  -webkit-mask-repeat: no-repeat;
  -webkit-mask-position: center;
  -webkit-mask-size: contain;
  opacity: 0.85;
}

.rail-label {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.top-tabbar-wrap {
  position: relative;
  margin-left: $railWidth;
  height: 54px;
  border-bottom: 1px solid var(--atk-admin-border);
  background: var(--atk-admin-bg-transl);
  backdrop-filter: blur(18px);

  @media (max-width: 859px) {
    display: none;
  }
}

.top-tabbar {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 100%;
  max-width: 1040px;
  margin: 0 auto;
  padding: 0 28px;
}

.tab {
  height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--atk-admin-subtle);
  font-size: 13px;
  font-weight: 550;
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
</style>
