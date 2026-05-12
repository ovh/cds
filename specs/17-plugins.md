---
title: gRPC plugins (action and integration)
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# gRPC plugins (action and integration)

This document specifies the **gRPC plugin protocol** that CDS uses to
extend the platform with action plugins (invoked by workers) and
integration plugins (invoked by the engine). It documents the
lifecycle, the `ActionPlugin` and `IntegrationPlugin` service
definitions, the plugin storage model, and the catalogues of
built-in plugins shipped under `contrib/`.

The worker-side invocation flow — how a step picks up a plugin
client, dials its socket, and ships logs — is documented in
[`11-workers.md`](./11-workers.md). The Go SDK that hosts plugin
clients is in [`16-sdk.md`](./16-sdk.md). Integration capabilities
(deployment, artifact-manager, …) are documented in
[`14-integrations.md`](./14-integrations.md).

Source code anchors. Plugin protocol in
`sdk/grpcplugin/grpcplugin.go` and `sdk/plugin.go`. Action plugin
proto and Go bindings in `sdk/grpcplugin/actionplugin/`. Integration
plugin proto and bindings in `sdk/grpcplugin/integrationplugin/`.
Plugin storage handlers: `engine/api/grpc_plugin.go` (v1) and
`engine/api/v2_plugin.go` (v2). Built-in action plugins in
`contrib/grpcplugins/action/`; integration plugins in
`contrib/integrations/`.

## 1. Scope

**In scope** — Plugin lifecycle (start, register, invoke, stop); the
`ActionPlugin` gRPC service (manifest, run, stream, worker HTTP
port, stop); the `IntegrationPlugin` gRPC service; plugin storage
(`GRPCPlugin.Binaries`, `GRPCPlugin.Type`); the built-in action
plugin catalogue (22 plugins); the built-in integration plugin
catalogue (6 systems).

