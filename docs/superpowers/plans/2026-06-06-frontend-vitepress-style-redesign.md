# Frontend VitePress Style Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the Vue frontend into a VitePress-inspired business workbench with a shared shell, local/system fonts, readable Chinese UI text, and consistent base controls.

**Architecture:** Add one shared layout component that owns the fixed top bar, sidebar module navigation, and responsive content frame. Move common visual tokens and reusable control styles into `frontend/assets/global.css`, then simplify individual pages so their scripts and API behavior stay intact while their templates and styles use the shared UI language.

**Tech Stack:** Vue 3 single-file components, Vue Router 4, Vite 5, plain CSS, existing `xlsx` and browser APIs.

---

## File Structure

- Create: `frontend/components/WorkbenchLayout.vue`
  - Owns the app shell: fixed top navigation, left module sidebar, mobile navigation toggle, page heading, optional description slot, and main content slot.
  - Exposes props: `title`, `description`, `wide`.
  - Depends only on `vue` and `vue-router`.

- Modify: `frontend/assets/global.css`
  - Defines design variables, font stacks, page background, shared `.panel`, `.btn`, `.input`, `.textarea`, `.table`, `.status-badge`, `.empty-state`, `.desc`, and responsive helpers.

- Modify: `frontend/components/SubPageLayout.vue`
  - Becomes a compatibility wrapper around `WorkbenchLayout` so current page imports keep working.
  - Exposes props: `title`, `description`, `wide`.

- Modify: `frontend/views/Home.vue`
  - Converts the homepage to the shared workbench shell.
  - Fixes Chinese text.
  - Shows module cards and a workflow overview.

- Modify: `frontend/views/DataPreparation.vue`
  - Fixes Chinese text in template and script messages.
  - Removes duplicated page-level visual styles that are covered by global classes.

- Modify: `frontend/views/UserProfile.vue`
  - Fixes Chinese text.
  - Keeps existing query API and filter state.
  - Uses global form, button, badge, table, and empty-state styles.

- Modify: `frontend/views/ProductCompletion.vue`
  - Fixes Chinese text and malformed string literals.
  - Keeps all existing observation, today, and poster API calls.
  - Uses global tabs, panels, buttons, inputs, table, status, and poster card styling.

- Modify: `frontend/views/ProductReport.vue`
  - Fixes Chinese text and malformed regex/string literals.
  - Keeps existing product-doc sync and month loading behavior.
  - Uses wide layout because product cards need more horizontal space.

- Modify: `frontend/views/OngoingProduct.vue`
  - Fixes Chinese text and malformed sheet-name logic.
  - Keeps client-side Excel parsing and summary calculation behavior.
  - Uses wide layout for result tables.

- Modify: `frontend/views/CustomerChurn.vue`
  - Fixes Chinese text and alert messages.
  - Uses global form and panel classes.

- Modify: `frontend/views/ChannelAnalysis.vue`
  - Fixes Chinese text and alert messages.
  - Uses global form and panel classes.

- Modify: `frontend/views/NominalBuyer.vue`
  - Fixes Chinese text and malformed textarea markup.
  - Uses global form, textarea, table, and panel classes.

- Leave unchanged unless build errors reveal a direct need:
  - `frontend/App.vue`
  - `frontend/main.js`
  - `frontend/router/index.js`
  - Backend files

---

### Task 1: Create Shared Workbench Shell

**Files:**
- Create: `frontend/components/WorkbenchLayout.vue`
- Modify: `frontend/components/SubPageLayout.vue`

- [ ] **Step 1: Create `WorkbenchLayout.vue` with route metadata and responsive sidebar**

Create `frontend/components/WorkbenchLayout.vue` with this structure:

```vue
<template>
  <div class="workbench-shell">
    <header class="workbench-topbar">
      <router-link to="/" class="workbench-brand" aria-label="返回工作台首页">
        <span class="brand-mark">BW</span>
        <span>
          <strong>业务工作台</strong>
          <em>航班服务 · 数据分析平台</em>
        </span>
      </router-link>

      <nav class="topbar-links" aria-label="顶部导航">
        <router-link to="/" class="topbar-link">首页</router-link>
        <router-link to="/product-completion" class="topbar-link">观察</router-link>
        <router-link to="/product-report" class="topbar-link">报告</router-link>
      </nav>

      <div class="topbar-actions">
        <button class="btn btn-secondary btn-sm" type="button" @click="openFeishu">
          打开飞书总表
        </button>
        <button class="icon-menu" type="button" :aria-expanded="sidebarOpen" aria-label="切换导航" @click="sidebarOpen = !sidebarOpen">
          <span></span>
          <span></span>
          <span></span>
        </button>
      </div>
    </header>

    <div class="workbench-body">
      <aside class="workbench-sidebar" :class="{ open: sidebarOpen }">
        <div class="sidebar-section">
          <p class="sidebar-kicker">Modules</p>
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            class="sidebar-link"
            @click="sidebarOpen = false"
          >
            <span class="sidebar-index">{{ item.index }}</span>
            <span>
              <strong>{{ item.title }}</strong>
              <em>{{ item.description }}</em>
            </span>
          </router-link>
        </div>
      </aside>

      <main class="workbench-main" :class="{ wide }">
        <div class="page-heading">
          <p class="page-kicker">Business Workbench</p>
          <h1>{{ title }}</h1>
          <p v-if="description" class="page-description">{{ description }}</p>
        </div>
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  wide: { type: Boolean, default: false },
})

const sidebarOpen = ref(false)

const navItems = [
  { index: '01', title: '数据准备', description: '同步飞书与本地数据', path: '/data-preparation' },
  { index: '02', title: '用户画像', description: '查询合投用户特征', path: '/user-profile' },
  { index: '03', title: '客户流失', description: '识别完结未复购客户', path: '/customer-churn' },
  { index: '04', title: '产品报告', description: '查看产品运行材料', path: '/product-report' },
  { index: '05', title: '派息/敲出观察', description: '跟踪存续产品观察日', path: '/product-completion' },
  { index: '06', title: '存续分析', description: '分析仍在持有产品', path: '/ongoing-product' },
  { index: '07', title: '渠道分析', description: '统计渠道成交表现', path: '/channel-analysis' },
  { index: '08', title: '名义购买人', description: '匹配私募管理人', path: '/nominal-buyer' },
]

function openFeishu() {
  alert('请在飞书中打开“航班服务交易总表”。')
}
</script>
```

