# Specification: Workflow Run Search

## Overview

The project run list (`/project/:key/run`) lets users find workflow runs through a single search
box. That box accepts two kinds of input, freely mixed:

- **Filters** — `key:value` tokens that constrain a specific field to an exact value.
- **Free text** — bare words that are matched as a case-insensitive substring against the most
  useful run fields at once, in the spirit of the CDS global search available from the home page.

The search box is also used by the global search; the shared behaviour lives in the
`app-input-filter` component.

---

## 1 — Search Box

### 1.1 Tokens

The raw search text is a space separated list of tokens. A token containing exactly one `:` is a
filter; any other token is a free text word. Spaces inside a filter key or value are encoded with a
non-breaking space so that a filter stays a single token.

### 1.2 Autocomplete

While typing, the box suggests:

- the current text as a "submit" entry,
- the available filter keys with an example value, when the caret is not inside a filter value,
- the known values of a filter, filtered by what has been typed, when the caret is inside a filter
  value.

Available filter keys and their known values are provided by the API, computed from the runs that
already exist in the project. Annotation keys used by the project's runs are exposed as filter keys
too.

### 1.3 Submission

The search never runs on keystroke. Submitting (Enter, or picking a suggestion) writes the search
into the URL query params, and the URL is the single source of truth that triggers the request. A
dedicated button re-runs the current search without touching the URL.

---

## 2 — Filters

| Key | Matches |
|---|---|
| `workflow` | workflow path `vcs/repository/name` |
| `workflow_repository` | repository holding the workflow, `vcs/repository` |
| `workflow_ref` | git ref of the workflow definition |
| `repository` | repository the run operates on, `vcs/repository` |
| `ref` | git ref of the run |
| `commit` | full commit sha |
| `actor` | user or VCS user that triggered the run |
| `author` | commit author |
| `status` | run status |
| `template` | workflow template path |
| *anything else* | run annotation, by `key:value` |

Semantics: several values for the same key are OR-ed, different keys are AND-ed, and annotations
are all required.

---

## 3 — Free Text Search

### 3.1 Fields

A free text word matches, case-insensitively and anywhere inside the value, any of:

- project key
- workflow path (`vcs/repository/name`) and `name#run_number`
- workflow ref
- run status
- run repository (`vcs/repository`), git ref, commit sha, commit author, commit message
- run version
- the user or VCS username that triggered the run
- the run annotations (`key:value`)

### 3.2 Semantics

- **All words must match**, each one independently, so words can be given in any order and each
  narrows the result set further. `awesome main` matches a run whose workflow name contains
  "awesome" and whose ref contains "main".
- Free text is AND-ed with the `key:value` filters.
- The `%`, `_` and `\` characters are matched literally, they are not wildcards.
- Free text is carried by the `query` URL param, so a search is shareable and can be saved like
  any other search.
- When a search is reloaded from the URL, the search box is rewritten in a canonical order: the
  `key:value` filters first, the free text last, so the filters in effect stay easy to spot.

These free text rules — all words required, order free, wildcards matched literally — are shared
with the CDS global search, which applies them to the label and the id of its results.

Consequence: an annotation whose key is literally `query` cannot be filtered with `query:value`,
since that token is read as free text.

### 3.3 Ordering

Free text does not influence ordering; results keep the sort selected by the user (started or last
modified, ascending or descending). This differs from the global search, which ranks results by
match relevance.

---

## 4 — Saved Searches

A search can be saved under a name, either privately (stored in user preferences) or shared at
project level. A saved search stores the raw search text and the sort, so it keeps both its filters
and its free text. Saved searches appear in the run list sidebar as links carrying the search as URL
query params.

---

## 5 — Live Updates

The run list subscribes to the run events of the project over the websocket, and the search it shows
is sent along with the subscription: an event is pushed only for a run that search matches, which the
API reads from the run the event carries.

Where such a run goes is the list's own decision, taken from the sort in effect. A run it already
displays is replaced where it stands; a run arriving is inserted at the position the sort gives it,
and dropped when that position falls past the page, since it then belongs to a later one. The result
count only moves for a run that is new to the search.

Only the first page follows the events. Further pages are left as they were read: a list showing one
of them has nowhere to put a run arriving, and coming back to the first page reads it again.

---

## 6 — Empty Results

A search returning no run is often a dead end reached from the global search: the user picked a
workflow that exists but has never run, and lands on an empty list.

When there is no run to show, the run list looks for the workflows the search was aiming at — taken
from the `workflow` filter when there is exactly one, otherwise from the free text — and searches the
project's workflow definitions for them. If any match, they replace the empty state with a single
centered block: a message stating that no run matches this search and that the workflows below do,
then the workflows themselves. The message is not separated from the list, they announce one
another; only the workflows are separated from each other by a line. Each one shows its name,
linking to its definition, and two compact actions — **look at its definition** in the explore view,
or **start a run** of it. Starting a run this way pre-selects that workflow only, without inheriting
the filters that returned nothing. Only the first few matches are listed, and the message is worded
for their number.

A workflow is named the same way everywhere it appears — run rows, run view header, matching
workflows — its name first, then its `vcs/repository` scope in a lighter style.

The ref of the workflow definition is only shown when naming a workflow, not when naming a run: next
to a run it would be read as the ref the run checked out, which it is not as soon as the workflow
targets another repository. A workflow defined on several refs shows the first one with a counter for
the others, the full list being in the tooltip.

The wording stays factual: it states that no run matches the search and that these workflows match,
never that the workflows have no run — the empty result may well come from another filter such as a
status. When no workflow matches, or when the search carries nothing to identify one, the plain
"no result" message is shown.

This lookup only happens when the result list is empty, and a failure to retrieve the suggestions
leaves the plain empty state untouched.

Starting a run from there leaves the empty state as soon as that run reaches the list through the
websocket: it counts as a result, so the message and the suggested workflows give way to the run,
and the list controls appear with it.

---

## 7 — Opening a Run

A run is opened by clicking anywhere on its line, not only on its name. This is a convenience for
the mouse: the line is not a control of its own, the run name inside it stays the keyboard and
screen reader entry point. Clicks aimed at another control of the line, modified clicks and clicks
ending a text selection are left alone.

---

## 8 — Pagination

Results are paginated, with the total number of matching runs returned next to the page so the
footer can display the result count and the pager. Both the filters and the free text are applied
before counting.
