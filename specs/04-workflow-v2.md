---
title: Workflow v2 (Ascode YAML Model)
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# Workflow v2 (Ascode YAML Model)

This document specifies the ascode workflow model — the current and
future model of CDS. Where [`03-workflow-v1.md`](./03-workflow-v1.md)
captures the legacy DAG, this spec captures the YAML-shaped model
stored in Git: `V2Workflow`, `V2Job`, `V2Action`, triggers,
expressions, gates, matrix, concurrency, semver. The runtime that
consumes this model is documented in
[`07b-run-engine-v2.md`](./07b-run-engine-v2.md); the on-disk
repository layout is documented in
[`05-ascode-entities.md`](./05-ascode-entities.md).

Source code anchors. Public types: `V2Workflow`, `V2Job`,
`V2JobRunsOn`, `V2JobService`, `V2JobServiceReadiness`, `V2JobGate`,
`V2JobStrategy`, `WorkflowStage`, `WorkflowConcurrency`,
`WorkflowSemver`, `WorkflowOn` and the seven `WorkflowHookType*` plus
`V2WorkflowHook`, `V2WorkflowHookData` in `sdk/v2_workflow.go`;
`V2Action`, `ActionRuns`, `ActionStep`, `ActionInput`, `ActionOutput`
in `sdk/v2_action.go`; `V2WorkerModel`, `V2WorkerModelDockerSpec`,
`V2WorkerModelOpenstackSpec`, `V2WorkerModelVSphereSpec` in
`sdk/v2_worker_model.go`. Hook event names and types in
`sdk/hooks_repository_event.go`. Expression parser in
`sdk/action_parser.go`; builtin functions in
`sdk/action_parser_funcs.go`. JSON schema export in
`sdk/jsonschema.go` and handler in `engine/api/v2_jsonschema.go`.

## 1. Scope

**In scope** — `V2Workflow` root (jobs map, stages, gates,
concurrencies, semver, integrations, variable sets, env); `V2Job`
(needs, `V2JobRunsOn`, services, strategy / matrix, gate, region,
concurrency, retry); steps (`run:` and `uses:`); `V2Action` and step
inputs / outputs; `V2WorkerModel` (Docker, OpenStack, vSphere specs);
the `on:` trigger block (`WorkflowOn`) and its event types;
`V2WorkflowHook` indexed in the database; the `${{ }}` expression
language and its builtin functions (in `sdk/action_parser.go` and
`sdk/action_parser_funcs.go`); the eleven context maps (`cds`, `git`,
`env`, `inputs`, `jobs`, `needs`, `steps`, `matrix`, `integrations`,
`gate`, `vars`); gates with reviewer rules; matrix expansion;
concurrency (workflow / job / project scopes); semver computation;
JSON-schema export endpoint; the `Lint` helpers; retention.

