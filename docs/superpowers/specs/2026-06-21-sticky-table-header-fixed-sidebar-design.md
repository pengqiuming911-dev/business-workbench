# 表格粘性表头 + 侧边栏固定设计

- 日期：2026-06-21
- 分支：待创建

## 1. 背景

当前页面的表格（返费、持仓、观察日历）没有有界的垂直滚动容器，表格随内容增长，整个页面通过浏览器原生滚动。用户往下滚动数据时，表头跟着页面一起滚走，无法看到当前数据对应的字段名。

此外，侧边栏需要确认在页面上下滚动时保持固定不动，让用户可以随时切换页面。

## 2. 目标

- 表格的表头在表格区域内固定，数据在表格内部滚动，用户始终能看到当前数据对应的字段。
- 表格高度自适应填满页面可用空间（视口减去顶栏、标题、筛选栏等）。
- 侧边栏在页面滚动时保持固定（当前已是 `position: fixed`，保持不变）。

## 3. 当前状态

### 3.1 侧边栏

`SidebarNav.vue` 已经使用 `position: fixed; top: 0; left: 0; bottom: 0`，在页面滚动时不会移动。此需求已满足，只需补充 `overflow-y: auto` 防止小屏幕下导航项被裁切。

### 3.2 表格

- `.table-wrap` 仅有 `overflow-x: auto`，无 `max-height`、无 `overflow-y`。
- 表格随内容无限增长，页面通过 `<body>` 原生滚动。
- 表头已有 `position: sticky; top: 0`（全局 CSS），但因为 `.table-wrap` 没有有界高度，sticky 相对的是视口，不是表格容器。
- 返费页面有两行分组表头（`top: 0` + `top: 44px`），逻辑正确但同样缺乏有界容器。

### 3.3 布局链

```
body (#app)
└─ .workbench-shell (min-height: 100vh)
   ├─ .sidebar (position: fixed — 脱离文档流)
   └─ .workbench-content (margin-left: 208px)
      ├─ .workbench-topbar (48px, 文档流内)
      └─ .workbench-main (padding: 24px 32px 72px)
```

页面滚动完全依赖 `<body>` 原生滚动，没有任何中间层有有界高度。

## 4. 设计方案

### 4.1 架构：Flexbox 布局链

从最外层到表格容器建立 flex 布局链，让表格自动填满页面剩余空间。

```
.workbench-shell  → height: 100vh; display: flex
  ├─ .sidebar (position: fixed，不参与 flex)
  └─ .workbench-content → flex: 1; display: flex; flex-direction: column; overflow: hidden
       ├─ .workbench-topbar → flex-shrink: 0（高度固定 48px）
       └─ .workbench-main → flex: 1; min-height: 0; display: flex; flex-direction: column; overflow: hidden
```

关键变更：
- `.workbench-shell`：从 `min-height: 100vh` 改为 `height: 100vh`，消除页面级滚动。
- `.workbench-content`：加 `display: flex; flex-direction: column; flex: 1; min-height: 0; overflow: hidden`。
- `.workbench-main`：加 `display: flex; flex-direction: column; flex: 1; min-height: 0; overflow: hidden`。

### 4.2 表格容器

在 flex 链建立后，表格页面内部结构：

```
.workbench-main (flex column, 有界高度)
  └─ 页面根元素 (flex: 1; min-height: 0; overflow-y: auto)
       ├─ .page-header → flex-shrink: 0
       ├─ .tab-bar → flex-shrink: 0
       ├─ .filter-bar → flex-shrink: 0
       ├─ .table-wrap → flex: 1; min-height: 0; overflow-y: auto ← 表格滚动容器
       │    └─ <table> 表头 sticky top: 0
       └─ .pagination → flex-shrink: 0 ← 始终可见在底部
```

注意：当前三个表格页面（ProductAnalysis、CustomerHolding、RebatePending）的分页控件在 `.table-wrap` 外部。这正好符合设计——`.table-wrap` 用 flex: 1 填满中间空间并内部滚动，分页控件用 flex-shrink: 0 始终显示在底部。