- [ ] **Step 2: Replace `SubPageLayout.vue` with a compatibility wrapper**

Replace the file with:

```vue
<template>
  <WorkbenchLayout :title="title" :description="description" :wide="wide">
    <slot />
  </WorkbenchLayout>
</template>

<script setup>
import WorkbenchLayout from './WorkbenchLayout.vue'

defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  wide: { type: Boolean, default: false },
})
</script>
```

- [ ] **Step 3: Run syntax validation**

Run:

```powershell
node --check frontend\main.js
```

Expected: no syntax errors.

- [ ] **Step 4: Commit shell component**

Run:

```powershell
git add frontend\components\WorkbenchLayout.vue frontend\components\SubPageLayout.vue
git commit -m "feat: add shared workbench layout"
```

Expected: commit succeeds with only those two files staged.

---

### Task 2: Add Global VitePress-Inspired Design System

**Files:**
- Modify: `frontend/assets/global.css`

- [ ] **Step 1: Replace global CSS with design variables and reusable classes**

Replace `frontend/assets/global.css` with:

```css
* {
  box-sizing: border-box;
}

html {
  min-width: 320px;
  background: #ffffff;
  color: #171717;
  font-size: 16px;
  text-rendering: optimizeLegibility;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

body {
  margin: 0;
  min-width: 320px;
  min-height: 100vh;
  font-family: "Inter", "Punctuation SC", ui-sans-serif, system-ui, -apple-system,
    BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei",
    "Noto Sans SC", sans-serif;
  color: var(--ink);
  background: var(--bg);
}

:root {
  --bg: #ffffff;
  --bg-alt: #f6f6f7;
  --surface: #ffffff;
  --surface-muted: #f8fafc;
  --ink: #20242a;
  --ink-strong: #111827;
  --ink-soft: #64748b;
  --ink-faint: #94a3b8;
  --border: #d9dee7;
  --border-soft: #e5e7eb;
  --brand: #2559db;
  --brand-hover: #1d4ed8;
  --brand-soft: rgba(37, 89, 219, 0.09);
  --success: #18794e;
  --success-soft: rgba(24, 121, 78, 0.1);
  --warning: #946300;
  --warning-soft: rgba(148, 99, 0, 0.12);
  --danger: #b8272c;
  --danger-soft: rgba(184, 39, 44, 0.1);
  --shadow-soft: 0 12px 32px rgba(15, 23, 42, 0.08);
  --radius: 8px;
  --topbar-height: 64px;
  --sidebar-width: 272px;
  --mono: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco,
    Consolas, "Liberation Mono", monospace;
}

button,
input,
select,
textarea {
  font: inherit;
}

button {
  border: 0;
}

a {
  color: inherit;
  text-decoration: none;
}

#app {
  min-height: 100vh;
}

.workbench-shell {
  min-height: 100vh;
  background: var(--bg);
}

.workbench-topbar {
  position: fixed;
  z-index: 40;
  top: 0;
  right: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--topbar-height);
  padding: 0 24px;
  border-bottom: 1px solid var(--border-soft);
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(14px);
}

.workbench-brand {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-mark {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--brand);
  font-family: var(--mono);
  font-size: 12px;
  font-weight: 700;
  background: var(--surface);
}

.workbench-brand strong {
  display: block;
  color: var(--ink-strong);
  font-size: 16px;
  line-height: 1.2;
}

.workbench-brand em {
  display: block;
  margin-top: 2px;
  color: var(--ink-soft);
  font-size: 12px;
  font-style: normal;
  line-height: 1.2;
}

.topbar-links {
  display: flex;
  align-items: center;
  gap: 6px;
}

.topbar-link {
  padding: 8px 10px;
  border-radius: var(--radius);
  color: var(--ink-soft);
  font-size: 14px;
  font-weight: 600;
}

.topbar-link:hover,
.topbar-link.router-link-exact-active {
  color: var(--brand);
  background: var(--brand-soft);
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.icon-menu {
  display: none;
  width: 36px;
  height: 36px;
  border-radius: var(--radius);
  background: var(--surface-muted);
}

.icon-menu span {
  display: block;
  width: 16px;
  height: 2px;
  margin: 3px auto;
  border-radius: 999px;
  background: var(--ink-strong);
}

.workbench-body {
  display: flex;
  min-height: 100vh;
  padding-top: var(--topbar-height);
}

.workbench-sidebar {
  position: fixed;
  top: var(--topbar-height);
  bottom: 0;
  left: 0;
  width: var(--sidebar-width);
  overflow-y: auto;
  border-right: 1px solid var(--border-soft);
  background: var(--bg-alt);
  padding: 32px 18px;
}

.sidebar-kicker,
.page-kicker {
  margin: 0 0 14px;
  color: var(--brand);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.14em;
  line-height: 1;
  text-transform: uppercase;
}

.sidebar-link {
  display: flex;
  gap: 12px;
  padding: 12px 10px;
  border-radius: var(--radius);
  color: var(--ink-soft);
}

.sidebar-link + .sidebar-link {
  margin-top: 4px;
}

.sidebar-link:hover,
.sidebar-link.router-link-exact-active {
  color: var(--ink-strong);
  background: #ffffff;
}

.sidebar-link.router-link-exact-active {
  box-shadow: inset 3px 0 0 var(--brand);
}

.sidebar-index {
  color: var(--ink-faint);
  font-family: var(--mono);
  font-size: 12px;
  font-weight: 700;
}

.sidebar-link strong {
  display: block;
  font-size: 14px;
  line-height: 1.35;
}

.sidebar-link em {
  display: block;
  margin-top: 2px;
  color: var(--ink-faint);
  font-size: 12px;
  font-style: normal;
  line-height: 1.35;
}

.workbench-main {
  width: 100%;
  max-width: 1120px;
  margin-left: var(--sidebar-width);
  padding: 38px 40px 72px;
}

.workbench-main.wide {
  max-width: none;
}

.page-heading {
  margin-bottom: 28px;
}

.page-heading h1 {
  margin: 0;
  color: var(--ink-strong);
  font-size: clamp(34px, 4vw, 56px);
  font-weight: 800;
  letter-spacing: 0;
  line-height: 1.08;
}

.page-description,
.desc {
  max-width: 900px;
  margin: 14px 0 0;
  color: var(--ink-soft);
  font-size: 15px;
  line-height: 1.8;
}

.section {
  display: grid;
  gap: 20px;
}

.panel,
.report-panel,
.result-panel,
.empty-state,
.empty {
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--surface);
}

.panel,
.report-panel,
.result-panel {
  padding: 24px;
}

.panel-title,
.section-title {
  margin: 0 0 16px;
  color: var(--ink-strong);
  font-size: 16px;
  font-weight: 800;
  line-height: 1.3;
}

.section-desc {
  margin: 0 0 14px;
  color: var(--ink-soft);
  font-size: 13px;
  line-height: 1.7;
}

.form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.form-row.align-top {
  align-items: flex-start;
}

.form-row > label:first-child {
  width: 104px;
  flex: 0 0 auto;
  color: var(--ink-soft);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.5;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px 16px;
  margin-bottom: 16px;
}

.input,
.textarea,
.month-select,
.file-label {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 8px 11px;
  color: var(--ink);
  font-size: 13px;
  background: #ffffff;
  outline: none;
}

.textarea {
  resize: vertical;
}

.input:focus,
.textarea:focus,
.month-select:focus,
.file-label:hover {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 36px;
  padding: 0 16px;
  border: 1px solid transparent;
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
  line-height: 1;
}

.btn-sm {
  min-height: 32px;
  padding: 0 12px;
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.btn-primary {
  color: #ffffff;
  background: var(--brand);
}

.btn-primary:hover:not(:disabled) {
  background: var(--brand-hover);
}

.btn-secondary,
.btn-outline {
  border-color: var(--border);
  color: var(--ink);
  background: #ffffff;
}

.btn-secondary:hover:not(:disabled),
.btn-outline:hover:not(:disabled) {
  border-color: var(--brand);
  color: var(--brand);
  background: var(--brand-soft);
}

.actions,
.search-actions,
.sync-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.table-wrap,
.table-panel {
  overflow-x: auto;
}

.table,
.data-table,
.overview-table,
.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.table th,
.table td,
.data-table th,
.data-table td,
.overview-table th,
.overview-table td,
.detail-table th,
.detail-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-soft);
  text-align: left;
  white-space: nowrap;
}

.table th,
.data-table th,
.overview-table th,
.detail-table th {
  color: var(--ink-soft);
  font-size: 12px;
  font-weight: 800;
  background: var(--surface-muted);
}

.table tbody tr:hover,
.data-table tbody tr:hover,
.overview-table tbody tr:hover {
  background: var(--surface-muted);
}

.num,
.col-right {
  text-align: right;
}

.col-center {
  text-align: center;
}

.badge,
.status-badge,
.auth-badge,
.poster-type-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 22px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.badge-green,
.status-ok,
.auth-ok,
.result-yes-dividend {
  color: var(--success);
  background: var(--success-soft);
}

.badge-red,
.result-yes-knockout {
  color: var(--danger);
  background: var(--danger-soft);
}

.status-loading {
  color: var(--warning);
  background: var(--warning-soft);
}

.status-none,
.auth-none,
.result-na {
  color: var(--ink-soft);
  background: var(--surface-muted);
}

.error,
.error-msg {
  color: var(--danger);
  font-size: 13px;
}

.success,
.success-msg {
  color: var(--success);
  font-size: 13px;
}

.hint,
.result-count,
.pending,
.placeholder,
.loading-msg,
.table-summary {
  color: var(--ink-soft);
  font-size: 13px;
}

.empty-state,
.empty {
  padding: 44px 24px;
  color: var(--ink-soft);
  text-align: center;
}

.code-cell,
.sidebar-index,
.flow-num {
  font-family: var(--mono);
}

@media (max-width: 960px) {
  .topbar-links {
    display: none;
  }

  .icon-menu {
    display: inline-grid;
    place-content: center;
  }

  .workbench-sidebar {
    z-index: 35;
    transform: translateX(-100%);
    transition: transform 0.18s ease;
    box-shadow: var(--shadow-soft);
  }

  .workbench-sidebar.open {
    transform: translateX(0);
  }

  .workbench-main {
    max-width: none;
    margin-left: 0;
    padding: 30px 20px 56px;
  }
}

@media (max-width: 640px) {
  .workbench-topbar {
    padding: 0 12px;
  }

  .workbench-brand em,
  .topbar-actions .btn {
    display: none;
  }

  .workbench-main {
    padding: 24px 14px 48px;
  }

  .page-heading h1 {
    font-size: 34px;
  }

  .form-row {
    align-items: stretch;
    flex-direction: column;
    gap: 7px;
  }

  .form-row > label:first-child {
    width: auto;
  }

  .panel,
  .report-panel,
  .result-panel {
    padding: 18px;
  }
}
```

