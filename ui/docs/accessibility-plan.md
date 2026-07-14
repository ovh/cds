# CDS UI — Screenreader Accessibility Plan

**Goal:** Full screenreader support (NVDA, JAWS, VoiceOver) for the CDS web UI (`ui/`), targeting **WCAG 2.2 Level AA** conformance (the current normative standard, and the baseline required by EN 301 549 / the European Accessibility Act). Keyboard operability is treated as a prerequisite of screenreader support throughout — a control NVDA cannot reach is a control NVDA cannot announce.

**Hard constraint for the first PR(s): zero visual change.** Sighted users must see pixel-identical rendering after the change. This plan therefore classifies every work item by visual impact (see §3) and defines a first-PR scope (§6) that is strictly non-visual. Visually observable improvements (better focus styles, a graph list view, reorder buttons, component migrations) are explicitly deferred to later, separately reviewable steps.

**Scope:** `ui/src/**` (Angular 21 app, ng-zorro-antd) and `ui/libs/workflow-graph` (standalone SVG DAG library). Both UI generations are in scope: v1 (`views/workflow`, `views/pipeline`, …) and v2 (`views/projectv2`), with v2 prioritized.

---

## 1. Current state (audit summary)

An audit of all 217 templates and 460 TypeScript files (2026-07) found:

### Global / structural
- **Zero hand-authored ARIA** in the entire app: no `aria-*`, no `role=`, no `[attr.aria…]` bindings. All existing semantics come implicitly from ng-zorro components.
- `ui/src/index.html:2` — `<html>` has **no `lang` attribute** (WCAG 3.1.1 failure on every page).
- **No landmark elements** anywhere: the shell (`src/app/app.component.html`) uses `nz-layout`/`nz-header`/`nz-content`, which render generic `<div>`s. No `<main>`, `<nav>`, `<header>`, `<footer>`, no skip link.
- Page titles: `Title` service is wired via route `data.title` in `app.component.ts:199-220` — good foundation, but coverage per route is incomplete.
- **No focus management, no live regions**: `@angular/cdk/a11y` (LiveAnnouncer, FocusMonitor, FocusTrap) is unused; only 2 `.focus()` calls in the whole app.
- **No a11y linting**: `.eslintrc.js` extends only `plugin:@angular-eslint/template/recommended`; the accessibility ruleset is not enabled.

### Semantics & controls
- **Forms are the broadest defect**: 526 `<nz-form-label>` but only 12 `nzFor` associations; 200 `<input>` with only 13 `id`s; 0 native `<label for>`. Screenreaders announce most fields with no name (e.g. `views/auth/signin/signin.html:13-24`).
- **28 fake links**: `<a class="pointing" (click)="…">` with no `href` — not focusable, not activatable by keyboard (e.g. `signin.html:28`, test filters in `views/workflow/run/node/test/table/test.table.html`).
- **~38 click handlers on non-interactive elements** (`div`/`span`/`li`/`td`/`i`), including clickable table rows in `shared/table/data-table.html`.
- **Icon-only buttons without accessible names** are common (navbar tool/settings triggers, `data-table.html:116` copy button, `workflow.notification.list.html:6,27`, graph controls). Some rely on `title=` only.
- Headings exist but are ad hoc: many pages start at `<h2>`/`<h3>` with no `<h1>`, frequent level skips.
- 3 `<img>` without `alt`; existing alts are weak (`alt="icon"`).

