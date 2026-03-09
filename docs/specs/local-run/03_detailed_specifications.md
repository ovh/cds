# Detailed Specifications — `cdsctl local run`

**Feature**: Local Execution of CDS v2 Workflows  
**Status**: Draft  
**Authors**: —  
**Last Updated**: 2026-03-09

---

## 1. CLI Implementation

### 1.1 Command Registration

The `local` command group is registered as a top-level `cdsctl` subcommand.

**File**: `cli/cdsctl/local.go`

```go
func localCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "local",
        Short: "Run CDS workflows with local code",
    }
    cmd.AddCommand(localRunCmd())
    return cmd
}

func localRunCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "run",
        Short: "Execute a v2 workflow using local repository code",
        Long:  `Archives the local git repository and triggers a CDS workflow run
using the local code instead of cloning from VCS. The execution happens
on CDS infrastructure (hatcheries/workers) with full fidelity.`,
        RunE: localRunFunc,
    }
    cmd.Flags().StringP("workflow", "w", "", "Workflow name (auto-detected if only one)")
    cmd.Flags().StringP("job", "j", "", "Run only this job")
    cmd.Flags().StringArrayP("var", "v", nil, "Variable override KEY=VALUE (repeatable)")
    cmd.Flags().String("branch", "", "Override branch name (default: current git branch)")
    cmd.Flags().Bool("dry-run", false, "Validate and create archive without executing")
    cmd.Flags().Bool("no-stream", false, "Don't stream logs, just return the run URL")
    cmd.Flags().StringArray("exclude", nil, "Additional glob patterns to exclude from archive")
    cmd.Flags().Bool("include-untracked", true, "Include untracked files in archive")
    cmd.Flags().String("project", "", "CDS project key (auto-detected from git remote)")
    return cmd
}
```

### 1.2 Main Execution Flow

```go
func localRunFunc(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Step 1: Collect git metadata
    gitInfo, err := collectGitInfo(ctx)
    if err != nil {
        return fmt.Errorf("not a git repository: %w", err)
    }

    // Step 2: Resolve CDS project
    projectKey, vcsServer, repository, err := resolveProject(ctx, client, gitInfo, flagProject)
    if err != nil {
        return err
    }

    // Step 3: Detect and select workflow
    workflowName, workflowYAML, err := selectWorkflow(ctx, flagWorkflow)
    if err != nil {
        return err
    }

    // Step 4: Create archive
    archivePath, sha256, stats, err := createArchive(ctx, gitInfo.RepoRoot, excludePatterns, includeUntracked)
    if err != nil {
        return err
    }
    defer os.Remove(archivePath)
    fmt.Fprintf(os.Stderr, "✓ Archive created (%s, %d files)\n", humanize.Bytes(stats.Size), stats.FileCount)

    if flagDryRun {
        fmt.Fprintln(os.Stderr, "Dry run: archive validated, not executing.")
        return nil
    }

    // Step 5: Upload archive
    archiveRef, err := uploadArchive(ctx, client, projectKey, archivePath, sha256)
    if err != nil {
        return err
    }
    fmt.Fprintf(os.Stderr, "✓ Archive uploaded\n")

    // Step 6: Trigger run
    runResult, err := triggerLocalRun(ctx, client, projectKey, vcsServer, repository, sdk.V2WorkflowRunLocalRequest{
        ArchiveRef:   archiveRef,
        WorkflowYAML: workflowYAML,
        JobFilter:    parseJobFilter(flagJob),
        Variables:    parseVariables(flagVars),
        V2WorkflowRunManualRequest: sdk.V2WorkflowRunManualRequest{
            Branch: branchName,
        },
    })
    if err != nil {
        return err
    }
    fmt.Fprintf(os.Stderr, "✓ Run #%d started (local)\n🔗 %s\n", runResult.RunNumber, runResult.UIURL)

    if flagNoStream {
        return nil
    }

    // Step 7: Stream logs
    return streamLogs(ctx, client, runResult.RunID)
}
```

---

## 2. Archive Creation

### 2.1 File Collection Algorithm

