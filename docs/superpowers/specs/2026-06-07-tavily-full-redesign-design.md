# Tavily Full Frontend Redesign Design

Date: 2026-06-07
Supersedes: 2026-06-07-tavily-style-frontend-redesign-design.md

## Goal

Fully redesign the existing Vue 3 frontend into a professional SaaS console inspired by Tavily's app.tavily.com/home dashboard. This includes visual overhaul, layout redesign, and new features: a data dashboard, global search (Cmd+K), and activity log.

## Approach

**Hand-written rebuild** using the existing Vue 3 + Vite + vanilla CSS stack. No UI component library. ECharts (lightweight) is added for charts. All components are written from scratch in the new style.

## Confirmed Scope

- Redesigned application shell: fixed left sidebar (240px) + compact top bar
- New Dashboard home page with stat cards, ECharts charts, observation feed, and sync status
- Global search via Cmd+K / Ctrl+K modal
- Activity log page at `/activity-log`
- Unified visual design system (Tavily-inspired color, typography, spacing, components)
- Restyle all 8 existing business module pages to the new design
- New backend API endpoints for dashboard data and search
- Install `lucide-vue-next` for icons

## Out Of Scope

- Authentication changes
- Backend business logic changes
- New AI/NLP features
- Marketing/landing pages

## Architecture

### Application Shell

Replace the current top-bar + hamburger sidebar with a **fixed left sidebar + compact top bar**.

```
+-------+------------------------------------------------+
| BW    | [search Cmd+K]                    [avatar]     |  top bar 56px
+-------+------------------------------------------------+
| Dash  |                                                |
| Data  |   page heading (title + description)           |
| User  |   +------------------------------------------+ |
| Churn |   |                                          | |
| Rpt   |   |        page content area                 | |
| Obs   |   |        max-width: 1200px                 | |
| Ong   |   |                                          | |
| Chan  |   +------------------------------------------+ |
| Nomi  |                                                |
|-------|                                                |
| Logs  |                                                |
+-------+------------------------------------------------+
  240px              content area
```

- Sidebar: always visible, 240px wide. Logo + 8 module links with icons + "Activity Log" at bottom. Active page highlighted with 3px left blue indicator.
- Top bar: brand name, global search shortcut, user avatar (Feishu).
- Mobile (720px below): sidebar collapses to 64px icon rail. Tap toggles overlay expansion.
- Remove `SubPageLayout.vue`. All pages render directly inside `WorkbenchLayout` slot.

### Shared Components

| Component | Purpose |
|-----------|---------|
| `WorkbenchLayout.vue` | Full rewrite. Sidebar + top bar + content slot |
| `SidebarNav.vue` | Extracted sidebar navigation with icons |
| `GlobalSearch.vue` | Cmd+K modal with search input + results list |
| `StatCard.vue` | Metric tile: big number + label + trend arrow |
| `PanelCard.vue` | Generic white card with optional title |
| `ActivityTimeline.vue` | Vertical timeline list for activity log entries |

Delete: `SubPageLayout.vue`, `FolderCard.vue` (if unused)

### File Structure

```
frontend/
  components/
    WorkbenchLayout.vue      (rewrite)
    SidebarNav.vue            (new)
    GlobalSearch.vue          (new)
    StatCard.vue              (new)
    PanelCard.vue             (new)
    ActivityTimeline.vue      (new)
  views/
    Dashboard.vue             (new, replaces Home as `/`)
    ActivityLog.vue           (new)
    DataPreparation.vue       (restyle)
    UserProfile.vue           (restyle)
    CustomerChurn.vue         (restyle)
    ProductReport.vue         (restyle)
    ProductCompletion.vue     (restyle)
    OngoingProduct.vue        (restyle)
    ChannelAnalysis.vue       (restyle)
    NominalBuyer.vue          (restyle)
    Home.vue                  (delete or redirect to Dashboard)
  router/index.js             (update routes)
  assets/
    main.css                  (new: design tokens + base styles)
    icons/                    (new: SVG icons if not using lucide)
  main.js                     (import main.css)
```