- [ ] **Step 2: Run frontend build**

Run:

```powershell
npm.cmd run build
```

from `D:\projects\business-workbench\frontend`.

Expected: Vite build either succeeds or fails only because page files still contain pre-existing malformed Chinese/string literals that later tasks fix.

- [ ] **Step 3: Commit global design system**

Run:

```powershell
git add frontend\assets\global.css
git commit -m "style: add workbench design system"
```

Expected: commit includes only `frontend/assets/global.css`.

---

### Task 3: Rebuild Home Page

**Files:**
- Modify: `frontend/views/Home.vue`

- [ ] **Step 1: Replace `Home.vue` template and script**

Use `WorkbenchLayout` directly, with readable Chinese labels and module cards. Keep route paths unchanged.

```vue
<template>
  <WorkbenchLayout
    title="业务工作台"
    description="统一管理航班服务数据准备、客户分析、产品报告、存续观察和渠道统计。"
  >
    <section class="home-hero">
      <div>
        <p class="home-label">Operational Console</p>
        <h2>从数据同步到观察报告，一屏进入核心流程。</h2>
        <p>
          先完成飞书数据同步，再进入用户画像、产品观察、存续分析和渠道分析等业务模块。
        </p>
      </div>
      <button class="btn btn-primary" type="button" @click="openExternal">打开飞书总表</button>
    </section>

    <section class="module-grid" aria-label="业务模块">
      <router-link v-for="item in modules" :key="item.path" :to="item.path" class="module-card">
        <span class="module-index">{{ item.index }}</span>
        <strong>{{ item.title }}</strong>
        <em>{{ item.description }}</em>
      </router-link>
    </section>

    <section class="report-panel">
      <h3 class="panel-title">建议流程</h3>
      <div class="flow">
        <router-link v-for="item in workflow" :key="item.path" :to="item.path" class="flow-card">
          <span class="flow-num">{{ item.index }}</span>
          <span class="flow-name">{{ item.title }}</span>
        </router-link>
      </div>
    </section>
  </WorkbenchLayout>
</template>

<script setup>
import WorkbenchLayout from '../components/WorkbenchLayout.vue'

const modules = [
  { index: '01', title: '数据准备', description: '同步飞书总表和合投用户表', path: '/data-preparation' },
  { index: '02', title: '用户画像', description: '按购买人、专户、竞品和行业筛选用户', path: '/user-profile' },
  { index: '03', title: '客户流失', description: '生成完结未复购客户分析', path: '/customer-churn' },
  { index: '04', title: '产品报告', description: '同步和查看产品运行材料', path: '/product-report' },
  { index: '05', title: '派息/敲出观察', description: '跟踪存续产品观察日和喜报', path: '/product-completion' },
  { index: '06', title: '存续分析', description: '分析存续产品金额、人数和类型', path: '/ongoing-product' },
  { index: '07', title: '渠道分析', description: '统计渠道成交人数、金额和复购表现', path: '/channel-analysis' },
  { index: '08', title: '名义购买人', description: '查询名义购买人与私募管理人关系', path: '/nominal-buyer' },
]

const workflow = modules.slice(0, 7)

function openExternal() {
  alert('请在飞书中打开“航班服务交易总表”，导出或同步后再继续分析。')
}
</script>
```

