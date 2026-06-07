# Tavily Full Frontend Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fully redesign the existing Vue 3 frontend into a professional SaaS console inspired by Tavily's app.tavily.com/home, adding a data dashboard, global search (Cmd+K), and activity log.

**Architecture:** Fixed left sidebar (240px) + compact top bar replacing the current hamburger menu layout. New Dashboard home page with stat cards, ECharts charts, observation feed, and sync status. Global search modal triggered by Cmd+K. Activity log page with backend SQLite table. All 8 existing business pages restyled with new component library while preserving existing API calls and business logic.

**Tech Stack:** Vue 3 + Vite + vanilla CSS + ECharts + lucide-vue-next. Backend: Express + sql.js (SQLite) with 4 new API endpoints.

---

### Task 1: Install Dependencies & Create Design Tokens

**Files:**
- Modify: `frontend/package.json` (install echarts, lucide-vue-next)
- Create: `frontend/assets/main.css` (new design token file)

- [ ] **Step 1: Install npm dependencies**

Run in `frontend/`:
```bash
npm install echarts lucide-vue-next
```

Expected: `echarts` and `lucide-vue-next` added to `package.json` dependencies.

- [ ] **Step 2: Create design token file `frontend/assets/main.css`**

```css
:root {
  --bg-page: #fefcf5;
  --bg-sidebar: #ffffff;
  --bg-card: #ffffff;
  --bg-hover: #f5f3ec;
  --bg-active: #edeae0;
  --brand: #2677ff;
  --brand-soft: #e9efff;
  --brand-hover: #1a63e8;
  --accent: #6c5ce7;
  --ink-strong: #1a1919;
  --ink: #383636;
  --ink-soft: #718096;
  --ink-faint: #a0aec0;
  --border: #e2e8f0;
  --border-soft: #edf2f7;
  --success: #01b574;
  --success-soft: #e6faf5;
  --warning: #ffb547;
  --warning-soft: #fff6da;
  --danger: #ee5d50;
  --danger-soft: #feefee;
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
  --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1);
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
  --radius-sm: 6px;
  --radius: 10px;
  --radius-lg: 14px;
}

*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html {
  font-family: var(--font-sans);
  font-size: 14px;
  color: var(--ink);
  background: var(--bg-page);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

body {
  min-height: 100vh;
}

a {
  color: inherit;
  text-decoration: none;
}

button {
  cursor: pointer;
  font-family: inherit;
}

input, select, textarea {
  font-family: inherit;
  font-size: inherit;
}

/* ── Typography ── */
.text-page-title { font-size: 28px; font-weight: 800; color: var(--ink-strong); line-height: 1.2; }
.text-section    { font-size: 18px; font-weight: 700; color: var(--ink-strong); }
.text-card-title { font-size: 15px; font-weight: 600; color: var(--ink-strong); }
.text-body       { font-size: 14px; font-weight: 400; color: var(--ink); line-height: 1.6; }
.text-label      { font-size: 12px; font-weight: 500; color: var(--ink-soft); }
.text-mono       { font-family: var(--font-mono); }

/* ── Buttons ── */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 36px;
  padding: 0 16px;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 600;
  border: 1px solid transparent;
  transition: all 150ms ease-out;
  line-height: 1;
}

.btn-primary {
  background: var(--brand);
  color: #fff;
  border-color: var(--brand);
}
.btn-primary:hover { background: var(--brand-hover); border-color: var(--brand-hover); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  background: #fff;
  color: var(--ink);
  border-color: var(--border);
}
.btn-secondary:hover { border-color: var(--brand); color: var(--brand); background: var(--brand-soft); }

.btn-outline {
  background: transparent;
  color: var(--ink);
  border-color: var(--border);
}
.btn-outline:hover { border-color: var(--brand); color: var(--brand); }

.btn-sm { height: 30px; padding: 0 10px; font-size: 12px; }
.btn-lg { height: 42px; padding: 0 24px; font-size: 15px; }

/* ── Inputs ── */
.input {
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: #fff;
  color: var(--ink);
  font-size: 13px;
  outline: none;
  transition: border-color 150ms, box-shadow 150ms;
}
.input:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.textarea {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: #fff;
  color: var(--ink);
  font-size: 13px;
  outline: none;
  resize: vertical;
  transition: border-color 150ms, box-shadow 150ms;
}
.textarea:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

select.input {
  cursor: pointer;
  appearance: auto;
}

/* ── Panels ── */
.panel-card {
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  padding: 20px 24px;
  transition: box-shadow 200ms;
}
.panel-card:hover { box-shadow: var(--shadow-md); }

.panel-card-header {
  font-size: 15px;
  font-weight: 600;
  color: var(--ink-strong);
  margin-bottom: 16px;
}

/* ── Tables ── */
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.data-table th {
  text-align: left;
  padding: 10px 12px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--ink-soft);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
.data-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-soft);
  color: var(--ink);
}
.data-table tr:hover td { background: var(--bg-hover); }

.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}

/* ── Badges ── */
.badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 700;
  line-height: 1.4;
}
.badge-green  { background: var(--success-soft); color: var(--success); }
.badge-red    { background: var(--danger-soft); color: var(--danger); }
.badge-amber  { background: var(--warning-soft); color: #92400e; }
.badge-blue   { background: var(--brand-soft); color: var(--brand); }

/* ── Form rows ── */
.form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.form-row label {
  font-size: 13px;
  font-weight: 500;
  color: var(--ink-soft);
  white-space: nowrap;
  min-width: 80px;
}

/* ── Status messages ── */
.error-msg {
  color: var(--danger);
  font-size: 13px;
  padding: 8px 12px;
  background: var(--danger-soft);
  border-radius: var(--radius-sm);
}
.success-msg {
  color: var(--success);
  font-size: 13px;
  padding: 8px 12px;
  background: var(--success-soft);
  border-radius: var(--radius-sm);
}

/* ── Empty / Loading ── */
.empty-state {
  text-align: center;
  padding: 48px 24px;
  color: var(--ink-soft);
  font-size: 14px;
}
.loading-state {
  text-align: center;
  padding: 48px 24px;
  color: var(--ink-soft);
  font-size: 14px;
}

/* ── Transitions ── */
.fade-enter-active, .fade-leave-active { transition: opacity 200ms ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
```

