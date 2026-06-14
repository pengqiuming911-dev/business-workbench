<template>
  <div class="workbench-shell">
    <SidebarNav
      :collapsed="sidebarCollapsed"
      :overlay-open="sidebarOverlay"
      @navigate="sidebarOverlay = false"
      @close="sidebarOverlay = false"
      @collapse="sidebarCollapsed = !sidebarCollapsed"
    />

    <div class="workbench-content" :class="{ expanded: sidebarCollapsed }">
      <header class="workbench-topbar">
        <div class="topbar-actions">
          <button class="topbar-btn" type="button" @click="searchOpen = true">
            <Search :size="18" :stroke-width="2" />
          </button>
          <button class="topbar-btn" type="button">
            <Bell :size="18" :stroke-width="2" />
          </button>
          <div class="topbar-avatar"></div>
        </div>
      </header>

      <main class="workbench-main">
        <slot />
      </main>
    </div>

    <AgentDrawer :open="drawerOpen" @close="drawerOpen = false" />

    <button
      class="agent-fab"
      type="button"
      @click="drawerOpen = !drawerOpen"
    >
      <X v-if="drawerOpen" :size="24" :stroke-width="2" color="#fff" />
      <MessageSquare v-else :size="24" :stroke-width="2" color="#fff" />
    </button>

    <GlobalSearch v-model:open="searchOpen" />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Search, Bell, MessageSquare, X } from '@lucide/vue'
import SidebarNav from './SidebarNav.vue'
import AgentDrawer from './AgentDrawer.vue'
import GlobalSearch from './GlobalSearch.vue'

const sidebarCollapsed = ref(false)
const sidebarOverlay = ref(false)
const searchOpen = ref(false)
const drawerOpen = ref(true)

function handleKeydown(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    searchOpen.value = !searchOpen.value
  }
}

function handleToggleDrawer() {
  drawerOpen.value = !drawerOpen.value
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
  window.addEventListener('toggle-agent-drawer', handleToggleDrawer)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('toggle-agent-drawer', handleToggleDrawer)
})
</script>

<style scoped>
.workbench-shell {
  min-height: 100vh;
}

.workbench-content {
  min-height: 100vh;
  margin-left: 230px;
  transition: margin-left 220ms ease;
}

.workbench-content.expanded {
  margin-left: 64px;
}

.workbench-topbar {
  position: sticky;
  top: 0;
  z-index: 60;
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 12px 32px 4px;
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.topbar-btn {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-soft);
  border-radius: 50%;
  background: var(--bg-card);
  color: var(--ink-soft);
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.topbar-btn:hover {
  background: var(--brand-soft);
  color: var(--brand);
}

.topbar-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--brand);
  opacity: 0.7;
  margin-left: 4px;
}

.workbench-main {
  max-width: 1100px;
  padding: 24px 32px 72px;
  margin: 0 auto 0 0;
}

/* FAB */
.agent-fab {
  position: fixed;
  bottom: 28px;
  right: 28px;
  z-index: 200;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: none;
  background: #10b981;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.agent-fab:hover {
  transform: scale(1.08);
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.45);
}

@media (max-width: 860px) {
  .workbench-content {
    margin-left: 0;
  }

  .workbench-topbar {
    padding: 10px 14px;
  }

  .workbench-main {
    padding: 16px 14px 48px;
  }
}
</style>
