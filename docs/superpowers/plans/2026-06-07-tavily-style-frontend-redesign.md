# Tavily-Inspired Frontend Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the Vue/Vite frontend into the approved option-B Tavily-inspired premium business console while preserving existing routes, APIs, and business behavior.

**Architecture:** Keep the current Vue app and route structure. Centralize visual language in `frontend/assets/main.css`, rebuild the application shell around `WorkbenchLayout` and `SidebarNav`, then update pages in functional groups while keeping script logic and API contracts stable.

**Tech Stack:** Vue 3, Vite, Vue Router, `@lucide/vue`, CSS variables, Playwright with local system Chrome, Chrome DevTools MCP.

---

## File Structure

- Modify `package.json` and `package-lock.json`: record `chrome-devtools-mcp` as a root dev dependency.
- Modify `frontend/package.json` and `frontend/package-lock.json`: record `@playwright/test` and `playwright` as frontend dev dependencies.
- Create `frontend/playwright.config.js`: configure local dev-server verification and system Chrome usage.
- Create `frontend/tests/visual-smoke.spec.js`: route-level smoke checks for desktop and mobile layouts.
- Modify `frontend/assets/main.css`: shared design tokens and reusable UI primitives.
- Modify `frontend/components/WorkbenchLayout.vue`: page shell, top bar, drawer state, layout slots.
- Modify `frontend/components/SidebarNav.vue`: Tavily-inspired sidebar, icons, readable Chinese navigation.
- Modify `frontend/components/PanelCard.vue`, `frontend/components/StatCard.vue`, `frontend/components/GlobalSearch.vue`, `frontend/components/ActivityTimeline.vue`, `frontend/components/PosterTemplate.vue`: align reusable components with shared tokens only when they visually conflict.
- Modify `frontend/views/Dashboard.vue`: business overview dashboard.
- Modify `frontend/views/DataPreparation.vue`: Feishu connection and sync console.
- Modify `frontend/views/UserProfile.vue`, `frontend/views/CustomerChurn.vue`, `frontend/views/ChannelAnalysis.vue`, `frontend/views/NominalBuyer.vue`: analysis page pattern.
- Modify `frontend/views/ProductReport.vue`, `frontend/views/OngoingProduct.vue`: product/report page pattern.
- Modify `frontend/views/ProductCompletion.vue`: product observation workspace.
- Modify `frontend/views/ActivityLog.vue`: align with shell/table pattern if visible styling conflicts.

Do not modify backend files.

---

### Task 1: Commit Verification Tooling

**Files:**
- Modify: `package.json`
- Modify: `package-lock.json`
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`

- [ ] **Step 1: Confirm dependency entries**

Run:

```powershell
Get-Content package.json
Get-Content frontend\package.json
```

Expected:

```json
"chrome-devtools-mcp": "^1.1.1"
```

in the root `devDependencies`, and:

```json
"@playwright/test": "^1.60.0",
"playwright": "^1.60.0"
```

in `frontend` `devDependencies`.

- [ ] **Step 2: Verify CLI availability**

Run:

```powershell
npx.cmd chrome-devtools-mcp --version
Set-Location frontend
npx.cmd playwright --version
```

Expected: Chrome DevTools MCP and Playwright print versions without installing new packages.

- [ ] **Step 3: Commit tooling changes**

Run:

```powershell
git add package.json package-lock.json frontend/package.json frontend/package-lock.json
git commit -m "chore: add frontend browser verification tooling"
```

Expected: commit succeeds and contains only package manifest/lockfile changes.

---

### Task 2: Add Playwright Smoke Test Harness

**Files:**
- Create: `frontend/playwright.config.js`
- Create: `frontend/tests/visual-smoke.spec.js`
- Modify: `frontend/package.json`

- [ ] **Step 1: Add frontend test scripts**

Update `frontend/package.json` scripts to include:

```json
"test:visual": "playwright test",
"test:visual:update": "playwright test --update-snapshots"
```

Keep existing `dev`, `build`, and `preview` scripts unchanged.

- [ ] **Step 2: Create Playwright config**

Create `frontend/playwright.config.js`:

```js
import { defineConfig, devices } from '@playwright/test'

