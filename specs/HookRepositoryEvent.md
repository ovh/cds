# SPEC — Hook Repository Event (Hooks Service)

## 1. Role of Hook Repository Events

A **HookRepositoryEvent** is the central object of the hooks service. It represents an event related to a code repository (push, pull-request, manual trigger, scheduler, dedicated webhook, or outgoing workflow-run) and orchestrates the entire processing chain from signal reception to the start of one or more workflow runs on the CDS API.

Each event traverses a state machine whose main steps are: reception, repository analysis, workflow hooks resolution, Git information retrieval, and then workflow runs triggering.

---

## 2. Entry Points (Event Creation)

Events are created by five HTTP handlers exposed by the hooks service:

### 2.1. Internal VCS Webhook

The CDS VCS service relays events received from providers (GitHub, Bitbucket, GitLab, Gitea, Forgejo) to the hooks service. The request is signed with an internal CDS public key. The raw payload is extracted and normalized according to the VCS provider type: each provider has a dedicated extractor that translates the native event into CDS data (ref, commit, modified files, normalized CDS event name and type, pull-request identifier, etc.).

### 2.2. Direct Webhook from a VCS Provider

A VCS provider can also directly call the hooks service via a dedicated URL containing the project, the VCS server type and name, and a unique identifier. Authentication is done via HMAC-256 signature with a shared secret. The created event is restricted to the project designated in the URL.

### 2.3. Dedicated Workflow Webhook

Similar to the direct webhook, but the URL additionally contains the repository and the workflow name. The created event is of type `webhook` and targets a specific workflow. The raw request body is preserved as a payload accessible in the workflow run context.

### 2.4. Manual Trigger

The UI or CLI sends a request containing the project, workflow, target ref/commit, and optionally job inputs. The created event is of type `manual`.

### 2.5. Outgoing Workflow-Run Event

When a workflow run emits an outgoing event (cross-workflow trigger), an intermediate `HookWorkflowRunOutgoingEvent` object is first created. Its processing resolves the `workflow-run` type workflow hooks to trigger, then creates a `HookRepositoryEvent` pre-populated with the already identified hooks.

### Common Creation Flow

Regardless of the entry point, creation follows the same flow:

1. Construction of the `HookRepositoryEvent` object with the **Scheduled** status
2. Extraction and normalization of VCS-specific data (for webhooks)
3. Saving the event in the Redis cache
4. Queuing for processing

---

## 3. Storage

Storage relies entirely on **Redis**:

| Data | Mechanism | Description |
|------|-----------|-------------|
| Events per repository | Redis Set | Each event is stored in a set identified by the VCS and the repository |
| Processing queue | Redis FIFO Queue | Queue of events to be consumed by the main processing goroutine |
| Maintenance queue | Redis FIFO Queue | Queue dedicated to manual events triggered during maintenance |
| Callback queue | Redis FIFO Queue | Queue of callbacks (completed analyses, completed Git operations) |
| Events in progress | Redis Set | Tracking of events being processed, to allow retry in case of failure |
| Distributed lock | Redis Key with TTL | Per-event lock (30 seconds) preventing concurrent processing by multiple instances |

No event data is persisted in the SQL database on the hooks side. Related entities (workflow hooks, analyses) are in the database on the API side.

---

## 4. Processing Goroutines

Four background goroutines consume events:

### 4.1. Main Processing

Infinite loop that dequeues events from the processing queue. For each event:

1. Acquisition of the distributed lock on the event
2. Execution of the state machine (see §5)
3. Saving the updated state
4. Releasing the lock
5. If the event is not finished, re-enqueue for the next step

This goroutine is paused during maintenance.

### 4.2. Callback Processing

Loop that dequeues callbacks from the API (completed analysis or completed Git operation). Each callback updates the event with the operation results (discovered workflows, models, signing key, semver, changesets) then re-enqueues it to continue processing.

### 4.3. Retry of Stuck Events

Periodic ticker that scans events marked as being processed. If an event has not been updated within a configurable delay, it is considered stuck and re-enqueued. During maintenance, only manual events marked `IsInMaintenance` are retried. This allows recovering events lost after a service restart.

### 4.4. Maintenance Queue Processing

Infinite loop identical to the main goroutine (§4.1) but consuming a dedicated Redis queue (`hooks:queue:repository:event:maintenance`). This goroutine is **never** paused by maintenance, allowing manual triggers performed by maintainers during a maintenance period to be processed. The same processing (state machine) is applied to events from this queue.

