# SPEC — Workflow Run v2 Execution (Engine)

## 1. Role of the Engine

The **engine** is the orchestration phase of a workflow run after crafting. It handles progressive job scheduling, dependency management, concurrency resolution, and transition to terminal states.

Its main responsibilities are:

- Determine which jobs are ready to be launched at each execution cycle
- Create `V2WorkflowRunJob` entities in the database, with their resolved contexts and configurations
- Handle matrix strategies (permutation generation)
- Resolve template-based jobs (expansion at execution time)
- Apply job-level and workflow-level concurrency rules
- Update the run's overall status at each cycle
- Trigger end notifications and outgoing workflows via the hooks service

---

## 2. Entry Points

The engine has two entry points feeding the same orchestration function:

**Immediate triggering via channel**: when a run completes crafting or a job completes, a processing request is pushed into an internal channel (`workflowRunTriggerChan`). A background worker consumes it and immediately starts orchestration.

**Triggering via Redis queue dequeuing**: a dequeue loop continuously polls a Redis queue (`WorkflowEngineKey`) with a 250ms delay between each attempt. Excess requests (full channel) are redirected to this queue. Each dequeued request is processed in a dedicated goroutine.

The `EnqueueWorkflowRun` function handles routing: it first attempts the channel, then falls back to the Redis queue if the channel is saturated.

---

## 3. Distributed Lock

Before any processing, a Redis lock is taken on the key `api:workflow:engine:{runID}` with a 5-minute time-to-live.

- If the lock is not acquired (another node is already processing this run): the request is re-enqueued and processing stops.
- The lock is released at the end of processing, whether successful or not.

This mechanism ensures that a run is never processed in parallel by multiple API nodes.

---

## 4. Initial Loading

Once the lock is acquired, the data required for processing is loaded:

- The run from the database
- The project with its integrations in clear text
- The VCS server associated with the run
- The repository associated with the run
- All run jobs for the current attempt
- All run results for the current attempt

An `allrunJobsMap` is then built: for each `jobID`, the last non-terminated run job is kept. It serves as a reference for duplicate and matrix checks.

---

## 5. Preliminary Guards

Several conditions cause the engine to no-op immediately:

- The run is in a terminal state (Fail, Success, Stopped, Cancelled, Skipped): no further processing is performed.
- The run is in `Blocked` status and has a workflow concurrency rule: the `workflowRunV2TriggerUnlocking` function is called to attempt to unblock the run (see section 9).

---

## 6. Handling Retrying Jobs

Before normal scheduling, each job with `Retrying` status is processed:

1. The old run job is marked `Fail` in the database.
2. A new run job is created with an incremented retry number and `Waiting` status.
3. The new job is persisted in a transaction.
4. An `EventRunJobEnqueued` event is published for the new job.
5. The run is re-enqueued and the engine execution stops there for this cycle.

This short-circuit mechanism ensures that only a retry is handled in a single engine cycle, without additional scheduling computation.

---

## 7. Forced Interruption (Stop / Cancel)

If the enqueue request status (`wrEnqueue.Status`) is a terminal status (typically `Stop` or `Cancel`), the `terminateWorkflowRun` function is called:

- All non-terminated jobs are marked `Stopped` or `Cancelled` according to the requested status.
- An information message is inserted on the run.
- The run is transitioned to the corresponding status.
- Execution continues to `endWorkflowV2Trigger` (section 12) without going through scheduling.

---

## 8. Building Existing Job Contexts

Before calculating which jobs to launch, the contexts of already completed jobs are reconstructed:

- `computeExistingRunJobContexts`: builds a `JobsResultContext` (map `jobID → result + outputs`) from completed run jobs, keeping only the latest attempt of each job.
- `computeGateContext`: for each job with a gate, builds the gate context (`GateInputs`) from run events.

These contexts serve as the basis for condition evaluations (`if`) and dependencies (`needs`) in the rest of the cycle.

---

## 9. Unblocking Runs Blocked by Workflow Concurrency

If the run is in `Blocked` status and has a workflow-level concurrency rule, `workflowRunV2TriggerUnlocking` is called:

1. A Redis lock on the concurrency key is acquired.
2. The `retrieveRunObjectsToUnLocked` function evaluates which runs should be unblocked or cancelled according to the concurrency rule.
3. If the current run should be cancelled: its status is set to `Cancelled`.
4. If the current run should be unblocked: its status is set to `Building`, a message is inserted, and it is re-enqueued.
5. If the run is not yet eligible for unblocking: the engine stops there.

---

## 10. Annotation Computation

Annotations declared in `workflow.annotations` are evaluated with the expression interpolator. The context includes statuses and outputs of already completed jobs.

Application rules:

- An annotation already present on the run is not recomputed.
- If interpolation fails, a warning message is inserted on the run, but processing continues.
- If the interpolated value is empty or equals `false` (case-insensitive), the annotation is not added.

