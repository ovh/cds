---
title: Project integrations
audience: maintainers + advanced users
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# Project integrations

This document specifies how CDS extends a project with external
systems through **integrations**: the abstract `IntegrationModel`,
the five built-in integration models (Kafka, RabbitMQ, OpenStack,
Artifactory, AWS) and their capability flags, and the five
`IntegrationType*` categories that govern how the run engine exposes
an integration to a worker.

The VCS adapter layer (GitHub, GitLab, Bitbucket, …) is a separate
concern documented in [`13-vcs.md`](./13-vcs.md). The configuration
side (how a project attaches an integration) lives in
[`02-project-and-tenancy.md`](./02-project-and-tenancy.md). The
plugins that implement integration callbacks live in
[`17-plugins.md`](./17-plugins.md).

Source code anchors. Integrations under `engine/api/integration/`;
`IntegrationModel` and `ProjectIntegration` in `sdk/integration.go`;
built-in seeds in `engine/api/integration/builtin.go`;
`IntegrationType*` constants in `sdk/integration.go`.

## 1. Scope

**In scope** — `IntegrationModel` shape; built-in catalogue (Kafka,
RabbitMQ, OpenStack, Artifactory, AWS); per-model capability flags
(`Hook`, `Storage`, `Deployment`, `Compute`, `Event`,
`ArtifactManager`); the five `IntegrationType*` categories;
capability-driven dispatch at run-time.

