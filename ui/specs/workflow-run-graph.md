# Specification: Workflow Run Graph

## Overview

The workflow run view renders an interactive **directed acyclic graph (DAG)** that visualizes the complete structure of a CDS workflow: its jobs, stages, matrix expansions, dependency edges, and runtime status. The graph is the central element of the run page — it drives navigation, status monitoring, and operational features such as **job restart** and **gate management**.

---

## 1 — Graph Rendering

### 1.1 Rendering Pipeline

The graph is built from the workflow YAML definition and runtime job data:

1. The YAML workflow is parsed into an internal model of jobs, stages, and their dependencies.
2. Each job run is matched to its definition by name. Matrix jobs (jobs with multiple parameter combinations) are grouped under a single matrix node.
3. The graph engine lays out nodes and edges using the **dagre** layout algorithm, then renders them as SVG.
4. After drawing, the graph auto-centers to fit the viewport. A resize observer re-centers on container changes.

### 1.2 Layout

The graph layout can be either **horizontal** (left-to-right) or **vertical** (top-to-bottom). Users can toggle the direction; doing so rebuilds the entire graph.

Layout spacing ensures a minimum separation between nodes on the same rank and between ranks.

---

## 2 — Node Types

### 2.1 Job Node

A rectangular node representing a single workflow job.

**Displays:**
- Job name
- Duration (live-updating while active, static when terminated)
- Status label (success, fail, building, inactive, etc.)
- Condition tooltip showing gate and if-expressions when defined

**Action buttons** (in the commands area):
- **Gate button**: Shown when the job defines a gate and the workflow is not active. For gates without inputs, a popconfirm triggers the gate directly. For gates with inputs, a button opens a drawer.
- **Restart button**: Shown when the job is terminated (but not skipped), the workflow is not active, and the job does not define a gate. Gate jobs are restarted through the gate mechanism instead.
- **Stop button**: Shown when the job is currently running.

**Status coloring:**

| Visual state | Statuses |
|---|---|
| Success (green) | Success |
| Fail (red) | Fail, Stopped |
| Building (animated) | Building, Waiting, Scheduling |
| Inactive (grey) | Skipped, Cancelled |

### 2.2 Matrix Node

A rectangular node (wider than a standard job node, variable height) representing a job that runs multiple times with different parameter combinations.

- A header displays the job name and optional gate controls
- Below, each variant is listed as a row with its own status, duration, and action buttons
- Variant keys are generated from all combinations of matrix parameters, sorted alphabetically

**Selection mode behavior:**
- A global select-all checkbox appears in the header when selection mode is active. Toggling it selects or deselects all variants at once.
- Each variant row displays an individual checkbox overlay for per-variant selection. This allows restarting specific matrix variants rather than the entire matrix.
- When all variants are selected, the select-all checkbox is checked. When none are selected, it is unchecked.

### 2.3 Stage Node

A container node that groups related jobs into a visual sub-graph with its own internal layout. Stages display:
- A title bar with the stage name
- A **center button** (expand icon) to zoom the viewport onto that stage
- An embedded sub-graph of the jobs it contains

Stages themselves are not navigable via keyboard — arrow keys skip over them and navigate to individual jobs within.

### 2.4 Fork / Join Node

Small circle nodes automatically inserted by the graph engine when edges diverge (fork) or converge (join). They show an aggregate status derived from their connected nodes. They have no controls and cannot be selected or navigated to.

**Status priority** (highest to lowest): Scheduling → Waiting → Building → Stopped → Fail → Success → Skipped.

---

## 3 — Edges

Edges represent dependencies between jobs. The rendering pipeline:

1. Builds an adjacency map from job dependencies
2. Inserts **fork nodes** where one job connects to multiple children
3. Inserts **join nodes** where multiple jobs converge onto one child
4. Deduplicates edges through the same fork/join pair
5. Colors edges based on source node status (green for success, red for fail, grey for inactive)
6. Active-status edges are drawn thicker for emphasis

On **hover**, all edges connected to the hovered node are highlighted.

---

## 4 — Zoom, Pan & Centering

### 4.1 User Interactions

- **Mouse wheel**: Zoom in/out at cursor position
- **Click + drag**: Pan the viewport
- Both panning and zooming are fully disabled during **lasso selection** (Shift-drag)

### 4.2 Centering Behaviors

| User action | Behavior |
|---|---|
| Click "Center graph" button | Reset viewport to show the entire graph at optimal scale (capped at 100%) |
| Click a stage's center icon | Zoom and center on that stage |
| Click a job node | Soft-center on that node (preserves current zoom, saves previous position for restore) |
| Arrow key navigation | Hard-center on the navigated-to node (no restore position saved) |
| Window/container resize | Re-center the currently focused node, or scale the existing view proportionally |

### 4.3 Focused Node Tracking

When a node is soft-centered (via click), the graph remembers which node is focused. On resize, this node is re-centered automatically. The focus is cleared by centering the full graph, centering a stage, hard-centering via keyboard, or deselecting a node.

