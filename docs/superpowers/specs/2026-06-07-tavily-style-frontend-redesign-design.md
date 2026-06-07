# Tavily-Inspired Frontend Redesign Design

Date: 2026-06-07

## Goal

Refactor the current Vue/Vite frontend into a premium business control console inspired by the Tavily app screenshot the user provided, while preserving all current business workflows, routes, API calls, calculations, and backend contracts.

The chosen direction is option B from the visual companion: Tavily-inspired warmth and polish, adapted to a dense business workbench. The UI should feel high-end, calm, and operational rather than decorative or marketing-like.

## Confirmed Scope

- Redesign the full frontend, not only the homepage.
- Preserve the current route structure and business modules.
- Preserve existing backend endpoints, HTTP methods, request payload assumptions, and response field assumptions.
- Keep existing page script logic stable unless a small script edit is required to fix visible Chinese text or a broken template binding.
- Fix visible Chinese text encoding problems in touched frontend files.
- Use the existing `@lucide/vue` icon package for module and action icons.
- Use Playwright with the local system Chrome for visual verification.
- Keep Chrome DevTools MCP installed and available for deeper browser inspection.

## Out Of Scope

- Backend changes.
- New API fields or changed API response shapes.
- New authentication behavior.
- Replacing the existing workflows with AI search or chat workflows.
- A pixel-perfect Tavily clone.
- Marketing landing-page sections.
- Large unrelated refactors.

## Visual Direction

The design should borrow from the provided Tavily screenshot:

- Warm near-white page background.
- Slightly tinted sidebar surface.
- Strong, simple dark brand mark.
- Rounded but controlled containers.
- Pill-shaped status/action elements where appropriate.
- Soft layered shadows used sparingly.
- Clean typography with strong hierarchy.
- Blue as the primary operational accent and green for healthy/ready states.

The adaptation for this project:

- Keep the density and clarity expected from a daily business operations tool.
- Use compact filters, tables, badges, and action toolbars.
- Avoid large decorative heroes, oversized cards, gradients, blobs, and purely atmospheric visuals.
- Repeated cards should use a radius of 8px or less unless the application shell needs a slightly softer container.

## Architecture

### Application Shell

`WorkbenchLayout` remains the main shell for the app.

It should provide:

- Persistent desktop sidebar with brand area and module navigation.
- Mobile drawer behavior for the sidebar.
- Compact top bar with current page context, global search entry, and utility actions.
- Consistent main content width and padding.
- A wide-mode option for operational tables and dashboard pages.
- Layout safeguards so controls and text do not overlap at desktop or mobile widths.

`SidebarNav` can remain separate if that matches the existing structure. `SubPageLayout` should stay a thin wrapper around the shell so pages do not duplicate navigation and heading logic.

### Shared UI Layer

`frontend/assets/main.css` should define the shared design system:

- Color tokens.
- Font stacks.
- Border radius and shadow tokens.
- Buttons and icon buttons.
- Inputs, selects, textareas, and filter toolbars.
- Tabs or segmented controls.
- Panels, metric tiles, source cards, and module rows.
- Tables and horizontal scroll containers.
- Badges and semantic status colors.
- Empty, loading, success, warning, and error states.

Page-scoped CSS should only handle page-specific structures such as observation calendars, poster grids, unusually wide tables, and report-specific layouts.

### Icons

Use `@lucide/vue`, already present in `frontend/package.json`.

Icons should be used for:

- Sidebar modules.
- Search and menu controls.
- Data source cards.
- Sync, refresh, generate, download, reset, edit, copy, and delete actions.
- Empty states and status callouts where useful.

Primary business actions still need readable Chinese labels; icon-only buttons should be reserved for familiar compact actions and should include accessible labels.

## Page Design

### Home

The homepage becomes a business overview dashboard.

It should include:

- A compact overview band with the app name, short operational purpose, and primary action.
- Metric/status cards for data preparation, product observation, product reports, and customer analysis.
- Module shortcuts with icons and concise descriptions.
- A recommended workflow: data sync, customer analysis, product reports, product observations, channel analysis.
- A neutral reminder if sync status is unavailable.

The page should not look like a marketing landing page.

### Data Preparation

This page becomes a connection and sync console:

- Feishu account connection state.
- Data source cards for transaction and co-invest user tables.
- Last sync time and row count.
- Clear primary sync actions.
- Inline success, error, disabled, and loading states.

