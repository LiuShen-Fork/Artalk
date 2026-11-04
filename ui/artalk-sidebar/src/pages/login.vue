<route lang="json">
{
  "meta": {
    "title": "Login - Artalk Sidebar"
  }
}
</route>

<script setup lang="ts">
import type { ArtalkType } from 'artalk'
import { getArtalk } from '../global'
import { useUserStore } from '../stores/user'

const router = useRouter()
const { t } = useI18n()

let userForm = ref({
  email: '',
  password: '',
})
let version = ref('')
let buildHash = ref('')
let loginErr = ref('')
let userSelector = ref<string[] | null>(null)

onMounted(() => {
  getArtalk()!
    .ctx.getApi()
    .version.getVersion()
    .then((res) => {
      version.value = res.data.version
      buildHash.value = res.data.commit_hash
    })
})

function onFocus() {
  loginErr.value = ''
}

function login(username?: string) {
  loginErr.value = ''

  const artalk = getArtalk()
  if (!artalk) throw new Error('Artalk instance not initialized')

  artalk.ctx
    .getApi()
    .user.login({
      name: username || '',
      email: userForm.value.email,
      password: userForm.value.password,
    })
    .then((res) => {
      artalk.ctx.getUser().update({
        ...res.data.user,
        token: res.data.token,
      })
      useUserStore().sync()
      router.replace('/comments')
    })
    .catch((e: ArtalkType.FetchError) => {
      if (e.data?.need_name_select) {
        userSelector.value = e.data?.need_name_select
      } else {
        loginErr.value = e.message || t('loginFailure')
      }
    })
}

function selectUser(username: string) {
  userSelector.value = null
  login(username)
}

const versionInfo = computed(() => {
  return `v${version.value + (buildHash.value ? ' / ' + buildHash.value : '')}`
})
</script>

<template>
  <div class="login-dialog">
    <div class="login-card">
      <a class="logo-link" href="https://artalk.js.org/" target="_blank" rel="noreferrer">
        <img class="logo" src="../assets/favicon.png" alt="Artalk" draggable="false" />
      </a>

      <div class="login-heading">
        <h1>Artalk 控制台</h1>
        <p>使用管理员账户继续</p>
      </div>

      <form class="login-form" @submit.prevent="login()">
        <label class="field">
          <span>{{ t('email') }}</span>
          <input
            v-model="userForm.email"
            type="text"
            autocomplete="username"
            :placeholder="t('email')"
            @focus="onFocus"
          />
        </label>
        <label class="field">
          <span>{{ t('password') }}</span>
          <input
            v-model="userForm.password"
            type="password"
            autocomplete="current-password"
            :placeholder="t('password')"
            @focus="onFocus"
          />
        </label>

        <div v-if="!!loginErr" class="err-msg atk-fade-in">{{ loginErr }}</div>

        <button type="submit">{{ t('login') }}</button>
      </form>

      <div class="copyright">
        Powered by
        <a href="https://artalk.js.org" target="_blank" rel="noreferrer">Artalk</a>
        <span>{{ versionInfo }}</span>
      </div>
    </div>

    <div v-if="userSelector" class="layer">
      <div class="user-selector atk-fade-in">
        <div class="text">{{ t('loginSelectHint') }}</div>
        <div class="user-list">
          <div v-for="(u, i) in userSelector" :key="i" class="item" @click="selectUser(u)">
            {{ u }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.login-dialog {
  position: fixed;
  top: 0;
  left: 0;
  z-index: 11;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 24px;
  background:
    radial-gradient(90% 60% at 100% 0%, rgba(15, 118, 110, 0.16), transparent 62%),
    radial-gradient(80% 60% at 0% 100%, rgba(191, 91, 59, 0.1), transparent 60%),
    var(--atk-admin-canvas);
}

.login-card {
  width: 100%;
  max-width: 340px;
  padding: 28px;
  border: 1px solid var(--atk-admin-border);
  border-radius: 22px;
  background: var(--atk-admin-surface);
  box-shadow: var(--atk-admin-shadow);
}

.logo-link {
  display: inline-flex;
  margin-bottom: 18px;
}

.logo {
  display: block;
  width: 46px;
  height: 46px;
  border-radius: 14px;
  user-select: none;
}

.login-heading {
  margin-bottom: 22px;

  h1 {
    margin: 0;
    color: var(--atk-admin-ink);
    font-size: 21px;
    font-weight: 700;
    letter-spacing: -0.02em;
  }

  p {
    margin: 5px 0 0;
    color: var(--atk-admin-subtle);
    font-size: 13px;
  }
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 13px;

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;

    span {
      color: var(--atk-admin-subtle);
      font-size: 12px;
      font-weight: 600;
    }
  }

  input {
    display: block;
    width: 100%;
    height: 42px;
    padding: 0 13px;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-xs);
    color: var(--atk-admin-ink);
    background: var(--atk-admin-surface-muted);
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;

    &::placeholder {
      color: var(--atk-admin-faint);
    }

    &:focus {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-surface);
      box-shadow: 0 0 0 3px var(--atk-admin-ring);
      outline: none;
    }
  }

  button {
    height: 42px;
    margin-top: 5px;
    border: 1px solid var(--atk-admin-ink);
    border-radius: var(--atk-admin-radius-xs);
    color: var(--atk-admin-surface);
    background: var(--atk-admin-ink);
    font-size: 14px;
    font-weight: 650;
    cursor: pointer;
    transition:
      background-color 0.18s ease,
      border-color 0.18s ease,
      transform 0.18s ease;

    &:hover {
      border-color: var(--atk-admin-sage);
      background: var(--atk-admin-sage);
    }

    &:active {
      transform: translateY(1px);
    }
  }

  .err-msg {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 9px 11px;
    border: 1px solid var(--atk-admin-danger-soft);
    border-radius: var(--atk-admin-radius-xs);
    color: var(--atk-admin-danger);
    background: var(--atk-admin-danger-soft);
    font-size: 12.5px;
  }
}

.copyright {
  margin-top: 22px;
  color: var(--atk-admin-faint);
  font-size: 11.5px;
  text-align: center;

  a {
    color: var(--atk-admin-sage);
    text-decoration: none;
  }

  span {
    display: block;
    margin-top: 3px;
  }
}

.layer {
  position: fixed;
  width: 100%;
  height: 100%;
  background: rgba(10, 20, 16, 0.36);
  backdrop-filter: blur(3px);
}

.user-selector {
  position: fixed;
  top: 50%;
  left: 50%;
  z-index: 999;
  width: min(300px, calc(100% - 40px));
  padding: 10px;
  border: 1px solid var(--atk-admin-border);
  border-radius: var(--atk-admin-radius);
  background: var(--atk-admin-surface);
  box-shadow: var(--atk-admin-shadow);
  transform: translate(-50%, -50%);

  & > .text {
    padding: 14px 14px 10px;
    color: var(--atk-admin-subtle);
    font-size: 13px;
  }

  .user-list > .item {
    padding: 10px 14px;
    border-radius: var(--atk-admin-radius-xs);
    color: var(--atk-admin-ink);
    cursor: pointer;
    transition: background-color 0.18s ease;

    &:hover {
      background: var(--atk-admin-surface-muted);
    }
  }
}
</style>