### 4.5. Outgoing Event Processing

A dedicated goroutine (`dequeueWorkflowRunOutgoingEvent`) dequeues outgoing workflow-run events from a Redis queue. It resolves which workflow hooks of type `workflow-run` should be triggered and creates the corresponding `HookRepositoryEvent` pre-populated with the resolved hooks.

### 4.6. Outgoing Event Retry

A periodic goroutine (`manageOldWorkflowRunOutgoingEvent`) scans outgoing events that are still in progress. If an outgoing event has not been updated within a configurable delay, it is re-enqueued, similar to §4.3 but for outgoing events.

---

## 5. State Machine

Event processing is driven by its status. On each pass through the processing loop, the step corresponding to the current status is executed. If the step completes successfully, the status is advanced to the next step and the event is re-enqueued.

### Statuses and Transitions

```
Scheduled
  ├─ push / manual ──────────────────→ Analyzing
  ├─ pull-request ───────────────────→ CheckAnalyzing
  └─ scheduler / webhook / workflow-run / pull-request-comment / model-update / workflow-update
                                     → WorkflowHooks

Analyzing ─────────────────────────────→ WorkflowHooks
CheckAnalyzing ────────────────────────→ WorkflowHooks

WorkflowHooks
  ├─ hooks found ────────────────────→ GitInfo
  └─ no hooks ───────────────────────→ Done

GitInfo ───────────────────────────────→ Workflow

Workflow
  ├─ all runs created ───────────────→ Done
  ├─ all hooks in error ─────────────→ Error
  └─ all hooks filtered ─────────────→ Done
```

### 5.1. Scheduled

Entry point of the state machine. Depending on the event type (`EventName`), the event is routed to the appropriate step:

- **push** and **manual**: require a repository analysis → transition to **Analyzing**
- **pull-request**: analysis was already done during the push, need to verify it's complete → transition to **CheckAnalyzing**
- **scheduler**, **webhook**, **workflow-run**, **pull-request-comment**, **model-update**, **workflow-update**: no analysis needed → direct transition to **WorkflowHooks**

### 5.2. Analyzing (Repository Analysis)

This step triggers the analysis of the repository content at the event's commit. It applies to push and manual event types.

1. The hooks service lists all CDS projects associated with the repository via the API
2. For each project, an analysis request is sent to the CDS API
3. The API clones the repository at the given commit, verifies the commit signature, scans files in the `.cds/` directory, and discovers entities (workflows, worker models, actions)
4. Once the analysis is complete on the API side, a callback is sent to the hooks service containing the results: discovered workflows, discovered models, skipped workflows, skipped hooks, analysis status, initiator identity, and signing key
5. The callback is enqueued in the callback queue, consumed by the dedicated goroutine, which updates the event with the results
6. On the next pass, if all analyses are complete → transition to **WorkflowHooks**

If an analysis fails, the error is stored but processing continues for the other projects.

### 5.3. CheckAnalyzing (Existing Analysis Verification)

This step applies to pull-request event types. Since the corresponding push already triggered an analysis, this step verifies that the analysis for the commit exists and is complete.

1. For each project associated with the repository, the analysis corresponding to the commit is looked up
2. If the analysis is not yet found, a retry counter is incremented. If the max counter is reached, the analysis is considered in error
3. If all analyses are complete → transition to **WorkflowHooks**

### 5.4. WorkflowHooks (Hooks Resolution)

This step determines which workflows should be triggered by the event. Behavior depends on the event type:

**For VCS events (push, pull-request, pull-request-comment)**: the hooks service calls the API to get the list of matching workflow hooks. The API examines all workflows of the concerned projects and returns those whose `on:` configuration matches the event (event name, type, ref, branch/tag filters).

**For manual events**: the hooks service loads the target workflow entity from the API, verifies it exists, and builds a single manual-type hook.

**For webhook events**: similar to manual, a single hook is built pointing to the workflow targeted in the URL.

**For scheduler events**: a single hook is built with the cron metadata (target VCS, target repository, target workflow, cron expression, timezone).

**For workflow-run events**: hooks were pre-populated during event creation by the outgoing event processing. The workflow entity is loaded to obtain the initiator's identity.

If no hooks are found → transition to **Done**. Otherwise → transition to **GitInfo**.

