<template>
  <aside class="sidebar" :class="{ collapsed, overlay: overlayOpen }">
    <div class="sidebar-inner">
      <div class="sidebar-brand">
        <RouterLink to="/" class="brand-link" @click="emit('navigate')">
          <span class="brand-mark">
            <img src="../assets/business-workbench-logo.jpg" alt="业务工作台 Logo" class="brand-logo" />
          </span>
          <span v-if="!collapsed" class="brand-name">业务工作台</span>
        </RouterLink>
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
import { RouterLink, useRoute } from 'vue-router'
import {
  Award,
  BarChart3,
  Bot,
  Database,
  FileText,
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
  { path: '/holding-analysis', title: '持有分析', icon: BarChart3 },
  { path: '/push-settings', title: '推送设置', icon: Send },
  { path: '/agent', title: '智能助手', icon: Bot },
]
</script>

<style scoped>
.sidebar {
  position: fixed;
  inset: 14px auto 14px 14px;
  width: 236px;
  z-index: 100;
  transition: transform 280ms cubic-bezier(0.4, 0, 0.2, 1), width 280ms cubic-bezier(0.4, 0, 0.2, 1);
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
  border-radius: var(--radius-lg);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.04), 0 8px 24px rgba(15, 23, 42, 0.04);
}

.sidebar-brand {
  min-height: 72px;
  display: flex;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-warm);
}

.brand-link {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-mark {
  width: 38px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(17, 24, 39, 0.12);
}

.brand-logo {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.brand-name {
  color: var(--ink-strong);
  font-size: 17px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: -0.02em;
}

.account-avatar {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  border-radius: 999px;
  color: #fff;
  background: linear-gradient(135deg, #ff8a00, #ff6b00);
  font-weight: 800;
  font-size: 15px;
  line-height: 1;
  box-shadow: 0 2px 8px rgba(255, 107, 0, 0.25);
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  padding: 10px 10px 14px;
}

.sidebar-link {
  min-height: 44px;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--nav-muted);
  text-decoration: none;
  border-radius: 10px;
  font-weight: 600;
  padding: 0 12px;
  letter-spacing: 0.005em;
  transition: color 180ms ease, background 180ms ease, box-shadow 180ms ease;
}

.sidebar-link:hover {
  color: var(--ink-strong);
  background: var(--bg-hover);
}

.sidebar-link.active {
  color: var(--brand);
  background: var(--bg-active);
  box-shadow: inset 0 0 0 1px rgba(37, 99, 235, 0.08);
}

.sidebar-link.active svg:first-child {
  color: var(--brand);
  fill: currentColor;
  stroke-width: 1.8;
}

.sidebar-link-text {
  flex: 1;
  min-width: 0;
  font-size: 13.5px;
}

.sidebar-footer {
  padding: 10px;
  border-top: 1px solid var(--border-warm);
}

.account-card {
  min-height: 50px;
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
  padding: 0 4px;
}

.account-card strong {
  flex: 1;
  min-width: 0;
  color: var(--ink);
  font-size: 13.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-exit {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  color: var(--ink-soft);
  background: transparent;
  transition: background 180ms ease, border-color 180ms ease, color 180ms ease;
}

.account-exit:hover {
  background: var(--bg-hover);
  border-color: var(--border);
  color: var(--ink);
}

.sidebar.collapsed .sidebar-brand,
.sidebar.collapsed .sidebar-nav,
.sidebar.collapsed .sidebar-footer {
  padding-left: 8px;
  padding-right: 8px;
}

.sidebar.collapsed .brand-name,
.sidebar.collapsed .account-card {
  display: none;
}

.sidebar.collapsed .brand-mark {
  width: 38px;
  height: 38px;
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
  background: rgba(15, 23, 42, 0.4);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
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