- [ ] **Step 2: Add scoped home styles**

Add this scoped style block:

```vue
<style scoped>
.home-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 24px;
  align-items: end;
  margin-bottom: 24px;
  padding: 28px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--surface);
}

.home-label {
  margin: 0 0 12px;
  color: var(--brand);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.home-hero h2 {
  max-width: 760px;
  margin: 0;
  color: var(--ink-strong);
  font-size: clamp(28px, 3.2vw, 44px);
  font-weight: 800;
  line-height: 1.12;
}

.home-hero p:last-child {
  max-width: 720px;
  margin: 14px 0 0;
  color: var(--ink-soft);
  line-height: 1.8;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 16px;
}

.module-card {
  display: grid;
  gap: 8px;
  min-height: 150px;
  padding: 18px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--surface);
  transition: border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
}

.module-card:hover {
  transform: translateY(-2px);
  border-color: var(--brand);
  box-shadow: var(--shadow-soft);
}

.module-index {
  color: var(--brand);
  font-family: var(--mono);
  font-size: 12px;
  font-weight: 800;
}

.module-card strong {
  color: var(--ink-strong);
  font-size: 18px;
}

.module-card em {
  color: var(--ink-soft);
  font-size: 13px;
  font-style: normal;
  line-height: 1.6;
}

.flow {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.flow-card {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--ink);
  background: #ffffff;
}

.flow-card:hover {
  border-color: var(--brand);
  color: var(--brand);
}

.flow-num {
  color: var(--brand);
  font-size: 11px;
  font-weight: 800;
}

.flow-name {
  font-size: 13px;
  font-weight: 700;
}

@media (max-width: 720px) {
  .home-hero {
    grid-template-columns: 1fr;
    padding: 20px;
  }
}
</style>
```

