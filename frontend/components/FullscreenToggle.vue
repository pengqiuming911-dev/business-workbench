<template>
  <button
    class="btn btn-secondary btn-sm fullscreen-btn"
    :title="isFullscreen ? '退出全屏' : '全屏显示表格'"
    @click="toggle"
  >
    <component :is="icon" :size="14" :stroke-width="2" />
    <span class="fullscreen-label">{{ isFullscreen ? '退出全屏' : '全屏' }}</span>
  </button>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { Maximize2, Minimize2 } from '@lucide/vue'

const props = defineProps({
  target: { type: String, required: true },
})

const isFullscreen = ref(false)
const icon = computed(() => isFullscreen.value ? Minimize2 : Maximize2)

function onFullscreenChange() {
  isFullscreen.value = document.fullscreenElement !== null
}

function toggle() {
  const el = document.querySelector(props.target)
  if (!el) return

  if (!document.fullscreenElement) {
    el.requestFullscreen().catch(() => {})
  } else {
    document.exitFullscreen().catch(() => {})
  }
}

onMounted(() => {
  document.addEventListener('fullscreenchange', onFullscreenChange)
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
})
</script>

<style scoped>
.fullscreen-btn {
  gap: 4px;
  white-space: nowrap;
}

.fullscreen-label {
  font-size: 12px;
}
</style>
