<template>
  <WorkbenchLayout :title="title" :description="description" :wide="wide">
    <slot />
  </WorkbenchLayout>
</template>

<script setup>
import WorkbenchLayout from './WorkbenchLayout.vue'

defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  wide: { type: Boolean, default: false },
})
</script>
