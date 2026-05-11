# SPEC — Workflow Run v2 Crafting

## 1. Role of Crafting

**Crafting** is the preparation phase of a workflow run before its execution. It occurs right after the run creation (status `Crafting`) and is responsible for:

- Building execution contexts (CDS, Git, Env)
- Resolving workflow templates
- Computing the semantic version (`cds.version`)
- Resolving and validating all dependencies of each job (actions, worker models, variable sets, integrations)
- Validating the workflow structure (lint)
- Applying workflow-level concurrency rules
- Preparing the `WorkflowData` snapshot (workflow + actions + worker models) that will be used throughout the execution
- Transitioning the run to the engine

---

## 2. Triggering

Crafting runs in two situations:

**Immediate triggering**: when a run is created (via hook, manual, or schedule), its ID is pushed into an internal channel. A background worker picks it up and starts crafting immediately.

**Ticker-based triggering** (recovery): a periodic timer loads from the database all runs that remain in `Crafting` status. This allows resuming runs whose processing may have failed, for example after an API restart. Each run is processed in a dedicated goroutine.

---

## 3. Distributed Lock

Before any processing, a Redis lock is taken on the key identifying the run, with a 5-minute time-to-live. If the lock is not acquired (another node is already processing this run), crafting is silently abandoned. The lock is released at the end of processing, whether successful or not.

---

## 4. Preliminary Checks

- Loading the run from the database.
- If the run no longer exists: no-op.
- If the run status is no longer `Crafting`: no-op (the run has already been processed).
- Loading the VCS server, repository, and project associated with the run.

---

## 5. Building Execution Contexts

Contexts are attached to the run and available in all `${{...}}` expressions during execution.

### 5.1 CDS Context (`cds.*`)

Contains run metadata:

| Field | Content |
|---|---|
| `project_key` | Project key |
| `run_id` | Unique run identifier |
| `run_number` | Sequential run number |
| `run_attempt` | Always `1` at creation |
| `run_url` | Web interface URL for this run |
| `workflow` | Workflow name |
| `workflow_ref` | Git ref of the workflow file (branch or tag) |
| `workflow_sha` | Commit SHA of the workflow file |
| `workflow_vcs_server` | VCS server hosting the workflow |
| `workflow_repository` | Repository hosting the workflow |
| `triggering_actor` | Identifier of the user or service that triggered the run |
| `event_name` | Name of the triggering event (e.g., `push`, `pull-request`) |
| `event` | Full payload of the triggering event |
| `version` | Computed semantic version (populated after semver resolution) |
| `version_next` | Next computed semantic version |
| `workflow_template` | Template name (if resolved from a template) |
| `workflow_template_ref` | Git ref of the template |
| `workflow_template_sha` | Commit SHA of the template |
| `workflow_template_vcs_server` | VCS server hosting the template |
| `workflow_template_repository` | Repository hosting the template |
| `workflow_template_project_key` | Project key of the template |
| `workflow_template_params` | Template parameters |
| `workflow_template_commit_web_url` | Template commit web URL |
| `workflow_template_ref_web_url` | Template ref web URL |
| `workflow_template_repository_web_url` | Template repository web URL |

### 5.2 Git Context (`git.*`)

Built from the triggering event information and supplemented by VCS calls if necessary.

| Field | Content |
|---|---|
| `server` | Name of the VCS server configured on the workflow |
| `repository` | Full repository name (`org/repo`) |
| `repositoryUrl` | Clone URL (SSH if SSH key configured, HTTPS otherwise) |
| `repository_web_url` | Repository web URL |
| `ref` | Full ref (`refs/heads/...` or `refs/tags/...`) |
| `ref_name` | Ref without prefix |
| `ref_type` | `branch` or `tag` |
| `ref_web_url` | Web URL of the branch or tag |
| `sha` | Full commit SHA |
| `sha_short` | First 7 characters of SHA |
| `commit_message` | Triggering commit message |
| `commit_web_url` | Commit web URL |
| `author` | Commit author name |
| `author_email` | Commit author email |
| `semver_current` | Current semantic version derived from Git tags |
| `semver_next` | Next semantic version derived from Git tags |
| `changesets` | List of modified files |
| `pullrequest_id` | PR identifier (if PR event) |
| `pullrequest_web_url` | PR web URL |
| `pullrequest_to_ref` | PR target ref |
| `pullrequest_to_ref_name` | PR target ref short name |
| `repository_origin` | Origin repository of the PR (for forks) |
| `connection` | `ssh` or `https` depending on VCS configuration |
| `ssh_key` | SSH key name (if configured) |
| `gpg_key` | GPG key name (if configured) |