```
Input: repoRoot, excludePatterns[], includeUntracked bool
Output: archivePath, sha256, stats

1. files = []

2. // Collect git-tracked files (includes staged + modified)
   trackedFiles = exec("git ls-files --cached --modified", cwd=repoRoot)
   files.append(trackedFiles)

3. // Collect untracked files (if enabled)
   if includeUntracked:
       untrackedFiles = exec("git ls-files --others --exclude-standard", cwd=repoRoot)
       files.append(untrackedFiles)

4. // Deduplicate
   files = unique(files)

5. // Apply .cdsignore exclusions
   if exists(repoRoot + "/.cdsignore"):
       ignorePatterns = parseGitignore(repoRoot + "/.cdsignore")
       files = files.filter(f => !ignorePatterns.match(f))

6. // Apply --exclude flag patterns
   for pattern in excludePatterns:
       files = files.filter(f => !globMatch(pattern, f))

7. // Always exclude .git directory
   files = files.filter(f => !strings.HasPrefix(f, ".git/"))

8. // Create tar.gz archive
   archivePath = tempDir + "/" + repoName + "-" + timestamp + ".tar.gz"
   createTarGz(archivePath, repoRoot, files)

9. // Compute SHA256
   sha256 = computeSHA256(archivePath)

10. // Validate size
    size = fileSize(archivePath)
    if size > 50MB:
        warn("Archive is large (%s). Consider using .cdsignore", humanize(size))
    if size > hardLimit:
        error("Archive size (%s) exceeds limit (%s)", humanize(size), humanize(hardLimit))

11. return archivePath, sha256, {Size: size, FileCount: len(files)}
```

### 2.2 `.cdsignore` File Format

The `.cdsignore` file follows `.gitignore` syntax and is placed at the repository root. It defines additional patterns to exclude from the local archive (on top of `.gitignore`).

```gitignore
# Large test fixtures
test/fixtures/large-dataset.sql
test/fixtures/*.dump

# Build artifacts (if not already in .gitignore)
build/
dist/
*.o
*.a

# Media files
*.mp4
*.avi
*.mov

# IDE directories
.idea/
.vscode/
```

### 2.3 Archive Format

| Property | Value |
|----------|-------|
| Format | gzip-compressed tar (`.tar.gz`) |
| Root directory | Files are stored relative to repo root (no wrapping directory) |
| Permissions | Preserved from filesystem |
| Symlinks | Followed (not stored as symlinks) |
| Max file size | No per-file limit; total archive limit applies |
| Filename | `{repo-name}-{unix-timestamp}.tar.gz` |

### 2.4 Archive Progress Display

```
⠋ Archiving local repository...
  Collecting files: 1,847 files
  Compressing: 12.3 MB
✓ Archive created (12.3 MB, 1,847 files)
```

For archives > 50 MB:
```
⠋ Archiving local repository...
  Collecting files: 12,456 files
  Compressing: 156.2 MB
⚠ Large archive (156.2 MB). Consider using .cdsignore to exclude unnecessary files.
✓ Archive created (156.2 MB, 12,456 files)
```

---

## 3. Git Metadata Collection

### 3.1 Collected Information

```go
type GitInfo struct {
    RepoRoot    string // Absolute path to git repo root
    RemoteURL   string // Origin fetch URL
    Branch      string // Current branch name
    SHA         string // HEAD commit SHA (full)
    SHAShort    string // HEAD commit SHA (7 chars)
    Author      string // Last commit author name
    AuthorEmail string // Last commit author email
    Message     string // Last commit message (first line)
    Tag         string // Tag pointing to HEAD (if any)
    IsDirty     bool   // True if working directory has uncommitted changes
}
```

### 3.2 Collection Commands