**Out of scope** — Worker-side invocation flow (which plugin is
called when a step runs) — see [`11-workers.md`](./11-workers.md);
Go SDK contract and HTTP layer (see [`16-sdk.md`](./16-sdk.md));
integration capability buckets and configuration (see
[`14-integrations.md`](./14-integrations.md)); RBAC for plugin
management (see [`09-rbac.md`](./09-rbac.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [Plugin protocol and lifecycle](#3-plugin-protocol-and-lifecycle)
4. [`ActionPlugin` service](#4-actionplugin-service)
5. [Built-in action plugin catalogue](#5-built-in-action-plugin-catalogue)
6. [`IntegrationPlugin` service](#6-integrationplugin-service)
7. [Integration plugin catalogue](#7-integration-plugin-catalogue)
8. [Plugin storage](#8-plugin-storage)
9. [Cross-spec pointers](#9-cross-spec-pointers)

## 3. Plugin protocol and lifecycle

The plugin SDK (`sdk/grpcplugin/grpcplugin.go`) is the shared
protocol that lets actions and integrations be authored in any
language. The CDS canonical implementations are in Go but the wire
contract is gRPC over a per-plugin Unix-domain socket.

| Symbol | Purpose |
| --- | --- |
| `Plugin` interface | Contract every plugin entry point must satisfy |
| `Common` value | Embedded base implementation reused by both action and integration plugins |
| `StartPlugin` | Boot routine: open a Unix socket, register the gRPC service, signal readiness on stdout |

### 3.1 Lifecycle

1. **Start** (`createGRPCPluginSocket` in
   `engine/worker/internal/plugin/plugin.go`) — the host (worker for
   action plugins, engine for integration plugins) executes the
   plugin binary, reads its stdout for `is ready to accept new
   connection\n` and extracts the socket path from the line.
2. **Register** — the plugin opens a gRPC server on the Unix socket
   and registers the service via reflection.
3. **Invoke** — the host connects as a gRPC client and issues the
   service RPCs.
4. **Stop** — the host calls `Stop()` over gRPC, then waits for the
   plugin process to exit.

The Unix socket path follows
`$HOME_CDS_PLUGINS/grpcplugin-socket-{UUID}.sock`.

### 3.2 Stdout / stderr forwarding

`enablePluginLogger` reads the plugin's stdout / stderr
line-by-line and forwards each line to the worker via
`c.w.SendLog` (action plugins) so plugins that just print to stdout
get their output into CDN without writing the gRPC stream.

### 3.3 Worker callback port

After connection, the host calls `WorkerHTTPPort` with the loopback
HTTP port the worker exposes for `/v2/output`, `/v2/result`, and
`/v2/result/synchronize`. The plugin uses this port to register run
results and step outputs without going through gRPC.

## 4. `ActionPlugin` service

`sdk/grpcplugin/actionplugin/`:

| File | Purpose |
| --- | --- |
| `actionplugin.proto` | gRPC service definition |
| `actionplugin.pb.go` | Generated Go bindings |
| `actionplugin.go` | Manifest / query / result helpers |

The service `ActionPlugin` exposes:

```protobuf
service ActionPlugin {
  rpc Manifest(Empty) returns (ActionPluginManifest);
  rpc Run(ActionQuery) returns (ActionResult);
  rpc Stream(ActionQuery) returns (stream StreamResult);
  rpc WorkerHTTPPort(WorkerHTTPPortQuery) returns (Empty);
  rpc Stop(Empty) returns (Empty);
}
```

| RPC | Purpose |
| --- | --- |
| `Manifest` | Plugin metadata (name, version, parameters, requirements) |
| `Run` | Execute the action with `ActionQuery` (parameters + secrets); synchronous |
| `Stream` | Same as `Run`, but ships `StreamResult { status, details, logs }` as the plugin works |
| `WorkerHTTPPort` | Worker tells the plugin where to find the loopback callback server |
| `Stop` | Graceful shutdown |

`ActionQuery` carries the parameters; `ActionResult` carries the
terminal status. `StreamResult` carries intermediate progress
messages.

## 5. Built-in action plugin catalogue

Plugins shipped with CDS live under `contrib/grpcplugins/action/`.
Each directory has a `main.go`. The `script` plugin is the one the
v2 worker invokes whenever a step uses `run:` rather than `uses:`.

| Plugin | Purpose |
| --- | --- |
| `addRunResult` | Create a run-result from an artifact |
| `artifactoryPromote` | Promote artifacts in Artifactory |
| `artifactoryRelease` | Cut an Artifactory release |
| `cache`, `cacheRestore`, `cacheSave` | Folder caching across runs |
| `checkout` | `git clone` the workflow's repository |
| `debianPush` | Push a `.deb` package to a Debian repo |
| `deployArsenal` | Deploy through OVH Arsenal |
| `dockerPush` | Push a Docker image to a registry |
| `downloadArtifact` | Download a previously-uploaded artifact |
| `helmPush` | Push a Helm chart |
| `junit` | Upload and parse a JUnit report |
| `keyInstall` | Install an SSH or GPG key in the worker filesystem |
| `plugin-archive` | Compress / decompress an archive |
| `plugin-arsenal-delete-alternative` | Remove an Arsenal alternative |
| `plugin-artifactory-release-bundle-create` | Create and sign a release bundle |
| `plugin-artifactory-release-bundle-distribute` | Distribute a bundle |
| `plugin-tmpl` | Render a Go template into a file |
| `pythonPush` | Publish a Python package |
| `script` | Run a shell / interpreter script |
| `uploadArtifact` | Upload an artifact to CDN |

## 6. `IntegrationPlugin` service

`sdk/grpcplugin/integrationplugin/`:

| File | Purpose |
| --- | --- |
| `integrationplugin.proto` | gRPC service definition |
| `integrationplugin.pb.go` | Generated Go bindings |
| `integrationplugin.go` | Manifest / query / result helpers |
| `example/main.go` | Reference implementation |

The service `IntegrationPlugin`:

```protobuf
service IntegrationPlugin {
  rpc Manifest(Empty) returns (IntegrationPluginManifest);
  rpc Run(RunQuery) returns (RunResult);
  rpc WorkerHTTPPort(WorkerHTTPPortQuery) returns (Empty);
  rpc Stop(Empty) returns (Empty);
}
```

`RunResult` carries `status`, `details`, and a `map[string]string`
of outputs that the worker promotes to step outputs. Integration
plugins are invoked by the engine for some callbacks (deployment,
artifact-manager flows) — see
[`14-integrations.md`](./14-integrations.md) for the capability
mapping.

## 7. Integration plugin catalogue

Integration plugins live under `contrib/integrations/`. They are
shipped per-integration-system and typically expose several
callbacks:

| Integration | Directory | Callbacks |
| --- | --- | --- |
| Arsenal | `contrib/integrations/arsenal/arsenal-deployment-plugin/` | deployment |
| Artifactory | `contrib/integrations/artifactory/artifactory-{build-info,upload-artifact,download-artifact,promote,release}-plugin/` | artifact-manager (5 callbacks) |
| Kubernetes | `contrib/integrations/kubernetes/plugin-kubernetes-deployment/` | deployment |
| Hello | `contrib/integrations/hello/hello-deployment-plugin/` | demo / example |
| OpenStack | `contrib/integrations/openstack/` | storage |
| AWS | `contrib/integrations/aws/` | storage |

Configuration files (`arsenal.yml`, `kubernetes.yml`, etc.) declare
the integration manifest seeded by the API in
[`02-project-and-tenancy.md`](./02-project-and-tenancy.md). The
runtime contract for these capabilities is in
[`14-integrations.md`](./14-integrations.md).

## 8. Plugin storage

A plugin is registered through the API
(`engine/api/grpc_plugin.go` v1; `engine/api/v2_plugin.go` v2). The
binary itself is stored in the object store and downloaded by the
worker on demand; the `Binaries` slice on `sdk.GRPCPlugin`
(`sdk/plugin.go`) lists one entry per `(OS, Arch)` combination with
`Cmd`, `Args`, and `Entrypoints`.

`GRPCPlugin.Type` (`sdk/plugin.go`) is one of:

- `action` — an action plugin.
- `integration-deploy_application`,
  `integration-upload_artifact`,
  `integration-download_artifact`,
  `integration-build_info`,
  `integration-release`,
  `integration-promote` — typed integration callbacks.

The typed integration variants let the engine resolve "which plugin
should I invoke for this callback on this integration" without
inspecting the plugin manifest at runtime.

## 9. Cross-spec pointers

- Worker-side plugin invocation flow → [`11-workers.md`](./11-workers.md)
- Go SDK contract and HTTP layer → [`16-sdk.md`](./16-sdk.md)
- Integration capability buckets and configuration → [`14-integrations.md`](./14-integrations.md)
- Microservices, request lifecycle → [`01-architecture.md`](./01-architecture.md)
- Workflow v2 schema (step `uses:` references) → [`04-workflow-v2.md`](./04-workflow-v2.md)
- Ascode entities (worker-model and action storage) → [`05-ascode-entities.md`](./05-ascode-entities.md)
- V2 run engine (plugin context) → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- RBAC v2 (`manage-plugin` global role) → [`09-rbac.md`](./09-rbac.md)
- Hatcheries (spawn workers that run plugins) → [`10-hatcheries.md`](./10-hatcheries.md)
- CDN (plugins write to CDN via the worker) → [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