- [ ] **Step 3: Update `frontend/main.js` to import `main.css` instead of old `global.css`**

Change the import in `frontend/main.js`:
```js
import './assets/main.css'
```

Remove the old `import './assets/global.css'` line. The old `global.css` file will be deleted later after migration is complete.

- [ ] **Step 4: Verify build passes**

```bash
cd frontend && npm run build
```

Expected: Build succeeds with no errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/assets/main.css frontend/main.js
git commit -m "chore: install echarts + lucide, add Tavily design tokens"
```

---

### Task 2: Rewrite Application Shell (WorkbenchLayout + SidebarNav)

**Files:**
- Rewrite: `frontend/components/WorkbenchLayout.vue`
- Create: `frontend/components/SidebarNav.vue`
- Modify: `frontend/router/index.js`

- [ ] **Step 1: Create `frontend/components/SidebarNav.vue`**

```vue
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
} from 'lucide-vue-next'

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
```

- [ ] **Step 2: Rewrite `frontend/components/WorkbenchLayout.vue`**

```vue
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
        <button
          class="sidebar-toggle"
          type="button"
          @click="toggleSidebar"
        >
          <Menu :size="20" />
        </button>

        <div class="topbar-search" @click="openSearch">
          <Search :size="16" />
          <span class="search-placeholder">搜索客户、产品、渠道...</span>
          <kbd class="search-kbd">{{ isMac ? '⌘' : 'Ctrl' }} K</kbd>
        </div>

        <div class="topbar-actions">
          <div class="topbar-avatar">
            <span class="avatar-circle">BW</span>
          </div>
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
import { Menu, Search } from 'lucide-vue-next'
import SidebarNav from './SidebarNav.vue'
import GlobalSearch from './GlobalSearch.vue'