`.table-wrap` 全局 CSS 改动：
- 加 `flex: 1; min-height: 0; overflow-y: auto`。
- 保留原有 `overflow-x: auto`（宽表横向滚动不变）。

表头不需要改动：
- 全局 `.data-table th` 已有 `position: sticky; top: 0; z-index: 4`。
- 返费页面两行分组表头 `top: 0` + `top: 44px` 逻辑正确。
- `.table-wrap` 成为有界滚动容器后，sticky 自然生效在表格区域顶部。

### 4.3 Tab 容器页面

`HoldingAnalysis.vue` 和 `RebateAnalysis.vue` 是 tab 容器，结构如下：

```
.holding-analysis-page (flex: 1; display: flex; flex-direction: column; min-height: 0)
  ├─ .page-header → flex-shrink: 0
  ├─ .tab-bar → flex-shrink: 0
  └─ <ProductAnalysis> / <CustomerHolding> (flex: 1; display: flex; flex-direction: column; min-height: 0)
       ├─ .filter-bar → flex-shrink: 0
       └─ .table-wrap → flex: 1; min-height: 0; overflow-y: auto
```

需要在 tab 容器和子页面根元素上都加 flex 布局。

### 4.4 非表格页面

Dashboard、DataPreparation、ProductReport、PushSettings、ActivityLog、UserProfile、AgentChat 等页面不需要表格粘性表头。

这些页面的根元素加 `flex: 1; min-height: 0; overflow-y: auto`，各自内部正常滚动，体验不变。

### 4.5 侧边栏

`SidebarNav.vue` 已经是 `position: fixed`，不需要布局改动。补充 `overflow-y: auto` 到 `.sidebar-inner`，防止小屏幕下导航项被裁切。

## 5. 涉及文件

### 5.1 全局布局（2 个文件）

| 文件 | 改动 |
|---|---|
| `frontend/components/WorkbenchLayout.vue` | shell / content / main 三层 flex 链 |
| `frontend/assets/main.css` | `.table-wrap` 加 flex 属性 |

### 5.2 表格页面（7 个视图）

| 文件 | 改动 |
|---|---|
| `frontend/views/HoldingAnalysis.vue` | 页面根元素 + tab 内容区加 flex 布局 |
| `frontend/views/ProductAnalysis.vue` | 根元素加 flex，筛选栏 flex-shrink: 0 |
| `frontend/views/CustomerHolding.vue` | 同上 |
| `frontend/views/RebateAnalysis.vue` | 同 HoldingAnalysis |
| `frontend/views/RebatePending.vue` | 根元素加 flex，筛选栏 flex-shrink: 0 |
| `frontend/views/RebateCompleted.vue` | 同上 |
| `frontend/views/ProductCompletion.vue` | 根元素加 flex，各 tab 内容区加 flex |

### 5.3 非表格页面（7 个视图）

| 文件 | 改动 |
|---|---|
| `frontend/views/Dashboard.vue` | 根元素加 `flex: 1; overflow-y: auto` |
| `frontend/views/DataPreparation.vue` | 同上 |
| `frontend/views/ProductReport.vue` | 同上 |
| `frontend/views/PushSettings.vue` | 同上 |
| `frontend/views/ActivityLog.vue` | 同上 |
| `frontend/views/UserProfile.vue` | 同上 |
| `frontend/views/AgentChat.vue` | 同上 |

### 5.4 侧边栏（1 个文件）

| 文件 | 改动 |
|---|---|
| `frontend/components/SidebarNav.vue` | `.sidebar-inner` 加 `overflow-y: auto` |

## 6. 验收标准

1. 表格页面（产品分析、客户持有、待返费、已返费、观察日历）：在表格内上下滚动时，表头固定在表格区域顶部，不随数据滚动。
2. 筛选栏、标签栏等始终可见在表格上方，不随表格滚动。
3. 侧边栏在页面操作时保持固定不动。
4. 非表格页面（Dashboard、DataPreparation 等）保持正常滚动行为，不受影响。
5. 宽表横向滚动行为不变（sticky 列仍然有效）。
6. 分页控件始终可见在表格下方，不随表格内部滚动。