**Out of scope** — Repository layout, ascode persistence, signature verification (see [`05-ascode-entities.md`](./05-ascode-entities.md)); the run state machine, queue, retry, dead-job watchdogs (see [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)); hook routing and per-provider webhook parsing (see [`06b-hooks-v2.md`](./06b-hooks-v2.md)); v2 templates (briefly here, fully covered in [`05-ascode-entities.md`](./05-ascode-entities.md)); v1 (see [`03-workflow-v1.md`](./03-workflow-v1.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [Workflow schema](#3-workflow-schema)
4. [Job schema](#4-job-schema)
5. [Actions and steps](#5-actions-and-steps)
6. [Worker model](#6-worker-model)
7. [Stages and the `needs` DAG](#7-stages-and-the-needs-dag)
8. [Triggers (`on:` block)](#8-triggers-on-block)
9. [Workflow-hook indexing](#9-workflow-hook-indexing)
10. [Expression language `${{ }}`](#10-expression-language--)
11. [Context reference](#11-context-reference)
12. [Gates](#12-gates)
13. [Matrix](#13-matrix)
14. [Concurrency](#14-concurrency)
15. [Semver](#15-semver)
16. [Retention](#16-retention)
17. [JSON Schema export](#17-json-schema-export)
18. [Validation](#18-validation)
19. [Cross-spec pointers](#19-cross-spec-pointers)

## 3. `V2Workflow` schema

A v2 workflow is one YAML document parsed into `V2Workflow`
(`sdk/v2_workflow.go`) declaring:

| Key | Field | Purpose |
| --- | --- | --- |
| `name` | `Name` | Required identifier, matches `EntityNamePattern` |
| `repository` | `Repository *WorkflowRepository` | Optional explicit binding to a VCS repository |
| `on` | `OnRaw` / `On *WorkflowOn` | Trigger block (see [section 8](#8-triggers-on-block)) |
| `stages` | `Stages map[string]WorkflowStage` | Optional grouping over jobs with their own dependencies |
| `gates` | `Gates map[string]V2JobGate` | Manual-approval declarations |
| `jobs` | `Jobs map[string]V2Job` | Map of job names → job specs (required when `From` is absent) |
| `env` | `Env map[string]string` | Workflow-level environment variables |
| `integrations` | `Integrations []string` | Project integration names made available to every job |
| `vars` | `VariableSets []string` | Variable sets made available to every job |
| `annotations` | `Annotations map[string]string` | Free-form labels |
| `semver` | `Semver *WorkflowSemver` | Automatic version computation |
| `concurrencies` | `Concurrencies []WorkflowConcurrency` | Pool declarations |
| `concurrency` | `Concurrency string` | Default pool reference used by jobs that do not declare their own |
| `from` | `From string` | Workflow-template reference (mutually exclusive with `Jobs`) |
| `parameters` | `Parameters map[string]string` | Template parameter values when `from:` is set |

A deprecated `Retention` field exists for backwards compatibility —
modern projects use `ProjectRunRetention` (see
[section 16](#16-retention)).

### 3.1 Minimal example

```yaml
name: build-and-test
on: [push, pull-request]
jobs:
  build:
    runs-on: library/docker-ubuntu
    steps:
      - uses: actions/checkout
      - run: make build
```

### 3.2 Full-featured example

```yaml
name: release-pipeline
repository:
  vcs: my-github
  name: ovh/cds
on:
  push:
    branches: [main]
    tags: [v*]
  pull-request:
    branches: [main]
    types: [opened, synchronized]
  schedule:
    - cron: "0 2 * * *"
      timezone: UTC
env:
  REGISTRY: ghcr.io
integrations:
  - my-slack
vars:
  - shared-vars
semver:
  from: npm
  path: package.json
  release_refs: [refs/heads/main]
concurrencies:
  - name: deploy-prod
    pool: 1
    cancel-in-progress: true
    if: ${{ git.ref_name == 'main' }}
gates:
  approve:
    if: ${{ success() }}
    inputs:
      env:
        type: string
        options:
          values: [staging, production]
    reviewers:
      groups: [ops]
stages:
  build:
    needs: []
  deploy:
    needs: [build]
jobs:
  compile:
    stage: build
    runs-on: library/docker-ubuntu
    steps:
      - uses: actions/checkout
      - run: make build
    outputs:
      version:
        value: ${{ steps.semver.outputs.version }}
  deploy:
    stage: deploy
    needs: [compile]
    gate: approve
    concurrency: deploy-prod
    runs-on:
      model: library/docker-ubuntu
      memory: "4096"
    steps:
      - run: ./deploy.sh ${{ gate.env }}
```

## 4. `V2Job` schema

`V2Job` (`sdk/v2_workflow.go`) carries:

| Key | Field | Purpose |
| --- | --- | --- |
| `name` | `Name` | Optional human label |
| `if` | `If` | Optional execution condition |
| `gate` | `Gate` | Optional reference to a workflow-level gate |
| `steps` | `Steps []ActionStep` | Required step list (mutually exclusive with `From`) |
| `needs` | `Needs []string` | Optional upstream job dependencies |
| `stage` | `Stage` | Optional stage assignment |
| `region` | `Region` | Optional region pin |
| `continue-on-error` | `ContinueOnError` | When true, a job failure does not fail the workflow |
| `runs-on` | `RunsOnRaw` / `RunsOn V2JobRunsOn` | Worker placement (see [section 4.1](#41-runs-on)) |
| `strategy` | `Strategy *V2JobStrategy` | Matrix expansion (see [section 13](#13-matrix)) |
| `integrations` | `Integrations []string` | Per-job integration overrides |
| `vars` | `VariableSets []string` | Per-job variable sets |
| `env` | `Env map[string]string` | Per-job environment variables |
| `services` | `Services map[string]V2JobService` | Sidecar containers (see [section 4.2](#42-services)) |
| `outputs` | `Outputs map[string]ActionOutput` | Declared outputs that downstream jobs can read |
| `from` | `From` | Job-template reference |
| `parameters` | `Parameters map[string]string` | Template parameter values |
| `concurrency` | `Concurrency` | Pool reference |
| `retry` | `Retry int64` | Retry count: `0`, `1`, or `2` (validated) |

### 4.1 `runs-on`

`V2JobRunsOn` (`sdk/v2_workflow.go`) accepts either a plain string
(the worker model identifier) or a structured object. The marshaller
tolerates both forms:

```yaml
# Shorthand
runs-on: library/docker-ubuntu

# Structured
runs-on:
  model: library/docker-ubuntu
  memory: "4096"
  flavor: large
```

Three fields are supported: `Model` (required worker model name),
`Memory` (optional), `Flavor` (optional OpenStack / vSphere sizing).

### 4.2 Services

`V2JobService` (`sdk/v2_workflow.go`) is a side container started
alongside the worker, with a `V2JobServiceReadiness` block (`Command`,
`Interval`, `Retries`, `Timeout`). Typical use: an ephemeral
PostgreSQL for integration tests.

```yaml
jobs:
  test:
    runs-on: library/docker-ubuntu
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: secret
        readiness:
          command: pg_isready -U postgres
          interval: 5s
          retries: 5
          timeout: 10s
    steps:
      - run: psql -h postgres -U postgres -c '\dt'
```

A service has an image, an environment map, and a readiness probe (command, interval, retries, timeout). A service whose readiness probe never returns success fails the whole job before any step runs.

### 4.3 Outputs

A job can declare outputs that downstream jobs read via the `needs` context:

```yaml
jobs:
  build:
    outputs:
      image:
        value: ${{ steps.docker-push.outputs.image }}
        type: string
      artifact:
        value: ${{ steps.upload.outputs.path }}
        type: path
  deploy:
    needs: [build]
    runs-on: library/docker-ubuntu
    steps:
      - run: deploy.sh ${{ needs.build.outputs.image }}
```

### 4.4 Retry

`V2Job.Retry` must be `0`, `1`, or `2` (validated by
`V2Workflow.Lint`). A retried run shares the same `RunNumber` but
increments `RunAttempt`.

## 5. Actions and steps

### 5.1 `V2Action`

`V2Action` (`sdk/v2_action.go`) is a reusable unit of execution stored
in `.cds/actions/<name>.yml`. It carries:

| Key | Field | Purpose |
| --- | --- | --- |
| `name` | `Name` | Identifier (matches `EntityNamePattern`) |
| `description` | `Description` | Free-form description |
| `inputs` | `Inputs map[string]ActionInput` (description, optional default) | |
| `outputs` | `Outputs map[string]ActionOutput` (description, value expression, `Type` — `string` or `path`) | |
| `runs.steps` | `Runs.Steps []ActionStep` | The step list |
| `runs.post` | `Runs.Post` | Optional cleanup script run after the step list |

Example action (`.cds/actions/build-go.yml`):

```yaml
name: build-go
description: Build a Go binary
inputs:
  go-version:
    description: Go version
    default: "1.21"
outputs:
  artifact:
    description: Path to the built binary
    value: ${{ steps.build.outputs.path }}
runs:
  steps:
    - uses: actions/checkout
    - uses: actions/setup-go
      with:
        go-version: ${{ inputs.go-version }}
    - id: build
      run: |
        go build -o app .
        echo "path=./app" >> $GITHUB_OUTPUT
```

### 5.2 `ActionStep`

`ActionStep` (`sdk/v2_action.go`) is the polymorphic step type. A step
is either an action reference (`Uses`) or an inline script (`Run`),
never both. A step also accepts:

- `ID` — optional identifier (`EntityActionStepID` pattern) used by
  other steps to read its outputs.
- `With map[string]interface{}` — input values forwarded to a `uses:`
  action.
- `If` — optional execution condition.
- `ContinueOnError` — when true, a failure does not fail the parent
  job.
- `Env map[string]string` — step-scoped environment variables.

A `uses:` reference accepts four forms:

| Form | Meaning |
| --- | --- |
| `actions/checkout` | Public action shipped with CDS |
| `actions/checkout@v3` | Versioned reference |
| `.cds/actions/build-go.yml` | Local repository action |
| `OTHER_PROJECT/vcs/repo/actions/x@ref` | Cross-project library action |

### 5.3 Inputs and outputs in expressions

- `${{ inputs.<name> }}` — the value passed at `with:` (action) or by the manual trigger (workflow inputs).
- `${{ steps.<id>.outputs.<name> }}` — read an output from a previous step in the same job.
- `${{ needs.<job-id>.outputs.<name> }}` — read an output from a job listed in `needs:`.
- `${{ jobs.<job-id>.result }}` and `${{ jobs.<job-id>.outputs.<name> }}` — broader access used in workflow-level expressions.

To write an output from a step, append a line to the platform's output file:

```bash
echo "path=./build/app" >> $GITHUB_OUTPUT
```

## 6. `V2WorkerModel`

`V2WorkerModel` (`sdk/v2_worker_model.go`) describes the execution
environment a hatchery uses to spawn a worker. It carries `Name`
(matches `EntityNamePattern`), `Description`, `OSArch` (one of
`OSArchRequirementValues`), `Type` (`docker`, `openstack`, `vsphere`),
and `Spec` parsed per type into `V2WorkerModelDockerSpec`,
`V2WorkerModelOpenstackSpec`, or `V2WorkerModelVSphereSpec`.

Examples:

```yaml
# Docker
name: my-go
osarch: linux/amd64
type: docker
spec:
  image: golang:1.21
  username: dockerhub-user
  password: ${{ secrets.DOCKER_PASSWORD }}
  envs:
    GOPROXY: https://proxy.golang.org
    CGO_ENABLED: "0"

# OpenStack
name: ubuntu-vm
osarch: linux/amd64
type: openstack
spec:
  image: Ubuntu 22.04
  flavor: b3-30

# vSphere
name: windows-builder
osarch: windows/amd64
type: vsphere
spec:
  image: win2022-template
  flavor: large
  username: ${{ secrets.VSPHERE_USER }}
  password: ${{ secrets.VSPHERE_PASSWORD }}
```

`OSArchRequirementValues` (`sdk/requirement.go`) mirrors the v1
catalogue and covers the usual platforms: `linux/amd64`,
`linux/arm64`, `linux/386`, `darwin/amd64`, the BSD variants for
`386` and `amd64`, and `windows/amd64`. The lint helper is
`V2WorkerModel.Lint` (`sdk/v2_worker_model.go`).

## 7. Stages and the `needs` DAG

`WorkflowStage` (`sdk/v2_workflow.go`) is an optional grouping over
jobs that carries its own `Needs []string`. Two modes coexist:

- **Without stages** — Every job declares its dependencies through `needs:`. The engine derives the DAG directly.
- **With stages** — Each job carries `stage:`. The stage may declare its own `needs:` to depend on whole stages. A job inside stage A can `needs:` other jobs inside stage A or implicitly inherit the dependency on stages listed in the stage's `needs:` field.

```yaml
stages:
  build:
    needs: []
  test:
    needs: [build]
  deploy:
    needs: [test]
jobs:
  compile:
    stage: build
    runs-on: library/docker-ubuntu
    steps: [...]
  unit-test:
    stage: test
    needs: [compile]
    runs-on: library/docker-ubuntu
    steps: [...]
  integration-test:
    stage: test
    needs: [compile]
    runs-on: library/docker-ubuntu
    steps: [...]
  deploy-prod:
    stage: deploy
    needs: [unit-test, integration-test]
    runs-on: library/docker-ubuntu
    steps: [...]
```

The DAG is validated at parse time: no self-loops, no references to unknown stages or jobs, and dependencies always stay within the same stage or its declared parent stages.

## 8. Triggers (`on:` block)

The `on:` block deserialises into `WorkflowOn` (`sdk/v2_workflow.go`).
Nine `WorkflowHookEventName*` constants
(`sdk/hooks_repository_event.go`) are recognised:

| Event name | Trigger source |
| --- | --- |
| `push` | A push event reaches the hooks service |
| `pull-request` | Pull request opened, reopened, closed, edited |
| `pull-request-comment` | Pull-request comment created, deleted, edited |
| `manual` | UI / API manual run (implicit; no `on:` entry required) |
| `webhook` | Generic incoming webhook |
| `workflow-run` | Another workflow finished |
| `schedule` | Cron tick |
| `workflow-update` | The workflow file itself changed |
| `model-update` | A referenced worker model changed |

The per-event configuration uses filters:

| Trigger | Configuration |
| --- | --- |
| `push` | Branch glob list, tag glob list, path regex list, optional commit-message regex |
| `pull-request` | Branch glob list, optional comment filter, path regex list, types filter (`opened`, `reopened`, `closed`, `edited`) |
| `pull-request-comment` | Same as pull-request plus a types filter (`created`, `deleted`, `edited`) |
| `schedule` | One or more entries, each with a cron expression and a timezone |
| `workflow-run` | Upstream workflow name, allowed upstream statuses, branch / tag filters |
| `model-update` | List of worker model names to watch, plus a target branch |
| `workflow-update` | Target branch |

### 8.1 Shorthand list form

The `on:` block can also be a string array — each entry expands to its
default config. The detector for this case is `IsDefaultHooks`
(`sdk/v2_workflow.go`):

```yaml
on: [push, pull-request, workflow-update]
```

This is equivalent to declaring each event with its zero value.

### 8.2 Pull-request types

`WorkflowHookEventType*` constants (`sdk/hooks_repository_event.go`):

| Constant | Value | Applies to |
| --- | --- | --- |
| `WorkflowHookEventTypePullRequestOpened` | `opened` | pull-request |
| `WorkflowHookEventTypePullRequestReopened` | `reopened` | pull-request |
| `WorkflowHookEventTypePullRequestClosed` | `closed` | pull-request |
| `WorkflowHookEventTypePullRequestEdited` | `edited` | pull-request |
| `WorkflowHookEventTypePullRequestCommentCreated` | `created` | pull-request-comment |
| `WorkflowHookEventTypePullRequestCommentDeleted` | `deleted` | pull-request-comment |
| `WorkflowHookEventTypePullRequestCommentEdited` | `edited` | pull-request-comment |

## 9. `V2WorkflowHook` (indexed hooks)

When the API parses a workflow and extracts its `on:` block, it
materialises one `V2WorkflowHook` row (`sdk/v2_workflow.go`) per
trigger so that incoming events can be matched in O(1) at the DB
level. The row carries `ProjectKey`, `VCSName` and `RepositoryName`,
`EntityID` and `WorkflowName`, `Ref` and `Commit`, `Type`, `Data`
(`V2WorkflowHookData`), and a `Head` flag.

The seven `WorkflowHookType*` constants (`sdk/v2_workflow.go`):

| Constant | Value | Triggered by |
| --- | --- | --- |
| `WorkflowHookTypeRepository` | `RepositoryWebHook` | Push, PR, PR comment from a VCS webhook |
| `WorkflowHookTypeWorkerModel` | `WorkerModelUpdate` | A referenced worker model was updated |
| `WorkflowHookTypeWorkflow` | `WorkflowUpdate` | The workflow file itself was updated |
| `WorkflowHookTypeManual` | `Manual` | UI / REST manual run |
| `WorkflowHookTypeWebhook` | `Webhook` | Generic webhook payload |
| `WorkflowHookTypeScheduler` | `Scheduler` | Cron tick |
| `WorkflowHookTypeWorkflowRun` | `WorkflowRun` | Upstream workflow finished |

`V2WorkflowHookData` carries trigger-specific configuration
(`BranchFilter`, `TagFilter`, `PathFilter`, `TypesFilter`, `Cron`,
`CronTimeZone`, `TargetBranch`, `WorkflowRunName`,
`WorkflowRunStatus`, etc.). Matching logic for incoming events is in
[`06b-hooks-v2.md`](./06b-hooks-v2.md).

## 10. Expression language `${{ }}`

Expressions are interpolated by the parser in `sdk/action_parser.go`.
The entry symbol is `${{ … }}`. The public surface offers four entry
points: `Validate`, `Interpolate`, `InterpolateToString`,
`InterpolateToBool`.

### 10.1 Operators

| Operator | Purpose | Implementation |
| --- | --- | --- |
| `==`, `!=` | Equality and inequality | `equal()` |
| `<`, `<=`, `>`, `>=` | Numeric and lexical comparison | `compare()` |
| `&&`, `\|\|` | Boolean and / or | `and()`, `or()` |
| `!` | Negation | `parseNotExpression` |
| `? :` | Ternary | `TermExpressionContext` |
| `[]` | Array or map indexing | `getArrayItemValueFromContext` |
| `.` | Property access | `getItemValueFromContext` |

### 10.2 Builtin functions

The registry is in `sdk/action_parser_funcs.go`. About thirty builtin
functions, grouped by purpose:

| Function family | Examples |
| --- | --- |
| String testing | `contains`, `startsWith`, `endsWith`, `match` (glob) |
| String transformation | `format`, `join`, `toLower`, `toUpper`, `toTitle`, `replace`, `trimAll`, `trimPrefix`, `trimSuffix` |
| Encoding | `b64enc`, `b64dec`, `b32enc`, `b32dec` |
| Serialisation | `toJSON`, `fromJSON`, `toArray` |
| Hashing | `hashFiles` (SHA-256 of file contents matching a glob) |
| Flow-control predicates | `success`, `failure`, `always`, `cancelled`, `stopped` |
| Result querying | `result(type, name)` — filter job results |
| Context utilities | `contextValue` (dynamic lookup), `default`, `coalesce` |

## 11. Context reference

Eleven contexts are exposed to expressions. The shapes live in
`sdk/contexts.go` and `sdk/v2_workflow_run.go`.

### 11.1 `cds` — `CDSContext`

`CDSContext` (`sdk/contexts.go`) carries the platform identity of the
current run:

| Field | Purpose |
| --- | --- |
| `event_name`, `event` | The event name and the full payload that triggered the run |
| `project_key` | Owning project |
| `run_id` | Run UUID |
| `run_number`, `run_attempt` | Monotonic and retry counters |
| `run_url` | UI deep-link |
| `workflow` | Workflow name |
| `workflow_ref`, `workflow_sha`, `workflow_vcs_server`, `workflow_repository` | Source coordinates |
| `triggering_actor` | User or service that triggered the run |
| `version`, `version_next` | Computed semver (see [section 15](#15-semver)) |
| `workflow_template`, `workflow_template_*` | Template provenance when the workflow is templated |
| `job`, `stage` | Current job / stage |
| `workspace` | Worker workspace path |

### 11.2 `git` — `GitContext`

`GitContext` (`sdk/contexts.go`) is filled at crafting time from the
VCS event:

| Field | Purpose |
| --- | --- |
| `server`, `repository`, `repository_origin`, `repositoryUrl`, `repository_web_url`, `ref_web_url`, `commit_web_url` | URL identifiers |
| `author`, `author_email`, `commit_message` | Commit metadata |
| `ref`, `ref_name`, `ref_type`, `sha`, `sha_short` | Git ref |
| `connection`, `username`, `token`, `ssh_key`, `gpg_key`, `email` | Auth credentials (secret values) |
| `semver_current`, `semver_next` | Semver shortcuts |
| `changesets` | List of changed files |
| `pullrequest_id`, `pullrequest_to_ref`, `pullrequest_to_ref_name`, `pullrequest_web_url` | PR metadata |

### 11.3 Other contexts

| Context | Scope availability | Source |
| --- | --- | --- |
| `env` | Workflow + job + step | Merged from workflow `env` and job `env` |
| `inputs` | Job (manual trigger only) | Manual trigger payload |
| `jobs` | Workflow expressions, job, step | Results of finished jobs |
| `needs` | Job, step | Subset of `jobs` matching `needs:` |
| `steps` | Step only | Previous steps in the same job |
| `matrix` | Job, step (when matrix is expanded) | One permutation of the matrix |
| `integrations` | Job, step | Resolved project integrations declared on the job |
| `gate` | Job, step (when gate is approved) | Values entered by the reviewer |
| `vars` | Workflow + job + step | Items of the declared variable sets |

Availability matrix:

| Context | Workflow expression | Job expression | Step expression |
| --- | --- | --- | --- |
| `cds`, `git`, `env`, `vars` | yes | yes | yes |
| `inputs` | no | yes (manual) | no |
| `jobs` | no | yes (after deps) | yes |
| `needs` | no | yes (when defined) | yes |
| `steps` | no | no | yes (previous steps) |
| `matrix` | no | yes (when strategy) | yes |
| `integrations` | no | yes | yes |
| `gate` | no | yes (after approval) | yes |

## 12. Gates

`V2JobGate` (`sdk/v2_workflow.go`) is a manual approval that holds a
job out of the scheduling set until a human (and / or a condition)
lets it through. A gate carries:

- `If` — an expression that must evaluate to true before the gate
  becomes eligible for review.
- `Inputs map[string]V2JobGateInput` — declares the typed values the
  reviewer must supply (each `V2JobGateInput` has a `Type` — `string`,
  `boolean`, `number` — an optional `Default`, optional `Options`
  (`V2JobGateOptions` with `Multiple` and `Values`), and a
  `Description`).
- `Reviewers` (`V2JobGateReviewers`) — `Groups []string` and
  `Users []string` allowed to approve.

A job opts in by referencing the gate by name (`V2Job.Gate`). When
the run engine reaches the job, `checkCanRunJob`
(`engine/api/v2_workflow_run_engine.go:2628`) evaluates the gate: it
checks the reviewer (allowed `Users`, `Groups`, or the `IsAdminWithMFA`
shortcut), then interpolates the gate `If` against a `gate` context
populated with the input defaults (an automatic `manual` boolean is
injected, `false` by default; user-supplied values override defaults
once approval arrives). If either check fails, `checkCanRunJob`
returns `canRun=false` and the job is filtered out of the scheduling
set — **the job is not inserted in `Blocked`**; it simply does not
appear in the queue until a reviewer pushes an approval (see flow
below). The `Blocked` status is reserved for the concurrency engine
(cf. [`07b-run-engine-v2.md` §10](./07b-run-engine-v2.md#10-concurrency-engine)).

### 12.1 Reviewer flow

```mermaid
sequenceDiagram
  participant Reviewer
  participant API
  participant Engine
  participant DB
  Reviewer->>API: postStartJobWorkflowRunHandler (gate inputs)
  API->>API: validate inputs against V2JobGate.Inputs; MergeGateDefaultInputs
  API->>API: check reviewer (Users / Groups / IsAdminWithMFA)
  API->>DB: store GateInputs + V2WorkflowRunJobEvent on the run
  API->>Engine: push V2WorkflowRunEnqueueGate (RunID, JobID, Inputs)
  Engine->>Engine: checkCanRunJob now returns canRun=true
  Engine->>DB: insert run job with status=Waiting
```

The handler `postStartJobWorkflowRunHandler` validates inputs against
`V2JobGate.Inputs`, applies defaults via `MergeGateDefaultInputs`, and
verifies the reviewer is allowed (by group, by user, or via the
admin-with-MFA shortcut). The job that is eventually scheduled enters
`Waiting`, like any other job — gates never produce `Blocked`.

```yaml
gates:
  approve-deploy:
    if: ${{ success() }}
    inputs:
      target_env:
        type: string
        description: Which environment?
        options:
          values: [staging, production]
      tag:
        type: string
        default: latest
    reviewers:
      groups: [ops-team]
      users: [alice]
jobs:
  deploy:
    gate: approve-deploy
    runs-on: library/docker-ubuntu
    steps:
      - run: deploy.sh ${{ gate.target_env }} ${{ gate.tag }}
```

Validation: `CheckGates` (`sdk/v2_workflow.go`) ensures each job's
`V2Job.Gate` resolves to a declared gate, each gate has a non-empty
`If`, and `Default` is an array when `Options.Multiple = true`.

## 13. Matrix

`V2JobStrategy.Matrix` (`sdk/v2_workflow.go`) is a map
`string → array of values` that the crafter expands into the full
Cartesian product of permutations. Each permutation becomes a separate
`V2WorkflowRunJob` with its `Matrix map[string]string` filled in.

```yaml
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-22.04, debian-12]
        arch: [amd64, arm64]
    runs-on: library/${{ matrix.os }}-${{ matrix.arch }}
    steps:
      - run: make build ARCH=${{ matrix.arch }}
```

Four runs are produced from the example above. The expansion logic is
in `engine/api/v2_workflow_run_engine.go` (`generateMatrix` and
`createMatrixedRunJobs`). Matrix values can themselves be expressions,
which is how dynamic matrices work:

```yaml
strategy:
  matrix:
    version: ${{ fromJson(needs.detect-versions.outputs.list) }}
```

The platform does not enforce a hard cap on the number of permutations; operators tune limits at the API configuration level when needed.

## 14. Concurrency

Three scopes of concurrency are supported:

- **Workflow-level** — Declared in `V2Workflow.Concurrencies[]` and
  referenced by name from `V2Workflow.Concurrency` and / or
  `V2Job.Concurrency`.
- **Job-level** — Same pool referenced from a single job.
- **Project-level** — `ProjectConcurrency`
  (`sdk/v2_project_concurrency.go`), shared across every workflow in
  the project.

`WorkflowConcurrency` (`sdk/v2_workflow.go`):

| Field | Purpose |
| --- | --- |
| `Name` | Pool identifier |
| `Order` (`ConcurrencyOrder`) | Queue drain order (`oldest_first` default; `newest_first`) |
| `Pool int64` | Maximum parallel runs / jobs (default 1) |
| `CancelInProgress` | When true, a new arrival kills the running run instead of queueing |
| `If` | Optional condition gating the pool |

The full concurrency engine lives in
`engine/api/v2_workflow_run_engine_concurrency.go`. A job that hits a
full pool sits at `V2WorkflowRunJobStatusBlocked` until a slot is
available; the run itself is in `V2WorkflowRunStatusBlocked` when a
workflow-level rule denies the slot at craft time. These are the
**only** two places in the engine that produce the `Blocked` status —
see [`07b-run-engine-v2.md` §10](./07b-run-engine-v2.md#10-concurrency-engine).

```yaml
concurrencies:
  - name: deploy-prod
    pool: 1
    order: newest_first
    cancel-in-progress: true
    if: ${{ git.ref_name == 'main' }}
jobs:
  deploy:
    concurrency: deploy-prod
    runs-on: library/docker-ubuntu
    steps: [...]
```

## 15. Semver

`WorkflowSemver` (`sdk/v2_workflow.go`) configures automatic version
computation:

| Field | Purpose |
| --- | --- |
| `From` (`WorkflowSemverType`) | Source family: `git`, `helm`, `cargo`, `npm`, `yarn`, `file`, `poetry`, `debian` |
| `Path` | File path inside the repository (required when `From != git`) |
| `ReleaseRefs []string` | Refs (glob patterns) that trigger a release bump |
| `Schema map[string]string` | Per-ref prerelease format |

The crafter resolves the version at run start
(`engine/api/v2_workflow_run_craft.go`) and exposes:

- `${{ cds.version }}` — current version
- `${{ cds.version_next }}` — proposed next version (minor bump by
  default)

```yaml
semver:
  from: npm
  path: package.json
  release_refs: [refs/tags/v.*]
  schema:
    refs/heads/main: release
    refs/heads/develop: beta
```

Validation rules (`CheckSemver`, `sdk/v2_workflow.go`):

- `From` must be one of `AvailableSemverType` (`SemverGit`,
  `SemverHelm`, `SemverCargo`, `SemverNpm`, `SemverYarn`,
  `SemverFile`, `SemverPoetry`, `SemverDebian`).
- `From != "git"` requires `Path`.
- `From == "git"` forbids both `Path` and `ReleaseRefs`.

## 16. Retention

V2 retention is configured per-project via `ProjectRunRetention`
(`sdk/retention.go`) rather than per-workflow (legacy v1 used
`Workflow.HistoryLength` / `PurgeTags`). The retention configuration
declares:

| Field | Purpose |
| --- | --- |
| Default rule | Applies when no workflow-specific rule matches |
| Per-workflow rules | One block per workflow with an optional workflow default and per-git-ref rules |
| Last execution | Last time the rules were applied |
| Last status | Outcome of the last application (`Success` / `Fail`) |
| Last report | Summary of the last application |

Each rule is `(duration in days, count)`. A workflow-retention rule can target a specific git ref (with a glob — `refs/heads/main`, `refs/tags/v.*`).

Routes (`engine/api/v2_workflow_run_retention.go`):

| Route | Method | Purpose |
| --- | --- | --- |
| `/projects/{projectKey}/runs-retention` | PUT | Update the retention rules |
| `/projects/{projectKey}/runs-retention/start` | POST | Apply the rules now |
| `/projects/{projectKey}/runs-retention/dry-run` | POST | Preview the deletions |

The actual purge is performed by the `Purge-Runs-V2` goroutine (see
[`01-architecture.md`](./01-architecture.md)).

The `Retention` field on `V2Workflow` is **deprecated** and kept only
for backwards compatibility with older imports.

## 17. JSON Schema export

The API exposes a JSON-schema endpoint that powers IDE autocompletion
and `cdsctl` validation. The generator (`sdk/jsonschema.go`):
`GetWorkflowJsonSchema`, `GetJobJsonSchema`, `GetActionJsonSchema`,
`GetWorkerModelJsonSchema`. The route handler is
`getJsonSchemaHandler` (`engine/api/v2_jsonschema.go`).

```
GET /v2/projects/{projectKey}/jsonschema/{type}
```

Supported types: `workflow`, `job`, `action`, `worker-model`,
`workflow-template`.

The schema is generated by reflection over the Go structs and enriched
at runtime with the project's actions, regions, and worker models —
so enums in the schema map exactly to what the user has access to.

## 18. Validation

`V2Workflow.Lint` (`sdk/v2_workflow.go`) is the entry point for
validation. It runs:

1. Template guard — if `From` is set, skip jobs validation (template
   instantiation happens later).
2. `CheckStageAndJobNeeds` — DAG topology.
3. Per-job `Retry` in `{0, 1, 2}`.
4. `CheckSemver` — semver shape.
5. `CheckGates` — gate references and reviewer config.
6. JSON Schema validation via `gojsonschema`.
7. Cron expression parsing for every `schedule:` trigger.

Two helper methods normalise inputs before persistence:

- `Clean` — strips whitespace in `run:` scripts.
- `MarshalJSON` / `UnmarshalJSON` — accept both the structured form
  and the shorthand list form for `on:` and `runs-on:`.

`WorkflowJobParents(w, jobID)` returns the transitive `needs:`
ancestors of a job and is the helper used by both the engine and the
UI when rendering the DAG.

## 19. Cross-spec pointers

- Microservices and request lifecycle → [`01-architecture.md`](./01-architecture.md)
- Project tenancy, variable sets, integrations → [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)
- Workflow v1 model (legacy) → [`03-workflow-v1.md`](./03-workflow-v1.md)
- `.cds/` folder layout, repository analysis, signatures, templates → [`05-ascode-entities.md`](./05-ascode-entities.md)
- Hooks service, per-provider webhook parsing, `HookRepositoryEvent` lifecycle → [`06b-hooks-v2.md`](./06b-hooks-v2.md)
- V2 run engine (state machine, queue, retry, concurrency engine) → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- RBAC v2 → [`09-rbac.md`](./09-rbac.md)
- Authentication (sessions, scopes, link) → [`08-auth.md`](./08-auth.md)
- Hatcheries → [`10-hatcheries.md`](./10-hatcheries.md)
- Workers → [`11-workers.md`](./11-workers.md)
- CDN, run results → [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md)
- VCS providers → [`13-vcs.md`](./13-vcs.md)
- Integrations → [`14-integrations.md`](./14-integrations.md)
- cdsctl → [`15-cli.md`](./15-cli.md)
- Go SDK → [`16-sdk.md`](./16-sdk.md)
- gRPC plugins → [`17-plugins.md`](./17-plugins.md)
- UI → [`18-ui.md`](./18-ui.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
