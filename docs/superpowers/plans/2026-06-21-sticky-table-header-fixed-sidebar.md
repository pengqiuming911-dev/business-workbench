# Sticky Table Headers + Fixed Sidebar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make table headers sticky within bounded table containers across all data table pages, and ensure the sidebar stays fixed during page scrolling.

**Architecture:** Establish a Flexbox layout chain from `.workbench-shell` → `.workbench-content` → `.workbench-main` → page root → `.table-wrap`. The chain gives `.table-wrap` a bounded height so `position: sticky` on `<th>` elements attaches to the table container instead of the viewport. Non-table pages get `flex: 1; overflow-y: auto` on their root to maintain independent scrolling.

**Tech Stack:** Vue 3 (scoped CSS), CSS Flexbox, existing `position: sticky` on table headers.

## Global Constraints

- The global `.data-table th` already has `position: sticky; top: 0; z-index: 4` — do NOT add or modify sticky positioning.
- Rebate tables have two-row grouped headers with `top: 0` and `top: 44px` — do NOT change these values.
- All table pages use `min-width` on their table class (3200–3600px) for horizontal scrolling — do NOT remove these.
- Pages with `:deep(.workbench-main) { max-width: none; }` must keep that rule.

---

## File Map

| File | Responsibility |
|---|---|
| `frontend/components/WorkbenchLayout.vue` | Shell / content / main flex chain |
| `frontend/assets/main.css` | Global `.table-wrap` flex properties |
| `frontend/components/SidebarNav.vue` | Sidebar inner overflow safety |
| `frontend/views/Dashboard.vue` | Non-table page root flex |
| `frontend/views/DataPreparation.vue` | Non-table page root flex |
| `frontend/views/ProductReport.vue` | Non-table page root flex |
| `frontend/views/PushSettings.vue` | Non-table page root flex |
| `frontend/views/ActivityLog.vue` | Non-table page root flex |
| `frontend/views/UserProfile.vue` | Non-table page root flex |
| `frontend/views/AgentChat.vue` | Non-table page root flex (special: root is WorkbenchLayout) |
| `frontend/views/HoldingAnalysis.vue` | Tab container flex chain |
| `frontend/views/RebateAnalysis.vue` | Tab container flex chain |
| `frontend/views/ProductAnalysis.vue` | Table page: add root class + flex + shrink children |
| `frontend/views/CustomerHolding.vue` | Table page: add root class + flex + shrink children |
| `frontend/views/RebatePending.vue` | Table page flex + shrink children |
| `frontend/views/RebateCompleted.vue` | Table page flex + shrink children |
| `frontend/views/ProductCompletion.vue` | Table page flex root + max-height fallback |

---

### Task 1: Global Flex Chain + Table Wrap Styles

**Files:**
- Modify: `frontend/components/WorkbenchLayout.vue` (scoped CSS, lines 75–130)
- Modify: `frontend/assets/main.css` (line 274–280)

**Interfaces:**
- Produces: `.workbench-shell` with `height: 100vh; display: flex` — all pages become flex children
- Produces: `.table-wrap` with `flex: 1; min-height: 0; overflow-y: auto` — bounded scroll container for tables

- [ ] **Step 1: Update WorkbenchLayout.vue scoped CSS**

Replace the existing `.workbench-shell`, `.workbench-content`, and `.workbench-main` rules in the `<style scoped>` block:

```css
.workbench-shell {
  height: 100vh;
  display: flex;
}

.workbench-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-left: 208px;
  transition: margin-left 220ms ease;
}

.workbench-content.expanded {
  margin-left: 64px;
}

.workbench-topbar {
  flex-shrink: 0;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 8px 32px 4px;
  border-bottom: 1px solid var(--border-soft);
  background: var(--bg-page);
}

.workbench-main {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  width: 100%;
  max-width: 1680px;
  padding: 24px 32px 72px;
  margin: 0 auto;
  box-sizing: border-box;
}
```

