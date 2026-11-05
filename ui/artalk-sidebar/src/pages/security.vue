<route lang="json">
{
  "meta": {
    "title": "Security - Artalk Sidebar"
  }
}
</route>

<script setup lang="ts">
import AdminPageHeader from '../components/AdminPageHeader.vue'
import VerifyCodeButton from '../components/VerifyCodeButton.vue'
import { getArtalk } from '../global'
import {
  accountApi,
  errorMessage,
  resolveServerPath,
  type AccountIdentity,
  type AccountProfile,
  type AuthProviderInfo,
} from '../lib/account-api'
import { startOAuthBind } from '../lib/oauth-bind'
import { useNavStore } from '../stores/nav'
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const router = useRouter()
const nav = useNavStore()
const userStore = useUserStore()

const loading = ref(true)
const loadError = ref('')

const profile = reactive({
  name: '',
  email: '',
  link: '',
})
const initialEmail = ref('')
const emailCode = ref('')
const hasPassword = ref(false)
const profileSaving = ref(false)
const profileError = ref('')
const profileNotice = ref('')

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordCode = ref('')
const passwordSaving = ref(false)
const passwordError = ref('')
const passwordNotice = ref('')

const identities = ref<AccountIdentity[]>([])
const providers = ref<AuthProviderInfo[]>([])
const bindingProvider = ref('')
const unbindTarget = ref<number | null>(null)
const identityError = ref('')
const identityNotice = ref('')

const emailChanged = computed(
  () => profile.email.trim().toLowerCase() !== initialEmail.value.toLowerCase(),
)
const oauthProviders = computed(() =>
  providers.value.filter((p) => p.name !== 'email' && p.name !== 'skip'),
)
const passwordMismatch = computed(
  () => !!confirmPassword.value && newPassword.value !== confirmPassword.value,
)
const canSubmitPassword = computed(
  () =>
    newPassword.value.length >= 8 &&
    !passwordMismatch.value &&
    (hasPassword.value ? !!currentPassword.value : !!passwordCode.value.trim()),
)

function identityFor(provider: string) {
  return identities.value.find((item) => item.provider === provider) || null
}

function applyProfile(user: AccountProfile) {
  profile.name = user.name
  profile.email = user.email
  profile.link = user.link
  initialEmail.value = user.email
  hasPassword.value = user.has_password

  const artalk = getArtalk()
  if (artalk) {
    artalk.ctx.getUser().update({
      name: user.name,
      email: user.email,
      link: user.link,
    })
  }
  userStore.$patch({
    name: user.name,
    email: user.email,
    link: user.link,
  })
}

async function loadAll() {
  loading.value = true
  loadError.value = ''

  try {
    const [profileRes, identityRes] = await Promise.all([
      accountApi.getProfile(),
      accountApi.getIdentities(),
    ])

    if (!profileRes.is_login || !profileRes.user) {
      router.replace('/login')
      return
    }

    applyProfile(profileRes.user)
    identities.value = identityRes.identities || []

    // The provider list is only exposed when social login is configured.
    // Profile and password management must keep working without it.
    try {
      const providerRes = await accountApi.getProviders()
      providers.value = providerRes.providers || []
    } catch {
      providers.value = []
    }
  } catch (e) {
    loadError.value = errorMessage(e) || t('opFailed')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  nav.updateTabs({})
  nav.scrollPageToTop()
  loadAll()
})

async function saveProfile() {
  if (profileSaving.value) return

  profileError.value = ''
  profileNotice.value = ''

  if (emailChanged.value && !emailCode.value.trim()) {
    profileError.value = t('emailChangeHint')
    return
  }

  profileSaving.value = true
  try {
    const res = await accountApi.updateProfile({
      name: profile.name.trim(),
      email: profile.email.trim(),
      link: profile.link.trim(),
      code: emailCode.value.trim() || undefined,
    })

    applyProfile(res.user)
    emailCode.value = ''
    profileNotice.value = t('settingSaved')
  } catch (e) {
    profileError.value = errorMessage(e) || t('opFailed')
  } finally {
    profileSaving.value = false
  }
}

