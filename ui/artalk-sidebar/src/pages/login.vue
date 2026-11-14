<route lang="json">
{
  "meta": {
    "title": "Login - Artalk Sidebar"
  }
}
</route>

<script setup lang="ts">
import type { ArtalkType } from 'artalk'
import VerifyCodeButton from '../components/VerifyCodeButton.vue'
import { getArtalk } from '../global'
import { useUserStore } from '../stores/user'

const router = useRouter()
const { t } = useI18n()

type LoginMode = 'login' | 'register' | 'reset'
const mode = ref<LoginMode>('login')

let userForm = ref({
  name: '',
  email: '',
  password: '',
  confirm: '',
})
let code = ref('')
let version = ref('')
let buildHash = ref('')
let loginErr = ref('')
let submitting = ref(false)
let userSelector = ref<string[] | null>(null)

const modeTitle = computed(() => {
  if (mode.value === 'register') return t('signUp')
  if (mode.value === 'reset') return t('resetPassword')
  return t('signIn')
})

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

function finishLogin(data: { user: any; token: string }) {
  const artalk = getArtalk()
  if (!artalk) throw new Error('Artalk instance not initialized')

  artalk.ctx.getUser().update({
    ...data.user,
    token: data.token,
  })
  useUserStore().sync()
  router.replace('/comments')
}

async function login(username?: string) {
  loginErr.value = ''

  const artalk = getArtalk()
  if (!artalk) throw new Error('Artalk instance not initialized')

  submitting.value = true
  try {
    const res = await artalk.ctx.getApi().user.login({
      name: username || '',
      email: userForm.value.email,
      password: userForm.value.password,
    })
    finishLogin(res.data)
  } catch (e: any) {
    if (e.data?.need_name_select) {
      userSelector.value = e.data?.need_name_select
    } else {
      loginErr.value = e.message || t('loginFailure')
    }
  } finally {
    submitting.value = false
  }
}

function submit() {
  loginErr.value = ''

  if (mode.value === 'login') {
    login()
    return
  }

  if (userForm.value.password !== userForm.value.confirm) {
    loginErr.value = t('passwordMismatch')
    return
  }

  const artalk = getArtalk()
  if (!artalk) throw new Error('Artalk instance not initialized')

  submitting.value = true
  ;(async () => {
    try {
      const res =
        mode.value === 'register'
          ? await artalk.ctx.getApi().auth.registerByEmail({
              name: userForm.value.name.trim(),
              email: userForm.value.email.trim(),
              password: userForm.value.password,
              code: code.value.trim(),
            })
          : await artalk.ctx.getApi().auth.registerByEmail({
              email: userForm.value.email.trim(),
              password: userForm.value.password,
              code: code.value.trim(),
            })

      finishLogin(res.data)
    } catch (e: any) {
      loginErr.value = e.message || t('loginFailure')
    } finally {
      submitting.value = false
    }
  })()
}

function switchMode(next: LoginMode) {
  mode.value = next
  loginErr.value = ''
  userSelector.value = null
  code.value = ''
  userForm.value.password = ''
  userForm.value.confirm = ''
}

function selectUser(username: string) {
  userSelector.value = null
  login(username)
}

function onCodeError(message: string) {
  loginErr.value = message
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
        <p>{{ mode === 'login' ? '使用账户继续' : modeTitle }}</p>
      </div>

      <form class="login-form" @submit.prevent="submit()">
        <label v-if="mode === 'register'" class="field">
          <span>{{ t('username') }}</span>
          <input
            v-model="userForm.name"
            type="text"
            autocomplete="nickname"
            :placeholder="t('username')"
            required
            @focus="onFocus"
          />
        </label>

        <label class="field">
          <span>{{ t('email') }}</span>
          <input
            v-model="userForm.email"
            type="email"
            autocomplete="username"
            :placeholder="t('email')"
            required
            @focus="onFocus"
          />
        </label>

        <label class="field">
          <span>{{ mode === 'login' ? t('password') : t('newPassword') }}</span>
          <input
            v-model="userForm.password"
            type="password"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            :minlength="mode === 'login' ? undefined : 8"
            :placeholder="mode === 'login' ? t('password') : t('newPassword')"
            required
            @focus="onFocus"
          />
        </label>

        <template v-if="mode !== 'login'">
          <label class="field">
            <span>{{ t('confirmPassword') }}</span>
            <input
              v-model="userForm.confirm"
              type="password"
              autocomplete="new-password"
              minlength="8"
              :placeholder="t('confirmPassword')"
              required
              @focus="onFocus"
            />
          </label>

          <div class="field">
            <span>{{ t('verificationCode') }}</span>
            <div class="code-row">
              <input
                v-model="code"
                type="text"
                inputmode="numeric"
                autocomplete="one-time-code"
                maxlength="6"
                :placeholder="t('verificationCode')"
                required
                @focus="onFocus"
              />
              <VerifyCodeButton :email="userForm.email" @error="onCodeError" />
            </div>
          </div>
        </template>

        <div v-if="!!loginErr" class="err-msg atk-fade-in">{{ loginErr }}</div>

        <button type="submit" :disabled="submitting">
          {{ submitting ? '…' : modeTitle }}
        </button>

        <div class="form-links">
          <template v-if="mode === 'login'">
            <span>
              {{ t('noAccountPrompt') }}
              <button type="button" class="link-btn" @click="switchMode('register')">
                {{ t('signUp') }}
              </button>
            </span>
            <button type="button" class="link-btn" @click="switchMode('reset')">
              {{ t('forgetPassword') }}
            </button>
          </template>
          <span v-else>
            {{ t('haveAccountPrompt') }}
            <button type="button" class="link-btn" @click="switchMode('login')">
              {{ t('signIn') }}
            </button>
          </span>
        </div>
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

  > button[type='submit'] {
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

  .code-row {
    display: flex;
    gap: 10px;

    input {
      flex: 1;
      min-width: 0;
    }

    .verify-code-btn {
      height: 42px;
    }
  }

  .form-links {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-top: 2px;
    color: var(--atk-admin-subtle);
    font-size: 12px;
  }

  .link-btn {
    height: auto;
    min-height: 0;
    margin: 0;
    padding: 0;
    border: 0;
    color: var(--atk-admin-sage);
    background: transparent;
    font-size: 12px;
    font-weight: 650;
    text-decoration: none;

    &:hover {
      border: 0;
      color: var(--atk-admin-sage-strong);
      background: transparent;
      text-decoration: underline;
    }

    &:active {
      transform: none;
    }
  }

  > button[type='submit']:disabled {
    cursor: not-allowed;
    opacity: 0.6;
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