### 5.5. GitInfo (Git Information Retrieval)

For each workflow hook identified in the previous step, additional information is retrieved via the API:

1. A request is sent to the API requesting semantic version resolution (current and next semver), changesets (files modified between source and target commit), and the commit message
2. The API creates a Git operation (repository clone, semver computation from tags, changeset extraction)
3. When the operation completes, the result comes back via a callback or by directly querying the operation
4. Each workflow hook is enriched with: current/next semver, list of modified files, commit message, commit author and email, target commit

If all hooks have received their information → transition to **Workflow**. If an operation fails, it is retried a limited number of times before being marked as error.

### 5.6. Workflow (Workflow Runs Triggering)

Last active step that creates workflow runs on the CDS API.

1. **Initiator resolution**: if the initiator is not yet known, the hooks service calls the API to determine the user who performed the commit, based on the GPG signing key. If the user cannot be identified and the workflow does not disable signature verification → the hook is marked as skipped

2. **For each workflow hook**:
   - **Path filtering**: if the hook defines path filters, they are applied to the list of modified files (changesets). If no file matches → the hook is skipped with the reason "no file matches path filters"
   - **Commit message filtering**: if the hook defines a commit filter, the message is verified. CI skip keywords (`[skip ci]`, `[ci skip]`, etc.) are also checked. If the message does not match → the hook is skipped with the reason "commit message does not match commit filter or contains a skip CI directive"
   - If filters pass: a workflow run creation request is sent to the API with all metadata (ref, sha, semver, changesets, event payload, hook type, initiator identity)
   - The API creates the `V2WorkflowRun` in the database, which then enters the crafting phase (see dedicated spec)
   - In return, the hooks service receives the identifier and number of the created run

3. **Final status**:
   - If at least one run was created → **Done**
   - If all hooks are in error → **Error**
   - If all hooks were filtered → **Done**

---

## 6. Supported Event Types

| Event Type | Source | Analysis | Hooks Resolution | Specifics |
|------------------|--------|---------|------------------|----------------|
| `push` | VCS Webhook | Yes (triggers analysis) | Via API (matching `on.push` hooks) | Most common type |
| `pull-request` | VCS Webhook | Verification (waits for push analysis) | Via API (matching `on.pull-request` hooks) | Supports subtypes: opened, reopened, closed, edited |
| `pull-request-comment` | VCS Webhook | No | Via API (matching `on.pull-request-comment` hooks) | Supports subtypes: created, deleted, edited |
| `manual` | UI / CLI | Yes (triggers analysis) | Direct loading of the workflow entity | Targets a single workflow. Supports job inputs |
| `webhook` | Direct HTTP Webhook | No | Direct loading of the workflow entity | Targets a single workflow. The HTTP body is preserved as payload |
| `scheduler` | Internal cron system | No | Direct construction from scheduler metadata | Targets a single workflow |
| `workflow-run` | Outgoing run event | No | Pre-populated from the outgoing event | Pre-resolved hooks. Enables workflow chaining |
| `model-update` | Worker model update | No | Via API | Triggers workflows using the updated model |
| `workflow-update` | Workflow update | No | Via API | Triggers scheduler resynchronization |

---

## 7. VCS Insight Report

At the end of processing each event (except `workflow-run` type), an Insight report is sent to the VCS provider via the API. This report is displayed on the commit or pull-request page on the VCS side and contains:

- The overall event status
- Details of each triggered analysis (with link to the repository settings in the CDS UI)
- Details of each triggered workflow run (with link to the run in the CDS UI)

If the number of analyses or workflows is too large, an aggregated summary is displayed with a link to the filtered runs view.

---

## 8. Fault Tolerance

- **Distributed lock**: each event is protected by a Redis lock (TTL 30 seconds) preventing concurrent processing by multiple hooks service instances
- **Automatic retry**: stuck events (no update within a configurable delay) are automatically re-enqueued by the retry goroutine. During maintenance, only manual events marked `IsInMaintenance` are retried; others are ignored until maintenance ends
- **Error counter**: each event maintains an error counter and a last error message. Errors are logged but processing can resume
- **Git operation verification**: if a callback is never received, the next pass through the GitInfo step directly queries the API to check the operation status
- **Maintenance mode**: in hooks service maintenance mode, the main processing goroutines are paused. However, manual triggers performed by maintainers remain possible thanks to a dedicated maintenance queue (see §8.1)

