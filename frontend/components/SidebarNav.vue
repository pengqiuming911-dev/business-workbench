<template>
  <aside class="sidebar" :class="{ collapsed: collapsed, overlay: overlayOpen }">
    <div class="sidebar-inner">
      <div class="sidebar-brand">
        <RouterLink to="/" class="brand-link" @click="emit('navigate')">
          <span class="brand-mark">
            <GitBranch :size="31" :stroke-width="3" />
          </span>
          <span v-if="!collapsed" class="brand-name">业务工作台</span>
        </RouterLink>
      </div>

      <div v-if="!collapsed" class="workspace-select">
        <span class="workspace-avatar">业</span>
        <span class="workspace-name">航班服务业务空间</span>
        <ChevronDown :size="16" />
      </div>

      <nav class="sidebar-nav" aria-label="主导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="sidebar-link"
          :class="{ active: currentPath === item.path }"
          @click="emit('navigate')"
        >
          <component :is="item.icon" :size="21" :stroke-width="2.2" />
          <span v-if="!collapsed" class="sidebar-link-text">{{ item.title }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <RouterLink
          to="/activity-log"
          class="sidebar-link"
          :class="{ active: currentPath === '/activity-log' }"
          @click="emit('navigate')"
        >
          <ScrollText :size="21" :stroke-width="2.1" />
          <span v-if="!collapsed" class="sidebar-link-text">操作日志</span>
        </RouterLink>

        <div v-if="!collapsed" class="account-card">
          <span class="account-avatar">q</span>
          <strong>qiuming peng</strong>
          <button class="account-exit" type="button" aria-label="退出登录">
            <LogOut :size="19" :stroke-width="2" />
          </button>
        </div>
      </div>
    </div>
  </aside>

  <div v-if="overlayOpen" class="sidebar-backdrop" @click="emit('close')"></div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import {
  Award,
  BarChart3,
  ChevronDown,
  Database,
  FileText,
  GitBranch,
  Home,
  LogOut,
  ScrollText,
  Send,
} from '@lucide/vue'

defineProps({
  collapsed: { type: Boolean, default: false },
  overlayOpen: { type: Boolean, default: false },
})

const emit = defineEmits(['navigate', 'close'])

const route = useRoute()
const currentPath = computed(() => route.path)

const navItems = [
  { path: '/', title: '业务总览', icon: Home },
  { path: '/data-preparation', title: '数据准备', icon: Database },
  { path: '/product-completion', title: '观察日历', icon: Award },
  { path: '/product-report', title: '产品报告', icon: FileText },
  { path: '/ongoing-product', title: '存续分析', icon: BarChart3 },
  { path: '/push-settings', title: '推送设置', icon: Send },
]
</script>

<style scoped>
.sidebar {
  position: fixed;
  inset: 16px auto 16px 16px;
  width: 232px;
  z-index: 100;
  transition: transform 220ms ease, width 220ms ease;
}

.sidebar.collapsed {
  width: 72px;
}

.sidebar-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-sidebar);
  border: 1px solid var(--border-warm);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.sidebar-brand {
  min-height: 76px;
  display: flex;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--border-warm);
}

.brand-link {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.brand-mark {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  color: #fff;
  background: var(--ink-strong);
  border-radius: 8px;
}

.brand-name {
  color: var(--ink-strong);
  font-size: 18px;
  font-weight: 760;
  line-height: 1;
  letter-spacing: 0;
}

.workspace-select {
  min-height: 46px;
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 12px;
  padding: 0 12px;
  color: var(--brand);
  border: 1px solid rgba(38, 119, 255, 0.24);
  border-radius: var(--radius);
  background: var(--brand-soft);
}

.workspace-avatar,
.account-avatar {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border-radius: 999px;
  color: #fff;
  background: #ff7900;
  font-weight: 800;
  font-size: 16px;
  line-height: 1;
}

.workspace-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 680;
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  padding: 8px 10px 12px;
  border-top: 1px solid var(--border-warm);
}

.sidebar-link {
  min-height: 42px;
  display: flex;
  align-items: center;
  gap: 11px;
  color: var(--nav-muted);
  text-decoration: none;
  border-radius: var(--radius);
  font-weight: 680;
  padding: 0 10px;
  transition: color 150ms ease, background 150ms ease;
}

.sidebar-link:hover {
  color: var(--ink-strong);
  background: var(--bg-hover);
}

.sidebar-link.active {
  color: var(--brand);
  background: var(--bg-active);
}

.sidebar-link.active svg:first-child {
  color: var(--brand);
  fill: currentColor;
  stroke-width: 1.8;
}

.sidebar-link-text {
  flex: 1;
  min-width: 0;
  font-size: 14px;
}

.sidebar-footer {
  padding: 10px;
  border-top: 1px solid var(--border-warm);
}

.account-card {
  min-height: 52px;
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
}

.account-card strong {
  flex: 1;
  min-width: 0;
  color: var(--ink);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-exit {
  width: 38px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #e0e3e7;
  border-radius: var(--radius);
  color: #2c3038;
  background: transparent;
}

.sidebar.collapsed .sidebar-brand,
.sidebar.collapsed .sidebar-nav,
.sidebar.collapsed .sidebar-footer {
  padding-left: 8px;
  padding-right: 8px;
}

.sidebar.collapsed .brand-name,
.sidebar.collapsed .workspace-select,
.sidebar.collapsed .account-card {
  display: none;
}

.sidebar.collapsed .brand-mark {
  width: 40px;
  height: 40px;
}

.sidebar.collapsed .sidebar-link {
  justify-content: center;
  gap: 0;
  padding: 0;
}

.sidebar-backdrop {
  position: fixed;
  inset: 0;
  z-index: 99;
  background: rgba(24, 24, 20, 0.35);
}

@media (max-width: 860px) {
  .sidebar {
    inset: 10px auto 10px 10px;
    width: min(292px, calc(100vw - 20px));
    transform: translateX(calc(-100% - 18px));
  }

  .sidebar.overlay {
    transform: translateX(0);
  }

  .sidebar.collapsed {
    width: min(292px, calc(100vw - 20px));
  }

  .sidebar.collapsed .brand-name,
  .sidebar.collapsed .workspace-select,
  .sidebar.collapsed .account-card {
    display: flex;
  }

  .sidebar.collapsed .sidebar-link {
    justify-content: flex-start;
    gap: 16px;
    padding: 0 10px;
  }
}
</style>