const chromePath = 'C:/Program Files/Google/Chrome/Application/chrome.exe'

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  use: {
    baseURL: 'http://127.0.0.1:5176',
    browserName: 'chromium',
    launchOptions: {
      executablePath: chromePath,
    },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: 'npm.cmd run dev -- --host 127.0.0.1 --port 5176',
    url: 'http://127.0.0.1:5176',
    reuseExistingServer: true,
    timeout: 60_000,
  },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1440, height: 900 } } },
    { name: 'mobile', use: { ...devices['Pixel 7'] } },
  ],
})
```

- [ ] **Step 3: Write failing visual smoke tests**

Create `frontend/tests/visual-smoke.spec.js`:

```js
import { expect, test } from '@playwright/test'

const routes = [
  ['Dashboard', '/'],
  ['Data preparation', '/data-preparation'],
  ['Product observation', '/product-completion'],
  ['Product report', '/product-report'],
  ['User profile', '/user-profile'],
  ['Customer churn', '/customer-churn'],
  ['Channel analysis', '/channel-analysis'],
  ['Nominal buyer', '/nominal-buyer'],
  ['Ongoing product', '/ongoing-product'],
]

test.describe('console shell', () => {
  for (const [name, path] of routes) {
    test(`${name} renders shell and readable content`, async ({ page }) => {
      await page.goto(path)

      await expect(page.locator('.workbench-shell')).toBeVisible()
      await expect(page.locator('.workbench-main')).toBeVisible()
      await expect(page.locator('body')).not.toContainText('涓')
      await expect(page.locator('body')).not.toContainText('閿')
    })
  }

  test('mobile sidebar opens and closes', async ({ page, isMobile }) => {
    test.skip(!isMobile, 'mobile-only behavior')

    await page.goto('/')
    await page.getByRole('button', { name: '打开导航' }).click()
    await expect(page.locator('.sidebar')).toHaveClass(/overlay|open/)
    await page.getByRole('link', { name: '数据准备' }).click()
    await expect(page).toHaveURL(/\/data-preparation$/)
  })
})
```

Expected at this point: tests may fail because shell classes, accessible names, or readable Chinese text are not yet implemented. This is the red stage.

- [ ] **Step 4: Run tests to verify red**

Run:

```powershell
Set-Location frontend
npm.cmd run test:visual
```

Expected: fail on missing shell/readable text/mobile labels.

- [ ] **Step 5: Commit the test harness**

Run:

```powershell
git add frontend/package.json frontend/playwright.config.js frontend/tests/visual-smoke.spec.js
git commit -m "test: add frontend visual smoke coverage"
```

Expected: commit succeeds even though the new tests are currently red.

---

### Task 3: Establish Shared Design System

**Files:**
- Modify: `frontend/assets/main.css`

- [ ] **Step 1: Replace root tokens**

Set the top of `frontend/assets/main.css` to include these tokens:

```css
:root {
  --bg-page: #fbfaf5;
  --bg-sidebar: #f6f4ee;
  --bg-surface: #ffffff;
  --bg-surface-muted: #f7f8f3;
  --bg-hover: #eef3f7;
  --ink-strong: #202124;
  --ink: #353330;
  --ink-soft: #6f7785;
  --ink-faint: #9aa5b5;
  --border: #e3e1d8;
  --border-soft: #ece9df;
  --brand: #2f7df6;
  --brand-hover: #1f66d8;
  --brand-soft: #e8f1ff;
  --success: #00a875;
  --success-soft: #e7f7ef;
  --warning: #b7791f;
  --warning-soft: #fff6df;
  --danger: #c24136;
  --danger-soft: #fff0ee;
  --shadow-soft: 0 18px 45px rgba(35, 43, 57, 0.10);
  --shadow-card: 0 10px 28px rgba(35, 43, 57, 0.07);
  --radius-sm: 6px;
  --radius: 8px;
  --radius-lg: 14px;
  --sidebar-width: 272px;
  --topbar-height: 68px;
  --font-sans: "Inter", "Punctuation SC", ui-sans-serif, system-ui, -apple-system,
    BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei",
    "Noto Sans SC", sans-serif;
  --font-mono: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco,
    Consolas, "Liberation Mono", monospace;
}
```

- [ ] **Step 2: Add shared primitives**

Ensure the file defines these classes with stable dimensions and no nested-card styling:

```css
.btn {}
.btn-primary {}
.btn-secondary {}
.btn-outline {}
.btn-sm {}
.icon-btn {}
.panel {}
.panel-header {}
.panel-title {}
.metric-card {}
.toolbar {}
.input {}
.textarea {}
.select {}
.tab-bar {}
.tab-btn {}
.table-wrap {}
.data-table {}
.badge {}
.badge-green {}
.badge-blue {}
.badge-amber {}
.badge-red {}
.empty-state {}
.loading-state {}
.error-msg {}
.success-msg {}
.hint {}
```

Use `border-radius: var(--radius)` for repeated cards and panels. Use `border-radius: 999px` only for badges, search fields, and pill actions.

- [ ] **Step 3: Add responsive safeguards**

Add:

```css
@media (max-width: 960px) {
  .workbench-content,
  .workbench-frame {
    margin-left: 0;
  }
}

