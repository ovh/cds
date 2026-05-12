---
title: Authentication
audience: maintainers
status: draft
version: spec-v1
last-reviewed: 2026-05-12
---

# Authentication

This document specifies how CDS proves who is making a request:
authentication drivers (user-facing plus the builtin / hatchery /
worker pseudo-drivers), the consumer model, scopes, sessions, JWT
signing with multi-key rotation, MFA, and the user-link system. It
also covers the legacy v1 group-based ACL — a v1-only authorization
model that lives alongside auth and does **not** interoperate with
the v2 RBAC subsystem.

The v2 RBAC subsystem (`RBAC` bundles, the seven scope tables, role
constants, glob matching, route rule helpers, the `rbacMiddleware`)
is a separate concern and lives in [`09-rbac.md`](./09-rbac.md).
The request-side surface — the order of middlewares, how the JWT
lands in the context, how scopes and RBAC compose — is documented in
[`01-architecture.md`](./01-architecture.md).

Source code anchors. Authentication framework lives under
`engine/api/authentication/` (one sub-package per driver); the
user-link surface lives under `engine/api/link/`. Public consumer /
session / driver types live in `sdk/token.go`. The user object lives
in `sdk/user.go`; v1 group permissions in `sdk/group.go`.

## 1. Scope

**In scope** — Auth drivers (local, LDAP, GitHub, GitLab, OIDC,
Corporate SSO, builtin, hatchery, worker); `AuthConsumerType` values;
`AuthConsumer`, `AuthUserConsumer`, `AuthHatcheryConsumer`; scopes
(`AuthConsumerScope*` constants); the `AuthSession` lifecycle; JWT
signing and the multi-key rotation strategy; MFA enforcement points;
the user-link system (`UserLink`); v1 group-based permission levels
(`PermissionRead = 4`, `PermissionReadExecute = 5`,
`PermissionReadWriteExecute = 7`) as the v1-only authorization model.

**Out of scope** — V2 RBAC (see [`09-rbac.md`](./09-rbac.md)); HTTP
middleware ordering and context propagation (see
[`01-architecture.md`](./01-architecture.md)); the project /
organisation / group data model (see
[`02-project-and-tenancy.md`](./02-project-and-tenancy.md));
per-service consumer types in inter-service traffic (see
[`01-architecture.md`](./01-architecture.md)); ascode RBAC enforcement
during repository analysis (see
[`05-ascode-entities.md`](./05-ascode-entities.md)); hatchery
worker-token issuance (see [`10-hatcheries.md`](./10-hatcheries.md)).

## 2. Table of contents