defineProps({
  wide: { type: Boolean, default: false },
})

const sidebarCollapsed = ref(false)
const sidebarOverlay = ref(false)
const searchOpen = ref(false)
const isMac = ref(false)

function toggleSidebar() {
  if (window.innerWidth <= 720) {
    sidebarOverlay.value = !sidebarOverlay.value
  } else {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }
}

function closeSidebar() {
  sidebarOverlay.value = false
}

function openSearch() {
  searchOpen.value = true
}

function handleKeydown(e) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    searchOpen.value = !searchOpen.value
  }
}

onMounted(() => {
  isMac.value = navigator.platform.toUpperCase().indexOf('MAC') >= 0
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
  margin-left: 240px;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  transition: margin-left 250ms ease;
}

.workbench-content.expanded {
  margin-left: 64px;
}

.workbench-topbar {
  position: sticky;
  top: 0;
  z-index: 50;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 24px;
  background: rgba(254, 252, 245, 0.85);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--border-soft);
}

.sidebar-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink-soft);
  transition: all 150ms;
}
.sidebar-toggle:hover {
  background: var(--bg-hover);
  color: var(--ink-strong);
}

.topbar-search {
  flex: 1;
  max-width: 480px;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: #fff;
  color: var(--ink-faint);
  cursor: pointer;
  transition: border-color 150ms;
}
.topbar-search:hover {
  border-color: var(--brand);
}

.search-placeholder {
  flex: 1;
  font-size: 13px;
}

.search-kbd {
  font-family: var(--font-sans);
  font-size: 11px;
  font-weight: 600;
  padding: 2px 6px;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--ink-soft);
  background: var(--bg-hover);
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.avatar-circle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--brand-soft);
  color: var(--brand);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
}

.workbench-main {
  flex: 1;
  padding: 24px 28px;
  max-width: 1200px;
  width: 100%;
}

@media (max-width: 720px) {
  .workbench-content {
    margin-left: 0;
  }
  .workbench-content.expanded {
    margin-left: 0;
  }
  .topbar-search {
    max-width: 240px;
  }
}
</style>
```

- [ ] **Step 3: Update router to use Dashboard as root**

Modify `frontend/router/index.js`:
```js
import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'