async function changePassword() {
  if (passwordSaving.value || !canSubmitPassword.value) return

  passwordError.value = ''
  passwordNotice.value = ''

  passwordSaving.value = true
  try {
    const res = await accountApi.updatePassword({
      current_password: hasPassword.value ? currentPassword.value : undefined,
      new_password: newPassword.value,
      code: hasPassword.value ? undefined : passwordCode.value.trim(),
    })

    getArtalk()?.ctx.getUser().update({ token: res.token })
    userStore.$patch({ token: res.token })
    hasPassword.value = res.user.has_password

    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordCode.value = ''
    passwordNotice.value = t('passwordUpdated')
  } catch (e) {
    passwordError.value = errorMessage(e) || t('opFailed')
  } finally {
    passwordSaving.value = false
  }
}

async function bind(provider: AuthProviderInfo) {
  if (bindingProvider.value) return

  identityError.value = ''
  identityNotice.value = ''
  bindingProvider.value = provider.name

  try {
    const { path } = await accountApi.startIdentityBind(provider.name)
    const result = await startOAuthBind(resolveServerPath(path))
    if (!result.ok) throw new Error(result.message || t('bindFailed'))

    await refreshIdentities()
    identityNotice.value = t('bindSuccess')
  } catch (e) {
    // The popup may close before its result message reaches us. Refresh the
    // list so a successful link is not reported as a failure.
    const wasLinked = identityFor(provider.name) !== null
    const message = errorMessage(e)
    if (!wasLinked) {
      await refreshIdentities()
    }
    if (!identityFor(provider.name)) {
      identityError.value =
        message === 'popup blocked' || message === 'canceled' || message === 'timeout'
          ? t('bindFailed')
          : message || t('bindFailed')
    } else {
      identityNotice.value = t('bindSuccess')
    }
  } finally {
    bindingProvider.value = ''
  }
}

async function refreshIdentities() {
  try {
    const res = await accountApi.getIdentities()
    identities.value = res.identities || []
  } catch (e) {
    identityError.value = errorMessage(e) || t('opFailed')
  }
}

async function confirmUnbind(identity: AccountIdentity) {
  identityError.value = ''
  identityNotice.value = ''

  try {
    await accountApi.removeIdentity(identity.id)
    unbindTarget.value = null
    await refreshIdentities()
  } catch (e) {
    identityError.value = errorMessage(e) || t('opFailed')
  }
}

function onCodeError(message: string) {
  profileError.value = message
}

function onPasswordCodeError(message: string) {
  passwordError.value = message
}
</script>