| Field | Command | Fallback |
|-------|---------|----------|
| `RepoRoot` | `git rev-parse --show-toplevel` | Error if not in a git repo |
| `RemoteURL` | `git remote get-url origin` | Try other remotes, then error |
| `Branch` | `git branch --show-current` | `git rev-parse --abbrev-ref HEAD` |
| `SHA` | `git rev-parse HEAD` | Error if no commits |
| `SHAShort` | `SHA[:7]` | — |
| `Author` | `git log -1 --format='%an'` | Empty string |
| `AuthorEmail` | `git log -1 --format='%ae'` | Empty string |
| `Message` | `git log -1 --format='%s'` | Empty string |
| `Tag` | `git describe --tags --exact-match HEAD 2>/dev/null` | Empty string |
| `IsDirty` | `git status --porcelain` (non-empty output) | false |

### 3.3 Dirty Working Directory

When the working directory has uncommitted changes (`IsDirty=true`), the CLI displays:

```
ℹ Working directory has uncommitted changes. The archive will include your working copy as-is.
```

This is informational only — local runs are designed to work with uncommitted code.

---

## 4. Worker Checkout Modification

### 4.1 Decision Logic

The checkout action must determine whether to perform a standard git clone or use a local archive.

**File**: `engine/worker/internal/action/builtin_checkout_application.go`

```go
func RunCheckoutApplication(ctx context.Context, wk workerruntime.Runtime,
    a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {

    jobContext := wk.GetJobContext()

    // LOCAL RUN: use archive instead of git clone
    if jobContext.LocalRun != nil && jobContext.LocalRun.ArchiveRef != "" {
        return runCheckoutFromLocalArchive(ctx, wk, jobContext.LocalRun.ArchiveRef)
    }

    // NORMAL RUN: standard git clone
    return runCheckoutFromVCS(ctx, wk, a, secrets)
}
```

### 4.2 Archive Download and Extraction

```go
func runCheckoutFromLocalArchive(ctx context.Context, wk workerruntime.Runtime,
    archiveRef string) (sdk.Result, error) {

    workspace := wk.GetJobContext().CDS.Workspace

    // Log the override
    wk.SendLog(ctx, workerruntime.LevelInfo,
        "Local run detected: downloading archive instead of git clone")

    // Download archive from CDN
    reader, err := wk.Client().CDNItemDownload(ctx, archiveRef, sdk.TypeItemLocalArchive)
    if err != nil {
        return sdk.Result{Status: sdk.StatusFail},
            fmt.Errorf("failed to download local archive: %w", err)
    }
    defer reader.Close()

    // Extract tar.gz into workspace
    wk.SendLog(ctx, workerruntime.LevelInfo,
        fmt.Sprintf("Extracting archive into %s", workspace))

    fileCount, err := extractTarGz(reader, workspace)
    if err != nil {
        return sdk.Result{Status: sdk.StatusFail},
            fmt.Errorf("failed to extract archive: %w", err)
    }

    wk.SendLog(ctx, workerruntime.LevelInfo,
        fmt.Sprintf("✓ Extracted %d files from local archive", fileCount))

    return sdk.Result{Status: sdk.StatusSuccess}, nil
}
```

### 4.3 Archive Extraction Safety

```go
func extractTarGz(reader io.Reader, destDir string) (int, error) {
    gzReader, err := gzip.NewReader(reader)
    if err != nil {
        return 0, fmt.Errorf("invalid gzip: %w", err)
    }
    defer gzReader.Close()

    tarReader := tar.NewReader(gzReader)
    fileCount := 0

    for {
        header, err := tarReader.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fileCount, fmt.Errorf("tar read error: %w", err)
        }

        // SECURITY: prevent path traversal attacks
        targetPath := filepath.Join(destDir, header.Name)
        if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
            return fileCount, fmt.Errorf("archive contains path traversal: %s", header.Name)
        }

        switch header.Typeflag {
        case tar.TypeDir:
            if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
                return fileCount, err
            }
        case tar.TypeReg:
            if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
                return fileCount, err
            }
            f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
            if err != nil {
                return fileCount, err
            }
            if _, err := io.Copy(f, tarReader); err != nil {
                f.Close()
                return fileCount, err
            }
            f.Close()
            fileCount++
        }
    }

    return fileCount, nil
}
```

---

## 5. Log Streaming

### 5.1 WebSocket Connection