- [ ] **Step 3: Build check**

Run:

```powershell
npm.cmd run build
```

from `frontend`.

Expected: home page no longer contributes malformed template or string errors.

- [ ] **Step 4: Commit home page**

Run:

```powershell
git add frontend\views\Home.vue
git commit -m "feat: redesign workbench home"
```

Expected: commit includes only `frontend/views/Home.vue`.

---

### Task 4: Fix Data Preparation And Simple Form Pages

**Files:**
- Modify: `frontend/views/DataPreparation.vue`
- Modify: `frontend/views/CustomerChurn.vue`
- Modify: `frontend/views/ChannelAnalysis.vue`
- Modify: `frontend/views/NominalBuyer.vue`

- [ ] **Step 1: Update `DataPreparation.vue` readable text and props**

Change the opening layout tag to:

```vue
<SubPageLayout
  title="数据准备"
  description="连接飞书账号后，将航班服务交易总表和合投用户表同步到本地数据库，供后续业务页面使用。"
>
```

Replace visible template text with:

```text
飞书账号连接
已连接
断开连接
未连接
连接飞书账号
跳转中...
数据同步
航班服务交易总表
上次同步：{{ formatTime(syncStatus.synced_at) }}，共 {{ syncStatus.row_count }} 条
尚未同步
已就绪
同步中...
未同步
同步数据
请先连接飞书账号
合投用户表
```

Replace script messages:

```js
result.value = '飞书授权失败：' + (params.get('msg') || '未知错误')
result.value = '获取授权链接失败，请确认后端服务已启动。'
if (!res.ok) throw new Error(data.error || '同步失败')
syncSuccess.value = `同步成功，共写入 ${data.rowCount} 条数据`
```

- [ ] **Step 2: Update `CustomerChurn.vue` readable text**

Use:

```vue
<SubPageLayout
  title="客户流失分析"
  description="生成客户存量峰值分析报告，识别已完结未复购客户，并为后续 Excel 导出预留入口。"
>
```

Visible labels:

```text
生成分析报告
分析截止日期
数据文件路径
请输入交易表_修正后.xlsx 路径
生成报告
下载 Excel
隐藏报告
查看报告
报告预览
报告内容将在接入后端后展示。
```

Alert messages:

```js
function generate() { alert('生成报告功能尚未接入后端。') }
function download() { alert('下载 Excel 功能尚未接入后端。') }
```

- [ ] **Step 3: Update `ChannelAnalysis.vue` readable text**

Use:

```vue
<SubPageLayout
  title="渠道分析"
  description="按最终渠道统计各渠道的成交人数、成交金额与复购表现。"
>
```

Visible labels:

```text
参数设置
开始年月
结束年月
数据文件
请输入交易表_修正后.xlsx 路径
生成渠道分析
分析结果
各渠道数据将在接入后端后展示。
```

Alert message:

```js
function run() { alert('渠道分析功能尚未接入后端。') }
```

- [ ] **Step 4: Update `NominalBuyer.vue` readable text and malformed textarea**

Use:

```vue
<SubPageLayout
  title="名义购买人 × 私募管理人"
  description="一次输入多个名义购买人，查看每个人分别在哪些私募管理人处购买过产品。"
>
```

Use this textarea markup:

```vue
<textarea
  v-model="names"
  rows="6"
  placeholder="请输入名义购买人姓名，每行一个"
  class="textarea"
/>
```

Visible labels:

```text
输入查询
名义购买人
（每行一个）
数据文件
请输入交易表_修正后.xlsx 路径
查询
查询结果
私募管理人
```

Alert message:

```js
function query() {
  alert('查询功能尚未接入后端。')
}
```

- [ ] **Step 5: Remove duplicated scoped CSS that conflicts with globals**

