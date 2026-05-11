# SPEC — Schedulers (Hooks Service)

## 1. Role of Schedulers

**Schedulers** allow automatically triggering workflow runs at regular intervals, based on a cron expression defined in the workflow YAML file (`on.schedule` block). They are entirely managed by the **hooks µService**, which stores definitions and next executions in Redis.

---

## 2. Lifecycle

### 2.1. Creation (Instantiation)

When a developer pushes code to the default branch of a repository, the API analyzes the repository content. If it detects `Scheduler` type hooks in the workflows, it:

1. Deletes the old schedulers for the affected workflow by calling the hooks service
2. Saves the new hooks in the database
3. Calls the hooks service to instantiate the new schedulers

The hooks service receives the list of scheduler hooks and for each workflow:
- Deletes all old definitions and planned executions
- Creates each new scheduler definition in Redis
- Calculates the next execution from the cron expression and configured timezone

### 2.2. Execution

A background routine, launched at hooks service startup, checks planned executions every 10 seconds. For each execution whose time has passed:

1. A distributed Redis lock is acquired to prevent double triggers in multi-instance environments
2. The scheduler definition existence is verified locally in Redis
3. The hook existence is verified on the API side via an HTTP call. If the hook no longer exists in the database, the definition and execution are deleted
4. A `HookRepositoryEvent` event is created with the `scheduler` type, then saved and placed in the event processing queue
5. The next execution is immediately recalculated and scheduled

The event then follows the standard hooks service event processing flow: repository analysis, callback to the API, then workflow run creation.

### 2.3. Update

An update occurs when a new push is made to the default branch and the API re-analyzes the repository. The process is identical to creation: old schedulers are deleted then recreated with the new definitions.

### 2.4. Deletion

Deletion occurs in several cases:
- Deletion of a workflow or entity: the API asks the hooks service to delete all schedulers for the workflow
- Deletion of a project: the API iterates over the project's scheduler hooks and deletes those for each affected workflow
- Individual deletion via the administration interface

Deletion removes the scheduler definition and its planned execution from Redis.

---

## 3. Storage

Storage relies entirely on Redis:

- **Definitions**: each scheduler is stored under an individual key containing the VCS, repository, workflow, and hook identifier
- **Planned executions**: a Redis set contains the next executions. Each set member represents a scheduler with its trigger time

No scheduler data is persisted in the SQL database on the hooks side. However, hooks are registered in the database on the API side.

---

## 4. Cron Expressions and Timezone

Cron expressions are parsed in extended format (5 to 7 fields), via the `cronexpr` library. An optional timezone can be associated with each scheduler. If specified, the next execution calculation is done in that timezone. If absent, the server's local time is used.

---

## 5. Fault Tolerance

- **Distributed lock**: each trigger is protected by a Redis lock with a 20-second TTL, preventing double executions in multi-instance hooks service deployments
- **API-side verification**: before each trigger, the hooks service verifies that the hook still exists in the database. This allows automatically cleaning up orphaned schedulers whose workflow or entity was deleted without the hooks service being notified
- **Maintenance mode**: when the hooks service is in maintenance mode, the trigger routine is paused

---

## 6. Resynchronization

A reconciliation mechanism ensures consistency between the Redis state and the API database (source of truth).

### 6.1. Startup Resync

At hooks service startup, a goroutine automatically runs a full resynchronization. This handles cases where schedulers were created or deleted while the hooks service was down.

### 6.2. Manual Resync

An admin route (`POST /admin/scheduler/resync`) allows triggering a manual resynchronization.

### 6.3. Reconciliation Steps

The resynchronization process executes the following steps:

1. **Load source of truth**: all scheduler hooks are loaded from the API database
2. **Load Redis state**: all current scheduler definitions and executions are loaded from Redis
3. **Repository verification**: for each repository referenced by a scheduler, verifies it exists in Redis (required for event processing to work)
4. **Add missing / update changed schedulers**: schedulers present in the API database but missing from Redis are created. Schedulers whose cron expression or timezone has changed are updated and their next execution is recalculated
5. **Remove orphaned schedulers**: schedulers present in Redis but absent from the API database are deleted (with a double-check via API to confirm)
6. **Clean orphaned executions**: planned executions whose scheduler definition no longer exists are removed
7. **Ensure pending executions**: verifies that every scheduler in the API database has a corresponding planned execution in Redis

The resync process is protected by a dedicated Redis lock with a 5-minute TTL to prevent concurrent reconciliations.

---

## 7. HTTP Routes

| Method | Route | Description |
|---------|-------|-------------|
| POST | `/v2/workflow/scheduler` | Instantiate a set of schedulers (called by the API) |
| DELETE | `/v2/workflow/scheduler/{vcsServer}/{repoName}/{workflowName}` | Delete all schedulers for a workflow |
| GET | `/admin/scheduler` | List all schedulers (administration) |
| GET | `/admin/scheduler/{vcsServer}/{repoName}/{workflowName}` | List schedulers for a workflow (administration) |
| POST | `/admin/scheduler/resync` | Trigger a manual resynchronization of all schedulers |
| GET | `/admin/scheduler/execution/{hookID}` | Get planned execution for a scheduler |
| DELETE | `/admin/scheduler/execution/{hookID}` | Delete a specific scheduler |

---

## 8. Processing Flow Summary

```
Push to default branch
  → Repository analysis by the API
  → Detection of Scheduler type hooks
  → Deletion of old schedulers
  → Instantiation of new schedulers (POST to hooks service)
  → Storage of definitions and calculation of next executions in Redis

Routine every 10 seconds
  → Read all planned executions
  → For each execution whose time has passed:
    → Distributed lock
    → Verify scheduler existence (local + API)
    → Create and enqueue a scheduler-type HookRepositoryEvent
    → Recalculate next execution
  → Event follows the standard flow: analysis → callback → workflow run
```