@media (max-width: 640px) {
  .toolbar,
  .form-row,
  .sync-actions,
  .search-actions,
  .actions {
    align-items: stretch;
    flex-direction: column;
  }

  .btn,
  .input,
  .select {
    width: 100%;
  }
}
```

- [ ] **Step 4: Run palette scan**

Run:

```powershell
rg -n "#(8B7355|D97757|6B5C4E|E8DDD0|F5F0E8|A8967E|C4B5A5|FAF7F4|F0EAE0)" frontend\assets\main.css
```

Expected: no matches.

- [ ] **Step 5: Build-check and commit**

Run:

```powershell
Set-Location frontend
npm.cmd run build
git add frontend/assets/main.css
git commit -m "style: establish premium console design system"
```

Expected: build passes and commit succeeds.

---

### Task 4: Rebuild Shell And Navigation

**Files:**
- Modify: `frontend/components/WorkbenchLayout.vue`
- Modify: `frontend/components/SidebarNav.vue`

- [ ] **Step 1: Update shell props and state**

`WorkbenchLayout.vue` should expose:

```js
defineProps({
  title: { type: String, default: '业务工作台' },
  description: { type: String, default: '' },
  wide: { type: Boolean, default: false },
})
```

Keep global search support if it already exists. Use mobile drawer state with `sidebarOverlay`.

- [ ] **Step 2: Implement shell template classes**

The rendered shell must include:

```vue
<div class="workbench-shell">
  <SidebarNav
    :collapsed="sidebarCollapsed"
    :overlay-open="sidebarOverlay"
    @navigate="closeSidebar"
    @close="sidebarOverlay = false"
  />
  <div class="workbench-content" :class="{ expanded: sidebarCollapsed }">
    <header class="workbench-topbar">...</header>
    <main class="workbench-main" :class="{ wide }">...</main>
  </div>
  <GlobalSearch v-model:open="searchOpen" />
