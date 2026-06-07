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

<style scoped>
/* Depends on .panel-card and .panel-card-header defined in global main.css */
</style>