The CLI uses the existing CDN WebSocket infrastructure to receive log events in real time.

```go
func streamLogs(ctx context.Context, client cdsclient.Interface, runID string) error {
    // Connect to CDN WebSocket
    conn, err := client.CDNWebsocketConnect(ctx)
    if err != nil {
        return fmt.Errorf("failed to connect to log stream: %w", err)
    }
    defer conn.Close()

    // Subscribe to this run's events
    if err := conn.Subscribe(sdk.CDNWSEvent{RunID: runID}); err != nil {
        return fmt.Errorf("failed to subscribe to run events: %w", err)
    }

    // Initialize terminal renderer
    renderer := newTerminalRenderer(os.Stdout, isTerminal(os.Stdout))

    // Poll run status concurrently
    statusCh := pollRunStatus(ctx, client, runID)

    for {
        select {
        case event := <-conn.Events():
            renderer.HandleEvent(event)

        case status := <-statusCh:
            renderer.SetRunStatus(status)
            if status.IsTerminal() {
                renderer.PrintSummary()
                if status == sdk.StatusFail {
                    return cli.ExitError{Code: 1}
                }
                return nil
            }

        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### 5.2 Terminal Renderer

The terminal renderer formats log output with visual hierarchy and color coding.

**State machine per job**:

```
Waiting → Running → Success
                  → Failure
                  → Skipped
```

**Output rules**:

| Element | TTY Output | Non-TTY Output |
|---------|-----------|----------------|
| Job header | `━━━ Job: build ━━━` (bold) | `--- Job: build ---` |
| Step start | `  ▶ Step 1/3: checkout` (blue) | `  > Step 1/3: checkout` |
| Log line | `    <text>` (indented) | `    <text>` |
| Step success | `    ✓ Done (0.8s)` (green) | `    OK (0.8s)` |
| Step failure | `    ✗ Failed (0.8s)` (red) | `    FAILED (0.8s)` |
| Job success | `✓ Job build: SUCCESS (21.4s)` (green) | `OK Job build: SUCCESS (21.4s)` |
| Job failure | `✗ Job build: FAILURE (21.4s)` (red) | `FAILED Job build: FAILURE (21.4s)` |
| Spinner | `⠋ Waiting for worker...` (animated) | `Waiting for worker...` |

### 5.3 Final Summary

```
✅ Workflow completed: SUCCESS (42.3s)
   Jobs: 3 passed, 0 failed, 1 skipped
   🔗 https://cds.example.com/project/MYPROJ/run/42
```

Or on failure:

```
❌ Workflow completed: FAILURE (38.1s)
   Jobs: 2 passed, 1 failed, 1 skipped
   Failed jobs:
     ✗ deploy (Step 2: deploy-to-staging)
   🔗 https://cds.example.com/project/MYPROJ/run/42
```

### 5.4 Reconnection Strategy

| Event | Action |
|-------|--------|
| WebSocket disconnected | Wait 1s, reconnect, re-subscribe |
| Reconnection failed | Exponential backoff (1s, 2s, 4s, 8s, max 30s) |
| Max retries (10) | Fall back to polling mode (GET logs every 5s) |
| During reconnection | Display `⚠ Connection lost, reconnecting...` |
| After reconnection | Fetch missed logs via HTTP, then resume streaming |

---

## 6. Server-Side: Local Run Crafting

### 6.1 Craft Modifications

The `craftWorkflowRunV2()` function in `engine/api/v2_workflow_run_craft.go` needs modification for local runs.

**Key differences from normal crafting**:

| Aspect | Normal Run | Local Run |
|--------|-----------|-----------|
| Workflow source | Read from VCS at the specified branch/commit | Use `workflow_yaml` from request (if provided), else VCS |
| Git context | Built from VCS webhook event | Built from client-provided git metadata |
| Job selection | All jobs run | Jobs filtered by `job_filter` (unfiltered jobs → `StatusSkipped`) |
| Trigger context | Event type (push, PR, cron...) | Always `"local"` |
| Run metadata | Standard | `IsLocalRun=true`, `LocalArchiveRef` set |

### 6.2 Job Filtering Logic

When `job_filter` is provided:

```go
func applyJobFilter(workflow *sdk.V2Workflow, filter []string) error {
    if len(filter) == 0 {
        return nil // No filter, run all jobs
    }

    // Validate all filtered job names exist
    filterSet := make(map[string]bool)
    for _, name := range filter {
        if _, exists := workflow.Jobs[name]; !exists {
            available := make([]string, 0, len(workflow.Jobs))
            for k := range workflow.Jobs {
                available = append(available, k)
            }
            sort.Strings(available)
            return fmt.Errorf("job '%s' not found. Available jobs: %s",
                name, strings.Join(available, ", "))
        }
        filterSet[name] = true
    }

    // Also include transitive dependencies (needs[])
    resolved := resolveJobDependencies(workflow, filterSet)

    // Mark non-selected jobs as Skipped
    for jobName := range workflow.Jobs {
        if !resolved[jobName] {
            // This job will be marked StatusSkipped during crafting
        }
    }

    return nil
}

