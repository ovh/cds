---
title: VCS providers
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# VCS providers

This document specifies how CDS talks to source-control providers
(GitHub, GitLab, Bitbucket Server, Bitbucket Cloud, Gerrit, Gitea,
Forgejo). It documents the VCS microservice, the repositories
microservice, the per-provider clients, commit-status reporting, the
user-link system that ties a CDS user to an external identity, and the
multi-VCS workflow pattern.

Integrations (Kafka, RabbitMQ, OpenStack, Artifactory, AWS, …) are a
separate concern documented in [`14-integrations.md`](./14-integrations.md).
The configuration side (how a project attaches a VCS server) lives in
[`02-project-and-tenancy.md`](./02-project-and-tenancy.md). The hook
side (per-provider webhook parsing) lives in
[`06b-hooks-v2.md`](./06b-hooks-v2.md) for v2 and
[`06a-hooks-v1.md`](./06a-hooks-v1.md) for v1. The auth driver that
lets a user sign in *as* a GitHub user is in
[`08-auth.md`](./08-auth.md).

Source code anchors. VCS service in `engine/vcs/` (entry point in
`engine/vcs/vcs.go`, router in `engine/vcs/vcs_router.go`, auth
middleware in `engine/vcs/vcs_auth.go`, per-provider packages
`engine/vcs/{github,gitlab,bitbucketserver,bitbucketcloud,gerrit,gitea,forgejo}/`).
Public types `VCSServer`, `VCSAuthorizedClient`, `VCSAuth`, `VCSRepo`,
`VCSBranch`, `VCSTag`, `VCSCommit`, `VCSPullRequest`, `VCSHook`,
`VCSContent`, `VCSCommitStatus`, `VCSBuildStatus`, `VCSRelease` in
`sdk/vcs.go`. Repositories service in `engine/repositories/`.
Operation types in `sdk/repositories_operation.go`. Link drivers
under `engine/api/link/` (`LinkDriver` in `engine/api/link/link.go`,
GitHub driver in `engine/api/link/github/`). `UserLink` in
`sdk/user_link.go`.

## 1. Scope

**In scope** — The VCS microservice (router, middleware,
configuration); the seven provider implementations with their auth
model; the `VCSServer` / `VCSAuthorizedClient` contract;
commit-status reporting; the repositories microservice (operations
queue, processor, vacuum cleaner, local filesystem cache);
`Operation`, its lifecycle (`Pending` → `Processing` → `Done` /
`Error`), and its three flavours (checkout, push, load-files); the
link system and its sole production driver (GitHub); multi-VCS
workflow patterns using cross-project ascode references.

