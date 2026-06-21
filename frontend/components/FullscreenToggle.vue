<template>
  <button
    v-if="!isFullscreen"
    class="btn btn-secondary btn-sm fullscreen-btn"
    title="全屏显示表格"
    @click="toggle"
  >
    <Maximize2 :size="14" :stroke-width="2" />
    <span class="fullscreen-label">全屏</span>
  </button>

  <Teleport to="body">
    <button
      v-if="isFullscreen"
      class="table-fullscreen-exit"
      type="button"
      title="退出全屏"
      @click="toggle"
    >
      <Minimize2 :size="12" :stroke-width="2" />
      <span>退出全屏</span>
    </button>
  </Teleport>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Maximize2, Minimize2 } from '@lucide/vue'

const props = defineProps({
  target: { type: String, required: true },
})

const isFullscreen = ref(false)

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
    el.style.setProperty('--sidebar-width', `${sidebarWidth}px`)
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

.table-fullscreen-exit {
  position: fixed;
  top: 12px;
  right: 16px;
  z-index: 1101;
  min-height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  font-size: 11px;
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.96);
  color: var(--ink-strong);
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(10px);
}

.table-fullscreen-exit:hover {
  background: #fff;
  border-color: var(--border);
}
</style>
