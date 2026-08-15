<script setup lang="ts">
defineProps<{
  title: string
  eyebrow?: string
  description?: string
}>()
</script>

<template>
  <section class="admin-page-head">
    <div>
      <div v-if="eyebrow" class="admin-eyebrow">{{ eyebrow }}</div>
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="admin-page-head-actions">
      <slot name="actions" />
    </div>
  </section>
</template>

<style scoped lang="scss">
.admin-page-head-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

@media (max-width: 850px) {
  .admin-page-head-actions {
    margin-top: 16px;
    justify-content: flex-start;
  }
}
</style>
