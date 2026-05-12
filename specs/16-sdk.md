---
title: Go SDK
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# Go SDK

This document specifies the Go SDK (`sdk/cdsclient/`) used by every
CDS service, hatchery, worker, and external consumer to talk to the
API. It covers the umbrella `Interface`, the ~30 sub-interfaces, the
factories, the HTTP layer (`RequestJSON`, `Stream`, websocket,
auto-refresh, retry), and the events websocket client.

The gRPC plugin protocol used to extend the worker (action and
integration plugins) is a separate concern documented in
[`17-plugins.md`](./17-plugins.md). The CLI (`cdsctl`) that builds on
top of the SDK is in [`15-cli.md`](./15-cli.md). The UI surface is in
[`18-ui.md`](./18-ui.md). The authentication backbone (drivers,
sessions, JWT) lives in [`08-auth.md`](./08-auth.md).

Source code anchors. The Go SDK lives under `sdk/cdsclient/`
(umbrella `Interface` in `sdk/cdsclient/interface.go`, HTTP layer in
`sdk/cdsclient/http.go`, client constructors in
`sdk/cdsclient/client.go`, config in `sdk/cdsclient/config.go`, mock
generation under `sdk/cdsclient/mock_cdsclient/`).

## 1. Scope

**In scope** — The `cdsclient.Interface` taxonomy (~30
sub-interfaces); the HTTP layer (`RequestJSON`, `Stream`, retry, auth
header injection); service / hatchery / worker factories
(`NewServiceClient`, `NewHatcheryServiceClient`, `NewWorker`,
`NewWorkerV2`); the events websocket client
(`WebsocketEventsListen`); mock generation.

