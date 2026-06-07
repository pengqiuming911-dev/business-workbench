<template>
  <div class="panel-card">
    <h3 v-if="title" class="panel-card-header">{{ title }}</h3>
    <slot />
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, default: '' },
})
</script>