---

## 5 — Keyboard Navigation

The graph maintains a **navigation graph** — a flattened representation of the DAG that abstracts away stages, matrix expansions, and fork/join nodes. Only job-level nodes are exposed for navigation.

### 5.1 Direction Mapping

**Horizontal layout:**

| Key | Action |
|---|---|
| `←` | Go to upstream (previous) job |
| `→` | Go to downstream (next) job |
| `↑` | Go to sibling above |
| `↓` | Go to sibling below |

**Vertical layout:**

| Key | Action |
|---|---|
| `↑` | Go to upstream (previous) job |
| `↓` | Go to downstream (next) job |
| `←` | Go to sibling left |
| `→` | Go to sibling right |

### 5.2 Additional Shortcuts

| Shortcut | Context | Action |
|---|---|---|
| `Enter` | A node is highlighted via keyboard | Open the job's detail panel |
| `Shift` (hold) | Workflow is terminated | Enter selection mode and enable lasso drag. If already in selection mode, re-enables lasso for additive area selection |
| `Shift` (release) | Selection mode active | Disable lasso. If nothing was selected, exit selection mode entirely |
| `Esc` | Selection mode active | Cancel selection and exit selection mode |
| `Esc` | No selection active | Close the side panel |

### 5.3 Help Tooltip

A help icon (?) in the toolbar shows a tooltip listing all keyboard shortcuts with styled key caps. The Shift shortcut line is only visible when the workflow run has terminated, since selection mode is only relevant for terminated runs.

---

## 6 — Hooks Display

Below the graph, the component displays workflow triggers (hooks):

- **During a run**: shows the trigger that started the run (manual, webhook, scheduler, etc.)
- **Without a run**: lists all hooks defined in the workflow

Clicking a hook opens its detail panel.

---

## 7 — Feature: Restart Jobs

### 7.1 Overview

Users can restart one or more jobs within a terminated workflow run. The feature introduces a **selection mode** where graph nodes become selectable, with dependency-aware constraints that prevent conflicting restart scenarios.

### 7.2 Preconditions

- The workflow run must be in a **terminal state** (Success, Fail, Stopped, or Cancelled)
- Any job can be restarted regardless of its individual status (failed, stopped, success, skipped, etc.)

### 7.3 Selection State Reset

When switching between runs (navigating to a different run), all restart selection state is cleared — selected jobs, gate data, selection mode, and graph visuals. This prevents stale selections from carrying over.

### 7.4 Selection Mode

Clicking "Restart Jobs" or pressing `Shift` activates selection mode. Each node enters one of three visual states:

| State | Meaning | User can click? |
|---|---|---|
| **Active** | Job is available for selection | Yes — checkbox shown |
| **Blocked** | Job is a descendant of a selected job | No — greyed out overlay |
| **Disabled** | Selection mode is off | No — no overlay shown |

### 7.5 Selection Constraints

When a job is selected, **all its downstream descendants are blocked**. This prevents conflicting scenarios where both a parent and its dependents would be restarted simultaneously — users must choose between restarting a parent (which triggers children naturally) or restarting specific children independently.

- On **select**: all transitive descendants are computed via a breadth-first search and become blocked
- On **deselect**: the blocked set is recalculated from scratch for all remaining selected jobs
- If a job was previously selected but becomes blocked (because an ancestor was selected), it is automatically deselected

**Matrix variant selection:**

For matrix nodes, selection operates at the individual variant level:
- The global select-all checkbox toggles all variants at once
- Individual variant checkboxes control per-variant selection
- When a matrix node is blocked (descendant of a selected parent), all its variant checkboxes are blocked
- Partial matrix selections are tracked separately from full-job selections

**Example:**
```
Job A
├── Job B (depends on A)
└── Job C (depends on A)
    └── Job D (depends on C)
```
- Select A → B, C, D become blocked
- Select C → D becomes blocked; A and B remain selectable
- Select then deselect A → B, C, D become selectable again
- B and C can both be selected (neither is a descendant of the other)

### 7.6 Lasso (Area) Selection

Hold `Shift` and drag to draw a rectangular selection area. All job nodes intersecting the rectangle are selected, subject to the descendant constraint (blocked nodes are skipped).

Visual feedback: a dashed blue rectangle with a semi-transparent blue fill. The cursor changes to crosshair.

**Reconciliation**: when the lasso moves and a previously-selected parent exits the rectangle, its children that are still under the rectangle become unblocked and are automatically selected. This ensures the lasso always selects everything it covers where possible.

Lasso can be re-engaged while selection mode is active — pressing Shift again enables additive area selection on top of existing individual selections.

**Matrix nodes:** The lasso selects individual matrix variant rows rather than entire matrix nodes. Only rows whose visual area intersects the lasso rectangle are selected. The lasso emits per-matrix diffs alongside regular job diffs, enabling partial matrix selections to be updated independently.

### 7.7 Selection Shortcuts

