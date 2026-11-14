<script setup lang="ts">
import { getArtalk } from '../global'

const props = defineProps<{
  email: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  error: [message: string]
}>()

const { t } = useI18n()

const sending = ref(false)
const remaining = ref(0)
let timer: number | undefined

onBeforeUnmount(() => {
  if (timer !== undefined) window.clearInterval(timer)
})

const label = computed(() => {
  if (sending.value) return '…'
  if (remaining.value > 0) return t('waitSeconds', { seconds: String(remaining.value) })
  return t('verifySend')
})

async function send() {
  if (sending.value || remaining.value > 0) return

  const email = props.email.trim()
  if (!email) {
    emit('error', t('email'))
    return
  }

  sending.value = true
  try {
    await getArtalk()!.ctx.getApi().auth.sendVerifyEmail({ email })

    remaining.value = 60
    timer = window.setInterval(() => {
      remaining.value -= 1
      if (remaining.value <= 0 && timer !== undefined) {
        window.clearInterval(timer)
        timer = undefined
      }
    }, 1000)
  } catch (e: any) {
    emit('error', e?.message || t('opFailed'))
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <button
    type="button"
    class="verify-code-btn"
    :disabled="disabled || sending || remaining > 0"
    @click="send"
  >
    {{ label }}
  </button>
</template>

<style scoped lang="scss">
.verify-code-btn {
  flex: none;
  min-width: 104px;
  height: 40px;
  padding: 0 12px;
  border: 1px solid var(--atk-admin-border-strong);
  border-radius: var(--atk-admin-radius-xs);
  color: var(--atk-admin-sage-strong);
  background: var(--atk-admin-surface);
  cursor: pointer;
  font-size: 12.5px;
  font-weight: 650;
  white-space: nowrap;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease;

  &:hover:not(:disabled) {
    border-color: var(--atk-admin-sage);
    background: var(--atk-admin-sage-soft);
  }

  &:disabled {
    cursor: not-allowed;
    color: var(--atk-admin-faint);
    background: var(--atk-admin-surface-muted);
  }
}
</style>
