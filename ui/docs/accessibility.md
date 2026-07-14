# CDS UI accessibility conventions

Target: **WCAG 2.2 Level AA**, screenreader support verified with **NVDA** (Chrome/Firefox).
Companion document: `accessibility-plan.md` (audit, phases, open items).

The current remediation wave is **strictly non-visual**: attribute-only retrofits, ARIA,
and announcements. Element swaps and any change with visual impact are tracked in the
plan as "Open / To Do (visual impact)".

## Rules for all new/changed templates

1. **Form controls**: every control gets an `id` with the `cds-field-` prefix and its
   `<nz-form-label nzFor="…">` pointing at it. Controls without a visible label get an
   `aria-label`. A placeholder is never the only label.
2. **Icon-only buttons**: always `aria-label="…"` on the button; decorative icons inside
   any labelled control get `aria-hidden="true"`. Keep `title`/`nz-tooltip` for sighted users.
3. **Never put `(click)` on a non-interactive element** (`div`, `span`, `li`, `td`, `i`,
   `a` without `href`). Use a real `<button>`/`<a [routerLink]>` for new code. For existing
   markup that must not change visually, add the `appClickable` directive
   (`shared/directives/clickable.directive.ts`) — it retrofits `role="button"`,
   `tabindex="0"` and Enter/Space activation.
4. **Screenreader-only text**: use the `cds-sr-only` class (defined in `src/styles.scss`).
5. **Announce async outcomes** with `LiveAnnouncer` (`@angular/cdk/a11y`): `'polite'` for
   success/info, `'assertive'` for errors. Toasts sent through `ToastService` are
   announced automatically — do not double-announce.
6. **State must be conveyed in text**, not only by icon or color: status icons carry
   `aria-label="Status: …"`; disclosure elements carry `aria-expanded`; sortable table
   headers carry `aria-sort`.
7. **Landmarks and headings**: pages must expose exactly one level-1 heading
   (`role="heading" aria-level="1"` on existing styled elements until the element-swap
   wave lands) and live inside the `role="main"` container provided by the app shell.
8. **Dialogs**: every modal needs an accessible name (`nzTitle` or `aria-labelledby`);
   focus must return to the triggering control on close.

## Linting & CI

- `@angular-eslint/template/*` accessibility rules are enabled in `.eslintrc.js` at
  `warn`; each rule is flipped to `error` when its sweep is finished. Do not introduce
  new warnings.
- Planned (see plan Phase 0): axe-core checks and a screenshot-diff harness in CI.

## Manual NVDA smoke pass (run before releases)

signin → home → select project → run list → open run → open failing job logs →
settings/profile. Verify: page title announced on navigation, landmarks (`D`) and
headings (`H`) navigate, every control reachable with Tab/arrows and announced with
role + name + state, toasts and run-status changes are spoken.