<template>
  <div class="security admin-page">
    <AdminPageHeader :title="t('security')" :description="t('securityDesc')" />

    <div v-if="loading" class="admin-panel security-state">
      <span class="state-spinner" />
      <span>{{ t('refresh') }}…</span>
    </div>

    <div v-else-if="loadError" class="admin-panel security-state error">
      <p>{{ loadError }}</p>
      <button class="admin-button" type="button" @click="loadAll">{{ t('refresh') }}</button>
    </div>

    <template v-else>
      <section class="admin-panel security-card">
        <header class="card-head">
          <div>
            <h2>{{ t('profileSection') }}</h2>
            <p>{{ t('emailChangeHint') }}</p>
          </div>
        </header>

        <form class="security-form" @submit.prevent="saveProfile">
          <div class="field-grid">
            <label class="field">
              <span>{{ t('username') }}</span>
              <input v-model="profile.name" type="text" autocomplete="nickname" required />
            </label>
            <label class="field">
              <span>{{ t('link') }}</span>
              <input v-model="profile.link" type="text" placeholder="https://" />
            </label>
          </div>

          <label class="field">
            <span>{{ t('email') }}</span>
            <input v-model="profile.email" type="email" autocomplete="email" required />
          </label>

          <div v-if="emailChanged" class="field">
            <span>{{ t('verificationCode') }}</span>
            <div class="code-row">
              <input
                v-model="emailCode"
                type="text"
                inputmode="numeric"
                autocomplete="one-time-code"
                maxlength="6"
                :placeholder="t('verificationCode')"
              />
              <VerifyCodeButton :email="profile.email" @error="onCodeError" />
            </div>
          </div>

          <div class="form-foot">
            <span v-if="profileError" class="form-msg error">{{ profileError }}</span>
            <span v-else-if="profileNotice" class="form-msg ok">{{ profileNotice }}</span>
            <span v-else class="form-msg" />
            <button class="admin-button primary" type="submit" :disabled="profileSaving">
              {{ profileSaving ? '…' : t('save') }}
            </button>
          </div>
        </form>
      </section>

      <section class="admin-panel security-card">
        <header class="card-head">
          <div>
            <h2>{{ t('password') }}</h2>
            <p>{{ hasPassword ? t('changePassword') : t('passwordUnset') }}</p>
          </div>
          <span class="admin-chip" :class="{ active: hasPassword }">
            {{ hasPassword ? t('passwordSet') : t('passwordUnset') }}
          </span>
        </header>

        <form class="security-form" @submit.prevent="changePassword">
          <label v-if="hasPassword" class="field">
            <span>{{ t('currentPassword') }}</span>
            <input
              v-model="currentPassword"
              type="password"
              autocomplete="current-password"
              required
            />
          </label>

          <div v-else class="field">
            <span>{{ t('verificationCode') }}</span>
            <div class="code-row">
              <input
                v-model="passwordCode"
                type="text"
                inputmode="numeric"
                autocomplete="one-time-code"
                maxlength="6"
                :placeholder="t('verificationCode')"
              />
              <VerifyCodeButton :email="profile.email" @error="onPasswordCodeError" />
            </div>
          </div>

          <div class="field-grid">
            <label class="field">
              <span>{{ t('newPassword') }}</span>
              <input
                v-model="newPassword"
                type="password"
                autocomplete="new-password"
                minlength="8"
                required
              />
            </label>
            <label class="field">
              <span>{{ t('confirmPassword') }}</span>
              <input
                v-model="confirmPassword"
                type="password"
                autocomplete="new-password"
                minlength="8"
                required
              />
            </label>
          </div>

          <p v-if="passwordMismatch" class="form-msg error">{{ t('passwordMismatch') }}</p>

          <div class="form-foot">
            <span v-if="passwordError" class="form-msg error">{{ passwordError }}</span>
            <span v-else-if="passwordNotice" class="form-msg ok">{{ passwordNotice }}</span>
            <span v-else class="form-msg" />
            <button
              class="admin-button primary"
              type="submit"
              :disabled="passwordSaving || !canSubmitPassword"
            >
              {{ passwordSaving ? '…' : t('save') }}
            </button>
          </div>
        </form>
      </section>

      <section class="admin-panel security-card">
        <header class="card-head">
          <div>
            <h2>{{ t('bindingsSection') }}</h2>
            <p>{{ t('bindingsDesc') }}</p>
          </div>
        </header>

        <div class="identity-list">
          <div v-for="provider in oauthProviders" :key="provider.name" class="identity-row">
            <img class="identity-icon" :src="provider.icon" alt="" />

            <div class="identity-meta">
              <strong>{{ provider.label }}</strong>
              <span :class="{ bound: !!identityFor(provider.name) }">
                {{ identityFor(provider.name) ? t('bound') : t('notBound') }}
              </span>
            </div>

            <div class="identity-actions">
              <template v-if="identityFor(provider.name)">
                <template v-if="unbindTarget === identityFor(provider.name)!.id">
                  <button
                    class="admin-button danger"
                    type="button"
                    @click="confirmUnbind(identityFor(provider.name)!)"
                  >
                    {{ t('unbind') }}
                  </button>
                  <button class="admin-button" type="button" @click="unbindTarget = null">
                    {{ t('cancel') }}
                  </button>
                </template>
                <button
                  v-else
                  class="admin-button"
                  type="button"
                  @click="unbindTarget = identityFor(provider.name)!.id"
                >
                  {{ t('unbind') }}
                </button>
              </template>
              <button
                v-else
                class="admin-button primary"
                type="button"
                :disabled="!!bindingProvider"
                @click="bind(provider)"
              >
                {{ bindingProvider === provider.name ? '…' : t('bind') }}
              </button>
            </div>
          </div>

          <div v-if="!oauthProviders.length" class="admin-empty">
            {{ t('bindingsEmpty') }}
          </div>
        </div>

        <p v-if="identityError" class="form-msg error identity-msg">{{ identityError }}</p>
        <p v-else-if="identityNotice" class="form-msg ok identity-msg">{{ identityNotice }}</p>
      </section>
    </template>
  </div>
