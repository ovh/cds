---
title: cdsctl
audience: end users + maintainers
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# cdsctl

This document specifies `cdsctl` — the official command-line client
for CDS. It covers the Cobra command tree, the 27 top-level commands,
configuration loading (`~/.cdsrc`, environment variables, OS
keychain), context management, and the authentication flow.

The Go SDK (`sdk/cdsclient/`) that `cdsctl` uses under the hood is
documented in [`16-sdk.md`](./16-sdk.md).
The UI surface is in [`18-ui.md`](./18-ui.md). The authentication
backbone (drivers, sessions, JWT) lives in
[`08-auth.md`](./08-auth.md).

Source code anchors. `cdsctl` lives under `cli/cdsctl/` (root at
`cli/cdsctl/main.go`, one Go file per top-level command). Internal
helpers (the `CDSContext` struct, keychain integration) live under
`cli/cdsctl/internal/`.

## 1. Scope

**In scope** — The `cdsctl` Cobra command tree (27 top-level
commands); global flags; configuration loading (`~/.cdsrc`,
environment variables, OS keychain integration); context switching
between multiple CDS installations; authentication flow against
`/auth/driver` and `/auth/consumer/{driver}/signin`; the
`experimental` subcommand for v2 surfaces.

**Out of scope** — Go SDK contract and HTTP layer (see
[`16-sdk.md`](./16-sdk.md)); gRPC plugins (see
[`17-plugins.md`](./17-plugins.md)); UI client (see
[`18-ui.md`](./18-ui.md)); auth drivers and session model (see
[`08-auth.md`](./08-auth.md)); RBAC enforcement (see
[`09-rbac.md`](./09-rbac.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [cdsctl architecture](#3-cdsctl-architecture)
4. [Command catalogue](#4-command-catalogue)
5. [The `experimental` subcommand](#5-the-experimental-subcommand)
6. [Configuration sources](#6-configuration-sources)
7. [Context management](#7-context-management)
8. [Authentication flow](#8-authentication-flow)
9. [`cdsctl workflow` reference](#9-cdsctl-workflow-reference)
10. [Cross-spec pointers](#10-cross-spec-pointers)

## 3. cdsctl architecture

`cdsctl` is built entirely on top of Spf13 Cobra
(`cli/cdsctl/main.go`). Each top-level command groups one CDS domain
(project, workflow, application, …). The `PersistentPreRun` hook in
`main.go` loads the configuration, instantiates the cdsclient, and
attaches it to the command context so every leaf handler can call the
API without setup boilerplate.

### 3.1 Global flags

| Flag | Purpose |
| --- | --- |
| `-c, --context` | Switch context (multi-installation support) |
| `-f, --file` | Override the config file path |
| `-n, --no-interactive` | Skip prompts; expect all inputs as flags |
| `--verbose` | Enable debug logging |
| `--insecure` | Allow self-signed TLS certificates |

### 3.2 Command lifecycle

1. **PersistentPreRun** — loads config, builds client, attaches to
   the command context.
2. **Leaf handler** — calls the SDK through the attached client.
3. **Output rendering** — most commands respect a `--format`
   parameter (`human` default, `json`, `yaml`).

## 4. Command catalogue

The 27 top-level commands span the entire CDS surface. Each lives in
`cli/cdsctl/<command>.go`.

| Command | File | Purpose |
| --- | --- | --- |
| `action` | `cli/cdsctl/action.go` | List / show / import / export / doc; builtin action helpers |
| `admin` | `cli/cdsctl/admin.go` | Database migrations, services, CDN admin, repositories admin, hooks admin, organization, integration models, maintenance, metadata, plugins, curl helper, features, workflows, users, groups |
| `application` | `cli/cdsctl/application.go` | v1 application CRUD + variables + keys |
| `consumer` | `cli/cdsctl/consumer.go` | Auth consumer CRUD + regen |
| `context` | `cli/cdsctl/context.go` | Multi-installation context switch |
| `doc` | `cli/cdsctl/doc.go` | Hidden: generate markdown for this catalogue |
| `encrypt` | `cli/cdsctl/encrypt.go` | Standalone encrypt / decrypt utility |
| `environment` | `cli/cdsctl/environment.go` | v1 environment CRUD + variables + keys |
| `events` | `cli/cdsctl/events.go` | Stream events from the websocket |
| `experimental` | `cli/cdsctl/experimental.go` | v2 commands: project, template, RBAC, workflow v2, region, organization, notification |
| `group` | `cli/cdsctl/group.go` | Group CRUD + member management |
| `health` | `cli/cdsctl/health.go` | Service health check |
| `login` | `cli/cdsctl/login.go` | Driver pick + signin + signup + password reset |
| `mcp` | `cli/cdsctl/mcp.go` | Model / plugin discovery for tooling |
| `pipeline` | `cli/cdsctl/pipeline.go` | v1 pipeline CRUD |
| `project` | `cli/cdsctl/project.go` | Project CRUD + notification + repository + VCS + integration + concurrency + key + variable |
| `queue` | `cli/cdsctl/queue.go` | Worker job queue inspection |
| `reset` | `cli/cdsctl/reset.go` | Reset session state |
| `session` | `cli/cdsctl/session.go` | Session management |
| `signup` | `cli/cdsctl/signup.go` | Account creation |
| `template` | `cli/cdsctl/template.go` | v1 workflow template lifecycle |
| `tools` | `cli/cdsctl/tools.go` | Misc tooling utilities |
| `update` | `cli/cdsctl/update.go` | Self-update |
| `user` / `usr` | `cli/cdsctl/user.go` | User CRUD + GPG keys + external links + contacts |
| `version` | `cli/cdsctl/version.go` | Show client + server versions |
| `worker` | `cli/cdsctl/worker.go` | Worker CRUD + worker model CRUD |
| `workflow` | `cli/cdsctl/workflow.go` | v1 workflow init / list / show / run / stop / export / import / pull / push / transform-as-code + labels + artifacts + logs + results |

## 5. The `experimental` subcommand

V2 commands live under `experimental` to signal they should not be
considered a stable surface yet (the flag will be dropped once v1
deprecation completes). The subcommand groups:

| Subcommand | Concern |
| --- | --- |
| `project` | V2 project operations (variable sets, run retention, repositories, concurrencies, notifications) |
| `template` | V2 workflow template lifecycle |
| `rbac` | RBAC bundle import / list / get / delete |
| `workflow` | V2 workflow run search + per-run / per-job access |
| `region` | Region CRUD |
| `organization` | Organization CRUD |
| `notification` | V2 notification configuration |
| `plugin` | Plugin registration |

## 6. Configuration sources

The CLI relies on **three layered credential sources** evaluated in
priority order: environment variables, the `~/.cdsrc` configuration
file, and the OS keychain.

### 6.1 Environment variables

| Variable | Effect |
| --- | --- |
| `CDS_API_URL` | Override the API host |
| `CDS_TOKEN` | Builtin consumer authentication token (long-lived) |
| `CDS_SESSION_TOKEN` | Session JWT (short-lived) |
| `CDS_USER` | Username for signin prompts |
| `CDS_FILE` | Override `~/.cdsrc` location |
| `CDS_CONTEXT` | Override the current context name |
| `CDS_HTTP_MAX_RETRY` | Retry count (default 2) |
| `CDS_CDN_URL` | Override the CDN URL |
| `CDS_VERBOSE` | Enable debug output |
| `CDS_INSECURE` | Skip TLS verification |

### 6.2 `~/.cdsrc`

The config file holds named contexts so a single binary can target
several CDS installations:

```yaml
current: prod
contexts:
  prod:
    host: https://api.cds.example.com
    token: <builtin-token>
    insecureSkipVerifyTLS: false
  staging:
    host: https://staging.cds.example.com
    session: <session-jwt>
```

The `CDSContext` struct (defined under `cli/cdsctl/internal/`)
captures: `Host`, `Token`, `Session`, `InsecureSkipVerifyTLS`. Helper
functions `GetContext` and `GetCurrentContext` resolve the active
context.

### 6.3 OS keychain

When neither env var nor file holds a usable token, cdsctl reaches
into the OS keychain — Keychain Access on macOS, `libsecret` on
Linux. The keyring entry name is the context name. `cdsctl login`
writes credentials to the keyring; `cdsctl session` and `cdsctl reset`
clean them.

## 7. Context management

`cdsctl context` (and the `-c` global flag) lets one binary target
multiple CDS installations. The current context is recorded in the
`current:` field of `~/.cdsrc`; switching it persists the new value.

```sh
cdsctl context add prod --host https://api.cds.example.com
cdsctl context add staging --host https://staging.cds.example.com
cdsctl context list
cdsctl context use prod
cdsctl -c staging project list   # one-shot override
```

Each context carries its own credentials in `~/.cdsrc` and / or in
the keychain entry named after the context.

## 8. Authentication flow

`cdsctl login` (`cli/cdsctl/login.go`):

1. Lists `/auth/driver` to discover the API's enabled drivers.
2. Prompts the user to pick a driver (or honours `--driver`).
3. Walks the OAuth / SSO flow against
   `/auth/consumer/{driver}/signin`.
4. Stores the resulting session JWT in `~/.cdsrc` and / or the
   keychain.

The full auth-driver model (drivers, scopes, sessions, JWT) is in
[`08-auth.md`](./08-auth.md).

### 8.1 Token kinds

The CLI handles two token kinds (also documented in
[`08-auth.md`](./08-auth.md#5-consumer-types-and-consumer-shape)):

| Kind | Source | Lifetime | Storage |
| --- | --- | --- | --- |
| Session JWT | `cdsctl login` | Days (driver-dependent) | `session:` field in `~/.cdsrc` or keychain |
| Builtin consumer token | `cdsctl consumer new` | Configurable, typically long-lived | `token:` field in `~/.cdsrc` or env `CDS_TOKEN` |

CI environments typically use the long-lived builtin token; humans
use the session JWT.

## 9. `cdsctl workflow` reference

The v1 workflow command is the most-used surface of the CLI. Its
subcommands:

| Subcommand | Purpose |
| --- | --- |
| `init` | Bootstrap a workflow from a template repository |
| `list` | List workflows |
| `show` | Show one workflow |
| `run` | Trigger a manual run |
| `stop` | Stop a running workflow |
| `export` | Export YAML |
| `import` | Import YAML |
| `pull` | Pull every YAML referenced by the workflow |
| `push` | Push every YAML referenced by the workflow |
| `transform-as-code` | Migrate a non-ascode workflow to ascode |
| `label` | Add / remove labels |
| `artifacts` | List artefacts from a run |
| `logs` | Stream logs from a run |
| `results` | List run results |

V2 workflow operations live under `cdsctl experimental workflow` (see
[section 5](#5-the-experimental-subcommand)).

## 10. Cross-spec pointers

- Go SDK contract, HTTP layer, factories → [`16-sdk.md`](./16-sdk.md)
- gRPC plugins → [`17-plugins.md`](./17-plugins.md)
- UI and websocket → [`18-ui.md`](./18-ui.md)
- Microservices and request lifecycle →
  [`01-architecture.md`](./01-architecture.md)
- Auth drivers, sessions, scopes, JWT, link →
  [`08-auth.md`](./08-auth.md)
- RBAC enforcement → [`09-rbac.md`](./09-rbac.md)
- Workflow v1 model → [`03-workflow-v1.md`](./03-workflow-v1.md)
- Workflow v2 schema → [`04-workflow-v2.md`](./04-workflow-v2.md)
- Ascode entities → [`05-ascode-entities.md`](./05-ascode-entities.md)
- V1 run engine → [`07a-run-engine-v1.md`](./07a-run-engine-v1.md)
- V2 run engine → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- Hatcheries → [`10-hatcheries.md`](./10-hatcheries.md)
- Workers → [`11-workers.md`](./11-workers.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