</div>
```

The mobile menu button accessible name must be `打开导航` so the Playwright smoke test can find it.

- [ ] **Step 3: Replace sidebar visible text**

Use these readable Chinese labels in `SidebarNav.vue`:

```js
const navItems = [
  { path: '/', title: '工作台首页', desc: '业务概览与快捷入口', icon: LayoutDashboard },
  { path: '/data-preparation', title: '数据准备', desc: '飞书连接与数据同步', icon: Database },
  { path: '/user-profile', title: '用户画像', desc: '合投用户特征查询', icon: UserRound },
  { path: '/customer-churn', title: '客户流失', desc: '完结未复购客户识别', icon: UserX },
  { path: '/product-report', title: '产品报告', desc: '产品运行材料查询', icon: FileText },
  { path: '/product-completion', title: '产品观察', desc: '派息与敲出观察', icon: Eye },
  { path: '/ongoing-product', title: '存续分析', desc: '仍在持有产品分析', icon: BarChart3 },
  { path: '/channel-analysis', title: '渠道分析', desc: '渠道成交表现', icon: PieChart },
  { path: '/nominal-buyer', title: '名义购买人', desc: '管理人关系匹配', icon: Users },
]
```

Use brand text:

```text
业务工作台
航班服务数据控制台
```

- [ ] **Step 4: Run tests to verify shell progress**

Run:

```powershell
Set-Location frontend
npm.cmd run test:visual -- --project=desktop
```

Expected: desktop tests should no longer fail because `.workbench-shell` or `.workbench-main` is missing. Remaining failures may be readable-text failures on pages not yet cleaned.

- [ ] **Step 5: Commit shell**

Run:

```powershell
git add frontend/components/WorkbenchLayout.vue frontend/components/SidebarNav.vue frontend/assets/main.css
git commit -m "feat: rebuild premium console shell"
```

Expected: commit succeeds.

---

### Task 5: Redesign Dashboard

**Files:**
- Modify: `frontend/views/Dashboard.vue`

- [ ] **Step 1: Keep route and layout**

Confirm `frontend/router/index.js` still maps `/` to `Dashboard.vue`. Do not rename the route.

- [ ] **Step 2: Replace dashboard visible text and data**

Use module labels:

```js
const statusCards = [
  { label: '数据准备', value: '飞书同步', detail: '先完成交易总表与合投用户表同步', path: '/data-preparation' },
  { label: '产品观察', value: '派息 / 敲出', detail: '查看今日观察、观察日历与喜报', path: '/product-completion' },
  { label: '产品材料', value: '运行报告', detail: '同步并查询产品运行材料', path: '/product-report' },
  { label: '客户分析', value: '画像 / 流失', detail: '查询客户特征与复购状态', path: '/user-profile' },
]
```

Use page title `业务工作台` and description `统一管理航班服务数据准备、客户分析、产品报告、存续观察和渠道统计。`

- [ ] **Step 3: Apply dashboard layout**

Use these page-specific classes:

```css
.dashboard-hero {}
.metric-grid {}
.dashboard-grid {}
.module-list {}
.module-row {}
.workflow-list {}
.workflow-step {}
```

Do not create card elements inside another decorative card. `module-row` and `workflow-step` may be repeated items inside one panel.

- [ ] **Step 4: Run route smoke test**

Run:

```powershell
Set-Location frontend
npm.cmd run test:visual -- --grep "Dashboard"
```

Expected: dashboard route test passes for shell and readable text.

- [ ] **Step 5: Commit dashboard**

Run:

```powershell
git add frontend/views/Dashboard.vue
git commit -m "feat: redesign business dashboard"
```

Expected: commit succeeds.

---

### Task 6: Redesign Data Preparation

**Files:**
- Modify: `frontend/views/DataPreparation.vue`

- [ ] **Step 1: Preserve exact endpoints**

Before editing, record endpoint lines:

```powershell
rg -n "fetch\\(" frontend\views\DataPreparation.vue
```

Expected endpoints include:

```js
fetch('/api/auth/status')
fetch('/api/db/sync-status')
fetch('/api/db/sync-coinvest-status')
fetch('/api/auth/login')
fetch('/api/auth/logout', { method: 'DELETE' })
fetch('/api/db/sync', { method: 'POST' })
fetch('/api/db/sync-coinvest', { method: 'POST' })
```

- [ ] **Step 2: Replace visible text**

Use page title `数据准备` and description `连接飞书账号，同步航班服务交易总表和合投用户表到本地数据库。`

Use labels:

```text
飞书账号连接
已连接
未连接
断开连接
连接飞书账号
数据源同步
航班服务交易总表
合投用户表
上次同步：
尚未同步
已就绪
同步中...
同步数据
请先连接飞书账号
```

- [ ] **Step 3: Normalize source cards**

Represent the data sources as sibling `.source-card` items inside `.source-grid`. Each card should show icon, title, sync metadata, badge, and action button.

- [ ] **Step 4: Confirm endpoints unchanged**

Run:

```powershell
rg -n "fetch\\(" frontend\views\DataPreparation.vue
```

Expected: same endpoint paths and methods as Step 1.

- [ ] **Step 5: Build and commit**

Run:

```powershell
Set-Location frontend
npm.cmd run build
git add frontend/views/DataPreparation.vue
git commit -m "feat: redesign data preparation console"
```

Expected: build passes and commit succeeds.

---

### Task 7: Normalize Analysis Pages

**Files:**
- Modify: `frontend/views/UserProfile.vue`
- Modify: `frontend/views/CustomerChurn.vue`
- Modify: `frontend/views/ChannelAnalysis.vue`
- Modify: `frontend/views/NominalBuyer.vue`

- [ ] **Step 1: Record existing fetch calls**

Run:

```powershell
rg -n "fetch\\(" frontend\views\UserProfile.vue frontend\views\CustomerChurn.vue frontend\views\ChannelAnalysis.vue frontend\views\NominalBuyer.vue
```

Expected: record current endpoints before editing. Do not change endpoint paths or methods.

- [ ] **Step 2: Set readable page headings**

Use:

```text
用户画像
按购买人、专户、竞品和行业筛选合投用户特征。
客户流失
识别产品完结后未复购的客户，并查看相关交易线索。
渠道分析
统计渠道成交人数、成交金额和复购表现。
名义购买人
查询名义购买人与私募管理人的匹配关系。
```

- [ ] **Step 3: Normalize filter sections**

Each page should use:

```vue
<div class="panel">
  <div class="panel-header">
    <h3 class="panel-title">筛选条件</h3>
    <span class="hint">数据来自本地同步结果</span>
  </div>
  <div class="toolbar">
    <!-- existing controls and buttons -->
  </div>