</template>

<style scoped lang="scss">
.security {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.security-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 160px;
  padding: 24px;
  color: var(--atk-admin-subtle);
  font-size: 13px;

  &.error {
    flex-direction: column;
    color: var(--atk-admin-danger);
  }

  p {
    margin: 0;
  }
}

.state-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--atk-admin-border-strong);
  border-top-color: var(--atk-admin-sage);
  border-radius: 50%;
  animation: security-spin 0.8s linear infinite;
}

@keyframes security-spin {
  to {
    transform: rotate(360deg);
  }
}

.security-card {
  overflow: hidden;
}

.card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 22px 0;

  h2 {
    margin: 0;
    color: var(--atk-admin-ink);
    font-size: 16px;
    font-weight: 700;
    letter-spacing: -0.01em;
  }

  p {
    margin: 6px 0 0;
    color: var(--atk-admin-subtle);
    font-size: 12.5px;
    line-height: 1.55;
  }
}

.admin-chip.active {
  border-color: var(--atk-admin-sage);
  color: var(--atk-admin-sage-strong);
  background: var(--atk-admin-sage-soft);
}

.security-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px 22px 22px;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 16px;
}

.field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 7px;

  > span {
    color: var(--atk-admin-subtle);
    font-size: 12px;
    font-weight: 650;
  }

  input {
    width: 100%;
    height: 40px;
    padding: 0 12px;
    border: 1px solid var(--atk-admin-border);
    border-radius: var(--atk-admin-radius-xs);
    color: var(--atk-admin-ink);
    background: var(--atk-admin-surface-muted);
    font-size: 13px;
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
}

.code-row {
  display: flex;
  gap: 10px;

  input {
    flex: 1;
    min-width: 0;
  }
}

.form-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  min-height: 38px;
}

.form-msg {
  min-width: 0;
  color: var(--atk-admin-subtle);
  font-size: 12.5px;
  line-height: 1.5;

  &.error {
    color: var(--atk-admin-danger);
  }

  &.ok {
    color: var(--atk-admin-sage-strong);
  }
}

.identity-list {
  display: flex;
  flex-direction: column;
  padding: 8px 10px 12px;
}

.identity-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: var(--atk-admin-radius-sm);
  transition: background-color 0.18s ease;

  & + & {
    border-top: 1px solid var(--atk-admin-border);
    border-top-left-radius: 0;
    border-top-right-radius: 0;
  }

  &:hover {
    background: var(--atk-admin-surface-muted);
  }
}

.identity-icon {
  width: 34px;
  height: 34px;
  flex: none;
  border: 1px solid var(--atk-admin-border);
  border-radius: 10px;
  background: var(--atk-admin-surface);
  object-fit: contain;
  padding: 5px;
}

.identity-meta {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;

  strong {
    overflow: hidden;
    color: var(--atk-admin-ink);
    font-size: 13.5px;
    font-weight: 650;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    color: var(--atk-admin-faint);
    font-size: 12px;

    &.bound {
      color: var(--atk-admin-sage-strong);
    }
  }
}

.identity-actions {
  display: flex;
  flex: none;
  gap: 8px;
}

.identity-msg {
  padding: 0 22px 18px;
}

.admin-empty {
  padding: 28px 16px 32px;
}

@media (max-width: 560px) {
  .card-head,
  .security-form {
    padding-right: 16px;
    padding-left: 16px;
  }

  .form-foot {
    align-items: stretch;
    flex-direction: column;

    .admin-button {
      width: 100%;
    }
  }

  .identity-msg {
    padding-right: 16px;
    padding-left: 16px;
  }
}
</style>