func resolveJobDependencies(workflow *sdk.V2Workflow, selected map[string]bool) map[string]bool {
    resolved := make(map[string]bool)
    for name := range selected {
        resolveDeps(workflow, name, resolved)
    }
    return resolved
}

func resolveDeps(workflow *sdk.V2Workflow, jobName string, resolved map[string]bool) {
    if resolved[jobName] {
        return
    }
    resolved[jobName] = true
    job := workflow.Jobs[jobName]
    for _, dep := range job.Needs {
        resolveDeps(workflow, dep, resolved)
    }
}
```

**Note**: When a job is filtered via `--job`, its transitive dependencies (via `needs[]`) are automatically included. For example, if `deploy` needs `build`, running `--job deploy` will also execute `build`.

---

## 7. Error Handling — Detailed Behaviors

### 7.1 CLI-Side Errors

| Error | Detection | User Message | Exit Code |
|-------|-----------|-------------|-----------|
| Not a git repo | `git rev-parse` fails | `Error: current directory is not a git repository` | 1 |
| No `.cds/` directory | `os.Stat(".cds")` | `Error: no .cds/ directory found. Is this a CDS project?` | 1 |
| No workflow files | `glob(".cds/workflows/*.yml")` empty | `Error: no workflow files found in .cds/workflows/` | 1 |
| Multiple workflows, no flag | Count > 1, no `--workflow` | Interactive prompt (TTY) or list (non-TTY) | — |
| Invalid workflow YAML | YAML parse error | `Error: invalid workflow YAML: <parse error>` | 1 |
| Archive too large | Size check | `Error: archive size (X MB) exceeds limit (Y MB). Use --exclude or .cdsignore` | 1 |
| No git remote | `git remote` empty | `Error: no git remote configured. Cannot resolve CDS project.` | 1 |
| Project not found | API 404 | `Error: repository 'URL' not found in CDS. Use --project to specify.` | 1 |
| Upload failure | HTTP error (after retries) | `Error: failed to upload archive after 3 attempts: <error>` | 1 |
| Run trigger failure | API error | `Error: failed to trigger local run: <error>` | 1 |
| Job not found (filter) | API 400 | `Error: job 'X' not found. Available jobs: a, b, c` | 1 |

### 7.2 Server-Side Errors

| Error | HTTP Status | Response Body |
|-------|------------|---------------|
| Archive SHA256 mismatch | 400 | `{"error": "archive integrity check failed: SHA256 mismatch"}` |
| Archive too large | 413 | `{"error": "archive size exceeds limit of 500 MB"}` |
| Archive expired | 410 | `{"error": "archive 'local-archive-xxx' has expired"}` |
| Invalid workflow YAML | 400 | `{"error": "workflow validation failed: <details>"}` |
| Job not found in filter | 400 | `{"error": "job 'xxx' not found in workflow. Available: a, b, c"}` |
| Permission denied | 403 | `{"error": "insufficient permissions to execute workflow"}` |
| Feature disabled | 403 | `{"error": "local run feature is disabled on this server"}` |

### 7.3 Runtime Errors

| Error | Behavior |
|-------|----------|
| Worker fails to download archive | Job fails with clear error message in logs |
| Archive extraction fails | Job fails; log shows extraction error |
| WebSocket connection lost | CLI reconnects automatically (see §5.4) |
| Run cancelled (Ctrl+C) | CLI prompts: "Cancel the remote run? [Y/n]" |

---

## 8. Ctrl+C Handling

### 8.1 Signal Handling Flow

```go
func handleInterrupt(ctx context.Context, cancel context.CancelFunc,
    client cdsclient.Interface, runID string) {

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Fprintln(os.Stderr, "\n⚠ Interrupted")

        if isTerminal(os.Stdin) {
            fmt.Fprint(os.Stderr, "Cancel the remote CDS run? [Y/n] ")
            reader := bufio.NewReader(os.Stdin)
            answer, _ := reader.ReadString('\n')
            answer = strings.TrimSpace(strings.ToLower(answer))

            if answer == "" || answer == "y" || answer == "yes" {
                fmt.Fprintln(os.Stderr, "Cancelling run...")
                _ = client.WorkflowV2RunStop(ctx, runID)
                fmt.Fprintln(os.Stderr, "✓ Run cancelled")
            } else {
                fmt.Fprintln(os.Stderr, "Detaching. Run continues at:")
                fmt.Fprintf(os.Stderr, "🔗 %s\n", runURL)
            }
        }
        cancel()
    }()
}
```

### 8.2 Behavior Summary

| Scenario | Ctrl+C Behavior |
|----------|----------------|
| During archive creation | Abort immediately, clean up temp file |
| During upload | Abort immediately |
| During log streaming (TTY) | Prompt: cancel run or detach |
| During log streaming (non-TTY) | Cancel the remote run |

---

## 9. Archive Cleanup (TTL)

### 9.1 Cleanup Goroutine

A background goroutine in the API engine periodically cleans up expired local archives.

```go
func (api *API) localArchiveCleanupRoutine(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := api.cleanupExpiredLocalArchives(ctx); err != nil {
                log.Error(ctx, "local archive cleanup: %v", err)
            }
        }
    }
}

