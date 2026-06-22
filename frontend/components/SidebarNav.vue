<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="sidebar-inner">
      <div class="sidebar-brand">
        <RouterLink to="/" class="brand-link">
          <img :src="logoImg" alt="衍选运营平台" class="brand-logo" />
          <span v-if="!collapsed" class="brand-text">衍选运营平台</span>
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
          class="account-card"
          :class="{ active: currentPath === '/user-profile', collapsed }"
          @click="emit('navigate')"
        >
          <img
            v-if="authUser?.avatar_url"
            :src="authUser.avatar_url"
            :alt="displayName"
            class="account-avatar"
          />
          <div v-else class="account-avatar account-avatar-fallback">
            <User :size="16" :stroke-width="2" />
          </div>
          <div v-if="!collapsed" class="account-copy">
            <span class="account-name">{{ displayName }}</span>
            <span class="account-meta">{{ displayMeta }}</span>
          </div>
        </RouterLink>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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

const authUser = ref(null)
const authorized = ref(false)

const displayName = computed(() => {
  if (!authorized.value) return '飞书未连接'
  return authUser.value?.name || authUser.value?.en_name || '当前账号'
})

const displayMeta = computed(() => {
  if (!authorized.value) return '点击连接飞书账号'
  return authUser.value?.email || authUser.value?.open_id || '查看当前登录信息'
})

const navItems = [
  { path: '/', title: '总览', icon: LayoutDashboard },
  { path: '/data-preparation', title: '数据准备', icon: FileSpreadsheet },
  { path: '/holding-analysis', title: '产品&持仓', icon: Activity },
  { path: '/rebate-analysis', title: '返费', icon: Receipt },
  { path: '/product-report', title: '销售物料', icon: FileText },
  { path: '/product-completion', title: '派息/敲出', icon: CalendarDays },
  { path: '/push-settings', title: '推送设置', icon: Send },
]

onMounted(() => {
  loadAuthUser()
  window.addEventListener('auth-status-changed', loadAuthUser)
})

onBeforeUnmount(() => {
  window.removeEventListener('auth-status-changed', loadAuthUser)
})

async function loadAuthUser() {
  try {
    const res = await fetch('/api/auth/status')
    const data = await res.json()
    authorized.value = Boolean(data.authorized)
    authUser.value = data.user || null
  } catch {
    authorized.value = false
    authUser.value = null
  }
}
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
  gap: 8px;
  border-top: 1px solid var(--border-soft);
  padding-top: 12px;
  margin-top: 8px;
}

.account-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  min-height: 56px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(148, 163, 184, 0.16);
  box-shadow: var(--shadow-sm);
  color: inherit;
  text-decoration: none;
  transition: background 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
}

.account-card:hover {
  background: rgba(255, 255, 255, 0.92);
  border-color: rgba(31, 58, 138, 0.16);
}

.account-card.active {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(230, 238, 255, 0.9));
  border-color: rgba(31, 58, 138, 0.18);
}

.account-card.collapsed {
  justify-content: center;
  padding: 8px 0;
}

.account-avatar {
  width: 34px;
  height: 34px;
  border-radius: 12px;
  object-fit: cover;
  flex: 0 0 auto;
}

.account-avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(31, 58, 138, 0.16), rgba(44, 92, 224, 0.12));
  color: var(--brand);
}

.account-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.account-name,
.account-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--ink-strong);
}

.account-meta {
  font-size: 11px;
  color: var(--ink-soft);
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