const routes = [
  { path: '/', component: Dashboard },
  { path: '/data-preparation', component: () => import('../views/DataPreparation.vue') },
  { path: '/customer-churn', component: () => import('../views/CustomerChurn.vue') },
  { path: '/product-report', component: () => import('../views/ProductReport.vue') },
  { path: '/product-completion', component: () => import('../views/ProductCompletion.vue') },
  { path: '/ongoing-product', component: () => import('../views/OngoingProduct.vue') },
  { path: '/channel-analysis', component: () => import('../views/ChannelAnalysis.vue') },
  { path: '/nominal-buyer', component: () => import('../views/NominalBuyer.vue') },
  { path: '/user-profile', component: () => import('../views/UserProfile.vue') },
  { path: '/activity-log', component: () => import('../views/ActivityLog.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes
})
```

- [ ] **Step 4: Create a minimal `frontend/views/Dashboard.vue` stub so the app boots**

```vue
<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">Dashboard</h1>
    <p class="text-body" style="margin-top: 12px;">Loading...</p>
  </WorkbenchLayout>
</template>

<script setup>
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
</script>
```

- [ ] **Step 5: Create a minimal `frontend/views/ActivityLog.vue` stub so the route resolves**

```vue
<template>
  <WorkbenchLayout>
    <h1 class="text-page-title">操作日志</h1>
  </WorkbenchLayout>
</template>

<script setup>
import WorkbenchLayout from '../components/WorkbenchLayout.vue'
</script>
```

- [ ] **Step 6: Verify build passes**

```bash
cd frontend && npm run build
```

Expected: Build succeeds.

- [ ] **Step 7: Start dev server and verify layout in browser**

```bash
cd frontend && npm run dev
```

Check that:
- Sidebar is visible on left with all module links
- Top bar shows with search area and avatar
- Clicking Cmd+K (or search area) does nothing yet (GlobalSearch component stub)
- Clicking sidebar links navigates
- Mobile width collapses sidebar properly

- [ ] **Step 8: Commit**

```bash
git add frontend/components/WorkbenchLayout.vue frontend/components/SidebarNav.vue frontend/views/Dashboard.vue frontend/views/ActivityLog.vue frontend/router/index.js
git commit -m "feat: rewrite app shell with fixed sidebar + top bar"
```

---

### Task 3: Create GlobalSearch, StatCard & PanelCard Components

**Files:**
- Create: `frontend/components/GlobalSearch.vue`
- Create: `frontend/components/StatCard.vue`
- Create: `frontend/components/PanelCard.vue`

- [ ] **Step 1: Create `frontend/components/GlobalSearch.vue`**

A Teleported modal with search input, debounced API call to `/api/search?q=`, keyboard navigation (up/down/enter/escape), and router navigation on selection. Uses `Search` icon from lucide. Groups results by type (客户/产品/渠道). Shows "暂无结果" when empty. Stores recently-accessed pages in a simple `ref` array (5 max, persisted to localStorage).

Component props: `modelValue` (v-model:open Boolean). Template: overlay div + modal with input + results list. Script: `ref` for query, results, loading, activeIndex. `watch(query)` with 300ms debounce calling fetch. `moveDown/moveUp/selectCurrent` methods. `close()` emits `update:open false`.

Style: full-screen overlay `rgba(0,0,0,0.4)`, modal centered `max-width 520px`, white bg, `border-radius 14px`, search input full-width, results as scrollable list max-height 320px.

- [ ] **Step 2: Create `frontend/components/StatCard.vue`**

Props: `title` (String), `value` (String/Number), `trend` (Number, optional, positive=green up, negative=red down, null=show dash). Uses `TrendingUp`/`TrendingDown`/`Minus` from lucide. Renders a panel-card with large number (28px 800 weight), label (12px caption), and trend indicator.

- [ ] **Step 3: Create `frontend/components/PanelCard.vue`**

Simple wrapper component. Props: `title` (String, optional). Renders a `.panel-card` div with optional `.panel-card-header`. Default slot for content.

- [ ] **Step 4: Commit**

```bash
git add frontend/components/GlobalSearch.vue frontend/components/StatCard.vue frontend/components/PanelCard.vue
git commit -m "feat: add GlobalSearch, StatCard, PanelCard components"
```

---

### Task 4: Backend — Add Dashboard Stats, Charts, Search & Activity Log APIs

**Files:**
- Modify: `backend/db.js` (add `activity_logs` table + query functions)
- Modify: `backend/index.js` (add 4 new endpoints)

- [ ] **Step 1: Add `activity_logs` table creation to `backend/db.js` `initDatabase()`**

Add after existing table creations:
```sql
CREATE TABLE IF NOT EXISTS activity_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  action TEXT NOT NULL,
  detail TEXT,
  created_at TEXT DEFAULT (datetime('now'))
);
```

Add exported functions:
```js
function logActivity(type, action, detail) {
  db.run('INSERT INTO activity_logs (type, action, detail) VALUES (?, ?, ?)', [type, action, detail || null]);
}