In these four files, remove local definitions for `.desc`, `.panel`, `.panel-title`, `.form-row`, `.input`, `.textarea`, `.btn`, `.btn-primary`, `.btn-outline`, `.report-panel`, `.placeholder`, `.table`, `.error-msg`, `.success-msg` when the global class covers the same styling.

Keep page-specific styles:

```css
.auth-row,
.source-row,
.source-icon,
.source-info,
.source-name,
.source-desc,
.source-status,
.file-input {
  /* Keep these selectors only when the page still uses them. */
}
```

- [ ] **Step 6: Build check**

Run:

```powershell
npm.cmd run build
```

from `frontend`.

Expected: these four pages no longer produce malformed template or script string errors.

- [ ] **Step 7: Commit simple form pages**

Run:

```powershell
git add frontend\views\DataPreparation.vue frontend\views\CustomerChurn.vue frontend\views\ChannelAnalysis.vue frontend\views\NominalBuyer.vue
git commit -m "fix: restore text on form pages"
```

Expected: commit includes only the four listed page files.

---

### Task 5: Fix User Profile Page

**Files:**
- Modify: `frontend/views/UserProfile.vue`

- [ ] **Step 1: Update layout props and visible Chinese**

Use:

```vue
<SubPageLayout
  title="用户画像"
  description="查询合投用户画像，支持按实际购买人、名义购买人、专户客户、竞品客户和行业筛选。"
  wide
>
```

Use these visible labels:

```text
搜索条件
实际购买人
名义购买人
模糊匹配
是否专户客户
客户是否竞品群
客户行业
全部
是
否
查询中...
搜索
重置
共 {{ rows.length }} 条
境内资产规模区间/万RMB
微信昵称
手机号
风险承受
历史存量峰值
峰值差额
待接入
未找到匹配的用户，请调整搜索条件
```

- [ ] **Step 2: Update script fallback and query error**

Replace garbled strings in `search()` with:

```js
if (!res.ok) throw new Error(data.error || '查询失败')
```

Ensure the filter option values for yes/no remain:

```vue
<option value="是">是</option>
<option value="否">否</option>
```

Ensure badge comparisons use:

```vue
:class="row.is_competitor === '是' ? 'badge-red' : 'badge-green'"
:class="row.is_dedicated_account === '是' ? 'badge-red' : 'badge-green'"
```

- [ ] **Step 3: Simplify local CSS**

Keep only page-specific CSS for table overflow and amount colors:

```css
.table-panel {
  overflow-x: auto;
}

.positive {
  color: var(--success);
  font-weight: 700;
}

.negative {
  color: var(--danger);
  font-weight: 700;
}
```

- [ ] **Step 4: Build check**

Run:

```powershell
npm.cmd run build
```

from `frontend`.

Expected: no `UserProfile.vue` template or script errors.

- [ ] **Step 5: Commit user profile**

Run:

```powershell
git add frontend\views\UserProfile.vue
git commit -m "fix: restore user profile text"
```

Expected: commit includes only `frontend/views/UserProfile.vue`.

---

### Task 6: Fix Product Observation Page

**Files:**
- Modify: `frontend/views/ProductCompletion.vue`

- [ ] **Step 1: Update layout props and tab labels**

Use:

```vue
<SubPageLayout
  title="产品派息/敲出观察"
  description="展示存续产品的派息与敲出观察情况，并生成对应日期的喜报海报。"
  wide
>
```

Tab labels:

```text
全量
今日观察
喜报
```

- [ ] **Step 2: Restore observation labels**

Use these table labels and messages:

```text
存续产品观察概览
航班编号
产品名称
私募管理人
持有状态
代码
入场价
入场日
存续月
锁定期(月)
最近观察日
标的价格
敲出价
派息线
是否敲出
是否派息
历史观察日明细
暂无观察日记录
共 {{ filteredProducts.length }} 个存续产品
暂无存续产品数据，请先在“数据准备”页面同步飞书数据。
```

Action labels:

```text
操作
数据来源
航班服务交易总表 · 产品表
本地数据库
搜索
按产品名称或航班编号搜索...
刷新标的价格
刷新中...
生成观察记录
生成中...
最后更新 {{ lastUpdated }}
```

- [ ] **Step 3: Restore today and poster labels**

Use:

```text
展示今日需要观察派息或敲出的存续产品。今日日期：{{ todayDate }}
加载中...
今日观察（{{ todayDate }}）
今日共 {{ todayProducts.length }} 个产品需观察
今日无产品需要观察派息/敲出。
自动生成敲出/派息喜报海报。可选择日期查询或生成对应日期的喜报。
喜报操作
筛选日期
生成喜报
重置为今日
{{ filterDate }} 暂无喜报。可点击“生成喜报”为该日期生成。
喜报（{{ filterDate }}）· 共 {{ posterList.length }} 张
敲出喜报
派息喜报
```

- [ ] **Step 4: Fix malformed script strings and status comparisons**

Use these exact replacements:

```js
if (!res.ok) throw new Error(data.error || '加载失败')
if (!res.ok) throw new Error(data.error || '刷新失败')
if (!res.ok) throw new Error(data.error || '生成失败')
successMsg.value = `价格刷新完成，${data.refreshed} 个成功${data.failed ? '，' + data.failed + ' 个失败' : ''}`
successMsg.value = `生成完成：新增 ${data.generated} 条${recalculated ? '，重算 ' + recalculated + ' 条' : ''}`
posterMsg.value = `生成完成：敲出 ${data.knockout} 张，派息 ${data.dividend} 张`
console.log('喜报图片已生成')
```