### 8.1. Maintenance Queue for Manual Triggers

When a maintainer manually triggers a workflow while maintenance is active, the event is handled specially:

1. The manual trigger handler detects that maintenance is active and sets the `IsInMaintenance` flag on the event (in `ExtractData.Manual`)
2. The enqueue DAO (`EnqueueRepositoryEvent`) routes the event to a **dedicated Redis queue** (`hooks:queue:repository:event:maintenance`) instead of the standard queue
3. A **dedicated goroutine** (`dequeueMaintenanceRepositoryEvent`) continuously consumes this queue, never pausing for maintenance
4. The event then follows the normal state machine (analysis, hooks resolution, GitInfo, workflow run triggering)

This mechanism allows unblocking an urgent deployment without disabling global maintenance.

The maintenance queue size is exposed as a monitoring metric (`MaintenanceQueue`).

---

## 9. Cleanup

A periodic goroutine deletes old events from the Redis cache when the number of events for a given repository exceeds the configured `RepositoryEventRetention` threshold. The oldest events are removed first, preventing indefinite storage growth.

---

## 10. HTTP Routes

### Event Reception

| Method | Route | Description |
|---------|-------|-------------|
| POST | `/v2/webhook/repository` | Reception of an internal webhook from the VCS service |
| POST | `/v2/webhook/repository/{projectKey}/{vcsServerType}/{vcsServer}/{uuid}` | Reception of a direct webhook from a VCS provider |
| POST | `/v2/webhook/workflow/{projectKey}/{vcsServer}/{repoName}/{workflowName}/{uuid}` | Reception of a dedicated workflow webhook |
| POST | `/v2/workflow/manual` | Manual workflow run trigger |
| POST | `/v2/workflow/outgoing` | Reception of an outgoing workflow-run event |

### Callbacks

| Method | Route | Description |
|---------|-------|-------------|
| POST | `/v2/repository/event/callback` | Analysis or Git operation callback from the API |

### Consultation and Administration

| Method | Route | Description |
|---------|-------|-------------|
| GET | `/v2/repository` | List repositories |
| GET | `/v2/repository/event/{vcsServer}/{repoName}` | List events for a repository |
| GET | `/v2/repository/event/{vcsServer}/{repoName}/{uuid}` | Event details |
| DELETE | `/v2/repository/event/{vcsServer}/{repoName}` | Delete events for a repository |
| DELETE | `/v2/workflow/outgoing/{projectKey}` | Delete outgoing events for a project |
| POST | `/admin/repository/event/{vcsServer}/{repoName}/{uuid}/stop` | Force stop an event (administration) |
| POST | `/admin/repository/event/{vcsServer}/{repoName}/{uuid}/restart` | Restart an event (administration) |
| DELETE | `/admin/repository/{vcsServer}/{repoName}` | Delete a repository (administration) |
| POST | `/admin/maintenance` | Toggle maintenance mode |

### Webhook Secret Management

| Method | Route | Description |
|---------|-------|-------------|
| POST | `/v2/repository/key/{projectKey}/{vcsServer}/{repoName}` | Generate secret for a repository webhook |
| POST | `/v2/workflow/key/{projectKey}/{vcsServer}/{repoName}/{workflowName}` | Generate secret for a workflow webhook |

---

## 11. Processing Flow Summary

```
Event reception (webhook, manual, scheduler, workflow-run)
  → Data extraction and normalization
  → HookRepositoryEvent creation (Scheduled status)
  → Save to Redis and enqueue

Processing goroutine
  → Dequeue event
  → Acquire distributed lock
  → Execute step corresponding to the current status

  Scheduled
    → Route based on event type

  Analyzing (push, manual)
    → Send analysis requests to the API for each project linked to the repository
    → Wait for analysis callbacks

  CheckAnalyzing (pull-request)
    → Verify that push analyses exist and are complete

  WorkflowHooks
    → Resolve workflow hooks to trigger
    → If no hooks → Done

  GitInfo
    → Retrieve semver, changesets, and commit message
    → Wait for Git operation callbacks

  Workflow
    → Resolve initiator (user identity)
    → Apply filters (paths, commit message)
    → Create workflow runs on the CDS API
    → Done / Error

  → Send Insight report to VCS provider
  → Deferred event cleanup
```