**Out of scope** — Project integrations (`IntegrationModel`, the
built-in integration catalogue, integration types matrix) — see
[`14-integrations.md`](./14-integrations.md). Webhook parsing (see
[`06a-hooks-v1.md`](./06a-hooks-v1.md) and
[`06b-hooks-v2.md`](./06b-hooks-v2.md)). Auth drivers and user
sign-in (see [`08-auth.md`](./08-auth.md)). Project-level VCS
configuration (see [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)).
RBAC enforcement on routes (see [`09-rbac.md`](./09-rbac.md)). V2 hook
indexing (see [`06b-hooks-v2.md`](./06b-hooks-v2.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [VCS service architecture](#3-vcs-service-architecture)
4. [VCS routes](#4-vcs-routes)
5. [Provider contract](#5-provider-contract)
6. [VCS auth middleware](#6-vcs-auth-middleware)
7. [Per-provider implementations](#7-per-provider-implementations)
8. [Commit-status reporting](#8-commit-status-reporting)
9. [Repositories service](#9-repositories-service)
10. [Operations](#10-operations)
11. [Link system](#11-link-system)
12. [Multi-VCS workflows](#12-multi-vcs-workflows)
13. [Cross-spec pointers](#13-cross-spec-pointers)

## 3. VCS service architecture

The VCS service is the stateless adapter between CDS and the seven
supported source-control providers (`Serve` in `engine/vcs/vcs.go`).
There is **one VCS service per CDS installation**, but it serves **N
VCS servers** — the API selects which one to query via the `{name}`
path parameter. The factory `getConsumer` (`engine/vcs/vcs.go`) reads
the per-request `VCSAuth` from the context and returns the matching
`VCSServer` implementation.

The `Service` struct (`engine/vcs/types.go`) carries:

| Component | Role |
| --- | --- |
| `Cfg` (`Configuration`) | Name, HTTP router config, base URL, API service config, Redis cache TTL + connection, optional outbound `ProxyWebhook` |
| `Router` | Exposes the uniform per-repository surface |
| `Cache` (Redis) | Deduplicates expensive provider calls (rate-limit-friendly) and stores commit-status fingerprints to avoid double-writes |
| `UI.HTTP.URL` | Used in commit-status target URLs |

## 4. VCS routes

The router (`engine/vcs/vcs_router.go`) is uniform across providers:

| Route | Methods | Handler |
| --- | --- | --- |
| `/vcs/{name}/repos` | GET | `getReposHandler` |
| `/vcs/{name}/repos/{owner}/{repo}` | GET | `getRepoHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/branches` | GET | `getBranchesHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/branches/` | GET | `getBranchHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/branches/commits` | GET | `getCommitsHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/tags` | GET | `getTagsHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/tags/{tagName}` | GET | `getTagHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/commits` | GET | `getCommitsBetweenRefsHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/commits/{commit}` | GET | `getCommitHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/commits/{commit}/statuses` | GET | `getCommitStatusHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/commits/{commit}/insight/{insightKey}` | POST | `postInsightHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/contents/{filePath}` | GET | `getListContentsHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/content/{filePath}` | GET | `getFileContentHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/archive` | POST | `archiveHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/pullrequests` | GET, POST | `getPullRequestsHandler`, `postPullRequestsHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/pullrequests/comments` | POST | `postPullRequestCommentHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/pullrequests/{id}` | GET | `getPullRequestHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/events` | GET, POST | `getEventsHandler`, `postFilterEventsHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/hooks` | GET, POST, PUT, DELETE | `(get|post|put|delete)HookHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/releases` | POST | `postReleaseHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/releases/{release}/artifacts/{artifactName}` | POST | `postUploadReleaseFileHandler` |
| `/vcs/{name}/repos/{owner}/{repo}/forks` | GET | `getListForks` |
| `/vcs/{name}/repos/{owner}/{repo}/search/pullrequest` | GET | `SearchPullRequestHandler` |
| `/vcs/{name}/status` | POST | `postStatusHandler` |

## 5. Provider contract

The two interfaces in `sdk/vcs.go` are what every provider must
satisfy:

| Interface | Role |
| --- | --- |
| `VCSServer` | Factory `GetAuthorizedClient(ctx, VCSAuth) (VCSAuthorizedClient, error)` |
| `VCSAuthorizedClient` | Wide surface used by the rest of the platform |

`VCSAuthorizedClientCommon` (`sdk/vcs.go`) groups its methods by
domain:

| Group | Methods |
| --- | --- |
| Repositories | `Repos`, `RepoByFullname`, `ListForks` |
| Refs | `Branches`, `Branch`, `Tags` |
| Commits | `Commits`, `CommitsBetweenRefs`, `Commit` |
| Pull requests | `PullRequest`, `PullRequests`, `PullRequestCreate`, `PullRequestComment`, `SearchPullRequest` |
| Hooks | `CreateHook`, `UpdateHook`, `GetHook`, `DeleteHook` |
| Statuses | `SetStatus`, `ListStatuses` |
| Events | `GetEvents` |
| Content | `GetArchive`, `ListContent`, `GetContent` |
| Releases | `Release`, `UploadReleaseFile` |
| Insights | `CreateInsightReport` |

Provider-agnostic value types defined in `sdk/vcs.go`: `VCSRepo`,
`VCSBranch`, `VCSTag`, `VCSCommit`, `VCSPullRequest`, `VCSHook`,
`VCSContent`, `VCSCommitStatus`, `VCSBuildStatus`, `VCSRelease`,
`GerritChangeEvent`. Every provider maps these onto its native API.

## 6. VCS auth middleware

Every VCS HTTP call carries the credentials in headers. The
middleware `engine/vcs/vcs_auth.go` decodes them and writes them into
the context under typed keys (`contextKeyVCSURL`,
`contextKeyVCSURLApi`, `contextKeyVCSType`, `contextKeyVCSUsername`,
`contextKeyVCSToken`).

The headers are base64-encoded so binary token contents survive the
HTTP layer. Gerrit adds three extra headers for its SSH side channel
(declared in `sdk/vcs.go`):

```
X-CDS-VCS-SSH-USERNAME
X-CDS-VCS-SSH-PORT
X-CDS-VCS-SSH-PRIVATE-KEY
```

`getVCSAuth` rehydrates the headers into an `sdk.VCSAuth` value that
`getConsumer` inspects to pick the right per-provider implementation.

## 7. Per-provider implementations

| Provider | Directory | Library | Auth | Notes |
| --- | --- | --- | --- | --- |
| GitHub | `engine/vcs/github/` | stdlib `net/http` + cache | Bearer token | Supports GitHub Enterprise via `githubURL` + `githubAPIURL`; rate-limit telemetry exposed |
| GitLab | `engine/vcs/gitlab/` | `github.com/xanzy/go-gitlab` | Personal access token | 60s HTTP timeout; `disableStatus` flag toggles commit-status pushes |
| Bitbucket Server | `engine/vcs/bitbucketserver/` | stdlib | OAuth 1.0a or PAT | Path format `project/slug` |
| Bitbucket Cloud | `engine/vcs/bitbucketcloud/` | stdlib | OAuth 2.0 + app password | Fixed API base URL (`api.bitbucket.org/2.0`) |
| Gerrit | `engine/vcs/gerrit/` | `andygrunwald/go-gerrit` | HTTP basic + SSH | Reviews instead of PRs; SSH used by the hooks-side poller |
| Gitea | `engine/vcs/gitea/` | `code.gitea.io/sdk/gitea` | PAT + basic auth | Path format `owner/slug` |
| Forgejo | `engine/vcs/forgejo/` | custom HTTP client | PAT + basic auth | Custom client (not a fork of the Gitea SDK), 60s timeout |

### 7.1 GitHub

The `githubClient` (`engine/vcs/github/github.go`) carries
`GitHubURL`, `GitHubAPIURL`, `ClientID`, `OAuthToken`, the cache, and
the per-request `(username, token)` pair. `GetAuthorizedClient` calls
`RateLimit(ctx)` so every cycle logs the rate-limit headers
(`vcs_github_ratelimit_remaining`, `vcs_github_ratelimit_limit`,
`vcs_github_ratelimit_reset`).

Statuses (`engine/vcs/github/client_status.go`) hit
`POST /repos/{owner}/{repo}/statuses/{commit}` and accept `success`,
`failure`, `pending`, `skipped`, `cancelled`. The status is cached
with a 61-minute TTL to deduplicate identical writes.

### 7.2 GitLab

`engine/vcs/gitlab/gitlab.go`. Uses `xanzy/go-gitlab`. Statuses
(`engine/vcs/gitlab/client_status.go`) map the CDS status enum:

| CDS status | GitLab status |
| --- | --- |
| `Waiting` / `Checking` / `Building` / `Pending` | `Pending` |
| `Success` | `Success` |
| `Fail` / `Unknown` | `Failed` |
| `Disabled` / `Cancelled` / `NeverBuilt` / `Skipped` | `Canceled` |

The implementation deduplicates by comparing `TargetURL`, `Status`,
`Ref`, `SHA`, `Name`, `Description` before writing.

### 7.3 Bitbucket Server

`engine/vcs/bitbucketserver/bitbucketserver.go`. Path format
`project/slug`. Authentication uses OAuth 1.0a with an HMAC-SHA-256
signed request or a personal access token in
`Authorization: Bearer`.

### 7.4 Bitbucket Cloud

`engine/vcs/bitbucketcloud/bitbucketcloud.go`. Fixed cloud API base
URL. Authentication is basic auth with `username:appPassword`. Status
states: `PENDING`, `SUCCESSFUL`, `FAILED`, `INPROGRESS`, `STOPPED`.

### 7.5 Gerrit

`engine/vcs/gerrit/gerrit.go`. HTTP via `andygrunwald/go-gerrit`; SSH
for streaming events (handled in the hooks service — see
[`06a-hooks-v1.md`](./06a-hooks-v1.md#9-gerrit-ssh-listener-v1-path)). Basic auth
applied via `client.Authentication.SetBasicAuth(username, token)`.
Insights are used in place of GitHub-style commit statuses.

### 7.6 Gitea

`engine/vcs/gitea/gitea.go`. Uses `code.gitea.io/sdk/gitea`. Statuses
(`engine/vcs/gitea/client_status.go`):
`POST /repos/{owner}/{repo}/statuses/{sha}` with values `pending`,
`success`, `failure`. UI context concatenates name and description.

### 7.7 Forgejo

`engine/vcs/forgejo/forgejo.go`. Forgejo is a Gitea fork, but the CDS
implementation uses a **custom HTTP client**
(`engine/vcs/forgejo/http_client.go`) rather than the upstream Gitea
SDK so the implementation can track API drift independently.
Authentication is basic auth with username and token.

## 8. Commit-status reporting

`POST /vcs/{name}/status` (handler `postStatusHandler` in
`engine/vcs/vcs_handlers.go`). The body is `sdk.VCSBuildStatus`
(`sdk/vcs.go`):

| Field | Purpose |
| --- | --- |
| `Title` | Short summary shown by the provider |
| `Description` | Longer description |
| `URLCDS` | Target URL pointing back at the run in the CDS UI |
| `Context` | Provider-side "context" identifying the status row (so subsequent updates replace it) |
| `Status` | One of the CDS run statuses, mapped per-provider |
| `RepositoryFullname` | `owner/name` |
| `GitHash` | Commit SHA |
| `GerritChange` | Optional Gerrit-specific payload |

The API calls this endpoint at run-engine transitions — when a v2 job
enters `Building`, `Success`, `Fail`, `Stopped`, or `Cancelled`. The
14 SDK statuses (`sdk/build.go`) are translated per-provider through
each `client_status.go`. Operators can disable status pushes globally
with the `disableStatus` flag (GitLab) or per-route by routing
through a different VCS name.

`ListStatuses` is the read side: it fetches what currently exists on
the commit so the API can avoid double-writes.

## 9. Repositories service

The `repositories` service is the long-running git-operation worker
(`Serve` in `engine/repositories/repositories.go`). It caches
checkouts on disk and serves them to the API when an analysis needs
file content beyond what the VCS service can stream.

`Serve` starts three internal goroutines:

- `processor` — drives the operation queue.
- `vacuumCleaner` — drops stale repository clones.
- `computeCacheSize` — keeps the metric in sync.

### 9.1 Routes

`engine/repositories/repositories_router.go`:

| Route | Method | Handler |
| --- | --- | --- |
| `/operations` | POST | `postOperationHandler` |
| `/operations/{uuid}` | GET | `getOperationsHandler` |
| `/admin/cache` | GET | `GetLocalCacheHandler` |
| `/admin/cache` | DELETE | `ClearLocalCacheHandler` |

### 9.2 Configuration

`engine/repositories/types.go` (`Configuration`): `Name`, `HTTP`,
`URL`, `Basedir` (root for clones), `RepositoriesRetention` (Go
duration, default `"24h"`), `Cache` (TTL + Redis config), `API`.

## 10. Operations

`Operation` (`sdk/repositories_operation.go`) is the work unit. It
carries: `UUID`, `VCSServer`, `RepoFullName`, `URL`,
`RepositoryStrategy` (auth credentials decrypted by the API before
send so the repositories service receives plaintext), `Setup`
(`OperationSetup`), `LoadFiles` (`OperationLoadFiles`), `Status`,
`Error` (`OperationError`), `Date`, `User` (`OperationUser`).

### 10.1 Status lifecycle

| Constant | Value | Meaning |
| --- | --- | --- |
| `OperationStatusPending` | 0 | Queued |
| `OperationStatusProcessing` | 1 | A processor took the operation |
| `OperationStatusDone` | 2 | Success |
| `OperationStatusError` | 3 | Failure (see `Operation.Error`) |

### 10.2 Operation flavours

The processor dispatches by inspecting the `Setup` and `LoadFiles`
fields:

| Flavour | Processor | File |
| --- | --- | --- |
| Checkout (with optional signature check) | `processCheckout` | `engine/repositories/processor_checkout.go` |
| Push | `processPush` | `engine/repositories/processor_push.go` |
| Load files (glob) | `processLoadFiles` | `engine/repositories/processor_loadfiles.go` |
| Signature check | `checkSignature` | inlined inside checkout |

Each flavour mutates the local clone, optionally reads files into
`LoadFiles`, and updates the status. The API polls
`/operations/{uuid}` until the status becomes terminal.

`OperationSetup` (`sdk/repositories_operation.go`):

- **Checkout** — branch, tag, or commit; optional signature
  verification flag; optional semver computation.
- **Push** — `FromBranch`, `ToBranch`, commit message, optional PR
  link.

### 10.3 Local cache

The service keeps clones at `Basedir/{repoID}`. `RepositoriesRetention`
(default `24h`) caps how long an idle clone is kept before
`vacuumCleaner` reclaims it.

Two levels of cache:

- **Redis** — operations and locks (`engine/repositories/dao.go`),
  10-minute TTL locks to prevent concurrent races on the same repo.
- **In-memory** — `gocache.Cache` with 10-minute TTL for metadata
  that does not survive restarts but speeds up the hot path.

## 11. Link system

The link system binds a CDS user to an external identity so RBAC
rules and audit trails can map between the two worlds. The contract
is in `engine/api/link/link.go` (`LinkDriver`); drivers live under
`engine/api/link/<driver>/`.

### 11.1 Contract

```
type LinkDriver interface {
    GetUserInfo(context.Context, AuthConsumerSigninRequest) (AuthDriverUserInfo, error)
    GetDriver() Driver
}
```

The registry `api.LinkDrivers map[AuthConsumerType]link.LinkDriver`
is populated at boot in `engine/api/api.go`.

### 11.2 Production drivers

Only one driver is wired in production: **GitHub**
(`LinkGithubDriver` in `engine/api/link/github/github.go`). GitLab,
Bitbucket Server, and Forgejo have their `UserLink.Type` values
reserved (`sdk/token.go`) but no linking driver is shipped — they
rely on the corresponding auth driver to populate the link
transitively.

### 11.3 Persisted shape

`UserLink` (`sdk/user_link.go`):

| Field | Purpose |
| --- | --- |
| `ID` | UUID |
| `AuthentifiedUserID` | CDS user UUID |
| `Type` | `github`, `gitlab`, `bitbucketserver`, `forgejo` |
| `ExternalID` | External user ID |
| `Username` | External username |
| `Created` | Creation timestamp |

### 11.4 Routes

| Route | Method | Handler | Purpose |
| --- | --- | --- | --- |
| `/link/driver` | GET | `getLinkDriversHandler` | List available linking drivers |
| `/link/{consumerType}/ask` | POST | `postAskLinkExternalUserWithCDSHandler` | Get redirect URL for OAuth |
| `/link/{consumerType}` | POST | `postLinkExternalUserWithCDSHandler` | Exchange code, insert `UserLink` |

The full flow (`engine/api/link.go`) is documented in
[`08-auth.md`](./08-auth.md).

## 12. Multi-VCS workflows

A v2 ascode workflow can reference resources across VCS servers and
projects. Two mechanisms enable this:

### 12.1 Cross-project ascode references

`uses: PROJECT_KEY/vcs-name/repo/action@ref` (or `worker-model@ref`,
or `workflow-template@ref`) resolves through the `EntityFinder` (see
[`05-ascode-entities.md`](./05-ascode-entities.md)). Each segment can
target a different project, VCS server, or repository — the call is
fully qualified.

### 12.2 Cross-VCS operations

The `Operation.VCSServer` field (`sdk/repositories_operation.go`) is
per-operation, so the repositories service routinely fans out to
multiple VCS servers. The DAO keys operations by UUID so collisions
are impossible; the local cache keys clones by repo URL hash
(`r.ID()`).

### 12.3 Library project

A project nominated as the "library" (configured per installation)
hosts shared actions, worker models, and templates. References of the
form `library/<entity-name>` short-circuit to this project. The
lookup is performed by `unsafeSearchEntityFromLibrary`
(`engine/api/entity_search.go`).

## 13. Cross-spec pointers

- Integrations (`IntegrationModel`, integration catalogue, types matrix) → [`14-integrations.md`](./14-integrations.md)
- Microservices, request lifecycle → [`01-architecture.md`](./01-architecture.md)
- Project-level VCS configuration → [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)
- Workflow v1 (legacy applications + `RepositoryStrategy`) → [`03-workflow-v1.md`](./03-workflow-v1.md)
- Workflow v2 ascode schema → [`04-workflow-v2.md`](./04-workflow-v2.md)
- Ascode entities, signature verification, cross-project resolution → [`05-ascode-entities.md`](./05-ascode-entities.md)
- V1 per-provider webhook parsing → [`06a-hooks-v1.md`](./06a-hooks-v1.md)
- V2 per-provider webhook parsing → [`06b-hooks-v2.md`](./06b-hooks-v2.md)
- V1 run engine → [`07a-run-engine-v1.md`](./07a-run-engine-v1.md)
- V2 run engine, commit-status writes at status transitions → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- Auth drivers, link system → [`08-auth.md`](./08-auth.md)
- RBAC for VCS users → [`09-rbac.md`](./09-rbac.md)
- Hatcheries → [`10-hatcheries.md`](./10-hatcheries.md)
- Workers (`actions/checkout` consumer) → [`11-workers.md`](./11-workers.md)
- CDN → [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md)
- cdsctl → [`15-cli.md`](./15-cli.md)
- Go SDK → [`16-sdk.md`](./16-sdk.md)
- gRPC plugins → [`17-plugins.md`](./17-plugins.md)
- UI → [`18-ui.md`](./18-ui.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