The next required action should be obvious when the account is not connected.

### Analysis Pages

User profile, customer churn, channel analysis, and nominal buyer should share a pattern:

- Shell-provided page context.
- Compact filter toolbar.
- Primary query/export action.
- Result summary or metric row.
- Table or empty state.

Long explanatory copy should be reduced where the controls already explain the workflow.

### Product Report And Ongoing Product

These pages should preserve existing behavior while normalizing the interface:

- Shared filter/action panel.
- Consistent panels, table wrappers, and badges.
- Horizontal scroll for wide data.
- No legacy beige/brown styling.

### Product Observation

The observation workspace should be visually reorganized without changing the workflow:

- Tabs become a segmented control.
- All-products view keeps search, price refresh, and observation generation in one toolbar.
- Calendar, today, and poster views use shared panels and empty states.
- Wide observation tables scroll horizontally and may use a sticky first column where useful.
- Poster grid keeps its page-specific structure but adopts the shared card, badge, and action language.

## Typography

Use local/system fonts only:

```css
font-family: "Inter", "Punctuation SC", ui-sans-serif, system-ui, -apple-system,
  BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei",
  "Noto Sans SC", sans-serif;
```

Use monospace for identifiers and codes:

```css
font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco,
  Consolas, "Liberation Mono", monospace;
```

Do not scale font sizes with viewport width. Letter spacing should stay at `0` except for deliberate small uppercase labels.

## Color

Use a warm premium console palette:

- Page background: warm off-white near `#fbfaf5`.
- Sidebar: slightly deeper warm surface.
- Cards and panels: white or near-white.
- Primary text: near-black.
- Secondary text: neutral gray.
- Border: soft warm gray.
- Primary accent: confident blue.
- Secondary accent: restrained green/cyan for healthy operational states.
- Warning: amber.
- Danger: red.

Avoid:

- Orange/brown dominance.
- Heavy beige themes.
- Purple or purple-blue gradients.
- Decorative gradient blobs or bokeh.
- One-note palettes.

## Data Flow

No data flow changes are planned.

Existing page scripts should keep the same endpoints, including:

- Feishu auth status, login, and logout.
- Database sync and sync status endpoints.
- Product report endpoints.
- Observation list, calendar, today, refresh, and generate endpoints.
- Poster list and generation endpoints.
- User profile, customer churn, channel analysis, ongoing product, and nominal buyer endpoints.

The redesign must not rename API fields, alter request methods, or change response assumptions.

## Error Handling

Use consistent state display:

- Request errors: compact danger callout or inline red message.
- Successful actions: compact success callout or green status.
- Loading states: local to the panel/action doing work.
- Empty states: explain the next action.
- Disabled actions: visibly disabled and clearly tied to missing prerequisites.

## Verification

Verification should include:

- `npm run build` from `frontend`.
- Start the Vite dev server.
- Use Playwright with system Chrome at `C:/Program Files/Google/Chrome/Application/chrome.exe`.
- Capture/check desktop and mobile screenshots.
- Spot-check these routes:
  - `/`
  - `/data-preparation`
  - `/product-completion`
  - `/product-report`
  - `/user-profile`
  - `/customer-churn`
  - `/channel-analysis`
  - `/nominal-buyer`
  - `/ongoing-product`
- Confirm Chinese UI text is readable in touched pages.
- Confirm wide tables scroll horizontally.
- Confirm mobile navigation opens and closes correctly.
- Use Chrome DevTools MCP if deeper DOM, console, performance, or network inspection is needed.

## Implementation Notes

Implement incrementally:

1. Establish shared visual tokens and UI classes in `frontend/assets/main.css`.
2. Rebuild the application shell and sidebar.
3. Redesign the homepage dashboard.
4. Normalize shared controls, panels, tables, badges, and states.
5. Update each business page template and visible text while keeping script logic stable.
6. Run build and browser verification.

If a page has complex logic, preserve its script section as much as possible. Prefer changes to template structure, class names, visible labels, icon imports, and scoped CSS.

## Spec Self-Review

- Completeness scan: no unresolved drafting markers remain.
- Scope check: this is a single frontend redesign effort; backend and API changes are explicitly out of scope.
- Consistency check: the visual direction matches selected option B and the current dependency state.
- Ambiguity check: "high-end" is defined as warm, polished, controlled, dense enough for operations, and not marketing-like.