function queryActivityLogs(type, limit) {
  let sql = 'SELECT * FROM activity_logs ORDER BY created_at DESC'
  const params = []
  if (type) { sql = 'SELECT * FROM activity_logs WHERE type = ? ORDER BY created_at DESC'; params.push(type) }
  sql += ' LIMIT ?'; params.push(limit || 50)
  return db.exec(sql, params)[0]?.values.map(r => ({ id: r[0], type: r[1], action: r[2], detail: r[3], createdAt: r[4] })) || []
}
```

Export both new functions.

- [ ] **Step 2: Add `GET /api/dashboard/stats` to `backend/index.js`**

After existing route definitions, add:
```js
app.get('/api/dashboard/stats', (req, res) => {
  const totalProducts = db.exec('SELECT COUNT(*) FROM products')[0]?.values[0][0] || 0
  const activeProducts = db.exec("SELECT COUNT(*) FROM products WHERE holding_status LIKE '%存续%' OR holding_status LIKE '%持有%'")[0]?.values[0][0] || 0
  const totalCustomers = db.exec('SELECT COUNT(DISTINCT customer_name) FROM customers')[0]?.values[0][0] || 0
  const totalChannels = db.exec('SELECT COUNT(DISTINCT channel_name) FROM channels')[0]?.values[0][0] || 0
  res.json({ totalProducts, activeProducts, totalCustomers, totalChannels })
})
```

- [ ] **Step 3: Add `GET /api/dashboard/charts` to `backend/index.js`**

```js
app.get('/api/dashboard/charts', (req, res) => {
  // Monthly trend: aggregate transactions by month
  const trendRows = db.exec(`
    SELECT strftime('%Y-%m', transaction_date) as month,
           SUM(subscribe_amount) as amount,
           COUNT(*) as count
    FROM transactions
    GROUP BY month ORDER BY month
  `)[0]?.values || []
  const monthlyTrend = trendRows.map(r => ({ month: r[0], amount: r[1] || 0, count: r[2] }))

  // Channel distribution: top 8 by total amount
  const chanRows = db.exec(`
    SELECT c.channel_name, SUM(t.subscribe_amount) as amount
    FROM channels c
    JOIN transactions t ON t.counterparty = c.channel_name
    GROUP BY c.channel_name ORDER BY amount DESC LIMIT 8
  `)[0]?.values || []
  const channelDistribution = chanRows.map(r => ({ channel: r[0], amount: r[1] || 0 }))

  res.json({ monthlyTrend, channelDistribution })
})
```

- [ ] **Step 4: Add `GET /api/search` to `backend/index.js`**

```js
app.get('/api/search', (req, res) => {
  const q = (req.query.q || '').trim()
  if (!q) return res.json({ results: [] })
  const like = `%${q}%`
  const results = []

  const custs = db.exec('SELECT id, customer_name FROM customers WHERE customer_name LIKE ? LIMIT 5', [like])[0]?.values || []
  custs.forEach(r => results.push({ type: 'customer', id: r[0], name: r[1], path: '/user-profile' }))

  const prods = db.exec('SELECT id, name FROM products WHERE name LIKE ? LIMIT 5', [like])[0]?.values || []
  prods.forEach(r => results.push({ type: 'product', id: r[0], name: r[1], path: '/ongoing-product' }))

  const chans = db.exec('SELECT id, channel_name FROM channels WHERE channel_name LIKE ? LIMIT 5', [like])[0]?.values || []
  chans.forEach(r => results.push({ type: 'channel', id: r[0], name: r[1], path: '/channel-analysis' }))

  res.json({ results })
})
```

- [ ] **Step 5: Add `GET /api/activity-logs` to `backend/index.js`**

```js
app.get('/api/activity-logs', (req, res) => {
  const type = req.query.type || null
  const limit = parseInt(req.query.limit) || 50
  const logs = queryActivityLogs(type, limit)
  res.json({ logs })
})
```

- [ ] **Step 6: Instrument existing sync endpoints to write activity log entries**

In the `POST /api/db/sync` handler, after successful sync, add:
```js
logActivity('sync', 'Transaction table synced', `${rowCount} rows`)
```

In `POST /api/db/sync-coinvest`, after success:
```js
logActivity('sync', 'Co-invest users synced', `${rowCount} rows`)
```

In `POST /api/drive/sync-product-docs`, after success:
```js
logActivity('sync', 'Product docs synced', `${docCount} docs, ${folderCount} folders`)
```

- [ ] **Step 7: Verify backend starts without errors**

```bash
cd backend && npm start
```

Test with curl or browser:
- `GET http://localhost:3001/api/dashboard/stats` — returns JSON with counts
- `GET http://localhost:3001/api/dashboard/charts` — returns JSON with arrays
- `GET http://localhost:3001/api/search?q=test` — returns results array
- `GET http://localhost:3001/api/activity-logs` — returns logs array