The restart button includes a dropdown menu (`…`) with bulk-selection shortcuts:

| Shortcut | Shown when | Behavior |
|---|---|---|
| "Select all failed jobs" | At least one job has failed | Selects all uniquely-named failed jobs |
| "Select skipped gate jobs" | See eligibility below | Selects eligible skipped gate jobs respecting descendant constraints |

#### "Select skipped gate jobs" eligibility

This shortcut is only available when the workflow run completed successfully (`Success` status) and there exists at least one gate job that was skipped but could have run. Specifically, a skipped gate job is eligible when it has at least one **succeeded parent** — meaning the job was skipped because the user didn't approve the gate, not because an upstream dependency failed.

A gate job has a "succeeded parent" when any of the following hold:
- **Job-level parents**: the gate job depends on a job (via `needs`) that succeeded
- **Stage-level parents**: the gate job is in a stage that depends on another stage, and at least one job in the parent stage succeeded
- **Root gate job**: the gate job has no upstream dependencies at all — it is always eligible if skipped

When multiple gate jobs are eligible, they are selected **incrementally** — after each selection, the descendant constraints are recalculated. This means if one gate job is a descendant of another, only the upstream one is selected.

### 7.8 Selection Counter

The validation button displays "Restart (N) jobs" in real-time. It is disabled when no jobs are selected.

### 7.9 Restart Flow

```
 Workflow in terminal state
         │
         ▼
 Activation
   • Click "Restart Jobs" button
   • Or press Shift key
   → Graph enters selection mode
         │
         ▼
 Selection
   • Click individual job overlays
   • Shift + drag (lasso)
   • Dropdown shortcuts
   Constraint: descendants are blocked
         │
    ┌────┴─────────────────────────┐
    │ Cancel (Esc / Cancel button  │
    │ / Shift release when empty)  │
    │ → Clears selections, exits   │
    └──────────────────────────────┘
         │
         ▼
 Validation — Click "Restart (N) jobs"
   • If any selected job has a gate with inputs → gate drawer opens
   • Otherwise → jobs are restarted immediately
         │
         ▼
 Gate Drawer (if needed)
   • Single unified drawer for ALL selected gated jobs
   • Inputs grouped by gate definition
   • When multiple jobs share the same gate: global section with
     opt-in per-job overrides
   • Pre-filled from previous run data
   • Confirm triggers execution
         │
         ▼
 Execution
   • Each selected job name is mapped to its run job(s)
   • For full matrix selections, the job name maps to all variant run instances
   • For partial matrix selections (individual variants), only the
     selected variants are restarted, not the entire matrix
   • All restarts execute in parallel
   • Graph refreshes automatically after completion
   • No selection limit imposed by the UI
```

---

## 8 — Feature: Gate Management

### 8.1 Overview

Gates are approval checkpoints on jobs. A gate can define **input fields** that must be filled before the job can proceed, or it can be a simple confirmation.

### 8.2 Single-Job Gate (graph interaction)

When a user clicks a gate button on a job node in the graph:

- **Gate without inputs**: a popconfirm triggers the job directly with no additional UI
- **Gate with inputs**: a drawer opens with the gate input form. The form is pre-filled from previous run attempts when available.

### 8.3 Drawer Behavior

The gate drawer is a side panel that contains:
- Gate input fields (rendered by a shared gate-inputs component)
- A "Run" button to confirm

**Dismiss behavior**: closing the drawer without submitting (clicking outside or pressing Escape) makes no API call and does **not** refresh the graph — the current view state is preserved. The graph only refreshes when the user explicitly submits.

### 8.4 Gate Pre-fill

When a gate has been triggered in a previous run attempt, its input values are loaded from the job events and pre-filled into the form. For multi-job gate drawers (restart flow), if all jobs sharing the same gate had identical previous values, those values populate the global section automatically.

---

## 9 — Feature: Job Condition Display

Job nodes display condition information in tooltips to help users understand execution prerequisites.

- Gate conditions and job if-expressions are shown
- Format: "with conditions: gate: `<expr>`, if: `<expr>`"
- If no conditions exist, no tooltip is shown

---

## 10 — Source File Reference

| Area | Key files |
|---|---|
| Graph engine (layout, zoom, shapes, lasso) | `graph.lib.ts` |
| Angular graph component (YAML parsing, run mapping, keyboard) | `graph.component.ts` |
| Navigation graph (keyboard nav, dependency traversal) | `graph.model.ts` |
| Job node | `node/job-node.component.ts` |
| Matrix node | `node/matrix-node.component.ts` |
| Stage node (sub-graph container) | `node/stage-node.component.ts` |
| Fork/join node | `node/fork-join-node.components.ts` |
| Status constants & helpers | `node/model.ts` |
| Host view (toolbar, panels, restart, shortcuts) | `run.component.ts`, `run.html`, `run.scss` |
| Trigger drawer component | `run-trigger.component.ts`, `run-trigger.html` |