Key changes from the original:
- `.workbench-shell`: `min-height: 100vh` → `height: 100vh; display: flex`
- `.workbench-content`: added `flex: 1; min-height: 0; display: flex; flex-direction: column; overflow: hidden`
- `.workbench-topbar`: added `flex-shrink: 0`
- `.workbench-main`: added `flex: 1; min-height: 0; display: flex; flex-direction: column; overflow: hidden`

- [ ] **Step 2: Update .table-wrap in main.css**

In `frontend/assets/main.css`, update the `.table-wrap` rule (around line 274):

```css
.table-wrap {
  position: relative;
  overflow-x: auto;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: #fff;
}
```

Changes from original: added `overflow-y: auto; flex: 1; min-height: 0;`

- [ ] **Step 3: Visual check — open the app**

Run: `cd D:\projects\business-workbench && npm run dev`

Expected: The app loads but pages may appear clipped or non-scrollable. This is expected — page-level flex styles are added in Tasks 2–6.

- [ ] **Step 4: Commit**

```bash
git add frontend/components/WorkbenchLayout.vue frontend/assets/main.css
git commit -m "style(layout): establish flex chain from shell to table-wrap"
```

---

### Task 2: Sidebar Overflow Safety

**Files:**
- Modify: `frontend/components/SidebarNav.vue` (scoped CSS, line 99)

**Interfaces:**
- Consumes: nothing from prior tasks
- Produces: `.sidebar-inner` with `overflow-y: auto` for small viewports

- [ ] **Step 1: Add overflow-y to sidebar-inner**

In `frontend/components/SidebarNav.vue`, update the `.sidebar-inner` rule in the `<style scoped>` block (around line 98):

```css
.sidebar-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px 12px;
  overflow-y: auto;
}
```

Change from original: added `overflow-y: auto`.

- [ ] **Step 2: Visual check**

Resize browser to a short viewport height (e.g., 500px). The sidebar nav items should scroll if they overflow, not get cut off.

- [ ] **Step 3: Commit**

```bash
git add frontend/components/SidebarNav.vue
git commit -m "style(sidebar): add overflow-y: auto to sidebar-inner"
```

---

### Task 3: Non-Table Pages Root Flex

**Files:**
- Modify: `frontend/views/Dashboard.vue`
- Modify: `frontend/views/DataPreparation.vue`
- Modify: `frontend/views/ProductReport.vue`
- Modify: `frontend/views/PushSettings.vue`
- Modify: `frontend/views/ActivityLog.vue`
- Modify: `frontend/views/UserProfile.vue`
- Modify: `frontend/views/AgentChat.vue`

**Interfaces:**
- Consumes: flex chain from Task 1
- Produces: each non-table page root gets `flex: 1; min-height: 0; overflow-y: auto` so they scroll independently

For each file below, add the CSS rule to the existing `<style scoped>` block. If the file has no `<style scoped>` block, add one.

- [ ] **Step 1: Dashboard.vue**

Add to `<style scoped>`:

```css
.dashboard-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

- [ ] **Step 2: DataPreparation.vue**

Add to `<style scoped>`:

```css
.data-preparation-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

- [ ] **Step 3: ProductReport.vue**

Add to `<style scoped>`:

```css
.product-report-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

- [ ] **Step 4: PushSettings.vue**

Add to `<style scoped>`:

```css
.push-settings-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

- [ ] **Step 5: ActivityLog.vue**

Add to `<style scoped>`:

