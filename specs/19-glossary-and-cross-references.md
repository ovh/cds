---
title: Glossary and Cross-References
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# Glossary and Cross-References

This document is the dense reference companion to the rest of the spec
series. It aggregates every term, status, event, hook type, result type,
expression operator, builtin function, and context field that the other
twelve documents introduce, so that a maintainer can find the canonical
name, the canonical Go identifier (for grep), and the spec that
documents it — all in a single lookup.

## 1. Scope

**In scope** — Alphabetical glossary covering both generations with Go
identifiers and source files; v1 status reference; v2 workflow-run and
job statuses; operation and hook-event statuses; the v2 event
catalogue; the v1 event taxonomy; the seven v2 hook types and the nine
v2 hook event names with their sub-types; the v2 run-result types; the
expression-language operators and the builtin functions; the eleven
context maps with their fields; a cross-reference table mapping every
concept to the spec that documents it.

**Out of scope** — Nothing new. This document only consolidates.

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [Glossary](#3-glossary)
4. [V1 status reference](#4-v1-status-reference)
5. [V2 status reference](#5-v2-status-reference)
6. [Other status references](#6-other-status-references)
7. [V2 events](#7-v2-events)
8. [V1 events](#8-v1-events)
9. [V2 hook types and event names](#9-v2-hook-types-and-event-names)
10. [V2 run-result types](#10-v2-run-result-types)
11. [Expression operators](#11-expression-operators)
12. [Expression builtin functions](#12-expression-builtin-functions)
13. [Context reference](#13-context-reference)
14. [Cross-reference table](#14-cross-reference-table)
15. [Spec series index](#15-spec-series-index)

## 3. Glossary

Each entry is tagged `[v1]`, `[v2]`, or `[both]`. File references point
to the canonical Go definition (without line numbers).

- **Action** `[both]` — Reusable unit of execution. v1: rows in the
  `action` table (`sdk/action.go`). v2: a YAML entity referenced via
  `uses:` (`sdk/v2_action.go`).
- **Ascode** `[v2]` — Pattern of storing CDS objects (workflows,
  actions, worker models, templates) as YAML files in a Git repository
  under `.cds/`. Runtime model: `Entity` in `sdk/entity.go`.
- **CDN item** `[both]` — Object stored by the CDN service. `CDNItem`
  in `sdk/cdn.go`; eight `CDNTypeItem*` types. See
  [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md).
- **Cancel-in-progress** `[v2]` — Concurrency option
  (`WorkflowConcurrency.CancelInProgress` in `sdk/v2_workflow.go`) that
  cancels in-flight runs / jobs when a new arrival shares the same
  pool.
- **Concurrency pool** `[v2]` — Named slot pool that limits parallel
  execution. Declared in `WorkflowConcurrency.Pool`.
- **Consumer** `[both]` — Identity used to authenticate against the
  API. Multiple types (`ConsumerLocal`, `ConsumerBuiltin`,
  `ConsumerHatchery`, `ConsumerGithub`, `ConsumerGitlab`, `ConsumerOIDC`,
  `ConsumerCorporateSSO`, `ConsumerLDAP`, `ConsumerBitbucketServer`,
  `ConsumerForgejo`, …) in `sdk/token.go`.
- **Convergent encryption** `[v2]` — Encryption scheme used by the CDN
  to deduplicate identical payloads without leaking that they are
  identical (`engine/cdn/storage/encryption/`).
- **Crafting** `[v2]` — First phase of a v2 run: resolve entities,
  evaluate triggers, materialise the job DAG. Status
  `V2WorkflowRunStatusCrafting` in `sdk/v2_workflow_run.go`. Code in
  `engine/api/v2_workflow_run_craft.go`.
- **DAG** `[both]` — Directed acyclic graph. v1: `Node`s connected by
  triggers, joins, forks. v2: jobs connected by `needs:`.
- **Entity** `[v2]` — YAML file from a project repository persisted in
  PostgreSQL. Four kinds (`EntityTypeWorkerModel`, `EntityTypeAction`,
  `EntityTypeWorkflow`, `EntityTypeWorkflowTemplate`) in
  `sdk/entity.go`. Full type: `Entity` in `sdk/entity.go`.
- **EntityFinder** `[v2]` — Cross-project entity resolver. Defined in
  `engine/api/entity_search.go`.
- **EntityWithObject** `[v2]` — Composition of `Entity` + the parsed
  concrete type (`V2Workflow`, `V2Action`, `V2WorkerModel`,
  `V2WorkflowTemplate`). In `sdk/entity.go`.
- **Event (v1)** `[v1]` — Bus event for v1 resources. Defined as Go
  struct types in `sdk/event_*.go`.
- **Event (v2)** `[v2]` — Bus event for v2 resources, identified by a
  string constant. Defined as `Event*` constants in `sdk/event_v2.go`.
- **Gate** `[v2]` — Manual approval that holds a job until reviewers
  approve and/or a condition resolves true. `V2JobGate` in
  `sdk/v2_workflow.go`. Gates do not produce the `Blocked` status —
  that status is concurrency-only (cf.
  [07b §10](./07b-run-engine-v2.md#10-concurrency-engine)).
- **Gate inputs** `[v2]` — Reviewer-supplied values landing in the
  `gate` context. Type `GateInputs` in `sdk/v2_workflow_run.go`.
- **Gate reviewers** `[v2]` — RBAC groups / users allowed to approve a
  gate. `V2JobGateReviewers` in `sdk/v2_workflow.go`.
- **GPG signature** `[v2]` — Commit signature verified during
  repository analysis. Implementation in
  `engine/api/v2_repository_analyze.go` (`analyzeCommitSignature…`).
- **Gorpmapper** `[both]` — In-house ORM on top of go-gorp at
  `engine/gorpmapper/`. Adds canonical-form templating, encryption,
  and signature verification (`SignedEntity`).
- **Hatchery** `[both]` — Service that spawns workers on demand. Five
  implementations under `engine/hatchery/{local,kubernetes,openstack,swarm,vsphere}/`.
  Interface in `sdk/hatchery/types.go`.
- **Hatchery session JWT** `[both]` — Short-lived JWT minted by the
  API at `/v2/auth/consumer/hatchery/signin`.
- **Hook (incoming)** `[both]` — Trigger fed by an external event. v1:
  `NodeHook` (`sdk/workflow_hook.go`). v2: `V2WorkflowHook`
  (`sdk/v2_workflow.go`).
- **Hook (outgoing)** `[v1]` — Side-effect call from a workflow node.
  v1-only (`NodeOutGoingHook`). v2 replaces it with `workflow-run`
  triggers chaining workflows.
- **HookRepositoryEvent** `[v2]` — Central object of the hooks service.
  Defined in `sdk/hooks_repository_event.go`. Drives the
  `HookEventStatus*` state machine.
- **Initiator** `[v2]` — The user, hook, or run that started a v2 run.
  `V2Initiator` in `sdk/v2_workflow_run.go`. Carries `(UserID,
  VCSUsername, IsAdminWithMFA)` through the run engine.
- **Integration** `[both]` — Project-attached configuration providing
  access to an external system. `ProjectIntegration`,
  `IntegrationModel` in `sdk/integration.go`.
- **Item type** `[both]` — CDN-side category of an item
  (`CDNTypeItemStepLog`, `CDNTypeItemJobStepLog`, `CDNTypeItemServiceLog`,
  `CDNTypeItemServiceLogV2`, `CDNTypeItemRunResult`,
  `CDNTypeItemRunResultV2`, `CDNTypeItemWorkerCache`,
  `CDNTypeItemWorkerCacheV2`) in `sdk/cdn.go`.
- **Job** `[both]` — A unit of execution scheduled to a worker. v1:
  `Job` in `sdk/job.go`. v2: `V2Job` in `sdk/v2_workflow.go`.
- **JWS signature** `[v2]` — JSON Web Signature used for inter-service
  auth, worker logs, and entity integrity. `Signature` in
  `sdk/cdn/signature.go`.
- **Library project** `[v2]` — Project nominated as the central host
  for shared actions / worker models / templates. References of the
  form `library/<name>` resolve here. Implementation:
  `unsafeSearchEntityFromLibrary` in `engine/api/entity_search.go`.
- **Locator** `[both]` — CDN-side physical path of an item.
  Convergent-encrypted when enabled. `Locator` field on `CDNItemUnit`
  in `sdk/cdn.go`.
- **Matrix** `[v2]` — Job strategy that fans a job into N permutations.
  `V2JobStrategy.Matrix` in `sdk/v2_workflow.go`.
- **Matrix permutation** `[v2]` — One instance produced by matrix
  expansion. Stored on `V2WorkflowRunJob.Matrix`.
- **Needs** `[v2]` — Per-job list of upstream jobs that must complete
  before this job runs. Replaces v1 stages and triggers. Field
  `V2Job.Needs`.
- **Operation** `[both]` — Long-running git task handled by the
  repositories service. `Operation` in
  `sdk/repositories_operation.go`.
- **Plugin** `[both]` — gRPC binary providing actions or integrations.
  Protocol in `sdk/grpcplugin/`. Type: `GRPCPlugin` in
  `sdk/plugin.go`.
- **Project** `[both]` — Tenant container. Holds workflows,
  integrations, keys, groups, regions, variables (v1) or variable
  sets (v2). `Project` in `sdk/project.go`.
- **RBAC v2** `[v2]` — Per-scope rule system replacing v1 group
  permissions. `RBAC` and `RBAC*` sub-types in `sdk/rbac.go` and
  `sdk/rbac_*.go`.
- **Region** `[both]` — Logical pool of hatcheries. Used for placement
  and RBAC. `Region` in `sdk/region.go`.
- **Repository (project)** `[v2]` — A Git repository attached to a
  project, scanned for `.cds/` content. `ProjectRepository` in
  `sdk/repository.go`.
- **Run-attempt** `[v2]` — Counter of retries within a run.
  `V2WorkflowRun.RunAttempt` in `sdk/v2_workflow_run.go`.
- **Run-number** `[both]` — Monotonic counter of runs of a given
  workflow.
- **Run result** `[both]` — Typed output of a job. 20 v2 types in
  `sdk/v2_workflow_run_detail.go` (`V2WorkflowRunResultType*`).
- **Semver context** `[v2]` — Auto-computed version derived from git
  history or repository files. `WorkflowSemver` in
  `sdk/v2_workflow.go`.
- **SignedEntity** `[both]` — Gorpmapper marker that attaches a JWS
  signature to a database row for integrity verification.
- **Stage** `[both]` — v1: parallel layer inside a pipeline
  (`sdk/stage.go`). v2: optional grouping over jobs (`WorkflowStage`
  in `sdk/v2_workflow.go`).
- **Step** `[both]` — One command inside a job. v2 steps can be `run:`
  (script) or `uses:` (action). `ActionStep` in `sdk/v2_action.go`.
- **Storage unit** `[both]` — CDN backend that stores items. Seven
  implementations: local, S3, Swift, NFS, Redis, WebDAV, encryption
  (`engine/cdn/storage/`).
- **Trigger** `[both]` — Edge that schedules execution. v1:
  `NodeTrigger` between two nodes. v2: an `on:` event matching a
  `V2WorkflowHook`.
- **V2WorkflowRunEnqueue** `[v2]` — Envelope sent through
  `workflowRunTriggerChan` to enter the engine phase. In
  `sdk/v2_workflow_run.go`.
- **VariableSet** `[v2]` — Project-scoped named set of variables and
  items, attachable to a workflow or a job. `ProjectVariableSet`,
  `ProjectVariableSetItem` in `sdk/project_variable.go`.
- **VCS server** `[v2]` — Configured connection to a GitHub / GitLab /
  Bitbucket / Gerrit / Gitea / Forgejo instance, attached to a
  project. `VCSProject` in `sdk/vcs.go`.
- **Worker model** `[both]` — Description of the execution environment
  for a job. v1: `Model` in `sdk/worker_model.go`. v2: `V2WorkerModel`
  in `sdk/v2_worker_model.go`.
- **Worker token JWT** `[both]` — JWT signed by a hatchery with its
  private RSA key. Verified by the API at `/auth/consumer/worker/signin`.
- **Workflow run** `[both]` — Instance of a workflow execution. v1:
  `WorkflowRun` in `sdk/workflow_run.go`. v2: `V2WorkflowRun` in
  `sdk/v2_workflow_run.go`.

## 4. V1 status reference

V1 status constants live in `sdk/build.go`. `StatusIsTerminated`
classifies the terminal subset.

| Constant | Value | Use |
| --- | --- | --- |
| `StatusPending` | `Pending` | Pre-craft enqueue |
| `StatusWaiting` | `Waiting` | Awaiting mutex / parent / trigger |
| `StatusChecking` | `Checking` | Deprecated (v1 pipeline-build) |
| `StatusBuilding` | `Building` | Currently running |
| `StatusSuccess` | `Success` | Terminal: succeeded |
| `StatusFail` | `Fail` | Terminal: failed |
| `StatusDisabled` | `Disabled` | Node disabled by config |
| `StatusNeverBuilt` | `Never Built` | Conditions filtered out |
| `StatusUnknown` | `Unknown` | Default zero value |
| `StatusSkipped` | `Skipped` | Stage / job skipped at runtime |
| `StatusStopped` | `Stopped` | User stop |
| `StatusBlocked` | `Blocked` | Reserved |
| `StatusCancelled` | `Cancelled` | Reserved |
| `StatusRetrying` | `Retrying` | Reserved |
| `StatusWorkerPending` | `Pending` | Worker not yet attached |
| `StatusWorkerRegistering` | `Registering` | Worker registering |
| `StatusCrafting` | `Crafting` | Crafting (also used by the v2 craft routine on shared types) |
| `StatusScheduling` | `Scheduling` | Awaiting worker (also shared) |

## 5. V2 status reference

### 5.1 `V2WorkflowRunStatus` (`sdk/v2_workflow_run.go`)

`IsTerminated` treats `Crafting`, `Building`, `Blocked` as non-terminal.

| Constant | Value | Terminal | Meaning |
| --- | --- | --- | --- |
| `V2WorkflowRunStatusCrafting` | `Crafting` | no | Entities being resolved |
| `V2WorkflowRunStatusBuilding` | `Building` | no | Jobs running or queued |
| `V2WorkflowRunStatusBlocked` | `Blocked` | no | Blocked by a workflow-level concurrency rule |
| `V2WorkflowRunStatusSuccess` | `Success` | yes | All jobs Success / Skipped |
| `V2WorkflowRunStatusFail` | `Fail` | yes | At least one Fail without continue-on-error |
| `V2WorkflowRunStatusStopped` | `Stopped` | yes | User stop |
| `V2WorkflowRunStatusSkipped` | `Skipped` | yes | No matching trigger / filtered |
| `V2WorkflowRunStatusCancelled` | `Cancelled` | yes | Concurrency cancel-in-progress |

### 5.2 `V2WorkflowRunJobStatus` (`sdk/v2_workflow_run.go`)

| Constant | Value | Terminal |
| --- | --- | --- |
| `V2WorkflowRunJobStatusUnknown` | `` (empty) | no (bootstrap only) |
| `V2WorkflowRunJobStatusBlocked` | `Blocked` | no |
| `V2WorkflowRunJobStatusWaiting` | `Waiting` | no |
| `V2WorkflowRunJobStatusScheduling` | `Scheduling` | no |
| `V2WorkflowRunJobStatusBuilding` | `Building` | no |
| `V2WorkflowRunJobStatusRetrying` | `Retrying` | no |
| `V2WorkflowRunJobStatusSuccess` | `Success` | yes |
| `V2WorkflowRunJobStatusFail` | `Fail` | yes |
| `V2WorkflowRunJobStatusStopped` | `Stopped` | yes |
| `V2WorkflowRunJobStatusSkipped` | `Skipped` | yes |
| `V2WorkflowRunJobStatusCancelled` | `Cancelled` | yes |

## 6. Other status references

### 6.1 Operation status (`sdk/repositories_operation.go`)

| Constant | Value | Meaning |
| --- | --- | --- |
| `OperationStatusPending` | `0` | Queued |
| `OperationStatusProcessing` | `1` | Processor took the operation |
| `OperationStatusDone` | `2` | Success |
| `OperationStatusError` | `3` | Failure |

### 6.2 Hook repository-event status (`sdk/hooks_repository_event.go`)

| Constant | Meaning |
| --- | --- |
| `HookEventStatusScheduled` | Initial state |
| `HookEventStatusAnalysis` | Analysing repository |
| `HookEventStatusCheckAnalysis` | Confirming analysis (PR events) |
| `HookEventStatusWorkflowHooks` | Matching workflow hooks |
| `HookEventStatusGitInfo` | Fetching commit signature + git context |
| `HookEventStatusWorkflow` | Triggering workflows |
| `HookEventStatusDone` | Terminal: success |
| `HookEventStatusError` | Terminal: error |
| `HookEventStatusSkipped` | Terminal: skipped |

### 6.3 Run-result status

`PENDING → COMPLETED` then optionally `PROMOTED` → `RELEASED`.
`CANCELLED` is the watchdog terminal value applied by
`CancelAbandonnedRunResults` (`engine/api/v2_workflow_run_job_routines.go`).

### 6.4 Repository analysis status (`sdk/repository.go`)

`InProgress` / `Success` / `Error` / `Skipped`.

### 6.5 CDN item status (`sdk/cdn.go`)

| Constant | Meaning |
| --- | --- |
| `CDNStatusItemIncoming` | Item still being written |
| `CDNStatusItemCompleted` | Finalised, ready for sync to long-term storage |

## 7. V2 events

All v2 event constants live in `sdk/event_v2.go`. Grouped by category.

### 7.1 Analysis

`EventAnalysisStart` (`AnalysisStart`), `EventAnalysisDone`
(`AnalysisDone`).

### 7.2 Run-job

`EventRunJobEnqueued`, `EventRunJobBlocked`, `EventRunJobCancelled`,
`EventRunJobSkipped`, `EventRunJobStopped`, `EventRunJobScheduled`,
`EventRunJobBuilding`, `EventRunJobManualTriggered`,
`EventRunJobRunResultAdded`, `EventRunJobRunResultUpdated`,
`EventRunJobEnded`.

### 7.3 Run

`EventRunCrafted`, `EventRunBuilding`, `EventRunEnded`,
`EventRunRestart`, `EventRunDeleted`.

### 7.4 Entity

`EventEntityCreated`, `EventEntityUpdated`, `EventEntityDeleted`.

### 7.5 VCS

`EventVCSCreated`, `EventVCSUpdated`, `EventVCSDeleted`.

### 7.6 Hatchery

`EventHatcheryCreated`, `EventHatcheryUpdated`,
`EventHatcheryTokenRegen`, `EventHatcheryDeleted`.

### 7.7 Repository

`EventRepositoryCreated`, `EventRepositoryDeleted`.

### 7.8 Organization

`EventOrganizationCreated`, `EventOrganizationDeleted`.

### 7.9 Region

`EventRegionCreated`, `EventRegionDeleted`.

### 7.10 Permission / RBAC

`EventPermissionCreated`, `EventPermissionUpdated`,
`EventPermissionDeleted`.

### 7.11 User

`EventUserCreated`, `EventUserUpdated`, `EventUserDeleted`,
`EventUserGPGKeyCreated`, `EventUserGPGKeyDeleted`.

### 7.12 Plugin

`EventPluginCreated`, `EventPluginUpdated`, `EventPluginDeleted`.

### 7.13 Integration model and instance

`EventIntegrationModelCreated`, `EventIntegrationModelUpdated`,
`EventIntegrationModelDeleted`, `EventIntegrationCreated`,
`EventIntegrationUpdated`, `EventIntegrationDeleted`.

### 7.14 Project

`EventProjectCreated`, `EventProjectUpdated`, `EventProjectDeleted`,
`EventProjectPurge`.

### 7.15 Notification

`EventNotificationCreated`, `EventNotificationUpdated`,
`EventNotificationDeleted`.

### 7.16 VariableSet

`EventVariableSetCreated`, `EventVariableSetDeleted`,
`EventVariableSetItemCreated`, `EventVariableSetItemUpdated`,
`EventVariableSetItemDeleted`.

### 7.17 Concurrency

`EventConcurrencyCreated`, `EventConcurrencyUpdated`,
`EventConcurrencyDeleted`.

## 8. V1 events

V1 events are Go struct types — the bus uses the struct name as the
discriminator. Defined in `sdk/event_*.go`.

| Domain | File | Notable types |
| --- | --- | --- |
| Workflow | `sdk/event_workflow.go` | `EventWorkflowAdd`, `EventWorkflowUpdate`, `EventWorkflowDelete`, `EventWorkflowPermission{Add,Update,Delete}`, `EventRetentionWorkflowDryRun` |
| Application | `sdk/event_application.go` | `EventApplicationAdd/Update/Delete`, `EventApplicationVariable*`, `EventApplicationPermission*`, `EventApplicationKey*`, `EventApplicationRepository*` |
| Pipeline | `sdk/event_pipeline.go` | `EventPipelineAdd/Update/Delete`, `EventPipelineParameter*`, `EventPipelinePermission*`, `EventPipelineStage*`, `EventPipelineJob*` |
| Environment | `sdk/event_environment.go` | `EventEnvironmentAdd/Update/Delete`, `EventEnvironmentVariable*`, `EventEnvironmentPermission*`, `EventEnvironmentKey*` |
| Project | `sdk/event_project.go` | `EventProjectAdd/Update/Delete`, `EventProjectVariable*`, `EventProjectPermission*`, `EventProjectKey{Add,Delete,Disable,Enable}`, `EventProjectVCSServer{Add,Delete}`, `EventProjectIntegration{Add,Update,Delete}`, `EventProjectRepository{Add,Delete,Analyze}` |
| Action | `sdk/event_action.go` | `EventActionAdd`, `EventActionUpdate` |
| Warning | `sdk/event_warning.go` | `EventWarning{Add,Update,Delete}` |
| AsCode (legacy bridge) | `sdk/event_ascode.go` | `EventAsCodeEvent` |
| Workflow template | `sdk/event_workflow_template.go` | `EventWorkflowTemplate{Add,Update}`, `EventWorkflowTemplateInstance{Add,Update}` |
| Operation | `sdk/event_operation.go` | `EventOperation` |

## 9. V2 hook types and event names

### 9.1 Hook types (`sdk/v2_workflow.go`)

| Constant | Value | Triggered by |
| --- | --- | --- |
| `WorkflowHookTypeRepository` | `RepositoryWebHook` | VCS push, pull-request, comment |
| `WorkflowHookTypeWorkerModel` | `WorkerModelUpdate` | Commit changing a worker model |
| `WorkflowHookTypeWorkflow` | `WorkflowUpdate` | Commit changing a watched workflow |
| `WorkflowHookTypeManual` | `Manual` | Manual run |
| `WorkflowHookTypeWebhook` | `Webhook` | Generic webhook into the hooks service |
| `WorkflowHookTypeScheduler` | `Scheduler` | Cron tick |
| `WorkflowHookTypeWorkflowRun` | `WorkflowRun` | Another workflow finishes |

### 9.2 Event names (`sdk/hooks_repository_event.go`)

| Constant | Value |
| --- | --- |
| `WorkflowHookEventNamePush` | `push` |
| `WorkflowHookEventNamePullRequest` | `pull-request` |
| `WorkflowHookEventNamePullRequestComment` | `pull-request-comment` |
| `WorkflowHookEventNameManual` | `manual` |
| `WorkflowHookEventNameWebHook` | `webhook` |
| `WorkflowHookEventNameWorkflowRun` | `workflow-run` |
| `WorkflowHookEventNameScheduler` | `scheduler` |
| `WorkflowHookEventNameWorkflowUpdate` | `workflow-update` |
| `WorkflowHookEventNameModelUpdate` | `model-update` |

### 9.3 Pull-request event sub-types (`sdk/hooks_repository_event.go`)

| Constant | Value | Applies to |
| --- | --- | --- |
| `WorkflowHookEventTypePullRequestOpened` | `opened` | pull-request |
| `WorkflowHookEventTypePullRequestReopened` | `reopened` | pull-request |
| `WorkflowHookEventTypePullRequestClosed` | `closed` | pull-request |
| `WorkflowHookEventTypePullRequestEdited` | `edited` | pull-request |
| `WorkflowHookEventTypePullRequestCommentCreated` | `created` | pull-request-comment |
| `WorkflowHookEventTypePullRequestCommentDeleted` | `deleted` | pull-request-comment |
| `WorkflowHookEventTypePullRequestCommentEdited` | `edited` | pull-request-comment |

## 10. V2 run-result types

`V2WorkflowRunResultType*` constants live in
`sdk/v2_workflow_run_detail.go`. Each kind comes with a typed
`Detail` struct.

| Kind | Constant | Detail struct |
| --- | --- | --- |
| coverage | `V2WorkflowRunResultTypeCoverage` | `V2WorkflowRunResultCoverageDetail` |
| tests | `V2WorkflowRunResultTypeTest` | `V2WorkflowRunResultTestDetail` |
| release | `V2WorkflowRunResultTypeRelease` | `V2WorkflowRunResultReleaseDetail` |
| generic | `V2WorkflowRunResultTypeGeneric` | `V2WorkflowRunResultGenericDetail` |
| variable | `V2WorkflowRunResultTypeVariable` | `V2WorkflowRunResultVariableDetail` |
| docker | `V2WorkflowRunResultTypeDocker` | `V2WorkflowRunResultDockerDetail` |
| debian | `V2WorkflowRunResultTypeDebian` | `V2WorkflowRunResultDebianDetail` |
| python | `V2WorkflowRunResultTypePython` | `V2WorkflowRunResultPythonDetail` |
| deployment | `V2WorkflowRunResultTypeArsenalDeployment` | `V2WorkflowRunResultArsenalDeploymentDetail` |
| helm | `V2WorkflowRunResultTypeHelm` | `V2WorkflowRunResultHelmDetail` |
| terraformProvider | `V2WorkflowRunResultTypeTerraformProvider` | `V2WorkflowRunResultTerraformProviderDetail` |
| terraformModule | `V2WorkflowRunResultTypeTerraformModule` | `V2WorkflowRunResultTerraformModuleDetail` |
| staticFiles | `V2WorkflowRunResultTypeStaticFiles` | `V2WorkflowRunResultStaticFilesDetail` |
| npm | `V2WorkflowRunResultTypeNpm` | `V2WorkflowRunResultNpmDetail` |
| maven | `V2WorkflowRunResultTypeMaven` | `V2WorkflowRunResultMavenDetail` |
| gradle | `V2WorkflowRunResultTypeGradle` | `V2WorkflowRunResultGradleDetail` |
| sbt | `V2WorkflowRunResultTypeSbt` | `V2WorkflowRunResultSbtDetail` |
| nuget | `V2WorkflowRunResultTypeNuget` | `V2WorkflowRunResultNugetDetail` |
| puppet | `V2WorkflowRunResultTypePuppet` | `V2WorkflowRunResultPuppetDetail` |
| conan | `V2WorkflowRunResultTypeConan` | `V2WorkflowRunResultConanDetail` |

## 11. Expression operators

Parser implementation: `sdk/action_parser.go`.

| Operator | Purpose | Implementation |
| --- | --- | --- |
| `==` / `!=` | Equality / inequality | `equal()` |
| `<` / `<=` / `>` / `>=` | Numeric / lexical comparison | `compare()` |
| `&&` / `\|\|` | Boolean and / or | `and()` / `or()` |
| `!` | Boolean negation | `parseNotExpression` |
| `? :` | Ternary | `TermExpressionContext` |
| `[]` | Array / map indexing | `getArrayItemValueFromContext` |
| `.` | Property access | `getItemValueFromContext` |

The public surface exposes four entry points (in `sdk/action_parser.go`):
`Validate`, `Interpolate`, `InterpolateToString`, `InterpolateToBool`.

## 12. Expression builtin functions

Registered in `sdk/action_parser_funcs.go`.

| Function | Purpose |
| --- | --- |
| `contains(search, item)` | Substring or array membership (case-insensitive, supports glob) |
| `startsWith(s, prefix)` | Prefix test |
| `endsWith(s, suffix)` | Suffix test |
| `format(tpl, …args)` | Placeholder substitution |
| `join(arr, sep?)` | Default separator `,` |
| `toJSON(v)` | Serialise |
| `fromJSON(s)` | Parse |
| `hashFiles(…glob)` | SHA-256 of matching files |
| `success()` | All previous steps / needs succeeded |
| `failure()` | At least one failed |
| `always()` | Constant true |
| `cancelled()` | Run cancelled |
| `stopped()` | Run stopped |
| `result(type, name)` | Filter job results by type and name |
| `toLower(s)`, `toUpper(s)` | Case |
| `toTitle(s)`, `title(s)` | Title case |
| `b64enc(s)`, `b64dec(s)` | Base64 |
| `b32enc(s)`, `b32dec(s)` | Base32 |
| `trimAll(s, cutset)` | Trim both ends |
| `trimPrefix(s, p)` | Drop prefix |
| `trimSuffix(s, p)` | Drop suffix |
| `toArray(v)` | Coerce |
| `match(s, glob)` | Glob match |
| `replace(s, old, new, n?)` | Substitution with optional max replacements |
| `contextValue(name, …path)` | Dynamic context lookup |
| `default(v, fallback?)` | Fallback when empty |
| `coalesce(a, b, …)` | First non-empty |

## 13. Context reference

Eleven contexts are visible inside `${{ … }}` expressions. Detailed
semantics live in [`04-workflow-v2.md`](./04-workflow-v2.md); this
section is the field-by-field reference.

### 13.1 `cds` — `CDSContext` (`sdk/contexts.go`)

| Field | Source |
| --- | --- |
| `event_name`, `event` | Triggering event |
| `project_key` | Owning project |
| `run_id`, `run_number`, `run_attempt`, `run_url` | Run identity |
| `workflow`, `workflow_ref`, `workflow_sha`, `workflow_vcs_server`, `workflow_repository` | Workflow source coordinates |
| `workflow_template`, `workflow_template_ref`, `workflow_template_sha`, `workflow_template_vcs_server`, `workflow_template_repository`, `workflow_template_project_key`, `workflow_template_params`, `workflow_template_commit_web_url`, `workflow_template_ref_web_url`, `workflow_template_repository_web_url` | Template provenance |
| `triggering_actor` | User / service that triggered |
| `version`, `version_next` | Computed semver |
| `job`, `stage` | Current job / stage |
| `workspace` | Worker workspace path |

### 13.2 `git` — `GitContext` (`sdk/contexts.go`)

| Field | Source |
| --- | --- |
| `server`, `repository`, `repository_origin`, `repositoryUrl`, `repository_web_url`, `ref_web_url`, `commit_web_url` | URL identifiers |
| `author`, `author_email`, `commit_message` | Commit metadata |
| `ref`, `ref_name`, `ref_type`, `sha`, `sha_short` | Git ref |
| `connection`, `username`, `token`, `ssh_key`, `gpg_key`, `email` | Auth credentials (secret values, masked) |
| `semver_current`, `semver_next` | Semver shortcuts |
| `changesets` | Changed files for this commit |
| `pullrequest_id`, `pullrequest_to_ref`, `pullrequest_to_ref_name`, `pullrequest_web_url` | PR metadata |

### 13.3 Other contexts

| Context | Type | Source |
| --- | --- | --- |
| `env` | `map[string]string` | Workflow + job `env:` |
| `inputs` | `map[string]interface{}` | Manual-trigger payload |
| `jobs` | `JobsResultContext` (`sdk/v2_workflow_run.go`) | Finished jobs in the run |
| `needs` | `NeedsContext` (`sdk/v2_workflow_run.go`) | Subset of `jobs` matching `needs:` |
| `steps` | `StepsContext` (`sdk/v2_workflow_run.go`) | Previous steps in the same job (conclusion + outcome) |
| `matrix` | `map[string]string` | One permutation of the matrix |
| `integrations` | `JobIntegrationsContexts` (`sdk/v2_workflow_run.go`) | Resolved integrations declared on the job |
| `gate` | `map[string]interface{}` | Reviewer-supplied gate inputs |
| `vars` | `map[string]interface{}` | Items of declared VariableSets |

The full availability matrix (which context is visible at which scope)
is in [`04-workflow-v2.md`](./04-workflow-v2.md).

## 14. Cross-reference table

Where to find each major type or concept:

| Topic | Canonical spec |
| --- | --- |
| Service topology, request lifecycle, goroutines, channels | [`01-architecture.md`](./01-architecture.md) |
| `Project`, `Organization`, `Group`, key, VCS attachment, integration, region, `VariableSet`, `ProjectNotification` | [`02-project-and-tenancy.md`](./02-project-and-tenancy.md) |
| `Workflow`, `WorkflowData`, `Node`, `NodeHook`, `WorkflowRun`, `WorkflowNodeRun`, `WorkflowNodeJobRun`, `WorkflowTemplate` (v1), YAML v1.0 / v2.0 | [`03-workflow-v1.md`](./03-workflow-v1.md) |
| `V2Workflow`, `V2Job`, `V2Action`, `V2WorkerModel`, `V2JobGate`, `V2JobStrategy`, `WorkflowConcurrency`, `WorkflowSemver`, `WorkflowOn`, expression language | [`04-workflow-v2.md`](./04-workflow-v2.md) |
| `Entity`, `EntityWithObject`, `EntityFinder`, `V2WorkflowTemplate`, `.cds/` layout, repository analysis, signature verification, cross-project resolution | [`05-ascode-entities.md`](./05-ascode-entities.md) |
| V1 hook taxonomy (10 built-in models), `NodeHook`, `NodeOutGoingHook`, `Task`, `TaskExecution`, `/webhook/{uuid}`, `/task/*`, Gerrit / Kafka / RabbitMQ listeners, outgoing hooks | [`06a-hooks-v1.md`](./06a-hooks-v1.md) |
| Hooks service architecture, `V2WorkflowHook`, 7 `WorkflowHookType*`, `HookRepositoryEvent` lifecycle, v2 matching, schedulers, HMAC secrets, outgoing workflow-run events | [`06b-hooks-v2.md`](./06b-hooks-v2.md) |
| V1 run engine: `WorkflowRun`, `WorkflowNodeRun`, `WorkflowNodeJobRun`, process engine, v1 statuses, retention | [`07a-run-engine-v1.md`](./07a-run-engine-v1.md) |
| V2 run engine: `V2WorkflowRun`, `V2WorkflowRunJob`, `V2WorkflowRunResult`, `V2Initiator`, craft + engine phases, queue, concurrency engine, watchdogs, retention | [`07b-run-engine-v2.md`](./07b-run-engine-v2.md) |
| Authentication: `AuthConsumer`, `AuthSession`, scopes, JWT signing + rotation, MFA, link system, v1 legacy ACL | [`08-auth.md`](./08-auth.md) |
| V2 RBAC: 7 scope tables (`RBACGlobal`, `RBACProject`, `RBACRegion`, `RBACHatchery`, `RBACWorkflow`, `RBACVariableSet`, `RBACRegionProject`), roles, glob, DAO, middleware, bypasses | [`09-rbac.md`](./09-rbac.md) |
| Hatchery contract, the five implementations, region binding, worker-model dispatch | [`10-hatcheries.md`](./10-hatcheries.md) |
| Worker binary, in-worker execution, services, plugin invocation flow, log streaming | [`11-workers.md`](./11-workers.md) |
| `CDNItem`, `CDNItemUnit`, item types, storage units, buffers vs storages, LRU, log streaming over TCP, log signature, GC | [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md) |
| VCS service, the seven providers, `VCSServer`, `VCSAuthorizedClient`, commit-status reporting, repositories service, operations, link system, multi-VCS workflows | [`13-vcs.md`](./13-vcs.md) |
| `IntegrationModel`, built-in catalogue (Kafka, RabbitMQ, OpenStack, Artifactory, AWS), `IntegrationType*` matrix | [`14-integrations.md`](./14-integrations.md) |
| cdsctl: command catalogue, config, context, auth flow | [`15-cli.md`](./15-cli.md) |
| Go SDK: `cdsclient.Interface`, HTTP layer, factories, events websocket client | [`16-sdk.md`](./16-sdk.md) |
| gRPC plugin protocol, `ActionPlugin` / `IntegrationPlugin` services, plugin storage, built-in action + integration catalogues | [`17-plugins.md`](./17-plugins.md) |
| Angular UI structure (views, NgRx store, services, websocket clients), engine-side websocket server | [`18-ui.md`](./18-ui.md) |
| Glossary, statuses, events, hook types, run-result types, expression operators / functions, contexts | [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md) (this file) |

## 15. Spec series index

| File | Topic |
| --- | --- |
| [`00-overview.md`](./00-overview.md) | Entry point + short glossary + V1 / V2 split |
| [`01-architecture.md`](./01-architecture.md) | Microservices, request lifecycle, inter-service auth, API goroutines |
| [`02-project-and-tenancy.md`](./02-project-and-tenancy.md) | Project, organization, groups, keys, integrations, regions, variable sets vs variables |
| [`03-workflow-v1.md`](./03-workflow-v1.md) | Legacy DAG: workflows, pipelines, stages, jobs, applications, environments, templates v1 |
| [`04-workflow-v2.md`](./04-workflow-v2.md) | Ascode YAML model: `V2Workflow`, `V2Job`, gates, matrix, concurrency, semver, expressions |
| [`05-ascode-entities.md`](./05-ascode-entities.md) | `.cds/` folder, `Entity` model, repository analysis, signatures, libraries |
| [`06a-hooks-v1.md`](./06a-hooks-v1.md) | Legacy v1 hooks: node-attached `Task` / `TaskExecution`, `/webhook/{uuid}`, `/task/*`, Kafka / RabbitMQ / Gerrit listeners, v1 schedulers, outgoing hooks |
| [`06b-hooks-v2.md`](./06b-hooks-v2.md) | Hooks service architecture, V2 hooks, `HookRepositoryEvent`, 7 hook types, matching algorithm, schedulers, HMAC, outgoing workflow-run events, maintenance |
| [`07a-run-engine-v1.md`](./07a-run-engine-v1.md) | Legacy v1 run engine: `WorkflowRun`, process engine, v1 statuses, queue, retention |
| [`07b-run-engine-v2.md`](./07b-run-engine-v2.md) | V2 run engine: craft + engine, state machine, queue, concurrency, gates, watchdogs, 20 result types |
| [`08-auth.md`](./08-auth.md) | Authentication: 9 drivers, sessions, JWT, scopes, link, v1 group ACL |
| [`09-rbac.md`](./09-rbac.md) | V2 RBAC: 7 scope tables, roles, glob, DAO, middleware, bypasses |
| [`10-hatcheries.md`](./10-hatcheries.md) | Hatcheries: contract, the five implementations, region binding, worker-model dispatch |
| [`11-workers.md`](./11-workers.md) | Worker binary, in-worker execution, plugin invocation, log streaming |
| [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md) | CDN service, items, storage units, log streaming, run results, retention |
| [`13-vcs.md`](./13-vcs.md) | VCS providers, repositories service, commit status, link system, multi-VCS |
| [`14-integrations.md`](./14-integrations.md) | `IntegrationModel`, built-in catalogue, integration types matrix |
| [`15-cli.md`](./15-cli.md) | cdsctl: command catalogue, config, context, auth flow |
| [`16-sdk.md`](./16-sdk.md) | Go SDK: `cdsclient.Interface`, factories, HTTP layer, websocket client |
| [`17-plugins.md`](./17-plugins.md) | gRPC plugins (action + integration): protocol, catalogues |
| [`18-ui.md`](./18-ui.md) | Angular UI and engine-side websocket server |
| [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md) | This file |