### Custom widgets
- **Workflow graph (v1 `views/workflow/graph/` and v2 `libs/workflow-graph/`)** — dagre-d3/SVG with Angular node components in `foreignObject`. Nodes are non-focusable `<div (click)>` (`job-node.html:1`); pan/zoom is d3-zoom (mouse-only); lasso multi-select is mouse-only. v2 has partial arrow-key navigation, but via `@HostListener('window:keydown')` (`graph.component.ts:187-245`) with **no tabindex, no roles, no focus indication, no announcements** — the graph is effectively invisible to a screenreader. v1 has no keyboard support at all.
- **Run log viewer (`views/projectv2/run/run-job.html`)** — custom renderer: collapsible step blocks are clickable `<div>`s without button/`aria-expanded` semantics; lines stream in over WebSocket with no live region; manual scroll-windowing.
- **Custom tabs (`shared/tabs/`)** — built as `<ul nz-menu><li nz-menu-item (click)>`: announced as a menu, not tabs; no `aria-selected`, no arrow-key support. Widely used for page sections.
- **Drag-and-drop (ng2-dragula)** — pipeline stage reordering (`views/pipeline/show/workflow/pipeline.workflow.html:24`) and job/action step reordering (`shared/action/action.html:89`) are mouse-only with no keyboard alternative.
- **Editors** — Monaco (via `nz-code-editor`) has strong built-in screenreader support but is never labeled by the app; CodeMirror 5 (`shared/codemirror.ts`, ~20 consumers) is largely inaccessible by design.
- **Status indicators** — `shared/status/status.icon.html` conveys build state by icon + color class only, no text alternative; graph edges/forks are color-only.
- **Live updates** — WebSocket-driven run/job status changes (`event-v2.service.ts`, `run-job.component.ts`) update the UI silently; toasts go through ng-zorro `NzNotificationService` (live-region behavior unverified).

### Strengths to build on
- Interactions overwhelmingly use real `<button>`/`nz-button` (350+).
- Data tables use `nz-table` with real `<thead>`/`<th>`.
- ng-zorro provides framework ARIA for `nz-modal` (focus trap, `role="dialog"`, Esc), `nz-select`, `nz-tabset`, `nz-dropdown`.
- No positive `tabindex` values anywhere (no tab-order traps).
- Run list/header status already uses distinct icon shapes per status, not color alone.
- v2 graph already has a keyboard navigation model (`navigationGraph`) to attach real focus semantics to.

---

## 2. Standards & references

| Standard | Use |
|---|---|
| WCAG 2.2 Level AA | Conformance target; every work item below maps to success criteria (SC) |
| WAI-ARIA 1.2 + ARIA Authoring Practices Guide (APG) | Widget patterns: tabs, dialogs, disclosure, listbox, toolbar, feed |
| EN 301 549 | EU procurement/EAA alignment (inherits WCAG 2.2 AA) |
| Angular CDK `a11y` package | `LiveAnnouncer`, `FocusMonitor`, `cdkTrapFocus`, `ListKeyManager` — preferred implementation vehicle |

Primary assistive-technology test matrix: **NVDA + Chrome** and **NVDA + Firefox** (primary), VoiceOver + Safari (secondary), JAWS spot checks.

---

## 3. Visual-impact classification & ground rules

Every work item is tagged with one of:

| Tag | Meaning | Ships in |
|---|---|---|
| **[A]** | **Zero visual impact in any state.** Attribute-only changes (`aria-*`, `role`, `id`, `lang`, `alt` on already-broken images), TypeScript-only changes (LiveAnnouncer, focus logic that doesn't add styles), off-screen DOM (`.sr-only` content), tooling/CI. | PR #1 wave (now) |
| **[B]** | **Pixel-identical for mouse users; visible only during keyboard interaction.** Focus outlines on elements newly made focusable, the skip link (rendered off-screen, appears only on Tab focus). The default rendering of every page is untouched. | PR #2 wave (own review, explicitly flagged) |
| **[C]** | **Visible UI change or element swap with styling risk.** New controls (list-view toggle, reorder buttons), element replacements (`<a>`→`<button>`, div→`<h2>`, custom tabs→`nz-tabset`), CSS additions. | **Open / To Do (visual impact)** — later, per-feature PRs with screenshots |

### Techniques that keep [A] items truly invisible

1. **Attribute-only retrofit instead of element swaps.** Where semantics call for a different element (`<button>`, `<h1>`, `<main>`), the first pass adds ARIA to the *existing* element instead: `role="button"` + `tabindex="0"` + Enter/Space handler on the clickable `<div>`/`<a>`; `role="heading" aria-level="1"` on the styled div; `role="main"`/`role="banner"`/`role="navigation"` on the existing layout containers. No DOM structure, classes, or CSS selectors change, so no style can shift. Element swaps become tracked [C] debt.
2. **No new wrapper elements in PR #1.** Wrapping `<router-outlet>` in `<main>` could break descendant/child CSS selectors — use `role="main"` on the existing `div.content.inner-content` in `app.component.html` instead.
3. **`.sr-only` utility class** (visually-hidden, standard clip-pattern) added once; all supplementary text for screenreaders (status text, table alternatives, instructions) uses it. Verify it isn't already themed by ng-zorro; name it `cds-sr-only` to avoid collisions.
4. **No global CSS changes** in [A]/[B] waves. No `:focus-visible` overhauls yet — newly focusable elements simply get the browser's default focus ring (keyboard-only, [B]). The unified focus-style pass is [C].
5. **`LiveAnnouncer` over focus moves** for route changes in wave 1: announcing the new page title is invisible; programmatically moving focus can paint a focus ring — that variant is [B].
6. **Proof, not promise:** the PR runs a screenshot-diff pass (the testcafe e2e suite already captures screenshots via `-s screenshots`; add a compare step or BackstopJS/Playwright screenshot run over the main routes) demonstrating zero rendered-pixel change. This is the PR's headline evidence.

### Known [A] risk points to verify during implementation
- Adding `role`/`aria` attributes to elements ng-zorro also decorates (`nz-menu-item`, `nz-table` internals): confirm ng-zorro doesn't overwrite or conflict at runtime (it renders its own roles on some components). Test per component.
- Adding `id`s to inputs: ensure no existing CSS `#id` selectors or tests collide; use a `cds-field-…` prefix.
- Monaco `accessibilitySupport`/`ariaLabel` options: config-only, but verify no layout reflow (Monaco can reserve space differently in a11y mode when screen reader is detected — that behavior is user-triggered, not default rendering).

---

## 4. Phased plan

Phases are ordered by (impact × reach) / effort. Sizing: S ≤ 2 dev-days, M ≤ 1 week, L = 2–4 weeks. Every item carries its visual-impact tag.

### Phase 0 — Tooling, guardrails, baseline (S–M) — all [A]

Stop the bleeding before fixing the backlog: every new template must be born accessible. Nothing in this phase touches the rendered app at all.

1. **[A] Enable a11y linting** — extend `plugin:@angular-eslint/template/accessibility` in `.eslintrc.js` for `*.html`. Initially set noisy rules (`label-has-associated-control`, `click-events-have-key-events`, `interactive-supports-focus`, `alt-text`, `valid-aria`, `role-has-required-aria`) to `warn`, flip to `error` per rule as each remediation sweep lands. Add to `lint-staged` so new code is gated immediately.
2. **[A] Automated a11y checks in CI** — integrate `axe-core`: jasmine/karma helper for component tests, and an axe pass over the main routes in the e2e suite (`@testcafe-community/axe`; if e2e migrates, `@axe-core/playwright`). Record the baseline violation count; CI fails on regression above baseline; baseline ratchets down.
3. **[A] Visual-regression harness** — screenshot capture + diff over the main routes (light and dark theme), wired into CI or at least into the PR workflow. This is both the zero-visual-change proof for this initiative and a lasting safety net for all future UI work.
4. **[A] Conventions & docs** — `ui/docs/accessibility.md`: the ARIA conventions of this codebase (attribute-retrofit rules from §3, `cds-sr-only`, id-prefix scheme, `aria-label` for icon buttons, LiveAnnouncer usage), plus a PR checklist. `@angular/cdk/a11y` needs no new dependency (`@angular/cdk` is already installed).
5. **[A] Baseline manual audit protocol** — script a 30-minute NVDA smoke pass (signin → home → project → run list → run → logs → settings) and record current behavior as the reference point.

**Exit criteria:** lint + axe + screenshot-diff running in CI with a ratcheting baseline; contributor doc merged. Zero runtime changes shipped.

### Phase 1 — Global structure: language, landmarks, titles, announcements (S–M)

1. **[A]** `index.html`: add `lang="en"` to `<html>` (SC 3.1.1). Add `role="status"` to the boot-loader div.
2. **[A] Landmarks via roles, not wrappers**: `role="banner"` on the existing navbar header element, `role="navigation" aria-label="Main"` on the navbar's menu container, `role="main"` on the existing content `div` around `<router-outlet>` in `app.component.html`. (SC 1.3.1, 2.4.1)
3. **[A] Route titles + announcements** — audit `app.routing.ts` and all lazy routing files for missing `data.title`; fill gaps (title bar text changes are not "visual rendering" of the app, but flag it in the PR). On `NavigationEnd`, announce the new title via cdk `LiveAnnouncer` so SPA navigation is perceivable. (SC 2.4.2)
4. **[B] Skip link** — visually-hidden-until-focused "Skip to main content" as first focusable element, targeting the main container (`tabindex="-1"` on target, `outline: none` scoped to it). Mouse users never see it; keyboard users see it on first Tab. (SC 2.4.1)
5. **[A] Heading semantics without element swaps** — where a page's visual title is a styled `div`/`nzTitle`, add `role="heading" aria-level="N"`; fix aria-level so each page exposes exactly one level-1 heading and no skips. Actual element conversion (`<h1>` etc.) is deferred [C] debt. (SC 1.3.1, 2.4.6)
6. **[C — deferred] Unified `:focus-visible` styling pass** across both themes. Until then, browser-default focus rings apply to newly focusable elements ([B]).

**Exit criteria:** NVDA landmark (`D`) and heading (`H`) navigation plus title announcements work on every top-level route; screenshot diff clean.

### Phase 2 — Forms & names sweep (M–L, mechanical but wide) — almost entirely [A]

The single broadest defect class (526 labels / 12 associations). Pure attribute work.

1. **[A] Label association** — one canonical pattern applied everywhere: `<nz-form-label nzFor="cds-field-x">` + `id="cds-field-x"` on the control. Where no visible label exists (search/filter inputs), `aria-label`; placeholders are never the only label. Sweep order: auth → v2 (`views/projectv2`) → settings → project/application/pipeline/environment → workflow v1 → shared components. (SC 1.3.1, 3.3.2, 4.1.2)
2. **[A] Icon-only buttons** — sweep all buttons whose content is only `nz-icon`/`<i>`: add `aria-label` (keep existing `title`/tooltip for sighted users). Priority: navbar triggers, `shared/table/data-table.html:116`, graph controls (`libs/workflow-graph/.../graph.html:40-53`), `workflow.notification.list.html`. (SC 4.1.2)
3. **[A] Images** — add the 3 missing `alt`s (`shared/card/card.html:2`, `shared/status/status.icon.html:4` → `alt=""` decorative, `shared/table/data-table.html:55`); replace meaningless `alt="icon"` values. (SC 1.1.1)
4. **[A] Status text alternatives** — `aria-label="Status: Success"` on status icons (`shared/status/`, run badges, graph nodes/edges); supplementary `cds-sr-only` text where labels can't attach. Color-only cues get an invisible text equivalent now; visible non-color cues (shapes/text) are [C] follow-up where still needed. (SC 1.1.1, 1.4.1)
5. **[A] Error identification** — associate validation error text to fields via `aria-describedby`; verify ng-zorro 21's `nzErrorTip` association, add `role="alert"` where missing. (SC 3.3.1, 3.3.3)

**Exit criteria:** axe `label`/`button-name`/`image-alt` counts at zero; corresponding ESLint rules flipped to `error`; NVDA forms-mode pass on signin, project creation, run-start, settings/profile; screenshot diff clean.

### Phase 3 — Interactive semantics: fake links, click-targets, tabs, tables (M)

First pass is attribute-retrofit ([A]/[B]); element swaps become tracked [C] debt for a dedicated styling-reviewed PR.

1. **[A→B] Fake links** — the 28 `<a class="pointing" (click)>`: keep the `<a>` element; add `tabindex="0"`, `role="button"` (for actions) or `[routerLink]` (for navigation), and Enter/Space handling via a shared directive (`appClickable`) so behavior is uniform. Focusability is [B] (focus ring on Tab). Element conversion to `<button class="link-style">` is [C]. (SC 2.1.1, 4.1.2)
2. **[A→B] Clickable non-interactive elements** (~38 `div`/`span`/`li`/`td`/`i` with `(click)`): same shared-directive retrofit — `role="button"`, `tabindex="0"`, key handling. (SC 2.1.1)
3. **[A→B] Data table interactions** (`shared/table/data-table.html`, run list): sortable `<th>` get `aria-sort`, `role="button"`, `tabindex="0"`, key handling on the existing click target; clickable rows get the same retrofit plus an `aria-label` describing the target. Real in-cell links replacing row-click are [C]. (SC 1.3.1, 2.1.1)
4. **[A→B] Custom tabs (`shared/tabs/`)** — retrofit APG tab semantics onto the existing markup: `role="tablist"` on the `ul`, `role="tab"`/`aria-selected` on the items, `role="tabpanel"` on the content container, arrow-key roving tabindex in the component class. Verify ng-zorro's `nz-menu` directive doesn't fight the overridden roles at runtime — if it does, drop the `nz-menu` directive while keeping identical classes/CSS (verify pixel parity via screenshot diff). Migration to `nz-tabset` is [C] and probably unnecessary once retrofitted. (SC 1.3.1, 2.1.1, 4.1.2)
5. **[A] Custom scroll containers** — `app-scrollview`/resizable panels: `tabindex="0"` (scrollable regions are keyboard-scrollable — [B] focus ring), `role="region"` + `aria-label` where they hold primary content. (SC 2.1.1)

**Exit criteria:** full keyboard walk-through (no mouse) of project explore, run list, run view, pipeline editor chrome, settings — every visible action reachable and operable; NVDA announces role/name/state for each; screenshot diff clean (default state).

### Phase 4 — Dynamic content: live regions, toasts, streaming (M) — all [A]

Screenreader users must *hear* what sighted users *see change*. All of this is TS-level or off-screen DOM: zero visual impact.

1. **[A] Toasts** — verify NVDA announces ng-zorro `NzNotificationService` output; if not, mirror toast text through cdk `LiveAnnouncer` inside `ToastService` (`shared/toast/ToastService.ts` is the single choke point — one fix covers ~40 call sites). Errors `assertive`, success/info `polite`. (SC 4.1.3)
2. **[A] Run/job status changes** — in the v2 run view (`run.component.ts` EventV2 subscription), announce meaningful transitions ("Job build: started / succeeded / failed") via `LiveAnnouncer`, debounced to avoid flooding; same for visible run-list state changes. (SC 4.1.3)
3. **[A] Log streaming (`run-job.component.ts`)** — do *not* pipe raw log lines into a live region (flooding). Announce step start/end and failure summaries; add `role="log"` + `aria-live="polite"` on the active step's existing container as an opt-in. Step headers get disclosure retrofit: `role="button"`, `aria-expanded`, `tabindex="0"` on the existing `<div>`s ([A→B]); windowing buttons announce loaded ranges via `cds-sr-only`/LiveAnnouncer. (SC 4.1.3, 1.3.1)
4. **[A] Loading states** — `role="status"` + `cds-sr-only` text on `nz-spin` blocks and the `displayResolver` spinner; long operations announce completion. (SC 4.1.3)
5. **[A] Modal audit** — ng-zorro handles trap/Esc/role; verify each custom modal sets an accessible name (`nzTitle`/`aria-labelledby`), that focus returns to the invoking control on close (service-created modals in `workflow.component`, `user.edit`, …), and that icon-only buttons inside got Phase-2 names. (SC 2.4.3)

**Exit criteria:** NVDA scripted scenario "start a run, hear it progress and finish, open the failing job, hear the failure" passes without looking at the screen; screenshot diff clean.

### Phase 5 — Complex widgets (L — the differentiating work) — mixed; visible parts deferred

#### 5a. Workflow graph (v2 `libs/workflow-graph` first, then v1)
The DAG cannot be made fully AT-equivalent as pixels; the strategy is **make the graph itself focusable and navigable** *plus* (later) **an equivalent structured alternative**.

1. **[A→B] Real focus semantics on the existing keyboard model**: move key handling from `window:keydown` onto a focusable graph container (`tabindex="0"`, `role="application"` with `aria-roledescription`/`aria-label` explaining arrow-key navigation). Each node root (`job-node.html`) gets `role="button"`, `tabindex="-1"` (roving), and an accessible name ("Job build-linux, status Success, stage Build") — attributes only. Reuse the existing `selected` CSS class as the focus indication (no new styles); announce selection changes via `LiveAnnouncer`. Enter/Space activation is already wired. (SC 2.1.1, 4.1.2)
2. **[A] Keyboard equivalents for pointer-only features**: zoom in/out/reset already have buttons — add `+`/`-`/`0` key bindings; multi-select restart via Space toggling the per-node checkbox that already exists (mouse lasso becomes redundant, not required). (SC 2.1.1)
3. **[A] Dependency info in accessible names**: node names include "needs: X, Y" so color-only edges become redundant for AT; a `cds-sr-only` run summary ("12 jobs: 10 success, 1 failed, 1 running") accompanies the graph. (SC 1.4.1, 1.1.1)
4. **[C — deferred] Equivalent "List view" toggle** rendering the run DAG as nested semantic lists/table (stage → job → status → dependencies), sharing selection state with the graph. Highest-value visible feature; also serves sighted keyboard power users. The v2 run page already has the data structures (`initRunJobs()`).
5. **v1 graph (`views/workflow/graph/`)**: apply item 1 or, if v1 is sunsetting, rely on the [C] list view — decide with product based on v1's remaining lifespan.

#### 5b. Editors
1. **[A] Monaco** (`nz-code-editor` consumers): set `ariaLabel` in each `EditorOptions` ("Workflow YAML editor", "Retention rule editor"); confirm `accessibilitySupport: 'auto'` isn't disabled; associate surrounding visible labels. Monaco's built-in a11y is good once labeled. (SC 4.1.2)
2. **[A now, C later] CodeMirror 5** (`shared/codemirror.ts`, ~20 consumers) is not meaningfully fixable in place. Interim [A]: `aria-label` on the wrapper + document the limitation. Real fix [C]: migrate consumers to `nz-code-editor` (Monaco, already a dependency) or CodeMirror 6; a plain-`<textarea>` fallback toggle is an alternative [C].

#### 5c. Drag-and-drop (dragula)
**[C — deferred]** Keyboard alternative requires visible controls: "Move up / Move down" buttons on pipeline stages (`pipeline.workflow.component.ts`) and action steps (`action.component.ts`), reusing existing reorder handlers, with LiveAnnouncer confirmation ("Stage Deploy moved to position 2 of 4"). Interim [A]: `cds-sr-only` note that reordering has no keyboard path yet + tracked debt. (SC 2.1.1, 2.5.7)

#### 5d. Charts (`shared/chart/`)
**[A]** `role="img"` + `aria-label` summary on the ngx-charts container, plus a `cds-sr-only` data table with the same series (application home stats). A visible "view as table" toggle is optional [C]. (SC 1.1.1)

**Exit criteria:** an NVDA user can, without a mouse: read a workflow DAG, navigate to a failing job, restart selected jobs, and edit workflow YAML. (Stage reordering waits for the [C] wave.)

### Phase 6 — Verification, hardening, governance (M, then continuous)

1. **[A]** Full manual audit against WCAG 2.2 AA: NVDA+Chrome and NVDA+Firefox across scripted journeys (signin/SSO, create project, explore repos, start run, monitor run, read logs, edit pipeline as-code, admin/settings, queue). VoiceOver secondary. Fix findings within their tags.
2. **[A]** Regression protection: promote all Phase-0 lint rules to `error`; axe CI baseline at zero for covered rules; DOM-level role/name/live-region assertions in e2e (NVDA itself can't run in CI).
3. **[A]** Documentation: accessibility conformance statement (VPAT/EN 301 549 if OVHcloud needs procurement docs); user docs for graph keyboard shortcuts.
4. **[A]** Governance: PR-template accessibility checklist; "accessible by default" contributor-guide section; NVDA smoke test in release QA; a11y triage label.
5. **[C backlog]** Schedule the deferred visible wave as individually reviewable PRs with before/after screenshots: focus-visible styling, skip link (if held back), element-swap debt (buttons/headings/links), graph list view, drag-drop reorder buttons, CodeMirror migration, non-color status cues.

---

## 5. Work breakdown summary

| Phase | Theme | Size | Visual impact | Key WCAG SC | Depends on |
|---|---|---|---|---|---|
| 0 | Lint + axe + screenshot-diff CI + conventions | S–M | [A] none | — (guardrails) | — |
| 1 | lang, landmark roles, titles, announcements, heading roles | S–M | [A] (+1 [B] skip link) | 1.3.1, 2.4.1/2/6, 3.1.1 | — |
| 2 | Form labels, icon-button names, alt text, status text, errors | M–L | [A] | 1.1.1, 1.3.1, 1.4.1, 3.3.x, 4.1.2 | 0 |
| 3 | Fake links, click-divs, tabs, tables retrofit | M | [A→B] | 1.3.1, 2.1.1, 4.1.2 | 0 |
| 4 | Live regions, toasts, streaming logs, modals | M | [A] | 4.1.3, 2.4.3 | 1 |
| 5 | Graph focus semantics, Monaco labels, chart alternatives | L | [A→B] (+[C] deferred) | 1.1.1, 1.3.1, 2.1.1 | 3, 4 |
| 6 | Audit, hardening, governance, [C] backlog scheduling | M + ongoing | [A] | all | 1–5 |

Phases 1–3 are parallelizable across contributors after Phase 0 lands. Phase 5a can start alongside 2–3 since it is isolated in `libs/workflow-graph`.

## 6. PR wave 1 — concrete scope (strictly zero visual change)

Everything tagged [A] above, deliverable as one PR or a small stack:

- Phase 0 entirely (lint `warn`, axe baseline, screenshot-diff harness, `docs/accessibility.md`, `cds-sr-only` utility).
- Phase 1 items 1–3, 5 (`lang`, landmark roles, titles + LiveAnnouncer, heading roles). Skip link ([B]) goes in wave 2.
- Phase 2 entirely (labels/ids, `aria-label` on icon buttons, `alt`, status labels, `aria-describedby` errors).
- Phase 4 entirely (toast/status/log announcements, `role="status"` loaders, modal name/focus-return audit).
- Phase 5b-1, 5d (Monaco `ariaLabel`, chart labels + sr-only table).

**PR evidence:** screenshot diff over main routes (both themes) showing zero pixel change; axe before/after counts; NVDA smoke-pass notes.

**Wave 2 ([B] — keyboard-visible only, flagged for design sign-off):** skip link; `tabindex` retrofits from Phase 3 and Phase 5a (focus rings appear on Tab only, default browser style).

**Wave 3+ ([C] — visible, per-feature PRs with screenshots):** focus-visible styling, element-swap debt, graph list view, drag-drop reorder buttons, CodeMirror migration, visible non-color status cues.

## 7. What changes for developers (and what they gain)

This initiative changes day-to-day frontend development in `ui/`. Communicate this up front — most of it is upside.

### New obligations
- **A11y lint gates**: `@angular-eslint/template/accessibility` rules run via `lint-staged` on every commit and in CI. New templates must have labeled controls, alt text, key handlers on click targets, valid ARIA. Rules arrive as `warn` and flip to `error` per completed sweep — no big-bang breakage of open branches.
- **New conventions to follow** (all documented in `ui/docs/accessibility.md`): `cds-field-…` id scheme + `nzFor` on every form control; `aria-label` mandatory on icon-only buttons; the `appClickable` directive instead of raw `(click)` on non-interactive elements; `LiveAnnouncer` for async outcomes; `cds-sr-only` for AT-only text.
- **PR checklist** gains an accessibility section; reviewers check roles/names/announcements like they check tests.
- **axe + screenshot baselines in CI**: a PR that introduces new axe violations or unexpected pixel diffs fails.

### Developer benefits (mostly free by-products)
- **Screenshot-diff harness (Phase 0.3) protects *all* UI work**, not just a11y — theme regressions, ng-zorro upgrades, and CSS refactors get caught automatically. This is arguably the biggest DX win of the whole plan.
- **Stable, semantic test selectors**: once controls have roles and accessible names, e2e and component tests can query "the button named *Restart job*" instead of brittle CSS-class chains — fewer flaky tests, and tests stop breaking on markup refactors.
- **Uniform interaction patterns**: one `appClickable` directive and one form-field pattern replace today's five ad-hoc ways of making things clickable/labeled — less copy-paste divergence, easier reviews.
- **Documented shared-component contracts**: the sweeps force overdue documentation of `shared/` components (data-table, tabs, status) and flush out dead code already found by the audit (`ng2-completer` is an unused dependency; `semantic.json` is leftover build config — both removable).
- **Better debugging of async flows**: LiveAnnouncer call sites make the "what changed and when" of WebSocket-driven state explicit and loggable.
- **Keyboard operability helps power users and devs**: full keyboard paths through run views and the graph speed up everyone, not just AT users.
- **Future-proofing**: EN 301 549 / EAA compliance becomes a checklist, not a fire drill, and ng-zorro/Angular upgrades get safer thanks to the axe + screenshot safety nets.

## 8. Implementation status (2026-07-14)

**✅ Implemented — [A] and [B] waves** (attribute-only retrofits, announcements, tooling; zero visual change in default rendering, browser-default focus rings visible only during keyboard use):

- `lang="en"`, `role="status"` boot loader (`index.html`); landmark roles (`role="banner"`, `role="navigation"`, `role="main"` + `id="main-content"`); skip link; `cds-sr-only` utility.
- Route-title announcements via cdk `LiveAnnouncer` (`app.component.ts`); toast announcements via `ToastService` (polite/assertive); banners/spinners as `role="status"`.
- `appClickable` directive (`shared/directives/clickable.directive.ts`) retrofitting `role="button"`/`tabindex`/Enter/Space onto clickable non-interactive elements; applied across views.
- Forms sweep: `nzFor` + `cds-field-…` ids, `aria-label` fallbacks; icon-only buttons named; `alt` fixes; status icons labelled (`aria-label="Status: …"`); data-table `aria-sort` + keyboard sort; tabs ARIA retrofit; run-log disclosure (`aria-expanded`, `role="log"`); run status change announcements; Monaco `ariaLabel`s; workflow-graph focus semantics, roving focus, node labels, keyboard zoom, selection announcements.
- ESLint template accessibility rules at `warn` (`.eslintrc.js`); conventions doc `ui/docs/accessibility.md`.

**🔲 Open / To Do (visual impact)** — every remaining [C] item; each needs its own PR with before/after screenshots and design sign-off:

- [ ] Unified `:focus-visible` styling across both themes (today: browser-default rings).
- [ ] Element-swap debt: fake links → real `<button>`/`<a routerLink>` styled as links; `role="heading"` divs → real `<h1>`–`<h4>`; landmark roles → native `<main>`/`<header>`/`<nav>` elements.
- [ ] Clickable table rows → real in-cell links (row-click stays as pointer enhancement).
- [ ] Custom tabs → `nz-tabset` migration (if the ARIA retrofit proves insufficient with NVDA).
- [ ] Workflow graph "List view" toggle — equivalent semantic list/table of the DAG, shared selection state.
- [ ] Drag-and-drop keyboard alternative: visible "Move up / Move down" controls on pipeline stages and action steps.
- [ ] CodeMirror 5 migration (to Monaco/`nz-code-editor` or CodeMirror 6); or visible plain-textarea fallback toggle.
- [ ] Visible non-color status cues where still missing (graph edges/forks).
- [ ] Charts: visible "view as table" toggle (sr-only fallback exists).
- [ ] Visible-on-focus enhancements pass (e.g. making the skip link styling match final design).
- [ ] axe-core CI integration + screenshot-diff harness (tooling; no runtime visual impact but not yet wired into CI).

## 9. Definition of done (per screen)

A screen is "screenreader-ready" when, under NVDA:
1. Landmarks, one level-1 heading, and a correct page title are announced on arrival.
2. Every interactive element is reachable by Tab/arrow keys, has a role, an accessible name, and announced state (expanded/selected/checked/sorted).
3. Every form field announces its label, required state, and validation errors.
4. No information is conveyed by color, icon, or position alone (an AT-perceivable equivalent exists).
5. Asynchronous outcomes (saves, run status, errors) are announced without user-initiated re-reading.
6. Focus never disappears: dialogs trap and restore it; deletions/route changes move it somewhere sensible.
7. axe reports zero violations for the enabled rule set — and the screenshot diff for the change is clean unless the PR is an approved [C]-wave change.
