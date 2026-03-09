# General Specifications — `cdsctl local run`

**Feature**: Local Execution of CDS v2 Workflows  
**Status**: Draft  
**Authors**: —  
**Last Updated**: 2026-03-09

---

## 1. Context and Motivation

### 1.1 Problem Statement

Today, CDS developers must commit and push their code changes to trigger a workflow run. This creates a slow feedback loop:

1. Developer makes changes (code or workflow YAML)
2. Developer commits and pushes to VCS
3. CDS detects the push event and triggers the workflow
4. Developer waits for the workflow to execute
5. If something fails, the developer must fix, commit, push again

This cycle is especially painful for:
- **Workflow authoring**: trial-and-error on `.cds/workflows/*.yml` files requires a commit for each iteration
- **Integration testing**: verifying that a code change passes CI requires a full push/PR cycle
- **Configuration changes**: testing variable or environment changes means polluting the commit history

### 1.2 Inspiration

Two open-source projects demonstrate the value of local workflow execution:

| Project | Description | Stars |
|---------|-------------|-------|
| [nektos/act](https://github.com/nektos/act) | Run GitHub Actions locally using Docker | 69k⭐ |
| [aws/codecatalyst-runner-cli](https://github.com/aws/codecatalyst-runner-cli) | Run AWS CodeCatalyst workflows locally | 4⭐ |

Both tools parse workflow YAML files, resolve job dependencies, and execute jobs in containers on the developer's machine.

### 1.3 Proposed Approach

Rather than executing workflows entirely on the developer's machine (which would require replicating CDS hatcheries, worker models, services, secrets, etc.), we propose a **"Remote Execution, Local Code"** model:

> The CLI archives the local repository, uploads it to the CDS server, and triggers a real workflow run — using the same hatcheries, workers, and infrastructure — but substituting the VCS checkout with the local archive.

This approach offers several advantages:
- **Full fidelity**: the execution environment is identical to production (same worker models, services, secrets, integrations)
- **No local Docker required**: no need to replicate the execution environment locally
- **Minimal server changes**: the core execution engine remains unchanged; only the code source is different
- **Security preserved**: all RBAC, secret management, and audit trails remain intact

---

## 2. Product Vision

A developer working on a CDS project should be able to test their code and/or workflow changes **before committing/pushing**, by launching a CDS workflow directly from their terminal.

The execution is **real** (same workers, same integrations, same secrets) but uses the **local code** instead of the VCS code.

### 2.1 Value Proposition

| For | Benefit |
|-----|---------|
| **Developers** | Instant feedback on code changes without polluting commit history |
| **DevOps engineers** | Test workflow modifications before merging |
| **Teams** | Reduce failed CI runs by validating locally first |
| **Platform** | Lower wasted compute from broken pushes |

### 2.2 One-Liner

> *"Think globally, run locally"* — Test your CDS workflows with local code changes before pushing.

---

## 3. Use Cases

### UC1: Fast Feedback Loop

**Actor**: Developer  
**Precondition**: Developer has local code modifications  
**Flow**:
1. Developer modifies source code files
2. Developer runs `cdsctl local run`
3. CLI archives the local repo and uploads it to CDS
4. CDS executes the workflow using the local code
5. Developer sees real-time logs in the terminal
6. Developer gets immediate SUCCESS/FAILURE feedback

**Postcondition**: Developer knows if changes pass CI before committing

### UC2: Workflow Debugging

**Actor**: DevOps Engineer  
**Precondition**: Engineer is modifying a `.cds/workflows/*.yml` file  
**Flow**:
1. Engineer edits the workflow YAML locally
2. Engineer runs `cdsctl local run`
3. CLI sends the modified workflow YAML along with the code
4. CDS uses the local workflow definition (not the VCS version)
5. Engineer validates the workflow behavior

**Postcondition**: Workflow changes are validated before merging

### UC3: Targeted Local CI/CD

**Actor**: Developer  
**Precondition**: Developer wants to run only a specific job  
**Flow**:
1. Developer runs `cdsctl local run --job unit-tests`
2. Only the specified job is executed (other jobs are skipped)
3. Developer gets feedback on the specific job

**Postcondition**: Faster feedback by running only what matters

### UC4: Variable Override

**Actor**: Developer  
**Precondition**: Developer wants to test with different configuration  
**Flow**:
1. Developer runs `cdsctl local run --var ENV=staging --var DEBUG=true`
2. CDS executes the workflow with the overridden variables
3. Developer validates behavior with the custom configuration

**Postcondition**: Developer can test different configurations without modifying workflow files

---

## 4. Scope

### 4.1 In Scope

| Area | Description |
|------|-------------|
| **Workflow format** | V2 workflows only (YAML as-code, `.cds/` directory) |
| **Execution model** | Remote execution on CDS infrastructure (hatcheries/workers) |
| **Code delivery** | Local code sent as `.tar.gz` archive |
| **Log output** | Real-time log streaming in the terminal |
| **Job selection** | Optional filtering to run specific job(s) |
| **Variables** | Override workflow variables from the command line |
| **History** | Local runs visible in CDS run history (marked as "local") |
| **Security** | Identical to normal runs (same RBAC, secrets, integrations) |

### 4.2 Out of Scope

| Area | Rationale |
|------|-----------|
| **V1 workflows** | V1 is legacy; new features target V2 only |
| **Local execution** | Would require replicating entire CDS runtime locally; too complex |
| **Git event simulation** | Trigger is always manual; simulating push/PR/tag events adds complexity without clear value |
| **Auto project creation** | CDS project and repo must already exist; auto-creation would bypass governance |

---

## 5. User Prerequisites

1. **`cdsctl` installed and authenticated** — configured with a CDS server URL and valid token
2. **CDS project exists** — the project must already be created in CDS with the repository configured
3. **Local git repository** — the user must be in a git repo containing `.cds/workflows/*.yml` files
4. **Git remote matches CDS** — the local repo's git remote must correspond to the repository registered in CDS

---

## 6. Target User Experience

### 6.1 Simple Case

```bash
$ cd my-project
$ cdsctl local run
⠋ Archiving local repository...
✓ Archive created (12.3 MB, 1,847 files)
⠋ Uploading to CDS...
✓ Uploaded. Run #42 started (local)
🔗 https://cds.example.com/project/MYPROJ/run/42

━━━ Job: build ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ▶ Step 1/3: checkout
    Using local archive (skipping git clone)
    ✓ Done (0.8s)
  ▶ Step 2/3: npm install
    added 847 packages in 12.4s
    ✓ Done (12.4s)
  ▶ Step 3/3: npm test
    Tests: 142 passed, 0 failed
    ✓ Done (8.2s)
✓ Job build: SUCCESS (21.4s)

━━━ Job: lint ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ▶ Step 1/2: checkout
    Using local archive (skipping git clone)
    ✓ Done (0.5s)
  ▶ Step 2/2: npm run lint
    ✓ Done (3.1s)
✓ Job lint: SUCCESS (3.6s)

✅ Workflow completed: SUCCESS (21.4s)
```

### 6.2 Advanced Case

```bash
# Run a specific job with variable overrides
$ cdsctl local run --workflow my-workflow --job build --var ENV=staging --var DEBUG=true

# Dry-run: validate and create archive without executing
$ cdsctl local run --dry-run

# Run without streaming logs (just get the run URL)
$ cdsctl local run --no-stream
✓ Uploaded. Run #42 started (local)
🔗 https://cds.example.com/project/MYPROJ/run/42
```

### 6.3 Error Cases

```bash
# No CDS workflow files found
$ cdsctl local run
✗ Error: No CDS workflow files found in .cds/ directory

# Multiple workflows, no --workflow flag
$ cdsctl local run
? Select a workflow:
  > build-and-test
    deploy-staging
    deploy-production

# Repository not found in CDS
$ cdsctl local run
✗ Error: Repository 'git@github.com:myorg/myrepo.git' not found in CDS.
  Use --project to specify the CDS project explicitly.

# Archive too large
$ cdsctl local run
⠋ Archiving local repository...
✗ Error: Archive size (612 MB) exceeds limit (500 MB).
  Use --exclude or .cdsignore to reduce archive size.
```

---

## 7. Command Reference

```
Usage:
  cdsctl local run [flags]

Flags:
  -w, --workflow string    Workflow name (auto-detected if only one in .cds/)
  -j, --job string         Run only this job (skip others)
  -v, --var stringArray    Variable override (KEY=VALUE), repeatable
      --branch string      Override branch name (default: current git branch)
      --dry-run            Validate workflow and create archive without executing
      --no-stream          Don't stream logs, just return the run URL
      --exclude strings    Additional glob patterns to exclude from archive
      --include-untracked  Include untracked files in archive (default: true)
      --project string     CDS project key (auto-detected from git remote)
  -h, --help               Help for local run
```

---

## 8. Acceptance Criteria

| # | Criterion | Priority |
|---|-----------|----------|
| AC1 | User can run `cdsctl local run` from a git repo and see a workflow execute with local code | Must |
| AC2 | Logs are streamed in real-time to the terminal | Must |
| AC3 | The checkout step uses the local archive instead of git clone | Must |
| AC4 | User can filter execution to a specific job with `--job` | Must |
| AC5 | User can override variables with `--var` | Must |
| AC6 | Local run is visible in CDS history, marked as "local" | Must |
| AC7 | Same RBAC rules apply as for manual runs | Must |
| AC8 | Archive respects `.gitignore` and optional `.cdsignore` | Should |
| AC9 | User can validate without executing with `--dry-run` | Should |
| AC10 | Interactive workflow selection when multiple workflows exist | Should |
| AC11 | Ctrl+C offers to cancel the remote run or just detach | Nice |
| AC12 | Archive size limit is enforced and configurable | Must |

---

## 9. Glossary

| Term | Definition |
|------|------------|
| **Local run** | A workflow run triggered from the CLI using local code, executed on CDS infrastructure |
| **Archive** | A `.tar.gz` file containing the local repository's working directory |
| **Checkout override** | The mechanism by which the worker uses the uploaded archive instead of cloning from VCS |
| **Job filter** | Selecting specific jobs to run while skipping others |
| **`.cdsignore`** | Optional file (`.gitignore` syntax) to exclude files from the local archive |
