<script setup lang="ts">
/**
 * Pagination Component
 *
 * Circular page buttons with a compact window, first/last shortcuts and
 * ellipsis for the gaps. The wrapper fades into the page background at the top.
 */
const props = defineProps<{
  /** Total item count */
  total: number

  /** Page size */
  pageSize: number

  /** Disabled */
  disabled?: boolean
}>()

const { pageSize, total, disabled } = toRefs(props)

const emit = defineEmits<{
  /** Page change event */
  (evt: 'change', offset: number): void
}>()

const curtPage = ref(1)
const maxPage = computed(() => Math.max(1, Math.ceil(total.value / Math.max(1, pageSize.value))))
const offset = computed(() => pageSize.value * (curtPage.value - 1))
const prevDisabled = computed(() => curtPage.value <= 1)
const nextDisabled = computed(() => curtPage.value >= maxPage.value)

type PageItem = number | 'ellipsis'

/**
 * Keep the current page in a small window: first, last, and the nearby pages.
 * Near either edge the window grows to five pages so it never looks sparse.
 */
const pageItems = computed<PageItem[]>(() => {
  const last = maxPage.value
  const current = curtPage.value

  if (last <= 7) return Array.from({ length: last }, (_, i) => i + 1)

  if (current <= 3) {
    return [1, 2, 3, 4, 5, 'ellipsis', last]
  }
  if (current >= last - 2) {
    return [1, 'ellipsis', last - 4, last - 3, last - 2, last - 1, last]
  }

  return [1, 'ellipsis', current - 1, current, current + 1, current + 2, 'ellipsis', last]
})

watch(maxPage, (last) => {
  if (curtPage.value > last) curtPage.value = last
})

function changePage(page: number) {
  if (disabled?.value) return
  const target = Math.min(maxPage.value, Math.max(1, page))
  if (target === curtPage.value) return

  curtPage.value = target
  emit('change', offset.value)
}

function prev() {
  if (disabled?.value || prevDisabled.value) return
  changePage(curtPage.value - 1)
}

function next() {
  if (disabled?.value || nextDisabled.value) return
  changePage(curtPage.value + 1)
}

function reset() {
  curtPage.value = 1
}

defineExpose({ prev, next, reset })
</script>

<template>
  <div v-if="maxPage > 1" class="atk-pagination-wrap">
    <nav class="atk-pagination" aria-label="Pagination">
      <button
        type="button"
        class="page-arrow"
        :disabled="disabled || prevDisabled"
        aria-label="Previous page"
        @click="prev()"
      >
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path
            d="M10 3.5 5.5 8l4.5 4.5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>

      <template v-for="(item, index) in pageItems" :key="`${item}-${index}`">
        <span v-if="item === 'ellipsis'" class="page-ellipsis" aria-hidden="true">…</span>
        <button
          v-else
          type="button"
          class="page-btn"
          :class="{ active: item === curtPage }"
          :aria-current="item === curtPage ? 'page' : undefined"
          :disabled="disabled"
          @click="changePage(item)"
        >
          {{ item }}
        </button>
      </template>

      <button
        type="button"
        class="page-arrow"
        :disabled="disabled || nextDisabled"
        aria-label="Next page"
        @click="next()"
      >
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path
            d="m6 3.5 4.5 4.5L6 12.5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </nav>
  </div>
</template>

<style scoped lang="scss">
.atk-pagination-wrap {
  display: flex;
  justify-content: center;
  width: 100%;
  padding: 30px 12px 16px;
  pointer-events: none;
}

.atk-pagination {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  overflow-x: auto;
  padding: 2px;
  pointer-events: auto;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
}

.page-btn,
.page-arrow {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 1px solid var(--atk-admin-border);
  border-radius: 50%;
  color: var(--atk-admin-subtle);
  background: var(--atk-admin-surface);
  box-shadow: 0 4px 14px -10px rgba(18, 46, 38, 0.45);
  font-size: 12.5px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  cursor: pointer;
  transition:
    color 0.18s ease,
    background-color 0.18s ease,
    border-color 0.18s ease,
    transform 0.18s ease;

  &:hover:not(:disabled) {
    border-color: var(--atk-admin-sage);
    color: var(--atk-admin-sage-strong);
    background: var(--atk-admin-sage-soft);
    transform: translateY(-1px);
  }

  &:active:not(:disabled) {
    transform: translateY(0);
  }

  &:focus-visible {
    outline: 2px solid var(--atk-admin-ring);
    outline-offset: 2px;
  }

  &:disabled {
    cursor: default;
    opacity: 0.42;
  }
}

.page-btn.active {
  border-color: var(--atk-admin-ink);
  color: var(--atk-admin-surface);
  background: var(--atk-admin-ink);
  box-shadow: 0 8px 18px -12px rgba(18, 46, 38, 0.7);

  &:hover:not(:disabled) {
    border-color: var(--atk-admin-ink);
    color: var(--atk-admin-surface);
    background: var(--atk-admin-ink);
    transform: none;
  }
}

.page-arrow svg {
  width: 14px;
  height: 14px;
}

.page-ellipsis {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 34px;
  color: var(--atk-admin-faint);
  font-size: 14px;
  letter-spacing: 0.04em;
  user-select: none;
}

@media (max-width: 480px) {
  .atk-pagination {
    gap: 4px;
  }

  .page-btn,
  .page-arrow {
    width: 30px;
    height: 30px;
    font-size: 12px;
  }

  .page-ellipsis {
    width: 16px;
    height: 30px;
  }
}
</style>