```css
.activity-log-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

- [ ] **Step 6: UserProfile.vue**

Add to `<style scoped>`:

```css
.user-profile-page {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

- [ ] **Step 7: AgentChat.vue**

AgentChat's root template element is `<WorkbenchLayout>`, not a `<div>`. Add to `<style scoped>`:

```css
:deep(.workbench-content) {
  flex: 1;
  min-height: 0;
}

.agent-shell {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
```

Note: The `:deep(.workbench-content)` targets AgentChat's inner WorkbenchLayout's content area. The `.agent-shell` class is on the first `<div>` inside AgentChat.

- [ ] **Step 8: Visual check — non-table pages**

Navigate to each non-table page (Dashboard, DataPreparation, ProductReport, PushSettings, ActivityLog, UserProfile, Agent). Verify:
- Page content scrolls normally when it overflows the viewport
- Sidebar remains fixed during scrolling
- No content is clipped or hidden

- [ ] **Step 9: Commit**

```bash
git add frontend/views/Dashboard.vue frontend/views/DataPreparation.vue frontend/views/ProductReport.vue frontend/views/PushSettings.vue frontend/views/ActivityLog.vue frontend/views/UserProfile.vue frontend/views/AgentChat.vue
git commit -m "style(views): add flex root to non-table pages for independent scroll"
```

---

### Task 4: HoldingAnalysis + ProductAnalysis + CustomerHolding

**Files:**
- Modify: `frontend/views/HoldingAnalysis.vue` (scoped CSS, line 61)
- Modify: `frontend/views/ProductAnalysis.vue` (template line 2 + scoped CSS, line 328)
- Modify: `frontend/views/CustomerHolding.vue` (template line 2 + scoped CSS)

**Interfaces:**
- Consumes: flex chain from Task 1, `.table-wrap` flex from Task 1
- Produces: flex chain from `.workbench-main` → tab container → child page → `.table-wrap`

- [ ] **Step 1: HoldingAnalysis.vue scoped CSS**

Replace the existing `<style scoped>` block content:

```css
:deep(.workbench-main) {
  max-width: none;
}

.holding-analysis-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.holding-analysis-page > .page-header {
  flex-shrink: 0;
}

.holding-analysis-page > .tab-bar {
  flex-shrink: 0;
}
```

Changes: kept `:deep(.workbench-main) { max-width: none; }`, added `.holding-analysis-page` flex rules.

- [ ] **Step 2: ProductAnalysis.vue — add root class + flex CSS**

First, add a class to the root `<div>` in the template (line 2). Change:
```html
<template>
  <div>
```
To:
```html
<template>
  <div class="product-analysis-page">
```

Then add these rules to the existing `<style scoped>` block. Add at the top:

```css
:deep(.workbench-main) {
  max-width: none;
}

.product-analysis-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.product-analysis-page > .filter-bar {
  flex-shrink: 0;
}

.product-analysis-page > .advanced-toggle {
  flex-shrink: 0;
}

.product-analysis-page > .advanced-bar {
  flex-shrink: 0;
}

.product-analysis-page > .pagination {
  flex-shrink: 0;
}
```

Note: The existing `.filter-bar`, `.advanced-toggle`, `.advanced-bar`, `.pagination` rules in scoped CSS remain unchanged — the `flex-shrink: 0` is added via the new `.product-analysis-page >` selectors above.

- [ ] **Step 3: CustomerHolding.vue — add root class + flex CSS**

First, add a class to the root `<div>` in the template (line 2). Change:
```html
<template>
  <div>
```
To:
```html
<template>
  <div class="customer-holding-page">
```

Then add these rules to the existing `<style scoped>` block:

```css
:deep(.workbench-main) {
  max-width: none;
}

.customer-holding-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.customer-holding-page > .filter-bar {
  flex-shrink: 0;
}

.customer-holding-page > .advanced-toggle {
  flex-shrink: 0;
}

.customer-holding-page > .advanced-bar {
  flex-shrink: 0;
}

.customer-holding-page > .update-hint {
  flex-shrink: 0;
}

.customer-holding-page > .pagination {
  flex-shrink: 0;
}
```

- [ ] **Step 4: Visual check — Product Analysis tab**

Navigate to 产品&持仓 → 产品分析 tab. Verify:
- Filter bar is visible at top, does not scroll
- Table fills remaining vertical space
- Scrolling inside the table: header stays fixed at top of table area
- Pagination is visible at the bottom, does not scroll with table
- Horizontal scrolling still works (scroll right to see all columns)
- Sticky first columns stay in place during horizontal scroll

- [ ] **Step 5: Visual check — Customer Holding tab**

Switch to 客户持有分析 tab. Verify the same behaviors as Step 4.

- [ ] **Step 6: Commit**

```bash
git add frontend/views/HoldingAnalysis.vue frontend/views/ProductAnalysis.vue frontend/views/CustomerHolding.vue
git commit -m "style(holding): flex chain for sticky table headers in product/customer tabs"
```

---

### Task 5: RebateAnalysis + RebatePending

**Files:**
- Modify: `frontend/views/RebateAnalysis.vue` (scoped CSS, line 61)
- Modify: `frontend/views/RebatePending.vue` (scoped CSS, line 809)

**Interfaces:**
- Consumes: flex chain from Task 1
- Produces: flex chain for rebate pending tab with two-row grouped sticky headers

- [ ] **Step 1: RebateAnalysis.vue scoped CSS**

Replace the existing `<style scoped>` block content:

```css
:deep(.workbench-main) {
  max-width: none;
}

.rebate-analysis-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.rebate-analysis-page > .page-header {
  flex-shrink: 0;
}

.rebate-analysis-page > .tab-bar {
  flex-shrink: 0;
}
```

- [ ] **Step 2: RebatePending.vue scoped CSS**

Add/update these rules in the existing `<style scoped>` block:

Add at the top (if not already present):

```css
.rebate-pending-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
```

Add `flex-shrink: 0` to each of these existing rules:
- `.action-bar`: add `flex-shrink: 0;`
- `.batch-panel`: add `flex-shrink: 0;`
- `.pagination`: add `flex-shrink: 0;` (the `.pagination` rule may not exist in scoped styles — if so, add it)

If `.pagination` does not exist in scoped styles, add:
```css
.pagination {
  flex-shrink: 0;
}
```

Also ensure the page-header and filter-bar don't shrink. The page-header uses the global `.page-header` class and the filter-bar uses global `.filter-bar` — they are already `flex-shrink: 0` by default in a flex context (default flex-shrink is 1, but since these have explicit heights from padding/content, they should be fine). To be safe, add:

```css
.rebate-pending-page > .page-header {
  flex-shrink: 0;
}

.rebate-pending-page > .filter-bar {
  flex-shrink: 0;
}
```

- [ ] **Step 3: Visual check — Rebate Pending tab**

Navigate to 返费 → 待返费分析 tab. Verify:
- Page header (if not embedded), filter bar, and action bar stay fixed at top
- Two-row grouped table header stays sticky at top of table area when scrolling
- First sticky column stays in place during horizontal scroll
- Pagination stays visible at the bottom

- [ ] **Step 4: Commit**

```bash
git add frontend/views/RebateAnalysis.vue frontend/views/RebatePending.vue
git commit -m "style(rebate): flex chain for sticky grouped headers in pending tab"
```

---

### Task 6: RebateCompleted

**Files:**
- Modify: `frontend/views/RebateCompleted.vue` (scoped CSS, line 1250)

**Interfaces:**
- Consumes: flex chain from Task 1
- Produces: flex chain for rebate completed tab (no pagination)

- [ ] **Step 1: RebateCompleted.vue scoped CSS**

Add/update these rules in the existing `<style scoped>` block:

Add at the top:

```css
.rebate-completed-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.rebate-completed-page > .page-header {
  flex-shrink: 0;
}

.rebate-completed-page > .filter-bar {
  flex-shrink: 0;
}

.rebate-completed-page > .action-bar {
  flex-shrink: 0;
}
```

Note: RebateCompleted has no pagination, so `.table-wrap` is the last flex child and fills all remaining space.

- [ ] **Step 2: Visual check — Rebate Completed tab**

Switch to 已返费分析 tab. Verify:
- Filter bar and action bar stay fixed at top
- Two-row grouped table header stays sticky at top of table area
- Table fills remaining vertical space (no pagination at bottom)
- Horizontal scroll and sticky columns work correctly

- [ ] **Step 3: Commit**

```bash
git add frontend/views/RebateCompleted.vue
git commit -m "style(rebate): flex chain for sticky headers in completed tab"
```

---

### Task 7: ProductCompletion

**Files:**
- Modify: `frontend/views/ProductCompletion.vue` (scoped CSS, line 568)

**Interfaces:**
- Consumes: flex chain from Task 1
- Produces: flex root for ProductCompletion page. Tables are inside PanelCard components which break the flex chain, so we use `max-height` fallback for table sticky headers.

- [ ] **Step 1: ProductCompletion.vue scoped CSS**

Add/update these rules in the existing `<style scoped>` block:

Add at the top:

```css
.product-completion-page {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.product-completion-page > .page-header {
  flex-shrink: 0;
}

.product-completion-page > .tab-bar {
  flex-shrink: 0;
}
```

For the tables inside PanelCard (which break the flex chain), add a max-height fallback:

```css
.panel-card .table-wrap {
  flex: none;
  max-height: 65vh;
}
```

This ensures `.table-wrap` inside PanelCard:
- Does NOT use `flex: 1` (no flex container parent to work with)
- Gets a bounded `max-height: 65vh` so sticky headers attach
- The `overflow-y: auto` from the global `.table-wrap` rule still applies

- [ ] **Step 2: Visual check — ProductCompletion tabs**

Navigate to 观察日历 page. Check each tab:

**全量 tab:**
- Operations panel stays visible at top
- Overview table inside PanelCard scrolls internally
- Table header stays sticky at top of the PanelCard table area
- First sticky column stays in place during horizontal scroll

**观察日历 tab:**
- Calendar renders normally (no table, no sticky needed)

**今日观察 tab:**
- Table inside PanelCard scrolls internally with sticky header

**喜报 tab:**
- Poster grid renders normally

- [ ] **Step 3: Commit**

```bash
git add frontend/views/ProductCompletion.vue
git commit -m "style(completion): flex root + max-height fallback for PanelCard tables"
```

---

### Task 8: Full Integration Verification

**Files:** None (verification only)

**Interfaces:**
- Consumes: all changes from Tasks 1–7

- [ ] **Step 1: Start dev server**

Run: `cd D:\projects\business-workbench && npm run dev`

Open browser to `http://localhost:5173`.

- [ ] **Step 2: Test all table pages — sticky headers**

For each table page, verify the sticky header behavior:

| Page | Path | Check |
|---|---|---|
| 产品分析 | `/holding-analysis?tab=product` | Scroll table vertically → header stays at top of table area |
| 客户持有 | `/holding-analysis?tab=customer` | Same |
| 待返费 | `/rebate-analysis?tab=pending` | Two-row grouped header stays sticky |
| 已返费 | `/rebate-analysis?tab=completed` | Two-row grouped header stays sticky |
| 观察日历 → 全量 | `/product-completion` | PanelCard table header stays sticky |

- [ ] **Step 3: Test horizontal scroll — sticky columns**

On any wide table page, scroll horizontally. Verify:
- Sticky first column(s) stay in place
- Header sticky columns align correctly with body sticky columns
- Z-index layering is correct (header sticky cols above body sticky cols)

- [ ] **Step 4: Test non-table pages — normal scroll**

Navigate to each non-table page and verify content scrolls normally:
- `/` (Dashboard)
- `/data-preparation`
- `/product-report`
- `/push-settings`
- `/activity-log`
- `/user-profile`
- `/agent`

- [ ] **Step 5: Test sidebar — stays fixed**

On every page, verify:
- Sidebar does not move when scrolling page content
- Sidebar does not move when scrolling inside a table
- Sidebar nav items are all visible (not cut off)

- [ ] **Step 6: Test pagination — always visible**

On pages with pagination (ProductAnalysis, CustomerHolding, RebatePending):
- Verify pagination controls are always visible at the bottom of the page
- Pagination does NOT scroll with the table content

- [ ] **Step 7: Final commit (if any fixes were needed)**

```bash
git add -A
git commit -m "fix: address integration issues from sticky header implementation"
```

If no fixes were needed, skip this step.