- [ ] **Step 8: Commit**

```bash
git add backend/db.js backend/index.js
git commit -m "feat: add dashboard stats, charts, search, and activity log APIs"
```

---

### Task 5: Build Dashboard Home Page

**Files:**
- Rewrite: `frontend/views/Dashboard.vue`

- [ ] **Step 1: Implement full Dashboard page**

Dashboard.vue structure:
1. Welcome greeting with today's date
2. Row of 4 `StatCard` components — fetch from `/api/dashboard/stats` on mount
3. Charts row (two columns): ECharts line chart (monthly trend) + ECharts pie chart (channel distribution) — fetch from `/api/dashboard/charts`
4. Dynamic panels row (two columns): Upcoming observations (fetch from `/api/observations/today`) + Sync status (fetch from `/api/db/sync-status` and `/api/db/sync-coinvest-status`)
5. Quick entry grid: 8 module links as simple icon+title cards

Import and use: `StatCard`, `PanelCard`, `WorkbenchLayout`. Import ECharts with tree-shaking: `import * as echarts from 'echarts/core'` + `BarChart`, `LineChart`, `PieChart`, `GridComponent`, `TooltipComponent`, `LegendComponent`, `CanvasRenderer`.

Charts: init on `onMounted` after data loads using `echarts.init()` on ref'd div elements. Dispose on `onUnmounted`. Use `watch` on data to update charts reactively.

- [ ] **Step 2: Verify in browser**

Navigate to `/`. Should show:
- Welcome banner with date
- 4 stat cards with real numbers from backend
- Line chart and pie chart rendering data
- Observation feed and sync status panels
- Quick entry module grid

- [ ] **Step 3: Commit**

```bash
git add frontend/views/Dashboard.vue
git commit -m "feat: implement Dashboard home page with stats, charts, and feeds"
```

---

### Task 6: Build Activity Log Page

**Files:**
- Rewrite: `frontend/views/ActivityLog.vue`
- Create: `frontend/components/ActivityTimeline.vue`

- [ ] **Step 1: Create `frontend/components/ActivityTimeline.vue`**

Props: `logs` (Array of `{ id, type, action, detail, createdAt }`). Renders a vertical timeline with dots and connecting lines. Each entry shows: time (formatted), type badge (color-coded: sync=blue, query=green, export=amber), action text, and detail text. Empty state when no logs.

- [ ] **Step 2: Implement `frontend/views/ActivityLog.vue`**

Fetch from `/api/activity-logs` on mount. Show filter buttons (全部/同步/查询/导出) that update a `type` filter. Pass filtered logs to `ActivityTimeline`. Wrap in `WorkbenchLayout`.

- [ ] **Step 3: Commit**

```bash
git add frontend/views/ActivityLog.vue frontend/components/ActivityTimeline.vue
git commit -m "feat: add activity log page with timeline component"
```

---

### Task 7: Restyle Data Preparation & User Profile Pages

**Files:**
- Restyle: `frontend/views/DataPreparation.vue`
- Restyle: `frontend/views/UserProfile.vue`