</div>
```

Preserve existing `v-model`, computed values, and event handlers.

- [ ] **Step 4: Normalize result sections**

Each result table should use:

```vue
<div class="panel">
  <div class="table-wrap">
    <table class="data-table">
      <!-- existing headers and rows -->
    </table>
  </div>
</div>
```

Empty states should use `class="empty-state"` with readable Chinese text.

- [ ] **Step 5: Confirm fetch calls unchanged**

Run the same `rg -n "fetch\\(" ...` command from Step 1.

Expected: endpoint paths and methods are unchanged.

- [ ] **Step 6: Build, smoke-test, and commit**

Run:

```powershell
Set-Location frontend
npm.cmd run build
npm.cmd run test:visual -- --grep "User profile|Customer churn|Channel analysis|Nominal buyer"
git add frontend/views/UserProfile.vue frontend/views/CustomerChurn.vue frontend/views/ChannelAnalysis.vue frontend/views/NominalBuyer.vue
git commit -m "feat: normalize analysis interfaces"
```

Expected: build passes; the four smoke tests pass for shell/readable text; commit succeeds.

---

### Task 8: Normalize Product Report And Ongoing Product

**Files:**
- Modify: `frontend/views/ProductReport.vue`
- Modify: `frontend/views/OngoingProduct.vue`

- [ ] **Step 1: Record existing fetch calls**

Run:

```powershell
rg -n "fetch\\(" frontend\views\ProductReport.vue frontend\views\OngoingProduct.vue
```

Expected: record current endpoints and methods before editing.

- [ ] **Step 2: Set readable page headings**

Use:

```text
产品报告
同步并查看产品运行材料，支持按产品和文件夹定位报告内容。
存续分析
分析仍在持有产品的金额、人数、类型和管理人分布。
```

- [ ] **Step 3: Normalize action panels and tables**

Use:

```vue
<div class="panel">
  <div class="panel-header">
    <h3 class="panel-title">筛选与操作</h3>
    <span class="hint">数据来自本地同步结果</span>
  </div>
  <div class="toolbar">
    <!-- existing controls -->
  </div>