**Out of scope** — gRPC plugin protocol and built-in plugin
catalogues (see [`17-plugins.md`](./17-plugins.md)); cdsctl (see
[`15-cli.md`](./15-cli.md)); UI and engine-side websocket server (see
[`18-ui.md`](./18-ui.md)); auth drivers and session model (see
[`08-auth.md`](./08-auth.md)); RBAC enforcement (see
[`09-rbac.md`](./09-rbac.md)); hatchery contract (see
[`10-hatcheries.md`](./10-hatcheries.md)); worker internals (see
[`11-workers.md`](./11-workers.md)); webhook routing (see
[`06a-hooks-v1.md`](./06a-hooks-v1.md) and
[`06b-hooks-v2.md`](./06b-hooks-v2.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [Go SDK overview](#3-go-sdk-overview)
4. [Sub-interfaces](#4-sub-interfaces)
5. [SDK factories](#5-sdk-factories)
6. [Config](#6-config)
7. [HTTP layer](#7-http-layer)
8. [Worker client](#8-worker-client)
9. [Events websocket client](#9-events-websocket-client)
10. [Mock generation](#10-mock-generation)
11. [Cross-spec pointers](#11-cross-spec-pointers)

## 3. Go SDK overview

`cdsclient.Interface` (`sdk/cdsclient/interface.go`) is the umbrella
type — a union of focused sub-interfaces. The pattern lets each
consumer import only what it needs (services pull `Raw` +
`AuthClient`; the worker pulls `WorkerClient`).
`cdsclient.New(config Config)` returns the aggregate.

The SDK is the canonical Go consumer of the CDS REST + websocket API.
Both `cdsctl` and every internal service (api, cdn, hooks, vcs,
repositories, elasticsearch, hatcheries, workers) reach the API
through it.

## 4. Sub-interfaces

| Interface | Domain |
| --- | --- |
| `Raw` | Low-level HTTP (`DoJSON`, `Stream`, `RequestWebsocket`) |
| `AuthClient` | Driver list, signin, signout |
| `ActionClient` | Action CRUD, builtin actions |
| `Admin` | Database migrations, service inspection, feature flags |
| `ApplicationClient` | v1 application CRUD + variables + keys |
| `EnvironmentClient` | v1 environment CRUD + variables + keys |
| `EventsClient` | `WebsocketEventsListen` |
| `ExportImportInterface` | YAML import / export for pipelines, workflows, applications |
| `GroupClient` | Group CRUD + member operations |
| `PipelineClient` | v1 pipeline CRUD |
| `ProjectClient` / `ProjectClientV2` | Project CRUD + v2 sub-resources (notifications, repositories, VCS, integrations, concurrencies, keys, variables, run retention) |
| `ProjectKeysClient` / `ProjectVariablesClient` | Detailed project key / variable management with encryption helpers |
| `V2QueueClient` / `QueueClient` | v2 and v1 job queues |
| `UserClient` | User CRUD + GPG keys |
| `V2WorkerClient` / `WorkerClient` | v2 and v1 worker lifecycle |
| `CDNClient` | CDN upload / download / stream |
| `WorkflowClient` | v1 workflow CRUD + runs + node operations + hooks |
| `WorkflowV2Client` | v2 workflow run search + per-run / per-job access + stop |
| `HookClient` | Hook resolution, scheduler listing, VCS event polling, repository event sign-key retrieval, insight reports |
| `HatcheryClient` | Hatchery CRUD + token regen |
| `HatcheryServiceClient` | Heartbeat, worker model fetch, job take / release, entity fetch, v2 worker listing, CDN config |
| `RBACClient` | RBAC import + delete + get + list + per-user / per-group permission |
| `RegionClient`, `OrganizationClient` | Region and organisation CRUD |
| `MaintenanceClient` | Toggle maintenance mode |
| `IntegrationClient` | Integration model CRUD |
| `DownloadClient` | Generic binary download |
| `RepositoriesManagerInterface` | Repository manager listing |
| `TemplateClient` / `TemplateV2Client` | v1 and v2 workflow template operations |
| `WebsocketClient` | Generic websocket helper |
| `ServiceClient` | Service configuration retrieval |
| `GRPCPluginsClient` / `GRPCPluginsV2Client` | Plugin registration + discovery |

Per-domain implementation files live alongside the interface
definitions as `client_<domain>.go` (e.g. `client_action.go`,
`client_workflow.go`, `client_workflowv2.go`).

## 5. SDK factories

Beyond `New(Config)`, the SDK ships specialised constructors for
service-to-service traffic. The same builtin-signin contract is
documented in [`01-architecture.md`](./01-architecture.md).

### 5.1 `New(Config)`

The user-facing factory. Returns a `cdsclient.Interface` configured
with a session JWT or a builtin token. Used by `cdsctl` and any
third-party Go consumer.

### 5.2 `NewServiceClient`

`sdk/cdsclient/client.go`. Used by `cdn`, `hooks`, `vcs`,
`repositories`, `elasticsearch`. Flow:

1. Read `ServiceConfig.Token` (long-lived builtin token).
2. POST `/auth/consumer/builtin/signin` with the token + a service
   registration payload.
3. Receive the session JWT, the service registration row, and the
   issuer's RSA public key.
4. Retry up to 60 times (one per minute) on 401 to tolerate API
   restarts during deployment.

### 5.3 `NewHatcheryServiceClient`

Same file. Hatcheries call this instead. The endpoint is
`/v2/auth/consumer/hatchery/signin`; the response carries the
hatchery ID, the public key, and the assigned region (see
[`10-hatcheries.md`](./10-hatcheries.md)).

### 5.4 `ServiceConfig`

In `sdk/cdsclient/config.go`. Carries: `Host`, `Token`,
`RequestSecondsTimeout`, `InsecureSkipVerifyTLS`, an optional `Hook`
(test hook), `Verbose`, and an optional `TokenV2` for hatchery v2
builtin tokens.

## 6. Config

The SDK is constructed from `Config` (`sdk/cdsclient/config.go`)
carrying: `Host`, `CDNHost`, `User`, `SessionToken` (JWT),
`BuiltinConsumerAuthenticationToken` (long-lived), `Verbose`,
`Retry`, `InsecureSkipVerifyTLS`, and an optional `*sync.Mutex`. The
helper `HasValidSessionToken` validates the JWT shape and expiry so
callers can short-circuit a refresh.

## 7. HTTP layer

`sdk/cdsclient/http.go` exposes five tiers:

| Method | Purpose |
| --- | --- |
| `RequestJSON(ctx, method, path, in, out, mods…)` | JSON CRUD |
| Verb shortcuts: `GetJSON`, `PostJSON`, `PutJSON`, `DeleteJSON` | Verb-specific helpers |
| `Request(ctx, method, path, body, mods…)` | Raw bytes |
| `Stream(ctx, client, method, path, body, mods…)` | Streaming with auto-retry |
| `StreamNoRetry` | Streaming without retry |
| `RequestWebsocket(ctx, goRoutines, path, send, recv, errs)` | Bidirectional websocket |

### 7.1 Auth header injection

When the target host matches `c.config.Host` and the path is not on
the `signinRouteRegexp` allowlist, the client injects
`Authorization: Bearer <SessionToken>` automatically. Signin paths
are exempt so the auth handshake itself does not require a JWT.

### 7.2 Auto-refresh

`StreamNoRetry` validates the session token before sending the
request. If the JWT is expired and the config has a
`BuiltinConsumerAuthenticationToken`, the client triggers an
auto-signin (`AuthConsumerSignin` for user / builtin consumers,
`AuthConsumerHatcherySigninV2` for hatcheries), swaps the JWT, and
retries.

### 7.3 Retry

`Stream` retries every transient failure up to `Config.Retry` times.
Non-retriable conditions: 4xx (except 429), and bodies that cannot be
re-read (the client retries only when `body` is an `io.ReadSeeker`
or a small enough buffer).

### 7.4 `RequestModifier`

A `RequestModifier` is a small function that decorates an outgoing
`http.Request`. Common helpers (`sdk/cdsclient/http.go`):
`SetHeader(key, value)`, `WithQueryParameter(key, value)`. Used by
callers that need to inject one-off headers (e.g.
`X-CDS-WORKER-SIGNATURE` for uploads).

## 8. Worker client

Two factories, one per generation, both in
`sdk/cdsclient/client.go`:

| Factory | Purpose |
| --- | --- |
| `NewWorker(endpoint, name, httpClient)` | v1 worker |
| `NewWorkerV2(endpoint, name, httpClient)` | v2 worker |

### 8.1 V1 worker calls

`sdk/cdsclient/client_worker.go`:

| Method | Endpoint |
| --- | --- |
| `WorkerRegister` | `POST /auth/consumer/worker/signin` |
| `WorkerUnregister` | `POST /auth/consumer/worker/signout` |
| `WorkerList` | `GET /worker` |
| `WorkerGet` | `GET /worker/{name}` |
| `WorkerRefresh` | `POST /worker/refresh` |
| `WorkerDisable` | `POST /worker/{id}/disable` |
| `WorkerSetStatus` | `POST /worker/waiting` |

### 8.2 V2 worker calls

`sdk/cdsclient/client_worker_v2.go`:

| Method | Endpoint |
| --- | --- |
| `V2WorkerRegister` | `POST /v2/queue/{region}/job/{jobID}/worker/signin` |
| `V2WorkerUnregister` | `POST /v2/queue/{region}/job/{jobID}/worker/signout` |
| `V2WorkerRefresh` | `POST /v2/queue/{region}/job/{jobID}/worker/refresh` |
| `V2WorkerList` | `GET /v2/worker` |
| `V2WorkerGet` | `GET /v2/worker/{name}` |
| `V2WorkerProjectGetKey` | `GET /v2/queue/{region}/job/{jobID}/key/{keyName}` |
| `V2QueueWorkerTakeJob` | `POST /v2/queue/{region}/job/{jobID}/worker/take` |

Signin uses a bootstrap Bearer (the worker token signed by the
hatchery — see [`10-hatcheries.md`](./10-hatcheries.md)). The
response header `X-CDS-JWT` returns the session token to use for
every subsequent call.

## 9. Events websocket client

`sdk/cdsclient/client_events.go` declares `EventsClient` with a
single high-level operation:
`WebsocketEventsListen(ctx, goRoutines, chanFiltersToSend, chanEventsReceived, chanErrorsReceived)`.
The implementation opens a `gorilla/websocket` connection to `/ws`,
sends `[]WebsocketFilter` (project keys, workflow names, run
identifiers) on `chanFiltersToSend`, and emits `WebsocketEvent`
records on `chanEventsReceived`. Errors land on
`chanErrorsReceived` without panicking. On disconnection the client
retries with a 1-second back-off until the parent context is
cancelled.

`WebsocketEvent` carries an `EventType`, a `Timestamp`, and a `Data`
payload.

The v2 events surface is larger and lives at `/v2/ws`; the UI
consumes it directly (see [`18-ui.md`](./18-ui.md)).

## 10. Mock generation

`sdk/cdsclient/mock_cdsclient/` holds the mocks generated by
`mockgen`. Two files cover the surface:

| File | Purpose |
| --- | --- |
| `interface_mock.go` | Mock for `cdsclient.Interface` and all sub-interfaces |
| `http_mock.go` | Mocks for the underlying HTTP transport |

Mocks are used pervasively in API and service tests to avoid hitting
a real CDS during unit runs.

## 11. Cross-spec pointers

- gRPC plugin protocol and built-in plugin catalogues → [`17-plugins.md`](./17-plugins.md)
- cdsctl → [`15-cli.md`](./15-cli.md)
- UI and engine websocket → [`18-ui.md`](./18-ui.md)
- Microservices, inter-service auth, request lifecycle → [`01-architecture.md`](./01-architecture.md)
- Auth drivers, sessions, scopes, JWT → [`08-auth.md`](./08-auth.md)
- RBAC v2 → [`09-rbac.md`](./09-rbac.md)
- Hatchery contract → [`10-hatcheries.md`](./10-hatcheries.md)
- Worker contract → [`11-workers.md`](./11-workers.md)
- Workflow v1 model → [`03-workflow-v1.md`](./03-workflow-v1.md)
- Workflow v2 model → [`04-workflow-v2.md`](./04-workflow-v2.md)
- Ascode entities → [`05-ascode-entities.md`](./05-ascode-entities.md)
- V1 run engine → [`07a-run-engine-v1.md`](./07a-run-engine-v1.md)
- V2 run engine → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
