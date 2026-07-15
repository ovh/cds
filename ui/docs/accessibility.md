# CDS UI accessibility

Target: **WCAG 2.2 Level AA**. Screenreader support is verified with **NVDA** (Chromium/Firefox).

This document describes how accessibility is implemented in the UI and the rules to
follow when adding or changing templates.

## How it works

### Page structure
- `<html lang="en">` is set in `src/index.html`.
- The app shell (`src/app/app.component.html`) exposes landmarks via roles on the
  existing layout elements: `role="banner"` (navbar), `role="navigation"` (menu),
  `role="main"` + `id="main-content"` (content container) telling screenreader users where they are.
- A "Skip to main content" link is the first focusable element; it is rendered
  off-screen and only becomes visible on keyboard focus.
- On navigation, the new page title is announced via CDK `LiveAnnouncer`
  (`app.component.ts`). Page titles come from route `data.title`.

### Screenreader-only content
- The `cds-sr-only` class (`src/styles.scss`) renders text visually hidden but
  available to assistive technology. Use it for supplementary text that has no
  visible equivalent.

### Announcements
- Toasts sent through `ToastService` (`shared/toast/ToastService.ts`) are announced
  automatically: `'polite'` for success/info, `'assertive'` for errors.
- Other async outcomes (e.g. run status changes) are announced with `LiveAnnouncer`
  (`@angular/cdk/a11y`), using the same politeness convention.
- Loading spinners and banners expose `role="status"`.

### Interactive elements
- The `appClickable` directive (`shared/directives/clickable.directive.ts`) makes a
  non-interactive element with a `(click)` handler keyboard-accessible: it adds
  `role="button"`, `tabindex="0"` and Enter/Space activation. It exists to retrofit
  legacy markup that must not change visually; new code uses real `<button>`/`<a>`.
- Shared tabs (`shared/tabs/`) implement the APG tab pattern: `role="tablist"`/`tab`,
  `aria-selected`, arrow-key navigation.
- The shared data table (`shared/table/`) exposes `aria-sort` on sortable headers
  and supports keyboard sorting.
- Run log steps are disclosures (`aria-expanded`); streaming log containers use
  `role="log"`.
- Status icons carry their state as text (`aria-label="Status: …"`).

### Workflow graph (`libs/workflow-graph`)
- The graph container is focusable and explains its keyboard model via
  `aria-roledescription`/`aria-label`.
- Arrow keys move real DOM focus between nodes (roving `tabindex`); each node has an
  accessible name including job, status and stage. Selection changes are announced.
- `+`/`-`/`0` zoom in/out and re-center. Key handling is scoped to the graph and
  does not intercept keys while typing in inputs.

### Forms
- Every form control has an `id` prefixed with `cds-field-`, associated to its
  label via `<nz-form-label nzFor="…">`. Controls without a visible label carry an
  `aria-label`.

## Rules for new/changed templates

1. **Form controls**: every control gets an `id` with the `cds-field-` prefix and its
   `<nz-form-label nzFor="…">` pointing at it. Controls without a visible label get an
   `aria-label`. A placeholder is never the only label.
2. **Icon-only buttons**: always `aria-label="…"` on the button; decorative icons inside
   any labelled control get `aria-hidden="true"`. Keep `title`/`nz-tooltip` for sighted users.
3. **Never put `(click)` on a non-interactive element** (`div`, `span`, `li`, `td`, `i`,
   `a` without `href`). Use a real `<button>`/`<a [routerLink]>` for new code. Only for
   existing markup that must not change visually, use the `appClickable` directive.
4. **Screenreader-only text**: use the `cds-sr-only` class.
5. **Announce async outcomes** with `LiveAnnouncer`: `'polite'` for success/info,
   `'assertive'` for errors. Toasts sent through `ToastService` are announced
   automatically — do not double-announce.
6. **State must be conveyed in text**, not only by icon or color: status icons carry
   `aria-label="Status: …"`; disclosure elements carry `aria-expanded`; sortable table
   headers carry `aria-sort`.
7. **Landmarks and headings**: each page exposes exactly one level-1 heading and lives
   inside the `role="main"` container provided by the app shell. Where the visual title
   is a styled non-heading element, use `role="heading" aria-level="1"` rather than
   changing the element.
8. **Dialogs**: every modal needs an accessible name (`nzTitle` or `aria-labelledby`);
   focus must return to the triggering control on close.

## Linting

`@angular-eslint/template/*` accessibility rules are enabled in `.eslintrc.js`. Rules
still reporting warnings on legacy templates run at `warn`; a rule is switched to
`error` once the codebase is clean for it. Do not introduce new warnings.

## Known limitations

Accessibility fixes that require visible UI changes are out of scope of the
attribute-only approach above and still open:

- No unified `:focus-visible` styling — newly focusable elements use the browser's
  default focus ring.
- Retrofitted elements (fake links, `role="heading"` divs, landmark roles) are not
  yet swapped for their native elements (`<button>`, `<h1>`–`<h4>`, `<main>`/`<nav>`).
- The workflow graph has no alternative list/table view.
- Drag-and-drop reordering (pipeline stages, action steps) has no keyboard alternative.
- CodeMirror 5 editors (`shared/codemirror.ts`) are largely inaccessible; Monaco
  (`nz-code-editor`) is the accessible editor and is labeled via `ariaLabel`.

## Manual NVDA smoke pass (run before releases)

signin → home → select project → run list → open run → open failing job logs →
settings/profile. Verify: page title announced on navigation, landmarks (`D`) and
headings (`H`) navigate, every control reachable with Tab/arrows and announced with
role + name + state, toasts and run-status changes are spoken.