**Out of scope** — VCS service, providers, commit-status, repositories
service, link system (see [`13-vcs.md`](./13-vcs.md)). Project-level
integration configuration (see [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)).
RBAC enforcement on integration management (see
[`09-rbac.md`](./09-rbac.md)). The gRPC plugin protocol that
integration plugins speak (see [`17-plugins.md`](./17-plugins.md)).
Kafka / RabbitMQ as inbound hook sources (see
[`06a-hooks-v1.md`](./06a-hooks-v1.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [Integration model](#3-integration-model)
4. [Built-in catalogue](#4-built-in-catalogue)
5. [Integration types matrix](#5-integration-types-matrix)
6. [Capability dispatch](#6-capability-dispatch)
7. [Cross-spec pointers](#7-cross-spec-pointers)

## 3. Integration model

### 3.1 `IntegrationModel`

`IntegrationModel` (`sdk/integration.go`) is the abstract description
of an integration. It carries `ID`, `Name`, `Author`, `Identifier`,
`Icon`, `DefaultConfig`, `AdditionalDefaultConfig`,
`PublicConfigurations`, `Disabled`, `Public`, plus a set of
capability booleans declaring which buckets the model implements:
`Hook`, `Storage`, `Deployment`, `Compute`, `Event`,
`ArtifactManager`.

The model is the **template**; a project instantiates it as a
`ProjectIntegration` (also in `sdk/integration.go`) that carries the
filled-in configuration values. The split between model and instance
is what lets the platform ship one Artifactory model and then have
N projects, each with its own URL and credentials.

### 3.2 DAO

`engine/api/integration/dao_model.go`:

| Function | Purpose |
| --- | --- |
| `LoadModels` | Read every `IntegrationModel` |
| `InsertModel` | Register a new model (admin API or plugin-driven) |

Built-in models are inserted at API boot via `CreateBuiltinModels`
(`engine/api/integration/builtin.go`); custom models can be added by
the admin API or by gRPC plugins (see
[`17-plugins.md`](./17-plugins.md)).

## 4. Built-in catalogue

Five models are seeded at boot from
`engine/api/integration/builtin.go`:

| Model | Source | Hook | Storage | Deployment | Event | Artifact mgr |
| --- | --- | --- | --- | --- | --- | --- |
| Kafka | `sdk.KafkaIntegration` | yes | no | no | yes | no |
| RabbitMQ | `sdk.RabbitMQIntegration` | yes | no | no | no | no |
| OpenStack | `sdk.OpenstackIntegration` | no | yes | no | no | no |
| Artifactory | `sdk.ArtifactoryIntegration` | no | no | no | no | yes |
| AWS | `sdk.AWSIntegration` | no | yes | no | no | no |

Each integration ships a default configuration whose `Type` field
declares the variable kind (text, password, region, …) — see
[`02-project-and-tenancy.md`](./02-project-and-tenancy.md) for the
configuration model.

### 4.1 Kafka

Used as both an event sink (CDS pushes events to a Kafka topic) and
an inbound hook source (v1 only — see
[`06a-hooks-v1.md`](./06a-hooks-v1.md#8-kafka-and-rabbitmq-listeners)).
The integration carries broker URL, topic, credentials, optional
TLS.

### 4.2 RabbitMQ

Same shape as Kafka but limited to inbound hooks (v1 only). Carries
exchange name, routing key, credentials.

### 4.3 OpenStack

A storage integration. Connects to OpenStack Swift / object storage
for CDN object storage. Carries auth URL, region, tenant, user,
password, container name.

### 4.4 Artifactory

An artifact-manager integration. Carries URL, token, repository
prefixes for each artifact type (Docker, Helm, Maven, …). The run
engine exposes the `artifact_manager` callbacks to workers when a
workflow declares the integration.

### 4.5 AWS

A storage integration. Carries access key, secret, region, bucket
name. Used by CDN's S3 storage unit when configured.

## 5. Integration types matrix

The five `IntegrationType*` constants in `sdk/integration.go`:

| Constant | Purpose |
| --- | --- |
| `IntegrationTypeEvent` | Stream CDS events to an external system (Kafka topic, etc.) |
| `IntegrationTypeCompute` | Provide compute resources (reserved for future use) |
| `IntegrationTypeHook` | Receive triggers from the external system (Kafka, RabbitMQ) |
| `IntegrationTypeStorage` | Store CDN items (S3-compatible, Swift, OpenStack object) |
| `IntegrationTypeDeployment` | Drive a deployment plugin (Arsenal, Kubernetes) |

The `ArtifactManager` flag is **orthogonal**: it indicates that the
integration speaks the artifact-management protocol (Artifactory
today, more in the future). A single integration can carry several
capability flags — Kafka, for instance, is both `Hook` and `Event`.

## 6. Capability dispatch

When a v2 workflow declares `integrations: [my-integration]`, the
run engine inspects the model's capability flags and exposes only
the relevant subset of operations to the worker.

| Capability | Surface exposed to worker |
| --- | --- |
| `Deployment` | `deployApplication` step (drives the matching deployment plugin — see [`17-plugins.md`](./17-plugins.md#7-integration-plugin-catalogue)) |
| `ArtifactManager` | `pushBuildInfo`, `promote`, `release` callbacks |
| `Event` | Kafka client provisioned at job startup; CDS publishes its event stream to the configured topic |
| `Hook` | Read-only from the worker's perspective; the hooks service is the producer (see [`06a-hooks-v1.md`](./06a-hooks-v1.md)) |
| `Storage` | Consumed by the CDN service, not the worker — see [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md) |

The run engine writes the capability-derived context into the job's
`integrations.*` namespace (see
[`07b-run-engine-v2.md`](./07b-run-engine-v2.md)); workers consume it
through the gRPC plugin invocation flow (see
[`11-workers.md`](./11-workers.md#9-plugin-invocation-flow)).

## 7. Cross-spec pointers

- VCS providers, repositories service, link → [`13-vcs.md`](./13-vcs.md)
- Project-level integration configuration → [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)
- gRPC plugin protocol and built-in plugin catalogues → [`17-plugins.md`](./17-plugins.md)
- Kafka / RabbitMQ as v1 hook sources → [`06a-hooks-v1.md`](./06a-hooks-v1.md)
- Workflow v2 ascode schema (`integrations:` field) → [`04-workflow-v2.md`](./04-workflow-v2.md)
- V2 run engine (`integrations.*` context) → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- Workers (consume the runtime integration context) → [`11-workers.md`](./11-workers.md)
- CDN storage units (consume `IntegrationTypeStorage`) → [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md)
- Auth drivers → [`08-auth.md`](./08-auth.md)
- RBAC → [`09-rbac.md`](./09-rbac.md)
- Microservices → [`01-architecture.md`](./01-architecture.md)
- Glossary → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