**Pattern for all page restyles:** Replace `SubPageLayout` import with `WorkbenchLayout`. Wrap content in `WorkbenchLayout`. Use `PanelCard` for sections. Use new design token classes (`text-page-title`, `form-row`, `input`, `btn`, `data-table`, `table-wrap`, `badge`, etc.). Remove all old scoped styles. Preserve all `<script setup>` logic exactly as-is.

- [ ] **Step 1: Restyle `DataPreparation.vue`**

Replace `<SubPageLayout>` wrapper with `<WorkbenchLayout>`. Use `PanelCard` for auth panel, transaction sync panel, and co-invest sync panel. Keep all existing script logic, API calls, and reactive state unchanged. Use design token classes for all styling.

- [ ] **Step 2: Restyle `UserProfile.vue`**

Replace `<SubPageLayout>` wrapper with `<WorkbenchLayout>`. Use `PanelCard` for filter panel and results panel. Use `data-table` + `table-wrap` for results table. Keep all existing script logic.

- [ ] **Step 3: Verify both pages in browser**

- `/data-preparation` — auth status, sync buttons, status badges render correctly
- `/user-profile` — filters, search, results table render correctly

- [ ] **Step 4: Commit**

```bash
git add frontend/views/DataPreparation.vue frontend/views/UserProfile.vue
git commit -m "style: restyle data preparation and user profile pages"
```

---

### Task 8: Restyle Analysis Pages (CustomerChurn, ChannelAnalysis, NominalBuyer)

**Files:**
- Restyle: `frontend/views/CustomerChurn.vue`
- Restyle: `frontend/views/ChannelAnalysis.vue`
- Restyle: `frontend/views/NominalBuyer.vue`

- [ ] **Step 1: Restyle `CustomerChurn.vue`**

Replace `SubPageLayout` with `WorkbenchLayout`. Use `PanelCard` for parameter panel and report panel. Use design token classes.

- [ ] **Step 2: Restyle `ChannelAnalysis.vue`**

Replace `SubPageLayout` with `WorkbenchLayout`. Use `PanelCard` for parameter panel and results. Use `data-table` + `table-wrap`.

- [ ] **Step 3: Restyle `NominalBuyer.vue`**

Replace `SubPageLayout` with `WorkbenchLayout`. Use `PanelCard` for input panel and results. Use `textarea` class for multi-line input. Use `data-table` + `table-wrap`.

- [ ] **Step 4: Commit**

```bash
git add frontend/views/CustomerChurn.vue frontend/views/ChannelAnalysis.vue frontend/views/NominalBuyer.vue
git commit -m "style: restyle analysis pages (churn, channel, nominal buyer)"
```

---

### Task 9: Restyle Product Pages (ProductReport, ProductCompletion, OngoingProduct)

**Files:**
- Restyle: `frontend/views/ProductReport.vue`
- Restyle: `frontend/views/ProductCompletion.vue`
- Restyle: `frontend/views/OngoingProduct.vue`

These are the largest/most complex pages. Preserve ALL script logic. Only restructure templates and styles.

- [ ] **Step 1: Restyle `ProductReport.vue`**

Replace `SubPageLayout` with `WorkbenchLayout`. Use `PanelCard` for parameter panel and report area. Keep product card grid with updated styling. Use `data-table` for month selector area.

- [ ] **Step 2: Restyle `ProductCompletion.vue`** (938 lines — largest page)

Replace `SubPageLayout` with `WorkbenchLayout`. Convert tabs to segmented control style (horizontal buttons with active state). Keep ALL 4 tab panels (全量/日历/今日/喜报) with their existing logic. Use `data-table` + `table-wrap` for observation tables. Keep `PosterTemplate` component integration. Restyle calendar grid.

- [ ] **Step 3: Restyle `OngoingProduct.vue`**

