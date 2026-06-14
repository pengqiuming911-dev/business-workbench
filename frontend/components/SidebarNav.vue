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
        <template v-for="item in navItems" :key="item.path || item.title">
          <!-- Regular nav item -->
          <RouterLink
            v-if="!item.children"
            :to="item.path"
            class="nav-item"
            :class="{ active: currentPath === item.path }"
            @click="emit('navigate')"
          >
            <component :is="item.icon" :size="20" :stroke-width="2" />
            <span v-if="!collapsed">{{ item.title }}</span>
          </RouterLink>

          <!-- Nav group with children -->
          <template v-else>
            <button
              class="nav-item nav-group-toggle"
              :class="{ active: isGroupActive(item), open: rebateOpen }"
              @click="collapsed ? null : (rebateOpen = !rebateOpen)"
            >
              <component :is="item.icon" :size="20" :stroke-width="2" />
              <span v-if="!collapsed">{{ item.title }}</span>
              <ChevronDown v-if="!collapsed" :size="14" :stroke-width="2" class="chevron" />
            </button>
            <div v-if="!collapsed && rebateOpen" class="nav-children">
              <RouterLink
                v-for="child in item.children"
                :key="child.path"
                :to="child.path"
                class="nav-item nav-child"
                :class="{ active: currentPath === child.path }"
                @click="emit('navigate')"
              >
                <span>{{ child.title }}</span>
              </RouterLink>
            </div>
          </template>
        </template>
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
import { ref, computed, watch } from 'vue'
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
  ChevronDown,
  Receipt,
} from '@lucide/vue'
import logoImg from '../assets/business-workbench-logo.jpg'

defineProps({
  collapsed: { type: Boolean, default: false },
  overlayOpen: { type: Boolean, default: false },
})

const emit = defineEmits(['navigate', 'close', 'collapse'])

const route = useRoute()
const currentPath = computed(() => route.path)

const rebateOpen = ref(false)

// Auto-expand rebate group when on a rebate page
watch(currentPath, (p) => {
  if (p.startsWith('/rebate')) rebateOpen.value = true
}, { immediate: true })

function isGroupActive(item) {
  return item.children?.some(c => currentPath.value === c.path)
}

const navItems = [
  { path: '/', title: '总览', icon: LayoutDashboard },
  { path: '/data-preparation', title: '简报', icon: FileSpreadsheet },
  { path: '/holding-analysis', title: '持有产品分析', icon: Activity },
  { path: '/product-report', title: '报告', icon: FileText },
  { path: '/push-settings', title: '投递', icon: Send },
  { path: '/channel-analysis', title: '分析', icon: BarChart3 },
  {
    title: '返费',
    icon: Receipt,
    children: [
      { path: '/rebate-pending', title: '待返费分析' },
      { path: '/rebate-completed', title: '已返费分析' },
    ],
  },
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

.nav-group-toggle {
  width: 100%;
  background: none;
  border: none;
  cursor: pointer;
  position: relative;
}

.nav-group-toggle .chevron {
  margin-left: auto;
  transition: transform 180ms ease;
  color: var(--ink-faint);
}

.nav-group-toggle.open .chevron {
  transform: rotate(180deg);
}

.nav-children {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.nav-child {
  padding-left: 44px !important;
  font-size: 15px !important;
  height: 38px !important;
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
