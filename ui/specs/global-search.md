# Specification: Global Search

## Overview

The global search is the cross-project entry point of CDS. It answers "where is that thing?" by
matching a free text query against the **names and paths** of the objects a user can read, and
returns them ranked by how well they match. It is reachable from two places sharing the same
behaviour:

- the **navbar search bar**, present on every page, which shows live suggestions and can jump
  straight to a result;
- the **search page** (`/search`), which lists the full paginated results.

It deliberately searches *definitions*, not runtime data. Looking for workflow runs is the job of
the project run list, see [workflow-run-search.md](./workflow-run-search.md).

---

## 1 — Searched Objects

Three kinds of results, each identified by a path-like id and displayed with a label:

| Type | Id | Label |
|---|---|---|
| `project` | project key | project name |
| `workflow` | `project/vcs/repository/name` | workflow name |
| `workflow-legacy` | `project/name` | workflow name |

Only the **head** version of an as-code workflow is searched, so a workflow is returned once even
when it exists on many git refs. Its known refs are attached to the result as **variants**, which
lets the UI offer a per-ref link.

Projects also carry a description, shown alongside the result.

---

## 2 — Query Syntax

The search box is the shared filter input, so the query is a space separated list of tokens where a
token is either a `key:value` filter or a free text word.

### 2.1 Filters

| Key | Restricts to |
|---|---|
| `project` | one or more project keys |
| `type` | one or more result types (`project`, `workflow`, `workflow-legacy`) |

Several values for the same key are OR-ed. The available keys and their possible values are provided
by the API, the project options being limited to the projects the user can read.

### 2.2 Free Text

- A word matches, case-insensitively and anywhere inside the value, the **label or the id** of a
  result.
- **All words must match**, so words can be given in any order and each one narrows the results
  further. `ovh cds` finds the objects whose name or path contains both.
- The `%`, `_` and `\` characters are matched literally, they are not wildcards.
- An empty query returns everything the user can read, subject to the filters.
- A bounded number of words is taken into account, so an oversized query cannot multiply the cost of
  a search. Extra words are ignored.

### 2.3 URL

A search lives in the URL: filters under their own key, free text under `query`, and the page number
under `page`. Searches are therefore shareable and survive a reload. When rebuilt from the URL, the
search box is written in a canonical order — filters first, free text last.

---

## 3 — Result Prioritisation

Results are ordered by three criteria, applied in turn:

1. **Match quality** — a result whose **label** matches all the words comes first, then one whose
   **id** matches all the words, then one that only matches when label and id are considered
   together (for instance one word matching the project key and another the workflow name). The
   intent is that typing a name surfaces the thing itself before the things whose *path* happens to
   contain it.
2. **Label length**, shortest first — with an exact-ish match, the shorter name is the more likely
   target. Searching `cds` puts `cds` ahead of `cds-workers`.
3. **Type**, projects first, then as-code workflows, then legacy workflows.

Ordering is independent of the number of matched words: the words only decide *whether* a result
matches, never how high it ranks.

---

## 4 — Authorization

The search never widens what a user can see:

- an **administrator or maintainer** searches every project;
- any other user searches only the projects they have the read role on.

A `project:` filter is intersected with that allowed set, so naming a project the user cannot read
yields nothing rather than an error. Authorization is enforced when building the searched project
list, not by filtering results afterwards.

---

## 5 — Navbar Search Bar

- Suggestions are refreshed as the user types, debounced so that a burst of keystrokes triggers a
  single request, and limited to a short list.
- A suggestion is labelled `label - id` and navigates directly to the object's default destination:
  the project page for a project, the run list filtered on that workflow for an as-code workflow,
  and the workflow page for a legacy workflow.
- Submitting instead of picking a suggestion opens the search page with the same query.

---

## 6 — Search Page

- Each result shows its type, label, id and description, with links to the destinations that make
  sense for it: explore the definition, or list the runs. As-code workflows can additionally be
  opened per git ref through their variants. Legacy workflows only offer their own page.
- Results are paginated, the total number of matches being returned next to the page so that the
  count and the pager can be displayed.

---

## 7 — Bookmarks

The home page pairs the search with the user's **bookmarks**, the objects they starred. Bookmarks
cover the same three kinds of objects but are a separate, curated list — they are not search
results and are not ranked.
