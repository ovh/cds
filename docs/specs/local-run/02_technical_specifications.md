# Technical Specifications — `cdsctl local run`

**Feature**: Local Execution of CDS v2 Workflows  
**Status**: Draft  
**Authors**: —  
**Last Updated**: 2026-03-09

---

## 1. Overall Architecture

### 1.1 System Overview

The feature follows a **"Remote Execution, Local Code"** model. The developer's CLI client archives and uploads local code, then triggers a standard CDS workflow run where the checkout step is substituted with the uploaded archive.

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Developer Workstation                         │
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌───────────────────────┐  │
│  │  Local Git   │───▶│  cdsctl      │───▶│  Archive .tar.gz     │  │
│  │  repo        │    │  local run   │    │  (code + .cds/)      │  │
│  └──────────────┘    └──────┬───────┘    └───────────┬───────────┘  │
│                             │                         │              │
│                             │ REST API                │ Upload       │
│                             ▼                         ▼              │
└─────────────────────────────┼─────────────────────────┼──────────────┘
                              │                         │
┌─────────────────────────────┼─────────────────────────┼──────────────┐
│                        CDS Server                     │              │
│                             │                         │              │
│  ┌──────────────────────────▼────────┐   ┌────────────▼───────────┐  │
│  │  API Engine                       │   │  CDN                   │  │
│  │  - New local run endpoint         │   │  - Archive storage     │  │
│  │  - Workflow run crafting          │   │  - Log streaming       │  │
│  │  - "local" flag marking           │   │                        │  │
│  └──────────────┬───────────────────┘   └────────────────────────┘  │
│                 │                                     ▲              │
│                 │ Queue job                           │ Logs         │
│                 ▼                                     │              │
│  ┌──────────────────────────────────┐                │              │
│  │  Hatchery + Worker              │────────────────┘              │
│  │  - Normal spawn                 │                               │
│  │  - Checkout → download archive  │                               │
│  │  - Execute steps normally       │                               │
│  └──────────────────────────────────┘                               │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Remote execution (not local) | Ensures full fidelity with production environment; avoids replicating hatcheries, secrets, services locally |
| Archive upload (not git patch) | Simpler to implement; handles untracked files; no need for a baseline commit on the server |
| CDN storage for archives | Reuses existing CDN infrastructure; workers already know how to download from CDN |
| Trigger is always manual | Avoids complexity of simulating git events; user intent is explicit |
| Same RBAC as manual runs | Security model is already proven; no new attack surface |

---

## 2. Execution Flow

### 2.1 Sequence Diagram

```
  Developer        cdsctl          CDS API         CDN           Hatchery       Worker
     │                │               │              │               │             │
     │  cdsctl local  │               │              │               │             │
     │     run        │               │              │               │             │
     │───────────────▶│               │              │               │             │
     │                │               │              │               │             │
     │                │──── git info ─┤              │               │             │
     │                │  (branch,sha) │              │               │             │
     │                │               │              │               │             │
     │                │── create archive (.tar.gz) ──┤              │             │
     │                │               │              │               │             │
     │                │── POST /local/archive ──────▶│              │             │
     │                │               │     store    │              │             │
     │                │◀── archive_ref ─────────────┤              │             │
     │                │               │              │               │             │
     │                │── POST /local/run ──────────▶│              │             │
     │                │               │  create run  │              │             │
     │                │               │  (marked     │              │             │
     │                │               │   local)     │              │             │
     │                │◀── run_id, url ─────────────┤              │             │
     │                │               │              │               │             │
     │                │               │── queue job ─┼──────────────▶│             │
     │                │               │              │               │── spawn ───▶│
     │                │               │              │               │             │
     │                │               │              │               │  take job   │
     │                │               │              │               │◀────────────│
     │                │               │              │               │             │
     │                │               │              │  download     │             │
     │                │               │              │  archive      │             │
     │                │               │              │◀──────────────┼─────────────│
     │                │               │              │──────────────▶│             │
     │                │               │              │               │             │
     │                │               │              │               │  execute    │
     │                │               │              │               │  steps      │
     │                │               │              │               │◀────────────│
     │                │── WebSocket (log streaming) ─┤              │             │
     │◀── logs ───────│               │              │               │  send logs  │
     │                │               │              │◀──────────────┼─────────────│
     │                │               │              │               │             │
     │◀── result ─────│               │              │               │  report     │
     │                │               │              │               │  result     │
     │                │               │              │               │             │
```