---

## 11. Selecting Jobs to Schedule (`retrieveJobToQueue`)

This function determines which workflow jobs are ready to be created in the database.

### 11.1 Initial Filtering

Only jobs for which no active run job exists yet are candidates. Exception: jobs with a matrix strategy whose permutations have not all been launched yet are passed back into the candidate list.

### 11.2 Stage Handling

If the workflow defines stages, their status is computed from the results of the jobs they contain. A job belonging to a stage whose status is `CannotBeRun` is excluded from scheduling.

### 11.3 `needs` Verification

A job is a candidate only if all jobs declared in its `needs` field have a result in the existing jobs context. A job whose prerequisite is not yet completed remains waiting.

### 11.4 `if` Condition and Gate Evaluation

For each candidate job, `checkJob` is called:

- **Variable set rights**: if the initiator is not an MFA admin, their rights on the variable sets used by the job are verified.
- **Gate**: if the job has a gate, the reviewers (users or groups authorized to validate) are checked. The `gate.if` condition is then evaluated with the inputs provided by the user. If the gate condition is not satisfied, the job is queued with `Skipped` status.
- **`job.if` condition**: evaluated after the gate. The default condition (absent) is equivalent to `${{ success() }}`.

A job whose condition is not satisfied or whose rights are insufficient is added with `Skipped` status.

---

## 12. Loading Variable Sets and Variables Context

Variable sets declared at the workflow level are loaded from the database and transformed into a variables context (`vars.*`). This context is shared between all jobs to be scheduled and is enriched at each job level by variable sets specific to that job.

---

## 13. Detecting Template-Based Jobs

A `hasTemplatedJob` flag is set if at least one job to be scheduled has a non-empty `from` field. This flag triggers an additional engine cycle at the end of processing to allow template expansion.

---

## 14. Resolving Concurrency Definitions

For each job to be scheduled, the applicable concurrency definition is resolved:

1. If the job references a named concurrency rule (`job.concurrency`), its definition is looked up in the workflow's concurrency list.
2. The concurrency rule's `if` condition is interpolated in the run's context.
3. If interpolation or resolution fails, an error message is inserted on the run.

The concerned Redis concurrency keys are then locked. If a lock is unavailable, the engine waits 2 seconds then re-enqueues the run.

---

## 15. Preparing Run Jobs (`prepareRunJobs`)

For each job to be scheduled, one or more `V2WorkflowRunJob` entities are created according to the following logic.

### 15.1 Per-Job Context

A complete `WorkflowRunJobsContext` is built for each job, including:

- The run's CDS, Git, and Env contexts (the `env` context merges workflow-level environment variables with job-level ones, with the latter taking priority)
- The `jobs.*` context of completed jobs
- The `needs.*` context (subset of completed jobs listed in `needs`)
- The `vars.*` context (job variable sets)
- The `gate.*` context (gate inputs if applicable)
- The `integrations.*` context (job and workflow integrations, with job integrations taking priority). This context is built from the project integrations matching the names declared on the job then on the workflow; dynamic names (containing `${{`) are ignored at this stage and resolved during interpolation.

### 15.2 Matrix Permutation Generation

If the job defines a `matrix` strategy, all value permutations are computed. Each scalar or array value can contain `${{...}}` expressions that are interpolated in the job's context. The full set of combinations is the Cartesian product of all lists.

Only permutations for which no run job already exists are created (those already launched in a previous cycle are ignored).

### 15.3 Template-Based Jobs (`from`)

When a job has a `from` field:

1. The template is resolved (`checkJobTemplate`): search for the template entity, lint the result.
2. The current workflow is modified in place (`handleTemplatedJobInWorkflow`):
   - The template job is removed from the workflow.
   - The template's jobs, stages, gates, annotations, and concurrencies are injected into the workflow.
   - `needs` dependencies in the rest of the workflow are updated to point to the template's final jobs.
   - The template's actions and worker models are merged into `WorkflowData`.
3. A `hasToUpdateRun` flag is set so the updated run is persisted.

This mechanism is identical for a template job with matrix: each permutation generates its own subset of jobs, and duplicates are detected.

### 15.4 Run Job Field Interpolation

For non-template jobs, the following dynamic fields are interpolated in order:

| Field | Behavior on error |
|---|---|
| `job.integrations` | Job fails |
| `job.name` | Job fails |
| `job.runs_on.region` | Job fails |
| `job.runs_on.model` | Job fails |
| `job.runs_on.flavor` | Job fails |
| `job.runs_on.memory` | Job fails |
| `job.services[*].image` | Job fails |
| `job.services[*].readiness.command` | Job fails |

If an integration redirects to a specific region (via `IntegrationConfigTypeRegion`), that region is applied to the run job. After resolving a dynamic integration name, the `integrations.*` context is rebuilt and the interpolation parser is recreated so that subsequent fields (like `runs_on.model`) can reference the newly resolved integration values.

