<template>
  <div class="panel-card">
    <div v-if="title || $slots['header-actions']" class="panel-card-header-row">
      <h3 v-if="title" class="panel-card-header">{{ title }}</h3>
      <slot name="header-actions" />
    </div>
    <slot />
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, default: '' },
})
</script>

<style scoped>
.panel-card-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  gap: 16px;
}

.panel-card-header-row .panel-card-header {
  margin-bottom: 0;
  letter-spacing: -0.01em;
}
</style>
