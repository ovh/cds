# Local In-Process Service Communication

## Overview

When all CDS services (API, CDN, VCS, Hooks, Repositories, Elasticsearch, Hatchery, UI) run in the same process, they can communicate directly in-process without HTTP network calls, API tokens, or cryptographic signatures.

This mode is automatically enabled when the API service is co-located with other services in the same `engine start` command. No configuration flag is required.

## What Changes

### Service → API Communication

Each service uses a `cdsclient.Interface` to call the API. In local mode, the HTTP client's transport is replaced with a `LocalRoundTripper` that calls the API's `http.Handler.ServeHTTP()` directly via an in-memory request/response cycle.

- No TCP connection is established
- No JWT token is generated or validated
- The service's identity (consumer ID and type) is injected via Go context values
- JSON serialization still happens at the handler level

### API → Service Communication

When the API needs to call another service (e.g., VCS for repository info, CDN for cache sync), it uses a local handler registry instead of making HTTP requests with RSA signatures.

- The registry maps service types to their `http.Handler`
- Requests are dispatched directly to the handler
- No RSA signature is generated or verified

### Authentication Bypass

Authentication is handled by injecting the service's consumer identity into the request context using Go's `context.WithValue()`. This is secure because:

- Context values can only be set programmatically (not from HTTP headers)
- External HTTP requests cannot inject local context values
- The API middleware loads the full consumer from the database, so all permission checks (`isCDN(ctx)`, `isHatchery(ctx)`, `isService(ctx)`) work identically

### Unified HTTP Gateway

A single HTTP server on one port serves all service routes with path prefixes:

| Prefix | Service |
|--------|---------|
| `/` | UI (Angular SPA) |
| `/cdsapi` | API |
| `/cdscdn` | CDN |
| `/cdshooks` | Hooks |
| `/vcs` | VCS |
| `/repositories` | Repositories |
| `/elasticsearch` | Elasticsearch |
| `/hatchery` | Hatchery |

Individual services skip starting their own HTTP listeners in gateway mode. The gateway uses the API's configured port.

### Embedded UI Static Files

The Angular UI files can be embedded in the Go binary using `go:embed`. At build time, the `embed_ui` Makefile target copies `ui/dist/browser/` into `engine/ui/dist/`. The `index.tmpl` template is transformed in memory with version, base URL, and Sentry URL substitutions.

When no embedded files are present, the service falls back to the filesystem-based approach (download from GitHub or use local build).

## Backward Compatibility

- Distributed deployments (services on separate machines) are unaffected
- All existing interfaces (`cdsclient.Interface`, service client) remain unchanged
- Configuration files do not require changes
- The mode is purely additive and opt-in via co-location