1. [Scope](#1-scope)
2. [Table of contents](#2-table-of-contents)
3. [Authentication framework](#3-authentication-framework)
4. [Drivers](#4-drivers)
5. [Consumer types and consumer shape](#5-consumer-types-and-consumer-shape)
6. [Scopes](#6-scopes)
7. [Sessions](#7-sessions)
8. [JWT signing and key rotation](#8-jwt-signing-and-key-rotation)
9. [MFA](#9-mfa)
10. [User link to external accounts](#10-user-link-to-external-accounts)
11. [User object](#11-user-object)
12. [HTTP middleware](#12-http-middleware)
13. [V1 legacy group permissions](#13-v1-legacy-group-permissions)
14. [Cross-spec pointers](#14-cross-spec-pointers)

## 3. Authentication framework

Every authentication driver implements the `AuthDriver` interface
(defined in `sdk/token.go`). Drivers live under
`engine/api/authentication/<driver>/`. The contract carries four
methods:

| Method | Purpose |
| --- | --- |
| `GetManifest` | Returns an `AuthDriverManifest` reporting the driver's consumer type, whether signup is disabled, and whether MFA is supported |
| `GetSessionDuration` | Returns how long an issued session is valid |
| `GetUserInfo` | Performs the handshake with the external system (OAuth callback, LDAP bind, SSO assertion, etc.) and returns a normalised `AuthDriverUserInfo` record. Takes an `AuthConsumerSigninRequest` |
| `GetDriver` | Returns the underlying `Driver` value |

The API maintains two registries of drivers, populated at boot:

- `AuthenticationDrivers` — drives sign-in. Keyed by
  `AuthConsumerType`.
- `LinkDrivers` — links an existing CDS user to an external account
  (see [section 10](#10-user-link-to-external-accounts)). Same key,
  different values.

## 4. Drivers

CDS ships several drivers. Most are user-facing; three (builtin,
hatchery, worker) are infrastructure pseudo-drivers used by `cdsctl`,
hatcheries, and workers respectively.

| Driver | Consumer type | Package | Session duration | Signup disabled | MFA |
| --- | --- | --- | --- | --- | --- |
| Local | `ConsumerLocal` | `engine/api/authentication/local/` | 30 days | configurable | no |
| LDAP | `ConsumerLDAP` | `engine/api/authentication/ldap/` | 30 days | configurable | no |
| GitHub | `ConsumerGithub` | `engine/api/authentication/github/` | 30 days | configurable | no |
| GitLab | `ConsumerGitlab` | `engine/api/authentication/gitlab/` | 30 days | configurable | no |
| OIDC | `ConsumerOIDC` | `engine/api/authentication/oidc/` | 30 days | configurable | no |
| Corporate SSO | `ConsumerCorporateSSO` | `engine/api/authentication/corpsso/` | 24 h | no | dynamic (`Config.Drivers.CorporateSSO.MFASupportEnabled`) |
| Builtin | `ConsumerBuiltin` | `engine/api/authentication/builtin/` | 1 h | yes | no |
| Hatchery | `ConsumerHatchery` | `engine/api/authentication/hatchery/` | 1 h | yes | no |
| Worker | (depends) | `engine/api/authentication/worker/` | passed at creation | n/a | no |

Two additional consumer types — `ConsumerBitbucketServer` and
`ConsumerForgejo` — exist as **link targets** for external account
binding but do not provide a sign-in driver: users sign in through
another driver and then link their Bitbucket Server or Forgejo
identity. Two test fixtures also exist (`ConsumerTest`,
`ConsumerTest2`).

### 4.1 Sign-in routes

| Route | Method | Auth |
| --- | --- | --- |
| `/auth/driver` | GET | none |
| `/auth/me` | GET | required |
| `/auth/scope` | GET | none |
| `/auth/consumer/local/signup` | POST | none |
| `/auth/consumer/local/signin` | POST | none |
| `/auth/consumer/local/verify` | POST | none |
| `/auth/consumer/local/askReset` | POST | none |
| `/auth/consumer/local/reset` | POST | none |
| `/auth/consumer/builtin/signin` | POST | none |
| `/auth/consumer/worker/signin` | POST | none |
| `/auth/consumer/worker/signout` | POST | required |
| `/auth/consumer/{consumerType}/askSignin` | GET | none |
| `/auth/consumer/{consumerType}/signin` | POST | optional |
| `/auth/consumer/{consumerType}/detach` | POST | required |
| `/auth/consumer/signout` | POST | required |
| `/v2/auth/consumer/hatchery/signin` | POST | none (see [`01-architecture.md`](./01-architecture.md)) |

Local routes are wired through `postAuthLocalSignupHandler`,
`postAuthLocalSigninHandler`, `postAuthLocalVerifyHandler`,
`postAuthLocalAskResetHandler`, `postAuthLocalResetHandler`. The
generic driver routes are `getAuthAskSigninHandler`,
`postAuthSigninHandler`, `postAuthDetachHandler`,
`postAuthSignoutHandler`. Worker signin lands on
`postRegisterWorkerHandler`; worker signout on
`postUnregisterWorkerHandler`.

## 5. Consumer types and consumer shape

### 5.1 Consumer types

The `AuthConsumerType` constants are defined in `sdk/token.go`:

| Constant | Value | Used for |
| --- | --- | --- |
| `ConsumerBuiltin` | `builtin` | Personal access tokens, services (API, CDN, hooks, VCS, repositories, ElasticSearch) |
| `ConsumerLocal` | `local` | Local signup |
| `ConsumerLDAP` | `ldap` | LDAP signin |
| `ConsumerCorporateSSO` | `corporate-sso` | SSO with optional MFA |
| `ConsumerGithub` | `github` | OAuth + link target |
| `ConsumerBitbucketServer` | `bitbucketserver` | Link target only |
| `ConsumerForgejo` | `forgejo` | Link target only |
| `ConsumerGitlab` | `gitlab` | OAuth + link target |
| `ConsumerOIDC` | `openid-connect` | OAuth / OIDC |
| `ConsumerHatchery` | `hatchery` | Hatchery service identity |
| `ConsumerTest` | `futurama` | Test fixture |
| `ConsumerTest2` | `planet-express` | Test fixture |

### 5.2 `AuthConsumer` (base)

The base `AuthConsumer` (in `sdk/token.go`) carries the common
identity surface: a UUID, name and description, the consumer type,
an optional parent ID (set when the consumer was minted by another),
the creation timestamp, a deprecated issued-at timestamp, a disabled
flag, an `AuthConsumerWarnings` audit list, the last authentication
timestamp, and the validity-period sequence
(`AuthConsumerValidityPeriods`, see
[section 5.5](#55-validity-periods)).

### 5.3 `AuthUserConsumer`

A user-bound consumer (`AuthUserConsumer`) embeds the base consumer
plus user-specific data (`AuthUserConsumerData`): the authenticated
user ID, an opaque driver-specific data blob (`AuthConsumerData`),
the list of group IDs (with an optional invalid-groups list to track
removed memberships), the scope details (`AuthConsumerScopeDetails`,
see [section 6](#6-scopes)), and — when the consumer represents a
service — the service name, type, region, and the
`ignore-job-with-no-region` flag.

Aggregates loaded on demand: the underlying `AuthentifiedUser`, the
`Groups`, the optional `Service` record, the optional `Worker`
record.

### 5.4 `AuthHatcheryConsumer`

A hatchery-bound consumer (`AuthHatcheryConsumer`) embeds the base
consumer plus hatchery-specific data (`AuthConsumerHatcheryData`):
the hatchery ID.

### 5.5 Validity periods

A consumer can be revoked and re-issued without recreating its
identity. The validity-period sequence
(`AuthConsumerValidityPeriods`, a slice of
`AuthConsumerValidityPeriod`) is a list of `(IssuedAt, Duration)`
pairs; a zero duration marks a revoked period. The latest period is
the currently active one, returned by `Latest()`. The `Sort()` method
maintains the slice newest-first.

When a consumer is regenerated (via the consumer-regen handler), a
new period is appended. In-flight tokens issued under the old period
stay valid until they expire naturally, and the multi-key JWT
verification covers any signature drift (see
[section 8](#8-jwt-signing-and-key-rotation)).

### 5.6 Consumer-creation helpers

The platform exposes one helper per consumer kind, all under
`engine/api/authentication/`:

| Helper | Purpose |
| --- | --- |
| `local.NewConsumerWithHash` | Local signup: hashes a password and creates the row |
| `NewConsumerExternal` | Mints a user consumer after an external driver returned user info |
| `builtin.NewConsumer` | Creates a personal access token or a service token |
| `NewConsumerHatchery` | Registers a hatchery |
| `NewConsumerWorker` | Issues a v1 worker token from a hatchery |
| `NewConsumerWorkerV2` | Issues a v2 worker token |

## 6. Scopes

Scopes are coarse capability tags attached to a consumer; they answer
*"can this token reach project endpoints at all?"*. They compose with
RBAC: a request must pass **both** scopes (logical OR over the
route's allowed list intersecting the consumer's scopes) and RBAC
(per-resource permission, [`09-rbac.md`](./09-rbac.md)).

The `AuthConsumerScope*` constants are in `sdk/token.go`:

| Constant | Used for |
| --- | --- |
| `AuthConsumerScopeUser` | Basic user actions |
| `AuthConsumerScopeAccessToken` | Token management |
| `AuthConsumerScopeAction` | Action CRUD |
| `AuthConsumerScopeAdmin` | Administrative endpoints |
| `AuthConsumerScopeGroup` | Group management |
| `AuthConsumerScopeTemplate` | Template management |
| `AuthConsumerScopeProject` | Project endpoints |
| `AuthConsumerScopeRun` | Run management |
| `AuthConsumerScopeRunExecution` | Run execution |
| `AuthConsumerScopeHooks` | Hook handlers |
| `AuthConsumerScopeWorkerModel` | Worker model management |
| `AuthConsumerScopeHatchery` | Hatchery endpoints |
| `AuthConsumerScopeService` | Service-to-service traffic |

A scope can be further refined with per-route restrictions through
`AuthConsumerScopeDetails`: a token can be limited to a single
endpoint within a scope (for instance, a token scoped to project
endpoints can still be locked to a specific create-workflow path
only).

## 7. Sessions

### 7.1 `AuthSession`

An `AuthSession` (in `sdk/token.go`) carries: an ID, the consumer ID
it is bound to, an expiry, the creation timestamp, an MFA flag, and
aggregates (the consumer, the groups, a `Current` flag for UI
purposes, and the last activity timestamp).

The signed JWT carries `AuthSessionJWTClaims`: the session ID, a
token-ID reference, and the standard JWT claims.

### 7.2 Lifecycle

The session module lives in `engine/api/authentication/session.go`:

| Helper | Purpose |
| --- | --- |
| `NewSession` | Mints a session for a consumer |
| `NewSessionWithMFA` | Mints a session and enables MFA tracking (default 15-minute activity TTL) |
| `NewSessionWithMFACustomDuration` | Same, with a custom MFA activity TTL |
| `CheckSession` | Validates expiration and MFA activity |
| `CheckSessionWithCustomMFADuration` | Validates with a custom MFA TTL |
| `NewSessionJWT` | Signs the session into a JWT (RS512) using the current signing key |
| `SessionCleaner` | Long-running goroutine — drops expired sessions; runs a corruption cleanup every 12 h |
| `SetSessionActivity` | Refreshes MFA activity in cache |
| `GetSessionActivity` | Reads MFA activity (used by `CheckSession`) |

### 7.3 MFA activity TTL

The MFA activity window is a sliding TTL: every authenticated request
refreshes it. When the TTL expires, the session loses its MFA flag
and any RBAC rule that gates on MFA starts failing for that session.
The activity timestamp is stored in cache, keyed by session ID
(`api:session:mfa:activity:{sessionID}`). The default window is 15
minutes.

## 8. JWT signing and key rotation

The signing surface lives in
`engine/api/authentication/authentication.go`.

### 8.1 Initialisation

At boot, `Init` loads a list of RSA private keys (`KeyConfig`, each
tagged with a `Timestamp` and a PEM-encoded `Key`). The keys are
sorted by timestamp ascending and one signer is built for each. The
newest key is the active signing key; the older keys remain loaded so
previously issued tokens can still be verified.

### 8.2 Signing

| Helper | Purpose |
| --- | --- |
| `GetSigningKey` | Returns the newest RSA private key |
| `SignJWT` | Signs an arbitrary JWT with the active key |
| `SignJWS` | Signs a content block with the active key for a given duration |

### 8.3 Verification

| Helper | Purpose |
| --- | --- |
| `VerifyJWT` | Tries every loaded key, newest first; returns the matching public key on success |
| `VerifyJWS` | Same multi-key tolerance for JWS payloads |

### 8.4 Rotation strategy

Adding a new key is a hot operation:

1. Append a new `KeyConfig` entry to the API configuration and reload
   it.
2. `Init` re-sorts so the new key becomes the active signing key.
3. Old keys stay loaded; tokens signed under them remain valid until
   they expire naturally.

To revoke a key, drop it from the configuration — every token signed
under that key will fail `VerifyJWT` on the next request.

## 9. MFA

Today only one driver actively supports MFA: Corporate SSO
(`corpsso`). The driver manifest reports
`SupportMFA = Config.Drivers.CorporateSSO.MFASupportEnabled`, so the
API can advertise MFA to the UI when it is enabled.

The `isMFA` admission helper (`engine/api/api_helper.go`) reads the
session's MFA boolean. The session was marked MFA when it was created
via `NewSessionWithMFA`, and the boolean is dropped when the activity
TTL expires.

RBAC rules can gate on MFA. For example, `hasRoleOnProject`
(`engine/api/router_rbac_rule_project.go`) consults `supportMFA(ctx)`
and `isMFA(ctx)`, and when the driver supports MFA but the session
does not have it, the helper checks a per-`project_key` feature flag
to decide whether to fail with `sdk.ErrMFARequired`.

`V2Initiator.IsAdminWithMFA` (see
[`07b-run-engine-v2.md`](./07b-run-engine-v2.md)) carries the MFA bit
through run-engine events so that audit trails know whether a manual
trigger was MFA-protected.

## 10. User link to external accounts

A `UserLink` (in `sdk/user_link.go`) joins a CDS user with a remote
identity. The persisted shape carries: a UUID, the authenticated CDS
user ID (`AuthentifiedUserID`), the external system type (`github`,
`gitlab`, `bitbucketserver`, `forgejo`), the external user ID, the
external username, and a creation timestamp.

This is what the ascode analyser (see
[`05-ascode-entities.md`](./05-ascode-entities.md)) uses to map a
GPG-signed commit to a CDS RBAC identity.

### 10.1 Routes

Handlers live under `engine/api/link.go`. The link-driver registry
(`api.LinkDrivers`, populated at boot in `engine/api/api.go`) is
keyed by `AuthConsumerType`.

| Route | Method | Auth | Handler |
| --- | --- | --- | --- |
| `/link/driver` | GET | required | `getLinkDriversHandler` |
| `/link/{consumerType}/ask` | POST | user | `postAskLinkExternalUserWithCDSHandler` |
| `/link/{consumerType}` | POST | user | `postLinkExternalUserWithCDSHandler` |
| `/admin/user/{username}/link/{consumerType}` | DELETE | admin | `deleteUserLinkHandler` |

### 10.2 Flow

1. The user calls `/link/{consumerType}/ask`. The API consults
   `api.LinkDrivers[consumerType]` and returns the OAuth / SSO
   redirect URL with a `LinkUser = true` flag in the state.
2. After the third-party redirects back, the user posts to
   `/link/{consumerType}` with the auth code. The driver's
   `GetUserInfo` is called and the API inserts a `UserLink` row.

Today only the GitHub link driver (`LinkGithubDriver`, in
`engine/api/link/github/`) is wired in production. The other external
systems rely on their auth-driver implementation indirectly.

## 11. User object

The CDS user is `sdk.AuthentifiedUser` (in `sdk/user.go`). Carries:

| Field | Purpose |
| --- | --- |
| `ID` | UUID |
| `Username` | Login |
| `Fullname` | Display name |
| `Ring` | One of `ADMIN`, `MAINTAINER`, `USER`; admin-level bypasses are documented in [`09-rbac.md`](./09-rbac.md#11-bypasses) |
| `Groups` | Aggregate of group memberships (loaded on demand) |
| `Created`, `LastUpdate` | Audit timestamps |

The Ring is the single global capability flag — ADMIN bypasses every
RBAC check (with `trackSudo` logging the override); MAINTAINER
automatically grants `ProjectRoleRead` on any project. USER is the
default.

## 12. HTTP middleware

The authentication middleware
(`engine/api/router_middleware_auth.go`) extracts and validates the
JWT, looks up the consumer + session, refreshes the MFA activity, and
attaches the resolved identity to the request context. Downstream
RBAC checking (documented in [`09-rbac.md`](./09-rbac.md)) consumes
the context.

The legacy v1 permission middleware
(`engine/api/router_middleware_auth_permission.go`) gates v1 routes
via `checkWorkflowPermissions`, `checkProjectPermissions` (see
[section 13](#13-v1-legacy-group-permissions)).

Middleware ordering (auth → permission / RBAC → handler) is
documented in [`01-architecture.md`](./01-architecture.md).

## 13. V1 legacy group permissions

V1 uses a numeric, group-based ACL model that predates RBAC. The
constants live in `sdk/group.go`:

| Constant | Value | Capability |
| --- | --- | --- |
| `PermissionRead` | 4 | read-only |
| `PermissionReadExecute` | 5 | read + execute (workflows / pipelines) |
| `PermissionReadWriteExecute` | 7 | full access |

These apply to **v1 resources only**: v1 workflows, pipelines,
applications, environments, old templates. V2 resources never read
this permission map. A user can hold v1 permissions on legacy
workflows and v2 RBAC roles on new ascode workflows in the same
project — **the two systems do not interoperate**.

### 13.1 Data model

| Type | Purpose | File |
| --- | --- | --- |
| `GroupPermission` | Group + Permission level | `sdk/group.go` |
| `ProjectGroup` | Project + Permission | `sdk/group.go` |
| `WorkflowGroup` | Workflow + Permission | `sdk/group.go` |

### 13.2 DAO

| File | Concerns |
| --- | --- |
| `engine/api/group/dao.go` | Group CRUD |
| `engine/api/group/group_permission.go` | Permission helpers |
| `engine/api/group/project_group.go` | Project ↔ group bindings |
| `engine/api/group/workflow_group.go` | Workflow ↔ group bindings |

### 13.3 Enforcement

V1 routes declare a `PermissionLevel` at registration time. The
permission middleware
(`engine/api/router_middleware_auth_permission.go`) checks the user's
group memberships against the resource's bindings:

| Helper | Purpose |
| --- | --- |
| `checkWorkflowPermissions` | Gate v1 workflow endpoints |
| `checkProjectPermissions` | Gate v1 project / application / environment / pipeline endpoints |

V2 routes declare `RbacCheckers` instead of `PermissionLevel`. A
route never declares both. The legacy data model is documented in
[`02-project-and-tenancy.md`](./02-project-and-tenancy.md).

## 14. Cross-spec pointers

- V2 RBAC (workflow-v2 authorization) → [`09-rbac.md`](./09-rbac.md)
- Microservices, middleware ordering, inter-service auth →
  [`01-architecture.md`](./01-architecture.md)
- Project, organisation, groups, regions, integrations →
  [`02-project-and-tenancy.md`](./02-project-and-tenancy.md)
- Workflow v1 (the legacy permission consumer) →
  [`03-workflow-v1.md`](./03-workflow-v1.md)
- Workflow v2 schema, gates, reviewers →
  [`04-workflow-v2.md`](./04-workflow-v2.md)
- Ascode entities (use the link system to map commit signers) →
  [`05-ascode-entities.md`](./05-ascode-entities.md)
- V1 hook routing → [`06a-hooks-v1.md`](./06a-hooks-v1.md)
- V2 hook routing → [`06b-hooks-v2.md`](./06b-hooks-v2.md)
- V1 run engine → [`07a-run-engine-v1.md`](./07a-run-engine-v1.md)
- V2 run engine, `V2Initiator`, gate approval → [`07b-run-engine-v2.md`](./07b-run-engine-v2.md)
- Hatchery + worker tokens → [`10-hatcheries.md`](./10-hatcheries.md)
- Worker session lifecycle → [`11-workers.md`](./11-workers.md)
- CDN, log streaming → [`12-cdn-and-artifacts.md`](./12-cdn-and-artifacts.md)
- VCS providers → [`13-vcs.md`](./13-vcs.md)
- Integrations → [`14-integrations.md`](./14-integrations.md)
- cdsctl → [`15-cli.md`](./15-cli.md)
- Go SDK → [`16-sdk.md`](./16-sdk.md)
- gRPC plugins → [`17-plugins.md`](./17-plugins.md)
- UI → [`18-ui.md`](./18-ui.md)
- Glossary, statuses, events → [`19-glossary-and-cross-references.md`](./19-glossary-and-cross-references.md)