### 15.5 Region Rights Verification

For each target region of a run job:

- The project must have the `execute` role on the region.
- The initiator (user or VCS user) must also have the `execute` role on the region, unless they are an MFA admin.

If rights are insufficient, the job is set to `Skipped` with an information message.

### 15.6 Job-Level Concurrency Application

For each non-skipped run job, `manageJobConcurrency` is called:

- The status of runs blocked by the same concurrency rule is evaluated.
- Depending on the rule (`cancel-in-progress`, `queue`, `forbid-new`), concurrent runs may be cancelled, blocked, or refused.
- If the current job should be blocked, its status is `Blocked` with an explanatory message.

### 15.7 Unblocking Blocked Jobs

For each existing job in `Blocked` status with a concurrency rule, `retrieveRunObjectsToUnLocked` is called. If the current job should be unblocked, its status changes to `Waiting` and it is added to the run jobs to be persisted.

---

## 16. Run Job Persistence (Transaction)

In a single transaction:

1. Each resulting run job is inserted or updated in the database.
2. Information messages (`runJobsInfo`) associated with run jobs are inserted.
3. If the run job is in a state requiring manual interaction (untriggered gate), a gate-type message is inserted on the run.
4. General run information messages (`runInfos`) are inserted.

If the transaction fails, it is rolled back and the error is raised.

---

## 17. Computing the Run's Final Status

If `jobsToQueue` is empty (no new job to place in queue) and no skipped or failed job is awaiting processing, `computeRunStatusFromJobsStatus` is called to determine the final status:

| Condition | Resulting Status |
|---|---|
| At least one non-terminated job | `Building` |
| At least one `Stopped` job | `Stopped` |
| At least one `Fail` job (without `continueOnError`) | `Fail` |
| At least one `Cancelled` job | `Cancelled` |
| All jobs `Skipped` | `Skipped` |
| All completed successfully | `Success` |
| Number of job IDs < number of jobs defined in the workflow | `Building` |

The status is then applied to the run and updated in the database.

---

## 18. Re-enqueuing and Cleanup

After persistence:

- Run objects to be cancelled (identified by concurrency management) are updated in the database.
- The main transaction is committed.
- The `needReEnqueue` flag is true if skipped jobs, jobs without steps, or template-based jobs were processed: a new engine cycle will be triggered to continue progression.

---

## 19. Cycle Closure (`endWorkflowV2Trigger`)

This function is called at the end of each cycle, regardless of its outcome.

### 19.1 Artifact Result Synchronization

In the background (goroutine), `synchronizeRunResults` is called: for each run result linked to an Artifactory integration, CDS and Git properties are pushed to the artifact, a JWS signature is computed, and an Artifactory Build Info is created or updated.

### 19.2 Job Concurrency End

For each run job completed during this cycle that has a concurrency rule, `manageEndConcurrency` is called to release occupied slots and potentially unblock other runs or jobs.

### 19.3 Run End

If the run is in a terminal state:

- The `EventRunEnded` event is published.
- `manageEndConcurrency` is called for workflow-level concurrency.
- The hooks service is notified via `POST /v2/workflow/outgoing` with the map of job statuses, allowing potential outgoing workflows to be triggered.

### 19.4 Re-enqueuing

If `reEnqueue` is true, the run is re-enqueued via `EnqueueWorkflowRun`.

### 19.5 WebSocket Events

A WebSocket event is published for each run job whose status changed during this cycle:

| Job status | Event published |
|---|---|
| `Fail` | `EventRunJobEnded` |
| `Blocked` | `EventRunJobBlocked` |
| `Cancelled` | `EventRunJobCancelled` |
| `Skipped` | `EventRunJobSkipped` |
| `Stopped` | `EventRunJobStopped` |
| Any other (e.g., `Waiting`) | `EventRunJobEnqueued` |

---

## 20. Handling Blocked Runs (`triggerBlockedWorkflowRun`)

A periodic mechanism scans runs in `Building` status and checks if all their active jobs are in `Blocked` status. If so and no job is currently executing, the run is re-enqueued with the initiator of the last completed job, allowing the engine to re-evaluate unblocking conditions.

---

## 21. Error Handling

| Situation | Behavior |
|---|---|
| Lock not acquired | Silent re-enqueue, no-op |
| Run already terminated | No-op |
| Data loading error | Error raised, cycle ends |
| Transaction failure | Rollback, error raised |
| Failed interpolation on a job field | Job set to `Fail` with message |
| Insufficient rights on region or varset | Job set to `Skipped` with message |
| Invalid post-template workflow lint | Error message inserted on run, run set to `Fail` |
| Artifact synchronization error | Error logged, processing of other artifacts continues |
| Hooks publication error | Error logged, non-blocking for the run |