### 2.2 Step-by-Step Flow

#### Step 1: Local Preparation (`cdsctl`)

| Sub-step | Action | Details |
|----------|--------|---------|
| 1a | Detect git repository | Read git remote URL, current branch, HEAD commit SHA |
| 1b | Resolve CDS project | Call API to map git remote URL → CDS project/VCS/repo |
| 1c | Read workflows | Parse `.cds/workflows/*.yml` files from local disk |
| 1d | Select workflow | Auto-detect if single workflow; prompt if multiple |
| 1e | Create archive | `.tar.gz` of working directory (respecting `.gitignore` + `.cdsignore`) |
| 1f | Compute hash | SHA256 of the archive for integrity verification |

#### Step 2: Archive Upload (`cdsctl` → CDS API/CDN)

| Sub-step | Action | Details |
|----------|--------|---------|
| 2a | Upload archive | `POST /v2/project/{projectKey}/local/archive` with chunked transfer |
| 2b | Server stores | CDN stores archive with `TypeItemLocalArchive` type and TTL |
| 2c | Return reference | Server returns `archive_ref` identifier |

#### Step 3: Run Trigger (`cdsctl` → CDS API)

| Sub-step | Action | Details |
|----------|--------|---------|
| 3a | Trigger run | `POST /v2/project/{projectKey}/local/run` with archive ref, workflow YAML, variables, job filter |
| 3b | Create run | Server creates `V2WorkflowRun` with `IsLocalRun=true` |
| 3c | Craft workflow | Standard crafting process (using local workflow YAML if provided) |
| 3d | Queue jobs | Jobs are enqueued normally (filtered jobs are marked `Skipped`) |

#### Step 4: Execution (Hatchery + Worker)

| Sub-step | Action | Details |
|----------|--------|---------|
| 4a | Spawn worker | Hatchery spawns a worker as usual |
| 4b | Take job | Worker picks up job from queue |
| 4c | Checkout override | At checkout step: download archive from CDN instead of git clone |
| 4d | Execute steps | Remaining steps execute normally |
| 4e | Report results | Worker sends results back to API |

#### Step 5: Log Streaming (CDS → `cdsctl`)

| Sub-step | Action | Details |
|----------|--------|---------|
| 5a | Connect WebSocket | CLI connects to CDN WebSocket for real-time events |
| 5b | Receive events | Log events streamed as they are produced |
| 5c | Display logs | Formatted output with colors, spinners, step grouping |
| 5d | Show result | Final SUCCESS/FAILURE summary with timing |

---

## 3. Impacted Components

### 3.1 CLI (`cli/cdsctl/`)

**New file**: `cli/cdsctl/local.go`

**Responsibilities**:
- Git repository detection and metadata collection
- Archive creation (`.tar.gz` with exclusions)
- CDS project resolution from git remote
- Archive upload to CDS
- Run trigger with parameters
- Real-time log streaming via WebSocket
- Terminal UX (colors, spinners, progress)

**Dependencies**: Existing `cdsclient` SDK for API calls.

### 3.2 SDK (`sdk/`)

**Modified files**:
- `sdk/v2_workflow.go` — New `V2WorkflowRunLocalRequest` struct
- `sdk/v2_workflow_run.go` — Add `IsLocalRun` and `LocalArchiveRef` fields to `V2WorkflowRun`

**New files**:
- `sdk/cdsclient/client_local.go` — Client methods for archive upload and local run trigger

**New types**:

```go
// V2WorkflowRunLocalRequest represents the payload for triggering a local run
type V2WorkflowRunLocalRequest struct {
    V2WorkflowRunManualRequest
    ArchiveRef   string            `json:"archive_ref"`    // CDN reference to uploaded archive
    WorkflowYAML string            `json:"workflow_yaml"`  // Local workflow YAML content (optional)
    JobFilter    []string          `json:"job_filter"`     // Run only these jobs (optional)
    Variables    map[string]string `json:"variables"`      // Variable overrides (optional)
}

// LocalRunContext provides local run metadata to the worker
type LocalRunContext struct {
    ArchiveRef string `json:"archive_ref"` // CDN reference to download
    RunBy      string `json:"run_by"`      // Username who triggered the local run
}
```

### 3.3 API Engine (`engine/api/`)

**New endpoints**:

| Method | Route | Handler | Auth |
|--------|-------|---------|------|
| `POST` | `/v2/project/{projectKey}/local/archive` | `postLocalArchiveHandler` | Bearer token + project write permission |
| `POST` | `/v2/project/{projectKey}/local/run` | `postLocalRunHandler` | Bearer token + workflow execute permission |

**Modified files**:
- `engine/api/api_routes.go` — Register new routes
- `engine/api/v2_workflow_run.go` — New handlers
- `engine/api/v2_workflow_run_craft.go` — Adapt crafting for local runs (use provided workflow YAML, apply job filter)

### 3.4 Worker (`engine/worker/`)

**Modified files**:
- `engine/worker/internal/action/builtin_checkout_application.go` — Add local archive download path
- `engine/worker/internal/runV2.go` — Propagate `LocalRunContext` in job metadata

**Behavior change**: The checkout action checks `runJobContext.LocalRun` before deciding between git clone and archive download.

### 3.5 CDN (`engine/cdn/`)

**Modified files**:
- Add `TypeItemLocalArchive` constant to CDN item types
- Configure TTL-based cleanup for local archives

**No new endpoints needed**: Archive upload is handled by the API engine, which stores into CDN. Archive download uses existing CDN download mechanisms.

---

## 4. Security Model

### 4.1 Authentication & Authorization

| Layer | Mechanism | Details |
|-------|-----------|---------|
| **CLI → API** | Bearer token | Standard `cdsctl` authentication via `~/.cdsrc` |
| **Project access** | RBAC | User must have write access to the CDS project |
| **Workflow execution** | RBAC | User must have permission to run the specific workflow |
| **Worker → CDN** | JWS signature | Worker uses existing signed requests to download archives |

### 4.2 Archive Integrity

```
Client-side:
  1. Create archive from local files
  2. Compute SHA256(archive)
  3. Send archive + SHA256 in X-CDS-Archive-SHA256 header

Server-side:
  1. Receive archive stream
  2. Compute SHA256 while reading
  3. Compare with client-provided hash
  4. Reject if mismatch → 400 Bad Request
```

### 4.3 Archive Size Limits

| Limit | Default | Configurable |
|-------|---------|-------------|
| Warning threshold | 50 MB | No (client-side) |
| Hard limit | 500 MB | Yes (server-side configuration) |

### 4.4 Archive Lifecycle

| Phase | Details |
|-------|---------|
| **Upload** | Stored in CDN with `TypeItemLocalArchive` type |
| **TTL** | 24 hours (server-configurable) |
| **Cleanup** | Background goroutine deletes expired archives |
| **Reference** | `local-archive-<uuid>` format |

### 4.5 Security Considerations

| Risk | Mitigation |
|------|------------|
| Malicious code in archive (to exfiltrate secrets) | Same risk as pushing malicious code to VCS; mitigated by RBAC |
| Archive bomb (zip bomb) | Size limit enforced; server rejects oversized uploads |
| Unauthorized access | Standard CDS RBAC; no new permissions model |
| Archive tampering | SHA256 integrity check end-to-end |
| Secret leakage in logs | Same log masking as normal runs |

---

## 5. CDS Project Resolution

### 5.1 Auto-Detection Algorithm

```
function resolveProject(localRepoPath):
    1. remotes = exec("git remote -v")
    2. for each remote in remotes:
         url = parseGitURL(remote.fetchURL)
         // Try to resolve via CDS API
         response = GET /v2/repository/resolve?url={url}
         if response.found:
           return (response.project_key, response.vcs_server, response.repository)
    3. if --project flag provided:
         return (flagValue, autodetect_vcs, autodetect_repo)
    4. error("Repository not found in CDS. Use --project to specify.")
```

### 5.2 Ambiguity Resolution

If the same repository is registered in multiple CDS projects:
- If `--project` flag is provided → use it
- If interactive TTY → prompt user to select
- If non-interactive → error with list of matching projects

---

## 6. Data Model Changes

### 6.1 Database Schema

```sql
-- Migration: Add local run support to v2_workflow_run
ALTER TABLE v2_workflow_run ADD COLUMN is_local_run BOOLEAN DEFAULT FALSE;
ALTER TABLE v2_workflow_run ADD COLUMN local_archive_ref TEXT;

-- Index for cleanup queries
CREATE INDEX idx_v2_workflow_run_local_archive
  ON v2_workflow_run(local_archive_ref)
  WHERE local_archive_ref IS NOT NULL;
```

### 6.2 Go Struct Modifications

