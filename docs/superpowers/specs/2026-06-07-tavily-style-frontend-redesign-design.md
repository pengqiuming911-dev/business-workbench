# Tavily Style Frontend Redesign Design

Date: 2026-06-07

## Goal

Refactor the existing Vue frontend into a professional SaaS control console inspired by Tavily's application workspace, while preserving the current business workflows, routes, API behavior, and backend contracts.

The redesign should make the application feel like a mature business workbench: clear left navigation, compact top actions, dashboard-style home, consistent forms and tables, readable Chinese UI text, and predictable operational flows.

## Confirmed Scope

- Redesign the whole frontend, including the home page, application shell, navigation, and all business pages.
- Use a professional SaaS console direction: left navigation, top utility area, overview dashboard, compact tools, and dense but readable data presentation.
- Add a frontend icon dependency, preferably `lucide-vue-next`, for module icons, action buttons, and status affordances.
- Fix visible Chinese text encoding issues in touched frontend files.
- Keep existing routes, business logic, data fetching, calculations, API paths, methods, and payload assumptions stable.
- Preserve the app as a business workbench, not an AI search clone or marketing page.

## Out Of Scope

- Backend changes.
- New authentication behavior.
- New API fields or changed response shapes.
- Replacing the business workflow with natural-language AI workflows.
- Pixel-perfect cloning of Tavily's logged-in UI.
- Adding landing-page marketing sections.

## Reference Direction

The reference URL is `https://app.tavily.com/home`. The logged-in page is not reliably available without an account, so the design uses the visible Tavily product direction and common SaaS console patterns rather than treating the reference as a pixel-perfect source.

The relevant direction is:

- Left-side product navigation.
- A compact top area for page context and primary actions.
- Dashboard-style home content.
- Minimal decoration, strong whitespace discipline, and thin borders.
- Clear module icons and concise labels.
- Operational density appropriate for daily use.

For this application, those ideas map to:

- Application modules in the left navigation.
- Business actions and page context in the top bar.
- Home page as an operational overview.
- Tables, forms, filters, tabs, status badges, and empty states as first-class interface elements.

## Architecture

### Application Shell

`WorkbenchLayout` should become the main application shell used across the frontend.

It should include:

- Fixed or persistent left sidebar with brand area and module navigation.
- Top bar with current page context, optional breadcrumb/title, global search entry, and primary action area.
- Main content region with consistent width rules and page padding.
- Mobile sidebar drawer behavior.
- Shared responsive constraints so content does not overlap at desktop or mobile sizes.

`SubPageLayout` should become a thin wrapper around `WorkbenchLayout` or be folded into the shell if that keeps the code simpler. The result should avoid duplicate navigation and page heading logic.

### Shared UI Layer

`frontend/assets/global.css` should hold the main design system:

- Color variables.
- Typography and font stacks.
- Layout sizing variables.
- Buttons and icon buttons.
- Inputs, selects, textareas, and filter rows.
- Tabs and segmented controls.
- Panels, metric tiles, and data cards.
- Tables, sticky columns, and horizontal scroll containers.
- Badges and semantic status colors.
- Empty, loading, success, and error states.

Page-scoped CSS should stay limited to page-specific structures such as calendars, poster grids, and unusually wide tables.

### Icons

Install and use `lucide-vue-next`.

Icons should be used for:

- Sidebar modules.
- Top bar actions.
- Search field affordance.
- Data source cards.
- Sync, refresh, generate, download, and reset actions where appropriate.
- Empty states or status callouts when useful.

Text labels should remain present for primary business actions, especially where Chinese labels are needed for clarity.

## Page Design

### Home

The home page should become an operational dashboard rather than a simple module index.

It should include:

- A compact overview band with the app name, short purpose, and primary action.
- Key status cards for data preparation, product observation, reports, and user/customer analysis.
- Module shortcuts with icons and concise descriptions.
- A recommended workflow area that links the usual sequence: data preparation, user/customer analysis, product reports, observations, and channel analysis.
- A neutral alert or reminder if data sync status is unavailable.

The home page should avoid marketing-style hero composition and avoid oversized decorative cards.

### Data Preparation

This page should become a clear connection and sync console:

- Feishu account connection status.
- Data source cards for the transaction table and co-invest user table.
- Last sync time and row count.
- Primary sync actions.
- Inline success, error, and disabled states.

The page should make the next required action obvious when the account is not connected.

### Search And Analysis Pages

The user profile, customer churn, channel analysis, and nominal buyer pages should follow a shared pattern:

- Page context from the shell.
- A compact filter toolbar.
- Primary action button.
- Result summary text or metric row.
- Table or empty state.

Large explanatory paragraphs should be reduced where the controls already make the workflow clear.

### Product Report And Ongoing Product Pages

These pages should preserve existing data behavior but normalize the interface:

- Shared toolbar for filters and actions.
- Consistent panels and tables.
- Clear status badges.
- Horizontal scroll for wide data.
- No duplicated beige/brown legacy styling.

### Product Observation Page

The observation page can remain functionally the same but should be visually reorganized:

- Tabs become a segmented control.
- The all-products view keeps search, refresh prices, and generate observations actions in a single toolbar.
- Calendar, today, and poster tabs use consistent panels and empty states.
- Wide observation tables keep horizontal scroll and sticky first column where useful.
- Poster grid remains page-specific but should use the shared card, badge, and action visual language.

## Visual Design

### Typography

Use local/system fonts only:

```css
font-family: "Inter", "Punctuation SC", ui-sans-serif, system-ui, -apple-system,
  BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei",
  "Noto Sans SC", sans-serif;
```

Use monospace for codes and identifiers:

```css
font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco,
  Consolas, "Liberation Mono", monospace;
```

### Color

Use a light SaaS palette:

- Page background: cool near-white or light gray.
- Surfaces: white.
- Primary text: near-black.
- Secondary text: neutral gray.
- Border: soft gray.
- Primary accent: blue or blue-cyan.
- Success: green.
- Warning: amber.
- Danger: red.

Avoid beige, brown, orange-heavy themes, large gradients, decorative blobs, and one-note purple/blue-purple palettes.

### Layout

- Sidebar width should be stable on desktop.
- Top bar should be compact and consistent.
- Page content should support wide operational tables.
- Repeated cards should use border radius of 8px or less.
- Controls should have stable dimensions so labels, icons, and hover states do not shift layout.
- Mobile views should collapse the sidebar and keep toolbars readable through wrapping or stacking.

## Data Flow

No data flow changes are planned.

Existing page scripts should continue using the same endpoints, including:

- Auth status, Feishu login, and logout.
- Database sync and sync status endpoints.
- Observation list, calendar, today, refresh, and generate endpoints.
- Poster list and generation endpoints.
- Existing report, user, customer, channel, and nominal buyer endpoints.

The refactor should not rename API fields, change request methods, or alter response assumptions.

## Error Handling

Error and state display should be consistent:

- Request errors use inline red danger text or a compact error callout.
- Successful actions use green success text or status callout.
- Empty states explain the next available action.
- Loading states appear locally where work is happening.
- Disabled actions clearly indicate unavailable prerequisites.

No new error-handling logic is required unless the current page already has unreadable text or unclear feedback in the touched area.

## Testing And Verification

Verification should include:

- Install `lucide-vue-next`.
- Run `npm run build` from `frontend`.
- Start the Vite dev server.
- Browser-check desktop and mobile viewports.
- Spot-check these routes:
  - `/`
  - `/data-preparation`
  - `/product-completion`
  - `/product-report`
  - `/user-profile`
- Confirm Chinese text is readable in touched pages.
- Confirm existing API actions still call the same methods.
- Confirm wide tables scroll horizontally and do not overlap surrounding UI.
- Confirm mobile navigation opens and closes cleanly.

## Implementation Notes

Implementation should be incremental:

1. Add the icon dependency and establish shared design tokens.
2. Rebuild the application shell and navigation.
3. Redesign the home dashboard.
4. Normalize shared controls, panels, forms, tabs, tables, badges, and state messages.
5. Update each business page template and visible text while keeping scripts stable.
6. Run build and browser verification.

Avoid unrelated backend work or feature expansion. If a page has complex existing logic, preserve the script section as much as possible and limit edits to template structure, visible text, class names, imports for icons, and scoped styles.
