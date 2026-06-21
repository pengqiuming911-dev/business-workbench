<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="sidebar-inner">
      <div class="sidebar-brand">
        <RouterLink to="/" class="brand-link">
          <img :src="logoImg" alt="业务工作台" class="brand-logo" />
          <span v-if="!collapsed" class="brand-text">业务工作台</span>
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
        <RouterLink
          to="/activity-log"
          class="nav-item"
          :class="{ active: currentPath === '/activity-log' }"
          @click="emit('navigate')"
        >
          <Clock :size="20" :stroke-width="2" />
          <span v-if="!collapsed">日志</span>
        </RouterLink>
        <RouterLink
          to="/user-profile"
          class="nav-item"
          :class="{ active: currentPath === '/user-profile' }"
          @click="emit('navigate')"
        >
          <User :size="20" :stroke-width="2" />
          <span v-if="!collapsed">客户画像</span>
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
  Clock,
  User,
  ChevronsLeft,
  Receipt,
  CalendarDays,
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
  { path: '/data-preparation', title: '数据准备', icon: FileSpreadsheet },
  { path: '/holding-analysis', title: '产品&持仓', icon: Activity },
  { path: '/rebate-analysis', title: '返费', icon: Receipt },
  { path: '/product-report', title: '销售物料', icon: FileText },
  { path: '/product-completion', title: '派息/敲出', icon: CalendarDays },
  { path: '/push-settings', title: '推送设置', icon: Send },
]
</script>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 208px;
  z-index: 100;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-soft);
  transition: width 220ms ease;
  overflow: hidden;
  box-shadow: inset -1px 0 0 rgba(255, 255, 255, 0.72);
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 18px 12px 14px;
  overflow-y: auto;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 6px 14px;
  margin-bottom: 8px;
  border-bottom: 1px solid rgba(226, 232, 240, 0.9);
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
  border-radius: 10px;
  object-fit: cover;
  box-shadow: var(--shadow-sm);
}

.brand-text {
  font-size: 16px;
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
  border-radius: 8px;
  background: transparent;
  color: var(--ink-faint);
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.8);
  color: var(--ink);
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 42px;
  padding: 0 12px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--ink-soft);
  text-decoration: none;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
  white-space: nowrap;
  border: 1px solid transparent;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.72);
  color: var(--ink-strong);
}

.nav-item.active {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), var(--brand-soft));
  color: var(--brand);
  border-color: rgba(31, 58, 138, 0.12);
  font-weight: 700;
  box-shadow: var(--shadow-sm);
}

.sidebar-bottom {
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-top: 1px solid var(--border-soft);
  padding-top: 12px;
  margin-top: 8px;
}

@media (max-width: 860px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .sidebar.collapsed {
    width: 208px;
  }
}
</style>
