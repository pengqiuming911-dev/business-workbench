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
        <button
          class="sidebar-toggle"
          type="button"
          @click="toggleSidebar"
        >
          <Menu :size="20" />
        </button>

        <div class="topbar-search" @click="openSearch">
          <Search :size="16" />
          <span class="search-placeholder">搜索客户、产品、渠道...</span>
          <kbd class="search-kbd">{{ isMac ? '⌘' : 'Ctrl' }} K</kbd>
        </div>

        <div class="topbar-actions">
          <div class="topbar-avatar">
            <span class="avatar-circle">BW</span>
          </div>
        </div>
      </header>

      <main class="workbench-main">
        <slot />
      </main>
    </div>

    <GlobalSearch v-model:open="searchOpen" />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Menu, Search } from '@lucide/vue'
import SidebarNav from './SidebarNav.vue'
import GlobalSearch from './GlobalSearch.vue'

defineProps({
  wide: { type: Boolean, default: false },
})

const sidebarCollapsed = ref(false)
const sidebarOverlay = ref(false)
const searchOpen = ref(false)
const isMac = ref(false)

function toggleSidebar() {
  if (window.innerWidth <= 720) {
    sidebarOverlay.value = !sidebarOverlay.value
  } else {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }
}

function closeSidebar() {
  sidebarOverlay.value = false
}

function openSearch() {
  searchOpen.value = true
}

function handleKeydown(e) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    searchOpen.value = !searchOpen.value
  }
}

onMounted(() => {
  isMac.value = navigator.platform.toUpperCase().indexOf('MAC') >= 0
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
  margin-left: 240px;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  transition: margin-left 250ms ease;
}

.workbench-content.expanded {
  margin-left: 64px;
}

.workbench-topbar {
  position: sticky;
  top: 0;
  z-index: 50;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 24px;
  background: rgba(254, 252, 245, 0.85);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--border-soft);
}

.sidebar-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-soft);
  transition: all 150ms;
}
.sidebar-toggle:hover {
  background: var(--bg-hover);
  color: var(--ink-strong);
}

.topbar-search {
  flex: 1;
  max-width: 480px;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: #fff;
  color: var(--ink-faint);
  cursor: pointer;
  transition: border-color 150ms;
}
.topbar-search:hover {
  border-color: var(--brand);
}

.search-placeholder {
  flex: 1;
  font-size: 13px;
}

.search-kbd {
  font-family: var(--font-sans);
  font-size: 11px;
  font-weight: 600;
  padding: 2px 6px;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--ink-soft);
  background: var(--bg-hover);
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.avatar-circle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--brand-soft);
  color: var(--brand);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
}

.workbench-main {
  flex: 1;
  padding: 24px 28px;
  max-width: 1200px;
  width: 100%;
}

@media (max-width: 720px) {
  .workbench-content {
    margin-left: 0;
  }
  .workbench-content.expanded {
    margin-left: 0;
  }
  .topbar-search {
    max-width: 240px;
  }
}
</style>