#### V2WorkflowRun

```go
type V2WorkflowRun struct {
    // ... existing fields ...

    IsLocalRun      bool   `json:"is_local_run" db:"is_local_run"`
    LocalArchiveRef string `json:"local_archive_ref,omitempty" db:"local_archive_ref"`
}
```

#### WorkflowRunJobsContext

```go
type WorkflowRunJobsContext struct {
    // ... existing fields ...

    LocalRun *LocalRunContext `json:"local_run,omitempty"`
}
```

#### CDN Item Type

```go
const (
    // ... existing types ...
    TypeItemLocalArchive CDNItemType = "local-archive"
)
```

---

## 7. API Contracts

### 7.1 Upload Archive

**`POST /v2/project/{projectKey}/local/archive`**

Request:
```http
POST /v2/project/MYPROJ/local/archive HTTP/1.1
Authorization: Bearer <token>
Content-Type: application/gzip
X-CDS-Archive-SHA256: e3b0c44298fc1c149afbf4c8996fb924...
X-CDS-Archive-Filename: myrepo-1709920000.tar.gz
Transfer-Encoding: chunked

<binary archive data>
```

Response (201 Created):
```json
{
  "ref": "local-archive-550e8400-e29b-41d4-a716-446655440000",
  "size": 12345678,
  "sha256": "e3b0c44298fc1c149afbf4c8996fb924...",
  "expires_at": "2026-03-10T19:25:00Z"
}
```

Error responses:
| Status | Condition |
|--------|-----------|
| 400 | SHA256 mismatch |
| 400 | Archive too large |
| 401 | Unauthorized |
| 403 | No project write permission |
| 413 | Payload too large (server limit) |

### 7.2 Trigger Local Run

**`POST /v2/project/{projectKey}/local/run`**

Request:
```json
{
  "archive_ref": "local-archive-550e8400-e29b-41d4-a716-446655440000",
  "vcs_server": "github",
  "repository": "myorg/myrepo",
  "workflow": "build-and-deploy",
  "workflow_yaml": "name: build-and-deploy\non:\n  push: ...",
  "branch": "feature/my-branch",
  "commit": "abc1234def5678",
  "job_filter": ["build", "unit-tests"],
  "variables": {
    "ENV": "staging",
    "DEBUG": "true"
  }
}
```

Response (201 Created):
```json
{
  "run_id": "550e8400-e29b-41d4-a716-446655440001",
  "run_number": 42,
  "ui_url": "https://cds.example.com/project/MYPROJ/run/42",
  "status": "Crafting"
}
```

Error responses:
| Status | Condition |
|--------|-----------|
| 400 | Invalid archive_ref, invalid workflow YAML, unknown job in filter |
| 401 | Unauthorized |
| 403 | No workflow execute permission |
| 404 | Project, VCS, repository, or workflow not found |
| 410 | Archive expired |

---

## 8. Interactions with Existing Features

| Feature | Behavior with Local Run |
|---------|------------------------|
| **Concurrency controls** | Local runs respect the same concurrency rules as normal runs |
| **Manual gates** | Work normally (but rarely useful in local context) |
| **Matrix strategy** | Supported — all matrix combinations execute |
| **Job services** | Work normally (e.g., postgres containers for integration tests) |
| **Artifact upload/download** | Work normally via CDN |
| **Build caches** | Work normally |
| **Notifications** | Sent as usual (consider `--no-notify` flag for future iteration) |
| **Hooks/Triggers** | Ignored — local run trigger is always "manual" |
| **Job retry** | Works normally |
| **Conditions (`if`)** | Evaluated with local git context |
| **`${{ git.* }}` context** | Populated from local git metadata (branch, SHA, author, message) |
| **`${{ cds.* }}` context** | Populated normally by the server |
| **Workflow `on:` triggers** | Ignored; the workflow runs regardless of trigger conditions |

---

## 9. Configuration

### 9.1 Server-Side Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `localRun.enabled` | `true` | Enable/disable the local run feature |
| `localRun.maxArchiveSize` | `500MB` | Maximum archive size accepted |
| `localRun.archiveTTL` | `24h` | Time-to-live for uploaded archives |
| `localRun.maxConcurrentRuns` | `5` | Max concurrent local runs per user |

### 9.2 Client-Side Configuration

No additional configuration beyond standard `cdsctl` setup (`~/.cdsrc`).

Optional `.cdsignore` file in repo root for archive exclusions.