**Automatic resolution of missing values**:
- If no `ref` is provided in the event: the repository's default branch is used.
- If no `sha` is provided: it is resolved from the branch or tag via the VCS API.

### 5.3 Env Context (`env.*`)

Copy of environment variables declared at the `workflow.env` level in the workflow YAML definition.

---

## 6. Workflow Template Resolution

If the `workflow.from` field is set, the workflow is generated from a template.

**Steps**:

1. Search for the template by name in accessible entities (same repository, linked repositories, library project).
2. Resolve the template with parameters declared in `workflow.parameters` → generates a complete `V2Workflow` definition.
3. Load the repository and VCS server hosting the template.
4. Enrich the CDS context with template metadata:
   - Template name and parameters
   - Ref, SHA, VCS server, repository, origin project
   - Commit, ref, and repository web URLs
5. **Update the `env` context**: environment variables declared in the resolved workflow (`workflow.env`) are merged into `run.Contexts.Env`. This allows the template to bring its own default env values, which add to or override those already present. This step is necessary because `buildRunContext` runs before template resolution, and thus cannot account for envs introduced by the template.
6. Recreate the entity finder in the template repository context (for resolving the template's actions and worker models).
7. Lint the resolved workflow (without the `from` field).

If an error occurs at any step, the run is set to fail (`Fail`).

---

## 7. Semantic Version Computation

This step is optional and only executed if the `workflow.semver` field is defined.

### 7.1 Supported Version Sources

| Source | File read | Key extracted |
|---|---|---|
| `git` | — (uses Git tags) | `git.semver_current` |
| `helm` | `Chart.yaml` | `version` field |
| `cargo` | `Cargo.toml` | `[package].version` |
| `npm` / `yarn` | `package.json` | `version` field |
| `file` | Custom path | First line of the file |
| `poetry` | `pyproject.toml` | `[project].version` or `[tool.poetry].version` |
| `debian` | `debian/changelog` | Format `package (version) distribution; urgency=…` |

For file-based sources, the content is retrieved from the VCS at the run's SHA. The content is decoded from base64 for GitHub, GitLab, Gitea, and Forgejo.

### 7.2 Determining the Release Branch

A ref is considered a **release ref** if:
- The `semver.release_refs` list is empty and the current ref is the repository's default branch, **or**
- The current ref matches one of the glob patterns defined in `semver.release_refs`.

For the `git` source, behavior is different: a tag-type ref is directly treated as a release.

### 7.3 Version Computation

**If the ref is a release ref AND the version has not yet been saved**:
- The version is directly the value extracted from the source file (clean version, e.g., `1.2.3`).
- This version will be saved in the database at the end of crafting to avoid recomputing it.

**In all other cases** (development branch, or version already saved):
- An interpolation pattern is applied.
- The first glob pattern from `semver.schema` matching the current ref is used.
- If no specific pattern matches, the `**/*` pattern from `semver.schema` is used.
- If no schema is defined, a default pattern is applied (includes run number and short SHA).
- The pattern is interpolated with the run's contexts (including `version` = version extracted from the source file).

The computed version must be valid according to the SemVer 2.0 specification.

**Without `workflow.semver`**: `cds.version` = `git.semver_current` and `cds.version_next` = `git.semver_next` from the triggering event.

---

## 8. Workflow Validation (Lint)

A structural validation of the workflow is performed after template resolution. It checks notably:
- Consistency of job dependencies (`needs`)
- Validity of referenced stages
- Gate consistency

If a lint error occurs, the run transitions to `Fail` status.

---

## 9. Integration Validation

Integrations are verified at two levels:

**Workflow level** (`workflow.integrations`):
- Each integration must exist on the project.
- Only one `artifact_manager` type integration is allowed per workflow.

**Job level** (`job.integrations`):
- Each integration must exist on the project.
- `artifact_manager` type integrations are **not** allowed at the job level (only at the workflow level).

If an error occurs, the run fails with an explanatory message.

---

## 10. Per-Job Dependency Resolution

For each job in the workflow, the following steps are executed. A job without steps and without `from` is ignored.

### 10.1 Region Propagation from Integrations

If the job references integrations but does not declare an explicit region, and the integration has a `region` type configuration, that value is used as the job's region.

### 10.2 Action Resolution (`uses`)

For each step declaring `uses: ...`:

1. Search for the action by name in accessible entities (same repository first, then linked repositories, then library project).
2. Rewrite `uses` in canonical form `actions/<complete-name>`.
3. Recursive resolution of sub-actions used by the found action's steps.
4. Add all found actions to the finder cache (local and remote).

If an action is not found or is ambiguous, the run fails.

### 10.3 Worker Model Validation

If the job's `runs_on.model` is not an interpolated expression and the job is not a template job:

1. Resolve the worker model by name in accessible entities. The full name is stored in the job.
2. Verify that the target region (explicit or default) exists.
3. Verify that at least one registered hatchery:
   - Supports this worker model type
   - Has RBAC rights on the target region

If no hatchery can execute the job in the requested region, the run fails with an explanatory message.

### 10.4 Variable Set Validation

For the workflow and for each job:
- Each referenced `variable_set` must exist on the project.
- Workflow and job variable sets are merged and deduplicated on the job.

### 10.5 Job Concurrency Interpolation

If the `job.concurrency` field contains a `${{...}}` expression, it is interpolated with the run's contexts at this stage.

---

## 11. Workflow Concurrency Interpolation

After the per-job resolution loop:

- For each `workflow.concurrencies` rule: if the name contains `${{...}}`, it is interpolated.
- Default values applied: `order = oldest_first`, `pool = 1`.
- If `workflow.concurrency` (the reference to a rule) contains `${{...}}`, it is interpolated.

---

## 12. Building the WorkflowData Snapshot

At the end of all resolutions, the run stores a complete and immutable snapshot of all entities required for execution:

- **`WorkflowData.Workflow`**: complete workflow definition (potentially resolved from a template).
- **`WorkflowData.Actions`**: map `<complete-name> → V2Action` of all actions used, directly or transitively.
- **`WorkflowData.WorkerModels`**: map `<complete-name> → V2WorkerModel` of all referenced worker models.

This snapshot guarantees that any subsequent modification of entities in the database does not affect the running run.

---

## 13. Workflow-Level Concurrency Management

If `workflow.concurrency` is set:

1. **Loading the definition**: the rule is searched first in the concurrencies declared in the workflow (`workflow.concurrencies`), then in concurrencies defined at the project level.

2. **Condition evaluation**: if the rule has an `if` field, it is interpolated. If the condition is false, the concurrency rule is ignored for this run.

3. **Concurrency lock**: a Redis lock is taken on the unique concurrency key to avoid race conditions during run registration.

4. **Rule application**: checks the state of active runs sharing the same concurrency key. Depending on the rules (`cancel-in-progress`, `oldest_first`, `newest_first`, `pool`), existing runs may be marked for cancellation.

---

## 14. Persistence and Finalization

All modifications are committed in a single transaction:

- Persist the run (with WorkflowData, contexts, concurrency, status)
- Persist collected information and warning messages
- If `mustSaveVersion = true` (first release of this version): save `cds.version` in the database
- Cancel runs affected by concurrency rules

After commit:
- A `EventRunBuilding` event is published on Redis (notifies UI and services).
- The run is enqueued in the engine channel to start job orchestration.

---

## 15. Error Handling

Two categories of errors:

**Business error (run stopped)**: any validation error (template not found, missing action, incompatible worker model, lint failure, non-existent variable set, invalid integration, non-semver-compatible version...) calls `stopRun` which:
- Persists the error message(s).
- Sets the run to `Fail` status (or `Skipped` if all messages are `Warning` level).
- Publishes an `EventRunEnded` event.

**Technical error**: infrastructure errors (DB, Redis, VCS unreachable...) return an error. The run stays in `Crafting` status and will be resumed by the ticker on the next cycle.