</div>
```

Use `.table-wrap` and `.data-table` for wide result tables.

- [ ] **Step 4: Confirm fetch calls unchanged**

Run:

```powershell
rg -n "fetch\\(" frontend\views\ProductReport.vue frontend\views\OngoingProduct.vue
```

Expected: same endpoint paths and methods as Step 1.

- [ ] **Step 5: Build, smoke-test, and commit**

Run:

```powershell
Set-Location frontend
npm.cmd run build
npm.cmd run test:visual -- --grep "Product report|Ongoing product"
git add frontend/views/ProductReport.vue frontend/views/OngoingProduct.vue
git commit -m "feat: normalize product report interfaces"
```

Expected: build passes; relevant smoke tests pass; commit succeeds.

---

### Task 9: Redesign Product Observation Workspace

**Files:**
- Modify: `frontend/views/ProductCompletion.vue`
- Modify: `frontend/components/PosterTemplate.vue` only if visual conflict remains.

- [ ] **Step 1: Record existing fetch calls**

Run:

```powershell
rg -n "fetch\\(" frontend\views\ProductCompletion.vue
```

Expected: record observation, refresh, generation, calendar, today, and poster endpoints.

- [ ] **Step 2: Set readable page text**

Use title `产品派息 / 敲出观察` and description `展示存续产品的派息与敲出观察情况，并生成对应日期的喜报海报。`

Use tab labels:

```text
全量
观察日历
今日观察
喜报
```

Use primary action labels:

```text
刷新标的价格
生成观察记录
搜索
按产品名称或航班编号搜索...
```

- [ ] **Step 3: Normalize workspace structure**

Use shared classes:

```vue
<div class="tab-bar">...</div>
<div class="panel">...</div>
<div class="toolbar">...</div>
<div class="table-wrap">
  <table class="data-table">...</table>
</div>
```

Keep page-specific classes only for:

```css
.calendar-panel {}
.calendar-weekdays {}
.calendar-grid {}
.calendar-cell {}
.calendar-product {}
.poster-grid {}
.poster-card {}
.poster-card-header {}
```

- [ ] **Step 4: Preserve status logic**

Keep existing comparisons for whether a product has knocked out or paid interest. If corrupted display strings prevent correct comparison, normalize both display text and comparison values to readable Chinese:

```js
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

- [ ] **Step 5: Confirm fetch calls unchanged**

Run:

```powershell
rg -n "fetch\\(" frontend\views\ProductCompletion.vue
```

Expected: endpoint paths and methods are unchanged from Step 1.

- [ ] **Step 6: Build, smoke-test, and commit**

Run:

```powershell
Set-Location frontend
npm.cmd run build
npm.cmd run test:visual -- --grep "Product observation"
git add frontend/views/ProductCompletion.vue frontend/components/PosterTemplate.vue
git commit -m "feat: redesign product observation workspace"
```

Expected: build passes; smoke test passes; commit succeeds. If `PosterTemplate.vue` was not changed, omit it from `git add`.

---

### Task 10: Align Remaining Reusable Components And Activity Log

**Files:**
- Modify: `frontend/components/PanelCard.vue`
- Modify: `frontend/components/StatCard.vue`
- Modify: `frontend/components/GlobalSearch.vue`
- Modify: `frontend/components/ActivityTimeline.vue`
- Modify: `frontend/views/ActivityLog.vue`

