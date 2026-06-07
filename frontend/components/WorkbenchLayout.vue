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

        <div class="topbar-actions">
          <span class="status-pill">
            <span class="status-dot"></span>
            运行正常
          </span>
          <div class="social-pill" aria-label="外部链接">
            <GitFork :size="18" :stroke-width="2.1" />
            <Send :size="18" :stroke-width="2.1" />
            <Mail :size="18" :stroke-width="2.1" />
          </div>
          <button class="moon-pill" type="button" aria-label="切换深色模式">
            <Moon :size="21" :stroke-width="2" />
          </button>
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
import {
  GitFork,
  Mail,
  Moon,
  PanelLeftClose,
  PanelLeftOpen,
  Send,
} from '@lucide/vue'
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
  margin-left: 280px;
  transition: margin-left 220ms ease;
}

.workbench-content.expanded {
  margin-left: 96px;
}

.workbench-topbar {
  position: sticky;
  top: 0;
  z-index: 60;
  min-height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 16px 32px 8px;
  pointer-events: none;
}

.sidebar-toggle,
.status-pill,
.social-pill,
.moon-pill {
  pointer-events: auto;
}

.sidebar-toggle {
  width: 50px;
  height: 50px;
  display: none;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  color: #6a758b;
  background: #fff;
  box-shadow: var(--shadow-sm);
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}

.status-pill,
.social-pill,
.moon-pill {
  min-height: 42px;
  display: inline-flex;
  align-items: center;
  color: var(--ink);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: #fff;
  box-shadow: var(--shadow-sm);
}

.status-pill {
  gap: 10px;
  padding: 0 14px;
  font-size: 14px;
  font-weight: 700;
}

.status-dot {
  width: 10px;
  height: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: #00b881;
}

.social-pill {
  gap: 16px;
  padding: 0 14px;
  color: #68748a;
}

.moon-pill {
  width: 42px;
  justify-content: center;
  border: 0;
  color: #657187;
}

.workbench-main {
  width: min(1240px, calc(100vw - 344px));
  margin: 0 auto 0 32px;
  padding: 0 0 72px;
}

@media (max-width: 1280px) {
  .workbench-content {
    margin-left: 268px;
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
    min-height: 72px;
    padding: 16px 14px 0;
  }

  .topbar-actions {
    gap: 8px;
  }

  .status-pill {
    min-height: 48px;
    padding: 0 14px;
    font-size: 14px;
  }

  .social-pill {
    display: none;
  }

  .moon-pill,
  .sidebar-toggle {
    width: 48px;
    min-height: 48px;
    height: 48px;
  }

  .sidebar-toggle {
    display: inline-flex;
  }

  .workbench-main {
    width: calc(100% - 28px);
    margin: 0 auto;
    padding-bottom: 36px;
  }

}
</style>