## Visual Design System

### Colors (Tavily-Inspired)

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
  --warning: #ffb547;
  --danger: #ee5d50;
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
  --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1);
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
  --radius-sm: 6px;
  --radius: 10px;
  --radius-lg: 14px;
}
```

### Typography

| Usage | Size | Weight |
|-------|------|--------|
| Page title | 28px | 800 |
| Section title | 18px | 700 |
| Card title | 15px | 600 |
| Body text | 14px | 400 |
| Labels / captions | 12px | 500 |

### Component Styles

- **Cards/Panels**: white bg, 1px border, 10px radius, shadow-sm. Hover: shadow-md.
- **Buttons**: primary = blue solid, 8px radius, 36px height. Secondary = outlined. 
- **Inputs**: 1px border, 8px radius. Focus: blue border + ring.
- **Sidebar links**: hover = light gray bg. Active = 3px left blue bar + slightly darker bg.
- **Tables**: no outer border, 1px row dividers, hover row highlight.

### Transitions

- Hover/state: 150ms ease-out
- Page transitions: 200ms fade-in
- Mobile sidebar: 250ms slide-in

## Dashboard Home Page

Replaces current `Home.vue` as the root route `/`.

### Layout

```
+--------------------------------------------------------+
| Welcome back, [username]              2026-06-07       |
+--------------------------------------------------------+
| [StatCard] [StatCard] [StatCard] [StatCard]            |
| Products   Active     Customers  Channels              |
|   128        42        356        18                   |
+--------------------------------------------------------+
| [ECharts line chart     ] [ECharts pie chart     ]     |
| Monthly transaction trend] [Channel distribution ]     |
+--------------------------------------------------------+
| [Observation feed       ] [Sync status panel    ]     |
| Upcoming observations    ] [Last sync timestamps ]     |
+--------------------------------------------------------+
| [Quick entry tiles: 8 modules, simplified cards]       |
+--------------------------------------------------------+
```

### Stat Cards (4)

| Card | Data Source |
|------|-------------|
| Total products | `products` table count |
| Active products | `ongoing_products` query |
| Total customers | `customers` distinct count |
| Total channels | `channels` distinct count |

Each card: large number + title + trend indicator (up/down arrow with percentage vs last month). If insufficient historical data exists for trend calculation, show a neutral dash "—" instead.

### Charts Row (2 columns)

- **Left: Monthly Transaction Trend** (ECharts line chart) — x=month, dual y-axis: amount + count
- **Right: Channel Distribution** (ECharts pie chart) — top 8 channels by amount, rest merged as "Other"

### Dynamic Panels (2 columns)

- **Left: Upcoming Observations** — products with observation dates in next 7 days. Show: product name, date, type (dividend/knockout).
- **Right: Sync Status** — each data source with last sync time + status badge (ready/stale/never).

### Quick Entry (bottom row)

8 simplified icon+title cards linking to each business module, smaller than current module cards.

## Global Search (Cmd+K)

Triggered by `Cmd+K` / `Ctrl+K` or clicking the search area in top bar.

### Behavior

- Centered modal overlay with search input
- Debounce 300ms on input
- Call `GET /api/search?q=keyword`
- Search across: customer names, product names, channel names
- Results grouped by type with icons
- Keyboard navigation: up/down arrows + enter to select
- Escape or click outside to close
- Show "recently accessed" when input is empty

### UI

```
+----------------------------------+
| [magnifying glass] Type to search |
+----------------------------------+
| RECENTLY ACCESSED                 |
|   Channel Analysis               |
|   Product Report                 |
|                                  |
| RESULTS (n)                       |
|   Zhang San - Customer           |
|   Xinxiang 3 - Product           |
|   CMB - Channel                  |
+----------------------------------+
```

## Activity Log Page

New route: `/activity-log`

### Display

Vertical timeline of operational events:

```
2026-06-07 14:32  [sync]  Transaction table synced (1,284 rows)
2026-06-07 10:15  [sync]  Co-invest users synced (89 rows)
2026-06-06 16:40  [query] Customer churn analysis run
2026-06-06 09:20  [sync]  Transaction table synced (1,276 rows)
```

- Support filtering by type (sync/query/export)
- Entry point in sidebar bottom

### Backend

New `activity_logs` table in SQLite:

```sql
CREATE TABLE activity_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,        -- 'sync', 'query', 'export'
  action TEXT NOT NULL,       -- short description
  detail TEXT,                -- JSON or text with extra info
  created_at TEXT DEFAULT (datetime('now'))
);
```

Instrument existing operations to write log entries:
- Data sync endpoints (transaction table, co-invest, product docs)
- Analysis/report generation
- Export operations

## Sub-Page Restyle

All 8 business module pages keep their existing functionality and API calls. Changes are template + style only.

### Common Pattern (for analysis pages)

```
+--------------------------------------------------------+
| Parameter Panel (PanelCard)                             |
| [input] [input] [input] [Generate]                      |
+--------------------------------------------------------+
| Results Panel (PanelCard)                               |
| [ECharts chart if applicable]                           |
| [Data table]                              [Export]     |
+--------------------------------------------------------+
```

### Specific Notes per Page

- **Data Preparation**: Connection status + sync panels in PanelCards. Keep existing auth flow and API calls.
- **User Profile**: Filter toolbar + results table.
- **Customer Churn**: Date filters + generate button + results table/chart.
- **Product Report**: Docs list in table with status badges.
- **Product Completion (Observation)**: Tabs as segmented control. Calendar, today, poster tabs. Keep all existing logic.
- **Ongoing Product**: Filters + table with chart summary.
- **Channel Analysis**: Date range + results with ECharts bar/line chart + table.
- **Nominal Buyer**: Search input + results table.

## New Backend APIs

### `GET /api/dashboard/stats`

Returns aggregated counts for the 4 stat cards.

```json
{
  "totalProducts": 128,
  "activeProducts": 42,
  "totalCustomers": 356,
  "totalChannels": 18
}
```

### `GET /api/dashboard/charts`

Returns pre-aggregated chart data.

```json
{
  "monthlyTrend": [
    { "month": "2026-01", "amount": 12000000, "count": 45 },
    ...
  ],
  "channelDistribution": [
    { "channel": "CMB", "amount": 8500000 },
    ...
  ]
}
```

### `GET /api/search?q=keyword`

Cross-table search across products, customers, channels.

```json
{
  "results": [
    { "type": "customer", "id": 1, "name": "Zhang San", "path": "/user-profile?customer=1" },
    { "type": "product", "id": 5, "name": "Xinxiang 3", "path": "/ongoing-product?product=5" },
    { "type": "channel", "id": 2, "name": "CMB", "path": "/channel-analysis?channel=2" }
  ]
}
```

### `GET /api/activity-logs?type=sync&limit=50`

Query activity log entries with optional type filter and limit.

```json
{
  "logs": [
    { "id": 1, "type": "sync", "action": "Transaction table synced", "detail": "1,284 rows", "createdAt": "2026-06-07T14:32:00Z" }
  ]
}
```

## Error Handling

- Request errors: inline red text or compact error callout within the relevant panel
- Success: green text or status callout
- Empty states: centered message with suggested action
- Loading: local spinner where work is happening
- Disabled: grayed out + explanatory hint

## Dependencies

- `echarts` (chart rendering, ~30KB gzip with tree-shaking)
- `lucide-vue-next` (icons)

## Data Flow

Existing frontend API calls remain unchanged. The 4 new backend endpoints are additive.

## Verification

1. `npm install` in frontend (installs echarts + lucide-vue-next)
2. `npm run build` in frontend — no errors
3. Start Vite dev server + backend
4. Browser check routes: `/`, `/data-preparation`, `/activity-log`, `/channel-analysis`, `/product-completion`
5. Verify Cmd+K search opens and returns results
6. Verify dashboard stats and charts render
7. Verify activity log shows entries after performing sync
8. Mobile viewport check at 375px width
9. Confirm all existing API calls still work (no changed endpoints/params)