func (api *API) cleanupExpiredLocalArchives(ctx context.Context) error {
    // Find expired archives
    expiredRuns, err := workflow_v2.LoadLocalRunsWithExpiredArchives(ctx, api.mustDB(), api.Config.LocalRun.ArchiveTTL)
    if err != nil {
        return err
    }

    for _, run := range expiredRuns {
        // Delete archive from CDN
        if err := api.cdnClient.DeleteItem(ctx, sdk.TypeItemLocalArchive, run.LocalArchiveRef); err != nil {
            log.Warn(ctx, "failed to delete archive %s: %v", run.LocalArchiveRef, err)
            continue
        }

        // Clear reference in database
        run.LocalArchiveRef = ""
        if err := workflow_v2.UpdateRun(ctx, api.mustDB(), &run); err != nil {
            log.Warn(ctx, "failed to update run %s: %v", run.ID, err)
        }
    }

    log.Info(ctx, "cleaned up %d expired local archives", len(expiredRuns))
    return nil
}
```

### 9.2 Cleanup SQL Query

```sql
SELECT *
FROM v2_workflow_run
WHERE is_local_run = TRUE
  AND local_archive_ref IS NOT NULL
  AND local_archive_ref != ''
  AND started < NOW() - INTERVAL '24 hours';  -- configurable TTL