- [ ] **Step 1: Inspect class conflicts**

Run:

```powershell
rg -n "#(8B7355|D97757|6B5C4E|E8DDD0|F5F0E8|A8967E|C4B5A5|FAF7F4|F0EAE0)|border-radius:\\s*(1[2-9]|[2-9][0-9])px" frontend\components frontend\views\ActivityLog.vue
```

Expected: list any legacy colors or oversized repeated-card radii.

- [ ] **Step 2: Normalize reusable components**

Use shared tokens only:

```css
background: var(--bg-surface);
border: 1px solid var(--border-soft);
border-radius: var(--radius);
box-shadow: var(--shadow-card);
color: var(--ink);
```

Do not create new visual systems inside these components.

- [ ] **Step 3: Build and commit**

Run:

```powershell
Set-Location frontend
npm.cmd run build
git add frontend/components/PanelCard.vue frontend/components/StatCard.vue frontend/components/GlobalSearch.vue frontend/components/ActivityTimeline.vue frontend/views/ActivityLog.vue
git commit -m "style: align shared console components"
```

Expected: build passes and commit succeeds. If a listed file was inspected but not changed, omit it from `git add`.

---

### Task 11: Full Verification And Polish

**Files:**
- Modify frontend files only if verification finds concrete issues.

- [ ] **Step 1: Run production build**

Run:

```powershell
Set-Location frontend
npm.cmd run build
```

Expected: command exits 0 and Vite reports built assets.

- [ ] **Step 2: Run visual smoke tests**

Run:

```powershell
Set-Location frontend
npm.cmd run test:visual
```

Expected: desktop and mobile projects pass.

- [ ] **Step 3: Search for corrupted text signatures**

Run:

```powershell
rg -n "涓|閿|閻|娑|妞|棣|鍧|鐢" frontend
```

Expected: no matches in visible UI strings. Inspect any matches manually before changing them because legitimate data comments or binary-like artifacts may not be user-visible.

- [ ] **Step 4: Search for legacy palette**

Run:

```powershell
rg -n "#(8B7355|D97757|6B5C4E|E8DDD0|F5F0E8|A8967E|C4B5A5|FAF7F4|F0EAE0)" frontend
```

Expected: no matches.

- [ ] **Step 5: Capture route screenshots**

Run:

```powershell
Set-Location frontend
node -e "const { chromium } = require('playwright'); const routes=['/','/data-preparation','/product-completion','/product-report','/user-profile']; (async()=>{ const browser=await chromium.launch({ executablePath:'C:/Program Files/Google/Chrome/Application/chrome.exe', headless:true }); for (const route of routes) { const page=await browser.newPage({ viewport:{ width:1440,height:900 } }); await page.goto('http://127.0.0.1:5176'+route); await page.screenshot({ path:'../verify-'+(route==='/'?'home':route.slice(1).replaceAll('/','-'))+'.png', fullPage:false }); await page.close(); } await browser.close(); })().catch(e=>{ console.error(e); process.exit(1); })"
```

Expected: screenshots are generated at repo root for manual visual review.

- [ ] **Step 6: Commit verification fixes**

If any files changed during verification:

```powershell
git add frontend
git commit -m "fix: polish frontend redesign verification issues"
```

Expected: commit succeeds. Skip this step if verification required no code changes.

---

## Self-Review

- Spec coverage: tasks cover verification tooling, design system, shell/navigation, dashboard, data preparation, analysis pages, product/report pages, product observation, reusable components, readable Chinese text, route smoke checks, mobile drawer behavior, wide tables, and palette cleanup.
- Scope control: all implementation tasks are frontend-only except package tooling already approved for verification.
- Completeness scan: no unresolved drafting markers remain.
- Type consistency: class names used in tests and tasks match planned shell/shared CSS names.
- Risk: several pages may have deeply corrupted existing Chinese text. The plan requires preserving API logic and only normalizing visible text and template structure unless broken string values affect display classes.
