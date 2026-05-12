---
title: RBAC (workflow v2)
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# RBAC (workflow v2)

This document specifies the **v2 RBAC subsystem** of CDS: signed
permission bundles, the seven scope tables, role constants, glob
pattern matching, the DAO loaders, the route-rule helpers, the
`rbacMiddleware`, admin / maintainer bypasses, and VCS-user
enforcement.

RBAC is a **v2-only concept**. V1 workflows, pipelines, and
applications use the legacy `PermissionRead = 4` /
`PermissionReadExecute = 5` / `PermissionReadWriteExecute = 7`
group-based ACL documented in
[`08-auth.md`](./08-auth.md#13-v1-legacy-group-permissions). The
two systems do not interoperate: a route declares either a
`PermissionLevel` (v1) or `RbacCheckers` (v2), never both.

Authentication itself (identity, sessions, scopes, link, JWT) is
documented in [`08-auth.md`](./08-auth.md); this document assumes a
valid `AuthentifiedUser` has been resolved by the auth middleware.

Source code anchors. RBAC types live in `sdk/rbac.go` and
`sdk/rbac_*.go`; the RBAC DAO under `engine/api/rbac/`; route rule
helpers in `engine/api/router_rbac_rule_*.go`; the middleware in
`engine/api/router_middleware_rbac.go`.

## 1. Scope

**In scope** — The `RBAC` bundle structure; the seven scope-specific
rule types (`RBACGlobal`, `RBACProject`, `RBACRegion`, `RBACHatchery`,
`RBACWorkflow`, `RBACVariableSet`, `RBACRegionProject`); role
constants per scope; database schema; DAO loaders (`HasRoleOn*`
helpers); route rule helpers (`engine/api/router_rbac_rule_*.go`);
the `rbacMiddleware`; admin and maintainer bypasses; VCS-user
enforcement; the `SignedEntity` envelope and signature verification;
glob pattern matching on workflows and variable sets; RBAC
validation; wildcards (`AllUsers`, `AllVCSUsers`, `AllWorkflows`,
`AllVariableSets`, `AllProjects`).

**Out of scope** — Authentication framework, drivers, sessions, JWT,
scopes, link (see [`08-auth.md`](./08-auth.md)); v1 group-based
permissions (see
[`08-auth.md`](./08-auth.md#13-v1-legacy-group-permissions));
ascode RBAC enforcement during repository analysis (see
[`05-ascode-entities.md`](./05-ascode-entities.md)); HTTP middleware
ordering (see [`01-architecture.md`](./01-architecture.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [RBAC root structure](#3-rbac-root-structure)
4. [The seven scope types](#4-the-seven-scope-types)
5. [Roles per scope](#5-roles-per-scope)
6. [Database schema](#6-database-schema)
7. [Storage and integrity](#7-storage-and-integrity)
8. [DAO layer](#8-dao-layer)
9. [Glob pattern matching](#9-glob-pattern-matching)
10. [RBAC middleware](#10-rbac-middleware)
11. [Bypasses](#11-bypasses)
12. [VCS-user enforcement](#12-vcs-user-enforcement)
13. [Validation](#13-validation)
14. [Interaction with auth](#14-interaction-with-auth)
15. [Cross-spec pointers](#15-cross-spec-pointers)

## 3. RBAC root structure

RBAC v2 entries are stored as named, signed bundles. An `RBAC` (in
`sdk/rbac.go`) has a UUID, a name, timestamps, and seven
scope-specific slices:

| Slice | Type |
| --- | --- |
| Global | `[]RBACGlobal` |
| Projects | `[]RBACProject` |
| Regions | `[]RBACRegion` |
| Hatcheries | `[]RBACHatchery` |
| Workflows | `[]RBACWorkflow` |
| VariableSets | `[]RBACVariableSet` |
| RegionProjects | `[]RBACRegionProject` |

Each bundle is one logical grant set; many bundles can coexist (per
team, per project, per release). The API loads only the slices it
needs through a `LoadOptions` struct (see [section 8](#8-dao-layer)).

`PermissionSummary` (`sdk/rbac.go`) is a normalised view returned to
the UI when displaying who can do what on a resource;
`RBACsToPermissionSummary` flattens wildcards to the literal `"*"`.

## 4. The seven scope types

The seven slice types follow a consistent shape: a `Role`, optional
`AllUsers` / `AllVCSUsers` / scope-specific wildcards, named users /
groups / organisations (resolved to IDs at write time), and
per-resource selectors. Each lives in its own file under `sdk/`.

### 4.1 `RBACGlobal` (`sdk/rbac_global.go`)

Carries the role and the named/resolved users and groups
(`RBACUsersName`, `RBACGroupsName`, plus resolved `RBACUsersIDs`,
`RBACGroupsIDs`). No resource selector. A global rule **must
designate at least one user or group** — `AllUsers` is not allowed at
this scope.

### 4.2 `RBACProject` (`sdk/rbac_project.go`)

Carries: the role, the list of project keys (`RBACProjectKeys`),
named/resolved users and groups, the VCS-user list (`RBACVCSUsers`),
and the `AllUsers` and `AllVCSUsers` flags. Recipients can be:

- a list of users or groups,
- all users (`AllUsers = true`),
- VCS users (`RBACVCSUsers`),
- all VCS users (`AllVCSUsers = true`).

`AllUsers` and an explicit list of users / groups are mutually
exclusive.

### 4.3 `RBACRegion` (`sdk/rbac_region.go`)

Carries: the role, the region (`RegionID` + computed `RegionName`),
the organisations selector (`RBACOrganizations`), the VCS-user list,
named/resolved users and groups, and the `AllUsers` and `AllVCSUsers`
flags.

A region rule **must reference at least one organization**. The
platform enforces a **consistency rule at write time**: every group
declared in the rule must belong to one of the declared
organisations. The check is run against the database during
validation; mismatches reject the rule.

### 4.4 `RBACHatchery` (`sdk/rbac_hatchery.go`)

Carries: the role, the `HatcheryID`, the `RegionID`, and computed
names (`HatcheryName`, `RegionName`). No user / group selector — a
hatchery binding is a system grant.

### 4.5 `RBACWorkflow` (`sdk/rbac_workflow.go`)

Carries: the role, the `ProjectKey`, named/resolved users and groups,
the list of workflow-name patterns (`RBACWorkflowsNames` of type
`RBACWorkflowNames`), the `AllWorkflows` flag, the VCS-user list, and
the `AllUsers` flag. Workflow patterns are stored as a JSONB array
and matched with the platform's `sdk/glob` library.

### 4.6 `RBACVariableSet` (`sdk/rbac_variableset.go`)

Carries: the role, the `ProjectKey`, named/resolved users and groups,
the list of variable-set-name patterns (`RBACVariableSetNames`), the
`AllVariableSets` flag, the VCS-user list, and the `AllUsers` flag.

### 4.7 `RBACRegionProject` (`sdk/rbac_region_project.go`)

Carries: the role, the `RegionID` + computed `RegionName`, the
`AllProjects` flag, and the list of project keys (`RBACProjectKeys`).

## 5. Roles per scope

### 5.1 Global

| Constant | Value | Usage |
| --- | --- | --- |
| `GlobalRoleProjectCreate` | `create-project` | Create new projects |
| `GlobalRoleManagePermission` | `manage-permission` | Manage RBAC permissions |
| `GlobalRoleManageOrganization` | `manage-organization` | Manage organizations |
| `GlobalRoleManageRegion` | `manage-region` | Manage regions |
| `GlobalRoleManageHatchery` | `manage-hatchery` | Manage hatcheries |
| `GlobalRoleManageUser` | `manage-user` | Manage users |
| `GlobalRoleManageGroup` | `manage-group` | Manage groups |
| `GlobalRoleManagePlugin` | `manage-plugin` | Manage plugins |

A global rule must designate at least one user or group.

### 5.2 Project

| Constant | Value | Usage |
| --- | --- | --- |
| `ProjectRoleRead` | `read` | Read project resources |
| `ProjectRoleManage` | `manage` | Administer the project |
| `ProjectRoleManageNotification` | `manage-notification` | Manage notifications |
| `ProjectRoleManageWorkerModel` | `manage-worker-model` | Manage worker models |
| `ProjectRoleManageAction` | `manage-action` | Manage actions |
| `ProjectRoleManageWorkflow` | `manage-workflow` | Manage workflows |
| `ProjectRoleManageWorkflowTemplate` | `manage-workflow-template` | Manage workflow templates |
| `ProjectRoleManageVariableSet` | `manage-variableset` | Manage variable sets |

### 5.3 Region

`RegionRoleList = "list"`, `RegionRoleExecute = "execute"`,
`RegionRoleManage = "manage"`.

### 5.4 Workflow

`WorkflowRoleTrigger = "trigger"`.

### 5.5 VariableSet

`VariableSetRoleUse = "use"`, `VariableSetRoleManageItem = "manage-item"`.

### 5.6 Hatchery

`HatcheryRoleSpawn = "start-worker"`.

### 5.7 Region-project

Reuses `RegionRoleExecute` from the region role set.

## 6. Database schema

The schema is one main `rbac` table plus seven scope-specific tables:

```
rbac
├── rbac_global
├── rbac_project
├── rbac_region
├── rbac_hatchery
├── rbac_workflow
├── rbac_variableset
└── rbac_region_project
```

Pivot tables (`rbac_*_users`, `rbac_*_groups`, `rbac_*_organizations`)
resolve names to IDs at write time. JSON columns (`vcs_users`,
`workflows`, `variablesets`) hold the dynamic lists.

## 7. Storage and integrity

Every row is signed via the platform's `SignedEntity` envelope. On
load, the signature is verified for each record; **a row whose
signature is corrupted is ignored silently with a log-level error**,
without interrupting the loading of other rules. This means a
partial corruption degrades gracefully — the bundle keeps every valid
sub-rule and only the tampered ones disappear.

The signature is written by the ascode analyser when an `RBAC` entity
is committed to a repository (see
[`05-ascode-entities.md`](./05-ascode-entities.md)); for direct API
writes (via the RBAC routes), the signature is added at insertion
time.

## 8. DAO layer

The RBAC DAO lives under `engine/api/rbac/`.

### 8.1 Main loaders (`dao_rbac.go`)

| Function | Purpose |
| --- | --- |
| `LoadAll(ctx, db, …LoadOptions)` | Every RBAC bundle |
| `LoadRBACByName(ctx, db, name, …LoadOptions)` | One by name |
| `LoadRBACByID(ctx, db, id, …LoadOptions)` | One by UUID |
| `LoadRBACByIDs(ctx, db, ids, …LoadOptions)` | Bulk lookup |

`LoadOptions` (`engine/api/rbac/loader.go`) controls which slices are
populated: `Default` loads Global + Project; `All` loads everything;
per-scope options exist for `LoadRBACWorkflow`, `LoadRBACVariableSet`,
etc.

Resource-scoped helpers exist for the hot paths:

- `LoadRBACByHatcheryID` (`dao_rbac_hatchery.go`).
- `LoadRBACByRegionID` (`dao_rbac_region.go`).

### 8.2 `HasRoleOn*` helpers

Per-scope helpers answer the gate question directly:

| Scope | File | Helpers |
| --- | --- | --- |
| Global | `dao_rbac_global.go` | `HasGlobalRole` |
| Project | `dao_rbac_project.go` | `HasRoleOnProjectAndUserID`, `HasRoleOnProjectAndVCSUser`, `LoadAllProjectKeysAllowedForVCSUser` |
| Workflow | `dao_rbac_workflow.go` | `HasRoleOnWorkflowAndUserID`, `HasRoleOnWorkflowAndVCSUsername`, `LoadAllWorkflowsAllowedForVCSUser` |
| VariableSet | `dao_rbac_variableset.go` | `HasRoleOnVariableSetAndUserID`, `HasRoleOnVariableSetsAndVCSUser` |
| Region | `dao_rbac_region.go` | `HasRoleOnRegion` |
| Hatchery | `dao_rbac_hatchery.go` | bespoke load |

### 8.3 Route rule helpers

| Scope | File | Public helpers |
| --- | --- | --- |
| Global | `engine/api/router_rbac_rule_global.go` | `globalPermissionManage`, `globalRegionManage`, `globalHatcheryManage`, `globalUserManage`, `globalGroupManage`, `globalOrganizationManage`, `globalPluginManage`, `globalProjectCreate` |
| Project | `engine/api/router_rbac_rule_project.go` | `projectRead`, `projectManage`, `projectManageNotification`, `projectManageWorkerModel`, `projectManageAction`, `projectManageWorkflow`, `projectManageWorkflowTemplate`, `projectManageVariableSet` |
| Workflow | `engine/api/router_rbac_rule_workflow.go` | `workflowTrigger` |
| VariableSet | `engine/api/router_rbac_rule_variableset.go` | `variableSetItemManage`, `variableSetItemRead` |
| Region | `engine/api/router_rbac_rule_region.go` | `regionRead`, `regionManage` |
| Hatchery | `engine/api/router_rbac_rule_hatchery.go` | `hatcherySpawn` |
| Admin | `engine/api/router_rbac_rule_admin.go` | `isAdmin` |

Each helper calls the corresponding `HasRoleOn*` DAO:

| Helper | DAO |
| --- | --- |
| `projectRead`, `projectManage` | `rbac.HasRoleOnProjectAndUserID` |
| `workflowTrigger` | `rbac.HasRoleOnWorkflowAndUserID` |
| `variableSetItemManage` | `rbac.HasRoleOnVariableSetAndUserID` |
| `regionRead` | `rbac.HasRoleOnRegion` |
| `hatcherySpawn` | bespoke load (`router_rbac_rule_hatchery.go`) |
| global roles | `rbac.HasGlobalRole` |

## 9. Glob pattern matching

`RBACWorkflow.RBACWorkflowsNames` and
`RBACVariableSet.RBACVariableSetNames` are JSONB arrays of `sdk/glob`
patterns:

```
*                          # every workflow
vcs/repo/*                 # every workflow under one repo
vcs/*/release-*            # wildcards inside path segments
```

Matching is performed inside the DAO using the platform's glob
library, so the database does not need to know about pattern
semantics.

### 9.1 Wildcards

The "all-*" flags short-circuit the glob list entirely:

| Flag | Effect |
| --- | --- |
| `RBACProject.AllUsers` | Grants the role to every CDS user |
| `RBACProject.AllVCSUsers` | Grants the role to every VCS identity |
| `RBACRegion.AllUsers`, `AllVCSUsers` | Same for regions |
| `RBACWorkflow.AllUsers` | Grants the role on the workflow set to every CDS user |
| `RBACWorkflow.AllWorkflows` | The role applies to every workflow in the project, ignoring the named patterns |
| `RBACVariableSet.AllUsers` | Every CDS user |
| `RBACVariableSet.AllVariableSets` | Every variable set in the project |
| `RBACRegionProject.AllProjects` | Every project may execute in the region |

## 10. RBAC middleware

The enforcement entry point is `rbacMiddleware`
(`engine/api/router_middleware_rbac.go`): it runs the `RbacCheckers`
chain declared on the route, returning an error on the first failure.
Admin requests bypass the checker chain after `trackSudo` logs the
override (see [section 11](#11-bypasses)).

The full middleware order is documented in
[`01-architecture.md`](./01-architecture.md). Each route declares its
checkers at registration time, typically alongside its handler:

```go
r.Handle("/v2/project/{projectKey}/workflow/{vcs}/{repo}/{workflow}/run",
    nil,
    r.POST(api.postWorkflowRunHandler,
        service.OverrideAuth(/* … */)),
    service.WithRbacCheckers(api.projectRead, api.workflowTrigger))
```

## 11. Bypasses

### 11.1 Admin

`isAdmin(ctx)` returns true if the user's `Ring == ADMIN` (see
[`08-auth.md`](./08-auth.md#11-user-object)). The middleware skips
the checker after logging `trackSudo`. The audit record captures the
caller, the route, and the resource path so the override is fully
attributable.

### 11.2 Maintainer

`isMaintainer(ctx)` (`engine/api/api_helper.go`) automatically grants
`ProjectRoleRead` on any project. The escalation is intentional so
maintainers can investigate without being added to every project. It
does not grant any `Manage`-level role.

### 11.3 MFA gating

Some helpers consult `supportMFA(ctx)` and `isMFA(ctx)` and fail with
`sdk.ErrMFARequired` when the driver supports MFA but the session
does not have it — gated by a per-project feature flag. See
[`08-auth.md`](./08-auth.md#9-mfa) for the MFA model.

## 12. VCS-user enforcement

When the actor is a VCS user (no CDS user binding — typical of
commit signers identified via the link system, see
[`08-auth.md`](./08-auth.md#10-user-link-to-external-accounts)),
the helper switches to the VCS-user variant of each DAO check:

| Standard helper | VCS-user variant |
| --- | --- |
| `HasRoleOnProjectAndUserID` | `HasRoleOnProjectAndVCSUser` |
| `HasRoleOnWorkflowAndUserID` | `HasRoleOnWorkflowAndVCSUsername` |
| `HasRoleOnVariableSetAndUserID` | `HasRoleOnVariableSetsAndVCSUser` |

The ascode analyser is the main caller of the VCS-user path (see
[`05-ascode-entities.md`](./05-ascode-entities.md)).

`RBACProject`, `RBACRegion`, `RBACWorkflow`, and `RBACVariableSet`
each carry an `RBACVCSUsers` field — a slice of `RBACVCSUser`
`(VCSServer, VCSUsername)` pairs. The DAO helpers query the JSONB
column with the `@>` containment operator, so the database is the
source of truth — there is no in-memory cross-checking.
`AllVCSUsers = true` is the wildcard that grants the role to every
VCS identity of the relevant server.

## 13. Validation

`IsValidRBAC(ctx, db, rbac)`
(`engine/api/rbac/rbac_validation.go`) walks each slice and applies
the following rules:

- The bundle name cannot be empty.
- Each sub-rule must have a valid role for its scope (the role
  string is matched against the scope's enum).
- Required fields are present according to the scope:
  - Users / groups (`RBACUsersName`, `RBACGroupsName`) when
    applicable.
  - `RegionID` and `RBACOrganizations` for region rules.
  - `HatcheryID` and `RegionID` for hatchery rules.
- `AllUsers = true` combined with a non-empty `RBACUsersName` /
  `RBACGroupsName` is rejected (mutual exclusivity).
- For region rules, organisation / group consistency is verified
  against the database: every group must belong to one of the
  declared organisations.

The signature behaviour on load (see
[section 7](#7-storage-and-integrity)) complements validation: a
write-time check guards against malformed rules, and a read-time
check guards against tampered rows.

## 14. Interaction with auth

RBAC reads identity from `AuthentifiedUser` via the auth middleware
(see [`08-auth.md`](./08-auth.md#12-http-middleware)). The pipeline
is:

1. Auth middleware extracts the JWT, looks up the consumer + session,
   refreshes MFA activity, and attaches the resolved identity to the
   request context.
2. Permission / RBAC middleware reads the context:
   - V1 routes consume the user's group memberships via
     `checkWorkflowPermissions` / `checkProjectPermissions`.
   - V2 routes run their `RbacCheckers` chain, which uses
     `HasRoleOn*` to gate access.
3. The handler runs with the verified identity.

Scopes (`AuthConsumerScope*`, in
[`08-auth.md`](./08-auth.md#6-scopes)) gate **which routes** a
token can reach at all; RBAC gates **which resources** within those
routes. Both must pass.

## 15. Cross-spec pointers

- Authentication framework, drivers, sessions, scopes, link, MFA →
  [`08-auth.md`](./08-auth.md)
- Microservices, middleware ordering, inter-service auth →
  [`01-architecture.md`](./01-architecture.md)
- Project, organisation, groups, regions, integrations →
  [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)
- Workflow v2 schema, gates, reviewers →
  [`04-workflow-v2.md`](./04-workflow-v2.md)
- Ascode entities, RBAC enforcement during analysis, `SignedEntity`
  envelope → [`05-ascode-entities.md`](./05-ascode-entities.md)
- V2 hook routing → [`06b-hooks-v2.md`](./06b-hooks-v2.md)
- V2 run engine, `V2Initiator`, gate approval → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- Hatchery + worker tokens → [`10-hatcheries.md`](./10-hatcheries.md)
- Worker session lifecycle → [`11-workers.md`](./11-workers.md)
- VCS providers → [`13-vcs.md`](./13-vcs.md)
- Integrations → [`14-integrations.md`](./14-integrations.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