Replace `SubPageLayout` with `WorkbenchLayout`. Use `PanelCard` for parameter panel. Use stat card grid for summary overview. Use `data-table` + `table-wrap` for analysis tables.

- [ ] **Step 4: Commit**

```bash
git add frontend/views/ProductReport.vue frontend/views/ProductCompletion.vue frontend/views/OngoingProduct.vue
git commit -m "style: restyle product pages (report, observation, ongoing)"
```

---

### Task 10: Cleanup, Delete Old Files & Final Verification

**Files:**
- Delete: `frontend/components/SubPageLayout.vue`
- Delete: `frontend/components/FolderCard.vue` (unused after ProductReport restyle)
- Delete: `frontend/views/Home.vue` (replaced by Dashboard.vue)
- Delete: `frontend/assets/global.css` (replaced by main.css)

- [ ] **Step 1: Remove obsolete files**

```bash
git rm frontend/components/SubPageLayout.vue
git rm frontend/components/FolderCard.vue
git rm frontend/views/Home.vue
git rm frontend/assets/global.css
```

- [ ] **Step 2: Verify no remaining imports to deleted files**

```bash
cd frontend && grep -rl "SubPageLayout" --include="*.vue" --include="*.js" .
cd frontend && grep -rl "FolderCard" --include="*.vue" --include="*.js" .
cd frontend && grep -rl "global.css" --include="*.vue" --include="*.js" .
cd frontend && grep -rl "Home.vue" --include="*.vue" --include="*.js" .
```

All searches should return zero results. If any remain, update the imports.

- [ ] **Step 3: Run full build**

```bash
cd frontend && npm run build
```

Expected: Build succeeds with zero errors.

- [ ] **Step 4: Start both servers and verify all routes in browser**

Start backend (`npm start`) and frontend (`npm run dev`). Visit each route:

| Route | Verify |
|-------|--------|
| `/` | Dashboard: stat cards, charts, observation feed, sync status, quick entries |
| `/data-preparation` | Auth connection, data sync, co-invest sync |
| `/user-profile` | Filters, search, results table |
| `/customer-churn` | Date filters, generate/download buttons, report panel |
| `/product-report` | Month selector, product cards, sync from Feishu |
| `/product-completion` | 4 tabs, observation table, calendar, posters |
| `/ongoing-product` | File upload, analysis summary + tables |
| `/channel-analysis` | Date range, analysis results |
| `/nominal-buyer` | Textarea input, query, results table |
| `/activity-log` | Timeline display, type filters |

- [ ] **Step 5: Test global search**

Press Cmd+K (or Ctrl+K). Verify:
- Modal opens centered on screen
- Typing triggers search (debounced)
- Results appear grouped by type
- Arrow keys navigate, Enter selects
- Escape closes modal
- Clicking outside closes modal

- [ ] **Step 6: Test mobile responsiveness**

Resize browser to 375px width. Verify:
- Sidebar collapses (hidden by default)
- Hamburger toggle opens sidebar overlay
- Clicking link closes overlay
- Content is readable on small screen

- [ ] **Step 7: Final commit**

```bash
git add -A
git commit -m "chore: cleanup old files, final verification pass"
```

---

## Execution Order Summary

```
Task 1: Dependencies + Design Tokens
  ↓
Task 2: App Shell (Sidebar + Layout)
  ↓
Task 3: Shared Components (Search, StatCard, PanelCard)
  ↓
Task 4: Backend APIs (stats, charts, search, activity log)
  ↓
Task 5: Dashboard Home Page
  ↓
Task 6: Activity Log Page
  ↓
Task 7: Restyle DataPrep + UserProfile
  ↓
Task 8: Restyle Churn + Channel + Nominal
  ↓
Task 9: Restyle ProductReport + Completion + Ongoing
  ↓
Task 10: Cleanup + Final Verification
```

Each task produces a working, buildable state. Commit after every task for easy rollback.