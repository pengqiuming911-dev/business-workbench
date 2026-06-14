<template>
  <div class="workbench-shell">
    <SidebarNav
      :collapsed="sidebarCollapsed"
      :overlay-open="sidebarOverlay"
      @navigate="closeSidebar"
      @close="sidebarOverlay = false"
    />

    <div class="workbench-content" :class="{ expanded: sidebarCollapsed }">
      <header class="workbench-topbar">
        <button class="sidebar-toggle" type="button" aria-label="切换导航" @click="toggleSidebar">
          <PanelLeftClose v-if="!sidebarCollapsed" :size="21" :stroke-width="2" />
          <PanelLeftOpen v-else :size="21" :stroke-width="2" />
        </button>
      </header>

      <main class="workbench-main">
        <slot />
      </main>
    </div>

    <GlobalSearch v-model:open="searchOpen" />
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { PanelLeftClose, PanelLeftOpen } from '@lucide/vue'
import SidebarNav from './SidebarNav.vue'
import GlobalSearch from './GlobalSearch.vue'

defineProps({
  wide: { type: Boolean, default: false },
})

const sidebarCollapsed = ref(false)
const sidebarOverlay = ref(false)
const searchOpen = ref(false)

function toggleSidebar() {
  if (window.innerWidth <= 860) {
    sidebarOverlay.value = !sidebarOverlay.value
  } else {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }
}

function closeSidebar() {
  sidebarOverlay.value = false
}

function handleKeydown(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    searchOpen.value = !searchOpen.value
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.workbench-shell {
  min-height: 100vh;
}

.workbench-content {
  min-height: 100vh;
  margin-left: 264px;
  transition: margin-left 280ms cubic-bezier(0.4, 0, 0.2, 1);
}

.workbench-content.expanded {
  margin-left: 86px;
}

.workbench-topbar {
  position: sticky;
  top: 0;
  z-index: 60;
  min-height: 64px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 12px 32px 4px;
  pointer-events: none;
}

.sidebar-toggle {
  pointer-events: auto;
  width: 46px;
  height: 46px;
  display: none;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  color: var(--ink-soft);
  background: #fff;
  box-shadow: var(--shadow-sm);
  transition: background 180ms ease, border-color 180ms ease;
}

.sidebar-toggle:hover {
  background: var(--bg-hover);
  border-color: var(--border);
}

.workbench-main {
  width: min(1200px, calc(100vw - 328px));
  margin: 0 auto 0 36px;
  padding: 0 0 72px;
}

@media (max-width: 1280px) {
  .workbench-content {
    margin-left: 250px;
  }

  .workbench-main {
    width: min(100% - 48px, 1120px);
    margin-left: 24px;
  }

  .workbench-topbar {
    padding-right: 36px;
    padding-left: 24px;
  }
}

@media (max-width: 860px) {
  .workbench-content,
  .workbench-content.expanded {
    margin-left: 0;
  }

  .workbench-topbar {
    min-height: 64px;
    padding: 12px 14px 0;
  }

  .sidebar-toggle {
    display: inline-flex;
    width: 46px;
    min-height: 46px;
    height: 46px;
  }

  .workbench-main {
    width: calc(100% - 28px);
    margin: 0 auto;
    padding-bottom: 36px;
  }
}
</style>
