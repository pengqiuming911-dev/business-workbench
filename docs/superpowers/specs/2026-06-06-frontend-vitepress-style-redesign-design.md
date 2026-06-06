# Frontend VitePress Style Redesign Design

Date: 2026-06-06

## Goal

Refactor the existing Vue frontend so the whole business workbench uses the visual language of the reference VitePress blog at `https://youweichen0208.github.io/blog/go/`, while preserving the current business workflows and API behavior.

The redesign should make the application feel like a clear technical workbench: fixed top navigation, left module navigation, strong typography, restrained colors, readable tables, and compact controls.

## Confirmed Scope

- Apply the redesign across the frontend shell and base UI: navigation, layout, typography, background, buttons, panels, forms, tables, tabs, empty states, and status badges.
- Keep existing business logic, data fetching, calculations, routes, and backend API contracts unchanged.
- Fix visible Chinese UI text while touching the affected frontend files.
- Use local/system fonts only. Do not add Google Fonts or other runtime font dependencies.
- Follow the reference site's style direction, but adapt it for a business dashboard instead of turning the app into a blog.

## Out of Scope

- Backend changes.
- New product features.
- Authentication behavior changes.
- Reworking API payload shapes.
- Adding external font or icon dependencies.
- Pixel-perfect cloning of the reference website.

## Reference Style

The reference site is a VitePress-based technical blog. The relevant characteristics are:

- A fixed 64px top navigation bar with a clean white background.
- A left sidebar for section navigation.
- A central content column with strong headings, generous vertical rhythm, and readable body text.
- Optional right-side outline on article pages.
- Mostly white and soft gray surfaces, with blue as the primary accent.
- Font stack centered around Inter-style sans text, Chinese sans fallback, and monospace for code.
- Subtle borders and low-shadow elevation instead of decorative gradients.

For this application, the same principles should be translated into a business workbench:

- Left sidebar maps to application modules.
- Central content maps to operational pages, tables, and forms.
- A right outline is not required for every page because the current pages are task-oriented rather than long-form documents.

## Architecture

### Shared Layout

Introduce or refactor toward a shared `WorkbenchLayout`-style shell used by the home page and subpages.

The shell should include:

- Fixed top bar with app name, compact subtitle, primary action area, and mobile menu entry.
- Left sidebar with the existing route modules.
- Main content region with page title, optional description, and page content slot.
- Responsive behavior that collapses the sidebar on smaller screens.

The current `SubPageLayout.vue` should either become this shared shell or delegate to it. The goal is to avoid keeping duplicate header/navigation CSS in both `Home.vue` and `SubPageLayout.vue`.

### Global Styling

Move common visual decisions into `frontend/assets/global.css`:

- CSS variables for color, spacing, typography, borders, and shadows.
- Base body font stack.
- Shared classes for buttons, panels, form fields, tabs, tables, badges, and empty states where practical.

Use component-scoped styles only for page-specific layout details.

### Page Integration

Update the pages to use the shared shell and base UI rules while preserving their existing scripts and state:

- `Home.vue`: becomes a module index and workflow overview inside the shared shell.
- `SubPageLayout.vue`: becomes the common shell for business pages.
- Business pages: keep data logic intact, but normalize visible text and align their panels/tables/buttons/tabs with the new design system.

The product observation page can remain functionally identical: tabs, search, refresh/generate actions, expandable table rows, today's observations, and poster generation remain available.

## Visual Design

### Typography

Use a local font stack:

```css
font-family: "Inter", "Punctuation SC", ui-sans-serif, system-ui, -apple-system,
  BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei",
  "Noto Sans SC", sans-serif;
```

Use monospace for codes, symbols, and technical identifiers:

```css
font-family: "IBM Plex Mono", ui-monospace, SFMono-Regular, Menlo, Monaco,
  Consolas, "Liberation Mono", monospace;
```

`Inter` and `IBM Plex Mono` may not be installed locally. They are listed as preferred local fonts only, with stable system fallbacks.

### Color

Adopt a light technical palette:

- Background: white and very light gray.
- Text: near-black for primary text, medium gray for secondary text.
- Borders: soft gray dividers.
- Primary accent: blue.
- Success: green.
- Warning: amber.
- Danger: red.

The current beige/brown/orange-heavy palette should be replaced because it does not match the reference site.

### Layout Rhythm

- Top bar height: 64px.
- Sidebar width: about 272px on desktop.
- Content width: responsive, optimized for tables and operational views.
- Use 24px to 32px page padding on desktop and tighter spacing on mobile.
- Keep cards/panels at 8px or less radius unless an existing element needs slightly more for readability.

### Components

Buttons:

- Primary actions use blue.
- Secondary actions use gray/neutral styling.
- Destructive or high-risk actions use red only when semantically needed.
- Disabled state should be visually clear.

Tables:

- Preserve horizontal scrolling for wide operational tables.
- Sticky headers and sticky first columns can remain.
- Use compact row height, readable line spacing, and clear hover states.

Forms:

- Inputs use white or light gray background, 1px border, and blue focus ring/border.
- Labels stay compact and aligned for repeated operational use.

Tabs:

- Use segmented controls with a subtle container and blue active state.

Badges:

- Use soft backgrounds and semantic text colors for status.

## Data Flow

No data flow changes are planned.

Existing page scripts should continue to call the same APIs:

- Observation list and refresh endpoints.
- Today's observation endpoint.
- Poster list and generation endpoints.
- Any existing data preparation and report endpoints.

The refactor should not rename API fields, change request methods, or alter response assumptions.

## Error Handling

Existing error states should remain visible, but their styling should become consistent:

- Error messages use the danger color.
- Success messages use the success color.
- Empty states use a neutral panel style.
- Loading states stay in place and use normal Chinese text.

No new error-handling logic is required unless a page currently displays malformed or unreadable text.

## Testing And Verification

Verification should include:

- `npm run build` from `frontend`.
- Manual browser check of the local Vite app.
- Desktop and mobile viewport review.
- Spot checks for key routes:
  - Home
  - Data Preparation
  - Product Observation
  - Product Report
  - User Profile
- Confirm Chinese text is readable in touched pages.
- Confirm existing API actions are still wired to the same methods.
- Confirm wide tables remain scrollable and do not overlap surrounding UI.

## Implementation Notes

The implementation should be incremental:

1. Establish shared design variables and layout.
2. Convert home and subpage shell.
3. Normalize component styles and visible Chinese text across pages.
4. Run build and browser verification.

Avoid unrelated refactors. If a page has substantial existing logic, keep the script section stable and limit changes to template text/class structure and styles.
