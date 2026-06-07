<template>
  <aside class="sidebar" :class="{ collapsed: collapsed, overlay: overlayOpen }">
    <div class="sidebar-brand">
      <RouterLink to="/" class="brand-link" @click="emit('navigate')">
        <span class="brand-mark">BW</span>
        <span v-if="!collapsed" class="brand-text">
          <strong>业务工作台</strong>
          <em>航班服务数据分析平台</em>
        </span>
      </RouterLink>
    </div>

    <nav class="sidebar-nav">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="sidebar-link"
        :class="{ active: currentPath === item.path }"
        @click="emit('navigate')"
      >
        <component :is="item.icon" :size="18" :stroke-width="1.8" />
        <span v-if="!collapsed" class="sidebar-link-text">
          <strong>{{ item.title }}</strong>
          <em>{{ item.desc }}</em>
        </span>
      </RouterLink>
    </nav>

    <div class="sidebar-footer">
      <div class="sidebar-divider"></div>
      <RouterLink
        to="/activity-log"
        class="sidebar-link"
        :class="{ active: currentPath === '/activity-log' }"
        @click="emit('navigate')"
      >
        <ScrollText :size="18" :stroke-width="1.8" />
        <span v-if="!collapsed" class="sidebar-link-text">
          <strong>操作日志</strong>
        </span>
      </RouterLink>
    </div>
  </aside>

  <div v-if="overlayOpen" class="sidebar-backdrop" @click="emit('close')"></div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import {
  LayoutDashboard, Database, UserRound, UserX,
  FileText, Eye, BarChart3, PieChart, Users, ScrollText
} from '@lucide/vue'

const props = defineProps({
  collapsed: { type: Boolean, default: false },
  overlayOpen: { type: Boolean, default: false },
})

const emit = defineEmits(['navigate', 'close'])

const route = useRoute()
const currentPath = computed(() => route.path)

const navItems = [
  { path: '/', title: 'Dashboard', desc: '数据总览', icon: LayoutDashboard },
  { path: '/data-preparation', title: '数据准备', desc: '同步飞书数据', icon: Database },
  { path: '/user-profile', title: '用户画像', desc: '查询用户特征', icon: UserRound },
  { path: '/customer-churn', title: '客户流失', desc: '未复购客户', icon: UserX },
  { path: '/product-report', title: '产品报告', desc: '运行材料', icon: FileText },
  { path: '/product-completion', title: '派息/敲出观察', desc: '跟踪观察日', icon: Eye },
  { path: '/ongoing-product', title: '存续分析', desc: '持有产品', icon: BarChart3 },
  { path: '/channel-analysis', title: '渠道分析', desc: '渠道表现', icon: PieChart },
  { path: '/nominal-buyer', title: '名义购买人', desc: '管理人匹配', icon: Users },
]
</script>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 240px;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-soft);
  display: flex;
  flex-direction: column;
  z-index: 100;
  transition: transform 250ms ease, width 250ms ease;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-brand {
  padding: 16px;
  border-bottom: 1px solid var(--border-soft);
}

.brand-link {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  background: var(--brand);
  color: #fff;
  font-size: 13px;
  font-weight: 800;
  flex-shrink: 0;
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  overflow: hidden;
}

.brand-text strong {
  font-size: 14px;
  font-weight: 700;
  color: var(--ink-strong);
}

.brand-text em {
  font-size: 11px;
  font-style: normal;
  color: var(--ink-soft);
}

.sidebar-nav {
  flex: 1;
  padding: 8px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  color: var(--ink);
  transition: all 150ms ease-out;
  position: relative;
  text-decoration: none;
}

.sidebar-link:hover {
  background: var(--bg-hover);
}

.sidebar-link.active {
  background: var(--bg-active);
  color: var(--brand);
}

.sidebar-link.active::before {
  content: '';
  position: absolute;
  left: -8px;
  top: 6px;
  bottom: 6px;
  width: 3px;
  border-radius: 2px;
  background: var(--brand);
}

.sidebar-link-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  overflow: hidden;
}

.sidebar-link-text strong {
  font-size: 13px;
  font-weight: 600;
}

.sidebar-link-text em {
  font-size: 11px;
  font-style: normal;
  color: var(--ink-soft);
}

.sidebar-footer {
  padding: 8px;
  border-top: 1px solid var(--border-soft);
}

.sidebar-divider {
  height: 1px;
  margin-bottom: 6px;
}

.sidebar-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 99;
}

@media (max-width: 720px) {
  .sidebar {
    transform: translateX(-100%);
    width: 240px;
  }
  .sidebar.overlay {
    transform: translateX(0);
  }
  .sidebar.collapsed {
    width: 240px;
  }
}
</style>