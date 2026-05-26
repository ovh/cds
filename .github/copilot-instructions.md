# CDS Copilot Instructions

## Repository structure

- **`engine/`** — Core server: API (`engine/api/`), CDN, VCS, hooks, hatchery manager, repositories, authentication, and more.
- **`engine/worker/`** — Worker binary that executes jobs on agents.
- **`sdk/`** — Shared Go library: all data models, `cdsclient` (REST API wrapper), interpolation, RBAC types, event types, plugin interfaces.
- **`cli/cdsctl/`** — CLI client built on the SDK.
- **`ui/`** — Angular frontend (NGXS state management).
- **`contrib/`** — gRPC action plugins, artifact managers (Artifactory, Helm), worker model examples. Each plugin is an independent Go module.
- **`tests/`** — Integration tests using Venom.

## Key Conventions

- Specs are stored in many folders named `specs` as markdown files. Specs should contains human readable content, technical details about the implementation should be avoided and placed in the codebase instead as comments. Also no code snippets should be included in the specs, if needed they should be placed in the codebase as test cases or examples.
- When adding a new feature or modifying an existing one, please check if there is a related spec and update it accordingly. If there is no related spec, please create one in the appropriate `specs` folder.
- Copilot should always use VSCode tools to read and write files rather than requesting the user to run shell commands.