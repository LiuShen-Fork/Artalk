<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useUserStore } from '../stores/user'
import { useNavStore } from '../stores/nav'
import { isOpenFromSidebar } from '../global'

const nav = useNavStore()
const router = useRouter()
const user = useUserStore()
const { t } = useI18n()
const { site: curtSite, is_admin: isAdmin, avatar, name: userName } = storeToRefs(user)
const { darkMode } = storeToRefs(nav)

const siteInitial = computed(() => (curtSite.value || '_').substring(0, 1))

const avatarClickHandler = () => {
  if (!isOpenFromSidebar()) logout()
}

const logout = () => {
  if (!window.confirm(t('logoutConfirm'))) return

  useUserStore().logout()
  nextTick(() => {
    router.replace('/login')
  })
}
</script>

<template>
  <header class="header">
    <button
      v-if="isAdmin"
      type="button"
      class="brand"
      :aria-label="t('siteManage')"
      @click="nav.showSiteSwitcher()"
    >
      <span class="brand-mark">{{ siteInitial }}</span>
      <span class="brand-copy">
        <span class="brand-name">Artalk</span>
        <span class="brand-site">{{ curtSite || t('ctrlCenter') }}</span>
      </span>
      <span class="chevron" />
    </button>

    <button v-else type="button" class="brand guest" @click="avatarClickHandler">
      <span class="brand-mark avatar-mark"><img :src="avatar" alt="" /></span>
      <span class="brand-copy">
        <span class="brand-name">{{ t('msgCenter') }}</span>
        <span class="brand-site">{{ userName || t('unknownUser') }}</span>
      </span>
    </button>

    <div class="header-actions">
      <button
        type="button"
        class="icon-btn theme-toggle"
        :aria-label="darkMode ? 'Switch to light mode' : 'Switch to dark mode'"
        :aria-pressed="darkMode"
        @click="nav.toggleDarkMode()"
      >
        <span class="theme-icon" :data-darkmode="darkMode ? 'on' : 'off'" />
      </button>
    </div>

    <!-- The host page renders its close button over this reserved area. -->
    <div class="close-btn" />
  </header>
  <SiteSwitcher v-if="isAdmin" />
</template>

<style scoped lang="scss">
.header {
  position: relative;
  z-index: 30;
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 10px;
  height: var(--atk-admin-header-h);
  padding-left: 14px;
  border-bottom: 1px solid var(--atk-admin-border);
  background: var(--atk-admin-bg-transl);
  backdrop-filter: blur(18px);
}

.brand {
  display: flex;
  flex: 0 1 auto;
  align-items: center;
  gap: 10px;
  min-width: 0;
  height: 44px;
  padding: 0 8px 0 0;
  border: 0;
  border-radius: var(--atk-admin-radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.18s ease;

  &:hover {
    background: var(--atk-admin-surface-muted);
  }

  &.guest {
    cursor: pointer;
  }
}

.brand-mark {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 11px;
  color: #fff;
  background: linear-gradient(145deg, var(--atk-admin-sage), var(--atk-admin-sage-strong));
  box-shadow: 0 8px 18px -10px rgba(15, 118, 110, 0.75);
  font-size: 15px;
  font-weight: 700;
  text-transform: uppercase;

  &.avatar-mark {
    overflow: hidden;
    background: var(--atk-admin-surface-muted);
    box-shadow: none;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }
}

.brand-copy {
  display: flex;
  flex: 0 1 auto;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.brand-name {
  color: var(--atk-admin-faint);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.12em;
  line-height: 1.3;
}

.brand-site {
  overflow: hidden;
  color: var(--atk-admin-ink);
  font-size: 14px;
  font-weight: 650;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chevron {
  flex: none;
  width: 7px;
  height: 7px;
  margin: 0 4px 0 2px;
  border-right: 1.5px solid var(--atk-admin-faint);
  border-bottom: 1.5px solid var(--atk-admin-faint);
  transform: rotate(45deg) translateY(-2px);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease;

  &:hover {
    border-color: var(--atk-admin-border);
    background: var(--atk-admin-surface-muted);
  }
}

.theme-icon {
  width: 20px;
  height: 20px;
  background-repeat: no-repeat;
  background-position: center;
  background-size: contain;

  &[data-darkmode='on'] {
    background-image: url('@/assets/icon-darkmode-on.svg');
  }

  &[data-darkmode='off'] {
    background-image: url('@/assets/icon-darkmode-off.svg');
  }
}

.close-btn {
  flex: none;
  width: 60px;
  height: var(--atk-admin-header-h);
}
</style>