```

---

## 10. Upload with Retry and Progress

### 10.1 Upload Implementation

```go
func uploadArchive(ctx context.Context, client cdsclient.Interface,
    projectKey, archivePath, sha256 string) (string, error) {

    file, err := os.Open(archivePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    stat, _ := file.Stat()

    // Wrap with progress bar (if TTY)
    var reader io.Reader = file
    if isTerminal(os.Stderr) {
        bar := progressbar.NewOptions64(stat.Size(),
            progressbar.OptionSetDescription("⠋ Uploading to CDS..."),
            progressbar.OptionSetWriter(os.Stderr),
            progressbar.OptionShowBytes(true),
        )
        reader = io.TeeReader(file, bar)
    }

    // Retry with exponential backoff
    var archiveRef string
    err = retry(3, 2*time.Second, func() error {
        var uploadErr error
        archiveRef, uploadErr = client.LocalArchiveUpload(ctx, projectKey, reader, sha256, filepath.Base(archivePath))
        if uploadErr != nil {
            // Reset reader for retry
            file.Seek(0, io.SeekStart)
        }
        return uploadErr
    })

    return archiveRef, err
}
```

### 10.2 Retry Logic

```go
func retry(maxAttempts int, initialDelay time.Duration, fn func() error) error {
    var lastErr error
    delay := initialDelay

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        lastErr = fn()
        if lastErr == nil {
            return nil
        }

        if attempt < maxAttempts {
            log.Warn("attempt %d/%d failed: %v. Retrying in %s...",
                attempt, maxAttempts, lastErr, delay)
            time.Sleep(delay)
            delay *= 2 // exponential backoff
        }
    }

    return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
```

---

## 11. Implementation Plan

### Phase 1: Foundations (SDK + Database)

| Todo | Description | Files |
|------|-------------|-------|
| `sdk-types` | Define `V2WorkflowRunLocalRequest`, `LocalRunContext`; add `IsLocalRun`/`LocalArchiveRef` to `V2WorkflowRun`; add `TypeItemLocalArchive` CDN type | `sdk/v2_workflow.go`, `sdk/v2_workflow_run.go`, `sdk/cdn.go` |
| `db-migration` | Add `is_local_run` and `local_archive_ref` columns to `v2_workflow_run` | `engine/sql/api/` |
| `cdn-archive-type` | Register `TypeItemLocalArchive` in CDN type system with TTL support | `engine/cdn/` |

### Phase 2: API Endpoints

| Todo | Description | Files |
|------|-------------|-------|
| `api-archive-upload` | `POST /v2/project/{projectKey}/local/archive` — chunked upload with SHA256 verification | `engine/api/v2_local.go` |
| `api-local-run` | `POST /v2/project/{projectKey}/local/run` — create local workflow run | `engine/api/v2_local.go` |
| `api-resolve-repo` | `GET /v2/repository/resolve` — resolve git URL to CDS project | `engine/api/v2_local.go` |
| `craft-local-run` | Adapt `craftWorkflowRunV2()` for local runs (YAML override, job filter, git context) | `engine/api/v2_workflow_run_craft.go` |

### Phase 3: Worker

| Todo | Description | Files |
|------|-------------|-------|
| `worker-checkout-archive` | Modify checkout action: detect local run, download archive from CDN, extract to workspace | `engine/worker/internal/action/builtin_checkout_application.go` |
| `worker-local-context` | Propagate `LocalRunContext` through job context | `engine/worker/internal/runV2.go` |

### Phase 4: CLI

| Todo | Description | Files |
|------|-------------|-------|
| `cli-local-cmd` | `cdsctl local run` command with all flags | `cli/cdsctl/local.go` |
| `cli-archive-create` | Archive creation logic (git ls-files, tar.gz, .cdsignore, exclusions) | `cli/cdsctl/local_archive.go` |
| `cli-git-detect` | Git metadata collection and CDS project resolution | `cli/cdsctl/local_git.go` |
| `cli-log-stream` | Real-time log streaming via WebSocket | `cli/cdsctl/local_logs.go` |
| `cli-ux` | Terminal rendering (colors, spinners, progress bars, summary) | `cli/cdsctl/local_renderer.go` |

### Phase 5: Finalization

| Todo | Description | Files |
|------|-------------|-------|
| `tests-integration` | End-to-end integration tests for the full local run flow | `tests/` |
| `docs` | User documentation and API reference | `docs/` |
| `cleanup-ttl` | Background goroutine to clean up expired local archives | `engine/api/v2_local_cleanup.go` |
