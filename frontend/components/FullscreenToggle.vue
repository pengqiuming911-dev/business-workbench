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
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { Maximize2, Minimize2 } from '@lucide/vue'

const props = defineProps({
  target: { type: String, required: true },
})

const isFullscreen = ref(false)
const icon = computed(() => isFullscreen.value ? Minimize2 : Maximize2)

function toggle() {
  isFullscreen.value = !isFullscreen.value
}

function onKeydown(e) {
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
  }
}

watch(isFullscreen, (val) => {
  const el = document.querySelector(props.target)
  if (!el) return
  el.classList.toggle('is-fullscreen', val)
  document.body.classList.toggle('table-fullscreen-active', val)

  if (val) {
    const sidebar = document.querySelector('.sidebar')
    const sidebarWidth = sidebar ? sidebar.getBoundingClientRect().width : 208
    el.style.setProperty('--sidebar-width', sidebarWidth + 'px')
  }
})

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.classList.remove('table-fullscreen-active')
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
