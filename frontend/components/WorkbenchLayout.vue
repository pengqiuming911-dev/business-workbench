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
      <main class="workbench-main">
        <slot />
      </main>
    </div>

    <AgentDrawer :open="drawerOpen" @close="drawerOpen = false" />

    <button
      v-show="!drawerOpen"
      class="agent-fab"
      type="button"
      @click="drawerOpen = true"
    >
      <MessageSquare :size="24" :stroke-width="2" color="#fff" />
    </button>

    <GlobalSearch v-model:open="searchOpen" />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { MessageSquare } from '@lucide/vue'
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
  height: 100vh;
  display: flex;
}

.workbench-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-left: 208px;
  transition: margin-left 220ms ease;
}

.workbench-content.expanded {
  margin-left: 64px;
}

.workbench-main {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  width: 100%;
  max-width: 1680px;
  padding: 8px 32px 16px;
  margin: 0 auto;
  box-sizing: border-box;
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

  .workbench-main {
    padding: 8px 14px 48px;
  }
}
</style>
