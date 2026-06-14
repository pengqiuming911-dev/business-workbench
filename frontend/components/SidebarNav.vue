<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="sidebar-inner">
      <div class="sidebar-brand">
        <RouterLink to="/" class="brand-link">
          <img :src="logoImg" alt="衍选" class="brand-logo" />
          <span v-if="!collapsed" class="brand-text">衍选</span>
        </RouterLink>
        <button v-if="!collapsed" class="collapse-btn" @click="emit('collapse')">
          <ChevronsLeft :size="18" :stroke-width="2" />
        </button>
      </div>

      <nav class="sidebar-nav">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: currentPath === item.path }"
          @click="emit('navigate')"
        >
          <component :is="item.icon" :size="20" :stroke-width="2" />
          <span v-if="!collapsed">{{ item.title }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-bottom">
        <RouterLink to="/activity-log" class="nav-item" :class="{ active: currentPath === '/activity-log' }" @click="emit('navigate')">
          <Clock :size="20" :stroke-width="2" />
          <span v-if="!collapsed">日志</span>
        </RouterLink>
        <RouterLink to="/user-profile" class="nav-item" :class="{ active: currentPath === '/user-profile' }" @click="emit('navigate')">
          <User :size="20" :stroke-width="2" />
          <span v-if="!collapsed">账户</span>
        </RouterLink>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  LayoutDashboard,
  FileSpreadsheet,
  Activity,
  FileText,
  Send,
  BarChart3,
  Clock,
  User,
  ChevronsLeft,
} from '@lucide/vue'
import logoImg from '../assets/business-workbench-logo.jpg'

defineProps({
  collapsed: { type: Boolean, default: false },
  overlayOpen: { type: Boolean, default: false },
})

const emit = defineEmits(['navigate', 'close', 'collapse'])

const route = useRoute()
const currentPath = computed(() => route.path)

const navItems = [
  { path: '/', title: '总览', icon: LayoutDashboard },
  { path: '/data-preparation', title: '简报', icon: FileSpreadsheet },
  { path: '/holding-analysis', title: '监控', icon: Activity },
  { path: '/product-report', title: '报告', icon: FileText },
  { path: '/push-settings', title: '投递', icon: Send },
  { path: '/channel-analysis', title: '分析', icon: BarChart3 },
]
</script>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 230px;
  z-index: 100;
  background: var(--bg-card);
  border-right: 1px solid var(--border-soft);
  transition: width 220ms ease;
  overflow: hidden;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px 12px;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 4px 20px;
}

.brand-link {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
}

.brand-logo {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  object-fit: cover;
}

.brand-text {
  font-size: 19px;
  font-weight: 700;
  color: var(--ink-strong);
  letter-spacing: -0.01em;
}

.collapse-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--ink-faint);
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.collapse-btn:hover {
  background: var(--border-soft);
  color: var(--ink);
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 44px;
  padding: 0 12px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 600;
  color: var(--ink-soft);
  text-decoration: none;
  transition: background 150ms ease, color 150ms ease;
  white-space: nowrap;
}

.nav-item:hover {
  background: var(--bg-hover, #f3f4f6);
  color: var(--ink-strong);
}

.nav-item.active {
  background: var(--brand-soft);
  color: var(--brand);
}

.sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: 2px;
  border-top: 1px solid var(--border-soft);
  padding-top: 12px;
  margin-top: 8px;
}

@media (max-width: 860px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .sidebar.collapsed {
    width: 230px;
  }
}
</style>