Status helpers:

```js
function isETF(product) {
  if (!product) return false
  return (product.name && product.name.includes('恒科ETF')) || (product.code && product.code.includes('恒科ETF'))
}

function knockoutClass(status) {
  if (status === '是') return 'result-yes-knockout'
  if (status === '否') return 'result-no'
  if (status === '不观察') return 'result-na'
  return ''
}

function dividendClass(status) {
  if (status === '是') return 'result-yes-dividend'
  if (status === '否') return 'result-no'
  return ''
}
```

- [ ] **Step 5: Keep ProductCompletion-specific styles only**

Keep local styles for these selectors because they describe page-specific behavior:

```css
.tab-bar
.tab-btn
.file-source
.file-badge
.file-from
.sticky-col
.data-row
.chevron
.detail-row
.detail-cell
.detail-label
.detail-empty
.poster-grid
.poster-card
.poster-card-header
.poster-product
.poster-type-badge.knockout
.poster-type-badge.dividend
```

Use global classes for `.desc`, `.panel`, `.panel-title`, `.btn`, `.input`, `.report-panel`, `.section-title`, `.overview-table`, `.empty-state`, `.error`, `.success`, and badges.

- [ ] **Step 6: Build check**

Run:

```powershell
npm.cmd run build
```

from `frontend`.

Expected: no `ProductCompletion.vue` syntax errors.

- [ ] **Step 7: Commit product observation page**

Run:

```powershell
git add frontend\views\ProductCompletion.vue
git commit -m "fix: restore product observation text"
```

Expected: commit includes only `frontend/views/ProductCompletion.vue`.

---

### Task 7: Fix Product Report Page

**Files:**
- Modify: `frontend/views/ProductReport.vue`

- [ ] **Step 1: Update layout props and visible labels**

Use:

```vue
<SubPageLayout
  title="产品运行报告"
  description="指定开始和结束年月，查看产品结构材料、月度产品文档和同步状态。"
  wide
>
```

Visible labels:

```text
参数设置
开始年月
结束年月
数据文件
请输入交易表_修正后.xlsx 路径
生成产品运行报告
报告区域
数据源：
-- 请选择月份 --
从飞书同步
同步中...
最后同步 {{ lastSyncTime }}
正在加载产品结构...
该月份暂无产品结构数据
请选择月份查看产品结构，或点击“从飞书同步”更新数据。
无内容
```

- [ ] **Step 2: Fix script regex and strings**

Use:

```js
const MONTH_PATTERN = /(\d{4})年\s*(\d{1,2})月?/
```

Use:

```js
function getDisplayName(docName) {
  return docName.replace(/^销售物料[:：\s]*/, '')
}
```

Use this `extractMonth()`:

```js
function extractMonth(doc) {
  const path = doc.parent_path || ''
  const match = path.match(MONTH_PATTERN)
  if (match) return `${match[1]}年${parseInt(match[2])}月`

  const nameMatch = doc.doc_name.match(MONTH_PATTERN)
  if (nameMatch) return `${nameMatch[1]}年${parseInt(nameMatch[2])}月`

  return '其他'
}
```

Replace fetch errors and logs:

```js
if (!res.ok) throw new Error('加载失败')
console.log('查询月份:', selectedMonth.value, '返回文档数:', data.length)
const filtered = data.filter(d => d.doc_name.includes('物料'))
console.log('过滤后的物料文档:', filtered.map(d => ({ name: d.doc_name, structured_json: d.structure_json ? '有内容' : '空', raw: d.raw_content?.slice(0, 50) })))
error.value = e.message || '加载产品失败'
console.log('同步结果:', result)
throw new Error(result.error || '同步失败')
alert(`同步成功！共 ${result.doc_count} 个文档，${result.folder_count} 个文件夹`)
console.log('可用月份:', availableMonths.value)
console.log('没有找到任何月份数据，请检查同步日志')
console.error('获取同步状态失败', e)
console.log('数据库中的所有文档数量:', data.length)
console.log('文档名示例:', data.slice(0, 5).map(d => ({ name: d.doc_name, path: d.parent_path })))
console.log('提取月份:', doc.doc_name, '->', month, '| path:', doc.parent_path)
console.log('最终月份列表:', JSON.stringify(availableMonths.value))
console.error('加载月份列表失败:', e)
function run() { alert('产品运行报告功能尚未接入后端。') }
```

Fix sort regex:

```js
const aMatch = a.match(/(\d{4})年(\d+)月/)
const bMatch = b.match(/(\d{4})年(\d+)月/)
```

- [ ] **Step 3: Keep product card styles, align colors to variables**

Update local product card CSS to use variables:

```css
.product-card {
  background: var(--surface);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  overflow: hidden;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.product-card:hover {
  border-color: var(--brand);
  box-shadow: var(--shadow-soft);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 11px 14px;
  color: #fff;
  background: var(--brand);
}
```

- [ ] **Step 4: Build check**

Run:

```powershell
npm.cmd run build
```

from `frontend`.

Expected: no `ProductReport.vue` syntax errors.

- [ ] **Step 5: Commit product report**

