<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'
import { isOpenFromSidebar } from '@/global'

const el = ref<HTMLElement | null>(null)

const nav = useNavStore()
const { siteSwitcherShow: curtShow, sites } = storeToRefs(nav)
const { site: curtSite } = storeToRefs(useUserStore())

const router = useRouter()
const { t } = useI18n()

onMounted(() => {
  nav.refreshSites()
})

interface IDisplaySite {
  label: string
  name: string
  logoText: string
}

function switchSite(siteName: string) {
  if (siteName === '__SITE_MANAGEMENT__') {
    router.replace('/sites')
  } else {
    curtSite.value = siteName
  }

  curtShow.value = false
}

const displaySites = computed(() => {
  const displays: IDisplaySite[] = []
  displays.push({ label: t('allSites'), name: '', logoText: '_' })
  sites.value.forEach((site) => {
    displays.push({
      label: site.name,
      name: site.name,
      logoText: site.name.substring(0, 1),
    })
  })
  displays.push({
    label: t('siteManage'),
    name: '__SITE_MANAGEMENT__',
    logoText: '+',
  })
  return displays
})

function outsideChecker(evt: MouseEvent) {
  const isClickInside = el.value?.contains(evt.target as any)
  if (!isClickInside) {
    curtShow.value = false
  }
}

watch(curtShow, (value) => {
  if (value) {
    setTimeout(() => {
      document.addEventListener('click', outsideChecker)
    }, 80)
  } else {
    document.removeEventListener('click', outsideChecker)
  }
})

function logout() {
  if (!window.confirm(t('logoutConfirm'))) return

  useUserStore().logout()
  nextTick(() => {
    router.replace('/login')
  })
}
</script>

<template>
  <Transition>
    <div v-show="curtShow" ref="el" class="atk-site-list-floater">
      <div class="atk-sites">
        <div
          v-for="(site, i) in displaySites"
          :key="i"
          class="atk-site-item"
          :class="{ 'atk-active': curtSite === site.name }"
          @click="switchSite(site.name)"
        >
          <div class="atk-site-logo">{{ site.logoText }}</div>
          <div class="atk-site-name">{{ site.label }}</div>
        </div>

        <!-- Logout Button -->
        <div v-if="!isOpenFromSidebar()" class="atk-site-item" @click="logout()">
          <svg
            class="atk-site-logo"
            stroke="currentColor"
            fill="currentColor"
            stroke-width="0"
            viewBox="-6 -6 36 36"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M5 22C4.44772 22 4 21.5523 4 21V3C4 2.44772 4.44772 2 5 2H19C19.5523 2 20 2.44772 20 3V6H18V4H6V20H18V18H20V21C20 21.5523 19.5523 22 19 22H5ZM18 16V13H11V11H18V8L23 12L18 16Z"
            ></path>
          </svg>
          <div class="atk-site-name">{{ t('logout') }}</div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped lang="scss">
.atk-site-list-floater {
  @extend .atk-slim-scrollbar;

  position: absolute;
  top: calc(var(--atk-admin-header-h) + 8px);
  left: 12px;
  z-index: 9999;
  width: min(300px, calc(100% - 24px));
  max-height: 46%;
  margin: 0;
  padding: 6px;
  overflow-y: auto;
  border: 1px solid var(--atk-admin-border);
  border-radius: var(--atk-admin-radius);
  background: var(--atk-admin-surface);
  box-shadow: var(--atk-admin-shadow);
  transform-origin: top left;

  .atk-sites {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .atk-site-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 7px 9px;
    border-radius: var(--atk-admin-radius-xs);
    cursor: pointer;
    transition: background-color 0.16s ease;

    .atk-site-logo {
      flex: none;
      width: 30px;
      height: 30px;
      margin: 0;
      border-radius: 9px;
      color: #fff;
      background: linear-gradient(145deg, var(--atk-admin-sage), var(--atk-admin-sage-strong));
      font-size: 12px;
      font-weight: 700;
      line-height: 30px;
      text-align: center;
    }

    .atk-site-name {
      overflow: hidden;
      margin: 0;
      color: var(--atk-admin-ink);
      font-size: 13.5px;
      font-weight: 550;
      white-space: nowrap;
      text-overflow: ellipsis;
    }

    &.atk-active,
    &:hover {
      background: var(--atk-admin-surface-muted);
    }

    &.atk-active {
      color: var(--atk-admin-sage-strong);
      background: var(--atk-admin-sage-soft);

      .atk-site-name {
        color: var(--atk-admin-sage-strong);
        font-weight: 650;
      }
    }
  }
}

.v-enter-from,
.v-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}
</style>