Run:

```powershell
git add frontend\views\ProductReport.vue
git commit -m "fix: restore product report text"
```

Expected: commit includes only `frontend/views/ProductReport.vue`.

---

### Task 8: Fix Ongoing Product Page

**Files:**
- Modify: `frontend/views/OngoingProduct.vue`

- [ ] **Step 1: Update layout props and visible labels**

Use:

```vue
<SubPageLayout
  title="存续产品分析"
  description="指定开始和结束年月，查看进场时间、金额、人次、人数及新客/增购/复购分布。"
  wide
>
```

Visible labels:

```text
参数设置
数据文件
点击选择航班交易服务总表.xlsx
开始年月
结束年月
计算中...
生成分析
总览
产品数量（航班）
总金额（万元）
交易人次
客户人数
1. 进场时间分布
统计区间内按进场（航班日期）年月的产品数量、金额、人次。
年月
产品数量
金额（万元）
人次
合计
2. 交易人次与人数
人次为每月交易笔数，人数为每月参与的唯一客户数（按姓名去重）。
人数（去重）
人均笔数
3. 新客 / 增购 / 复购分布
按交易类型统计金额与笔数。
类型
笔数
金额占比
```

- [ ] **Step 2: Fix script comments and string literals**

Replace the sheet selection block with:

```js
const sheetName = wb.SheetNames.includes('交易表') ? '交易表' : wb.SheetNames[0]
```

Replace row comments with readable comments:

```js
// 原始行数据：[航班编号, 姓名, 金额, 类型, 存续状态, year, month]
```

Replace parser errors and validation:

```js
errorMsg.value = '文件解析失败：' + err.message
errorMsg.value = '请选择开始和结束年月'
errorMsg.value = '所选年月范围内无存续数据'
```

Replace business strings:

```js
r[4] !== '完结'
const t = r[3] || '未知'
```

- [ ] **Step 3: Convert local colors to variables**

Keep summary-grid and table-specific CSS, but replace hard-coded beige/orange colors with:

```css
.s-card {
  background: var(--surface-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 14px 16px;
  text-align: center;
}

.s-val {
  color: var(--brand);
}

.data-table td {
  color: var(--ink);
}

.total-row td {
  background: var(--surface-muted);
}
```

- [ ] **Step 4: Build check**

Run:

```powershell
npm.cmd run build
```

from `frontend`.

Expected: no `OngoingProduct.vue` syntax errors.

- [ ] **Step 5: Commit ongoing product**

Run:

```powershell
git add frontend\views\OngoingProduct.vue
git commit -m "fix: restore ongoing product text"
```

Expected: commit includes only `frontend/views/OngoingProduct.vue`.

---

### Task 9: Final Build And Browser Verification

**Files:**
- May modify any frontend file touched in earlier tasks if verification reveals layout defects.

- [ ] **Step 1: Run production build**

Run:

```powershell
npm.cmd run build
```

from `D:\projects\business-workbench\frontend`.

Expected: build succeeds and emits `dist` output.

- [ ] **Step 2: Start local Vite dev server**

Run:

```powershell
npm.cmd run dev -- --host 127.0.0.1 -p 5173
```

from `frontend`.

Expected: Vite serves the app at `http://127.0.0.1:5173/`.

- [ ] **Step 3: Verify desktop routes in browser**

Open these routes:

```text
http://127.0.0.1:5173/
http://127.0.0.1:5173/data-preparation
http://127.0.0.1:5173/user-profile
http://127.0.0.1:5173/product-completion
http://127.0.0.1:5173/product-report
http://127.0.0.1:5173/ongoing-product
```

Expected:

- Top bar is fixed and readable.
- Sidebar is visible on desktop and active route is clear.
- Page heading text is normal Chinese.
- Tables scroll horizontally when wide.
- Buttons and inputs use the blue/gray design system.
- No visible beige/orange dominant theme remains.

- [ ] **Step 4: Verify mobile layout**

Use a viewport around `390 x 844`.

Expected:

- Sidebar is hidden by default.
- Menu button opens sidebar.
- Top bar text does not overlap.
- Page heading and controls fit within the viewport.
- Tables remain horizontally scrollable instead of squeezing columns.

- [ ] **Step 5: Commit verification fixes**

If any visual fixes were needed, run:

```powershell
git add frontend
git commit -m "fix: polish responsive workbench layout"
```

Expected: commit contains only frontend files changed during verification.

If no fixes were needed, do not create an empty commit.

---

## Self-Review

Spec coverage:

- Shared shell: covered by Task 1.
- Global design variables and base controls: covered by Task 2.
- Home redesign: covered by Task 3.
- Visible Chinese text restoration: covered by Tasks 4 through 8.
- Business logic preservation: each page task explicitly keeps existing APIs and calculations.
- System fonts only: covered by Task 2.
- Build and browser verification: covered by Task 9.

Placeholder scan:

- The plan contains no TBD markers.
- The plan does not use undefined component names except `WorkbenchLayout`, which is created in Task 1.
- Commands and expected outcomes are specified for each task.

Type and prop consistency:

- `WorkbenchLayout` and `SubPageLayout` both use `title`, `description`, and `wide`.
- Existing pages can continue importing `SubPageLayout`.
- Route paths match `frontend/router/index.js`.
