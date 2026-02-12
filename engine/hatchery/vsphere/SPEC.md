# CDS Hatchery vSphere — Specification

## 1. Overview

The CDS Hatchery vSphere is a component of the CDS (Continuous Delivery Service) platform responsible for
automatically spawning CDS workers on a VMware vSphere infrastructure. It creates virtual machines by cloning
VMware templates, boots them, and launches a CDS worker process inside the guest OS via VMware Guest Operations.

The hatchery implements the `hatchery.InterfaceWithModels` interface and supports both CDS Worker Model V1
(managed by the CDS API) and Worker Model V2 (defined as-code in repositories).

## 2. Architecture

### 2.1 Component Diagram

```
┌─────────────┐         ┌──────────────────┐         ┌──────────────┐
│   CDS API   │◄────────│  Hatchery vSphere│────────►│  vSphere API │
│             │  gRPC /  │                  │ govmomi  │  (vCenter /  │
│             │  HTTP    │  - Spawn loop    │          │   ESXi)      │
│             │         │  - Provisioning  │          │              │
│             │         │  - Cleanup loops │          │              │
└─────────────┘         └──────────────────┘         └──────────────┘
```

### 2.2 Source Files

| File | Responsibility |
|------|----------------|
| `types.go` | Configuration structs and `HatcheryVSphere` struct definition |
| `hatchery.go` | Hatchery lifecycle (init, config, CanSpawn), worker cleanup, provisioning loops |
| `spawn.go` | VM spawning logic, template creation (V1), provisioning, worker bootstrap |
| `client.go` | VM listing/filtering, clone spec preparation, guest operations |
| `vsphere.go` | `VSphereClient` interface and govmomi SDK wrapper implementation |
| `init.go` | Hatchery initialization, govmomi client creation, background goroutines setup |
| `ip.go` | IP address management (allocation, reservation, availability) |

## 3. Configuration

### 3.1 Configuration Struct

The hatchery is configured via `HatcheryConfiguration`, serialized as TOML.

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `commonConfiguration` | `HatcheryCommonConfiguration` | — | Yes | Shared hatchery config (see section 3.1.1) |
| `user` | `string` | — | Yes | vSphere user for API authentication |
| `endpoint` | `string` | — | Yes | vSphere endpoint (e.g. `pcc-11-222-333-444.ovh.com`) |
| `password` | `string` | — | Yes | vSphere password |
| `datacenterString` | `string` | — | Yes | vSphere datacenter name |
| `datastoreString` | `string` | — | No | vSphere datastore name (uses default if empty) |
| `networkString` | `string` | — | No | vSphere network name (uses default if empty) |
| `cardName` | `string` | `e1000` | No | Virtual ethernet card type |
| `iprange` | `string` | — | No | IP range for static IP assignment (format: `a.a.a.a/b,c.c.c.c/e`) |
| `gateway` | `string` | — | No | Gateway IP for spawned workers |
| `dns` | `string` | — | No | DNS server IP |
| `subnetMask` | `string` | `255.255.255.0` | No | Subnet mask |
| `workerTTL` | `int` | `120` | No | Worker time-to-live in minutes |
| `workerRegistrationTTL` | `int` | `10` | No | Worker registration timeout in minutes |
| `workerProvisioningInterval` | `int` | `120` (2 min) | No | Provisioning loop interval in seconds |
| `workerProvisioningPoolSize` | `int` | `1` | No | Max concurrent provisioning operations |
| `workerProvisioning` | `[]WorkerProvisioningConfig` | — | No | List of models to pre-provision |
| `guestCredentials` | `[]GuestCredential` | — | No | Guest OS credentials per model |
| `defaultWorkerModelsV2` | `[]DefaultWorkerModelsV2` | — | No | Default V2 models for V1 jobs (binary matching) |

### 3.1.1 Common Provision Configuration

The `commonConfiguration.provision` block contains settings shared across all hatchery types.
The fields most relevant to the vSphere hatchery are:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `maxWorker` | `int` | `10` | Maximum allowed simultaneous workers (see section 6.4) |
| `maxConcurrentProvisioning` | `int` | `10` | Maximum allowed simultaneous workers in pending/starting state |
| `maxConcurrentRegistering` | `int` | `2` | Maximum allowed simultaneous model registrations (`-1` to disable) |
| `registerFrequency` | `int` | `60` | Check interval (seconds) for model registration |
| `region` | `string` | `""` | Region label; jobs can require a specific region |
| `ignoreJobWithNoRegion` | `bool` | `false` | If true, ignore jobs that do not specify a region prerequisite |
| `maxAttemptsNumberBeforeFailure` | `int` | `5` | Max spawn attempts for the same job before marking it as failed (`-1` to disable) |

**Validation rules**:
- `maxConcurrentProvisioning` must be ≤ `maxWorker`
- `maxConcurrentRegistering` must be ≤ `maxWorker`

### 3.2 Worker Provisioning Config

```go
type WorkerProvisioningConfig struct {
    ModelPath   string  // CDS worker model path (V1 only, e.g. "group/model")
    ModelVMWare string  // VMware template name (V2 only, e.g. "debian12")
    Number      int     // Number of VMs to keep pre-provisioned
}
```

### 3.3 Guest Credentials

```go
type GuestCredential struct {
    ModelPath   string  // CDS worker model path (V1 only)
    ModelVMWare string  // VMware template name (V2 only)
    Username    string  // Guest OS username
    Password    string  // Guest OS password
}
```

### 3.4 Default Worker Models V2

```go
type DefaultWorkerModelsV2 struct {
    WorkerModelV2 string    // V2 worker model reference
    Binaries      []string  // If a job binary requirement matches, use this model
}
```

Used to bridge V1 jobs (which select models by binary requirements) to V2 worker models.

## 4. Worker Models

### 4.1 Worker Model V1

Worker Model V1 is managed by the CDS API. Each model has a type (`vsphere`), a name, and a
`ModelVirtualMachine` struct:

```go
type ModelVirtualMachine struct {
    Image    string  // vSphere template name to clone from
    Flavor   string  // Currently unused by the vSphere hatchery
    PreCmd   string  // Script to run before the worker binary
    Cmd      string  // Worker binary invocation command
    PostCmd  string  // Script to run after worker exits (e.g. "sudo shutdown -h now")
    User     string  // Guest OS username (can be overridden by GuestCredentials config)
    Password string  // Guest OS password (can be overridden by GuestCredentials config)
}
```

The hatchery creates (and caches) a **template VM** for each V1 model. This template is built by:
1. Cloning the base image
2. Running `PostCmd` inside the guest
3. Waiting for shutdown
4. Marking the VM as a vSphere template

Template re-creation is triggered when `NeedRegistration` is true or when `UserLastModified` differs
from the timestamp stored in the template's annotation.

### 4.2 Worker Model V2

Worker Model V2 is defined as-code (YAML) in repositories:

```yaml
name: my-worker
type: vsphere
osarch: linux/amd64
spec:
  image: "debian12"
  username: "admin"
  password: "${{ secrets.VM_PASSWORD }}"
```

```go
type V2WorkerModelVSphereSpec struct {
    Image    string  // Name of the vSphere template to clone
    Username string  // Guest OS username
    Password string  // Guest OS password
}
```

For V2, the template must already exist in vSphere. The hatchery does **not** create or manage templates
for V2 models. Guest credentials can be specified either in the model spec or in the hatchery
`GuestCredentials` configuration (which takes precedence).

## 5. VM Annotations

Every VM created by the hatchery carries a JSON annotation stored in `VirtualMachineConfigSpec.Annotation`.
This annotation is the primary mechanism for tracking VM state and ownership.

```go
type annotation struct {
    HatcheryName            string    // Name of the hatchery that created this VM
    WorkerName              string    // CDS worker name assigned to this VM
    RegisterOnly            bool      // True if VM is for model registration only
    Provisioning            bool      // True if VM is a pre-provisioned idle worker
    WorkerModelPath         string    // CDS worker model path (V1, e.g. "group/model")
    VMwareModelPath         string    // VMware template name (V2)
    WorkerModelLastModified string    // Unix timestamp of model last modification
    Model                   bool      // True if VM is a model template (do not destroy)
    Created                 time.Time // Creation timestamp
    JobID                   string    // CDS job ID assigned to this worker
    IPAddress               string    // Static IP assigned to this VM
}
```

All hatchery operations (cleanup, provisioning lookup, duplicate detection) rely on parsing these
annotations from `VirtualMachine.Config.Annotation`.

## 6. Lifecycle

### 6.1 Initialization

On startup (`InitHatchery`), the hatchery:

1. Initializes the common hatchery subsystem
2. Creates a govmomi client connected to the vSphere endpoint
3. Instantiates the `VSphereClient` wrapper bound to the configured datacenter
4. Parses the IP range (if configured) into a list of available IP addresses
5. Starts background goroutines:
   - **Provisioning loop** (if `workerProvisioning` is configured): runs every `workerProvisioningInterval` seconds, executes `provisioningV2()` then `provisioningV1()`
   - **Kill awol servers loop**: runs every 2 minutes, removes stale/expired VMs
   - **Kill disabled workers loop**: runs every 2 minutes, removes disabled workers

### 6.2 Spawning a Worker

The main spawn flow (`SpawnWorker`) proceeds as follows:

```
SpawnWorker(spawnArgs)
│
├── 1. Resolve template VM
│   ├── V2: Load template by spec.Image name (must exist)
│   └── V1: Load template by model name, or create it if NeedRegistration
│
├── 2. Try to find a pre-provisioned VM (if not register-only)
│   ├── Found: rename → start → wait for IP → launch worker script → DONE
│   └── Not found: continue to fresh clone
│
├── 3. Fresh clone path
│   ├── Build annotation
│   ├── prepareCloneSpec() → clone specification (network, IP, datastore)
│   ├── Clone template VM into datacenter folder
│   ├── Wait for guest operations readiness
│   └── Launch worker script via guest operations → DONE
│
└── 4. Error handling: shutdown + mark for deletion on any failure
```

#### 6.2.1 Clone Specification (`prepareCloneSpec`)

The clone specification defines how the VM is created:

- **Network**: Loads the first ethernet card from the template's devices, reconfigures it with the
  configured card type and network backing
- **Resource Pool**: Uses the datacenter's default resource pool
- **Datastore**: Relocates the VM to the configured datastore
- **Disk**: Uses `MoveChildMostDiskBacking` disk move type (linked clone)
- **Customization**: Linux prep with auto-generated hostname. If IP range is configured, assigns
  a static IP with subnet mask, gateway, and DNS
- **Annotation**: Serializes the `annotation` struct as JSON into `VirtualMachineConfigSpec.Annotation`
- **Power On**: The clone is powered on immediately (`PowerOn: true`)
- **VM Tools**: Configured to run after power on

**Important**: CPU, RAM, and disk size are **not** specified in the clone spec. The cloned VM inherits
all resource settings from the source template.

#### 6.2.2 Worker Script Launch (`launchScriptWorker`)

After the VM is cloned and has obtained an IP:

1. Wait for the VM IP address to be available
2. Generate worker configuration (API endpoint, tokens, etc.)
3. Build the launch script: `PreCmd + Cmd + PostCmd`, templated with worker config
4. Check VM readiness by running `env` command via guest operations
5. Execute the launch script via `StartProgramInGuest` with guest credentials
6. The script is run as: `/bin/echo -n ;<script>`, with `CDS_CONFIG` passed as environment variable

#### 6.2.3 Guest Operations Authentication

Guest OS credentials are resolved in order:
1. From `GuestCredentials` config, matched by `ModelVMWare` (V2) or `ModelPath` (V1)
2. If not found in config, from the worker model spec (`Username`/`Password`)

If neither provides valid credentials, spawning fails.

### 6.3 Pre-Provisioning

Pre-provisioning creates idle VMs ahead of time so that job assignment is faster.

#### 6.3.1 V2 Provisioning (`provisioningV2`)

1. Lock the provisioning cache
2. List all VMs prefixed with `provision-v2`, count per VMware model path
3. For each model in `WorkerProvisioning` config with a `ModelVMWare`:
   - Calculate deficit: `config.Number - currentCount`
   - Queue provisioning tasks for the deficit
4. Execute up to `WorkerProvisioningPoolSize` concurrent provisioning goroutines
5. Each provisioning operation:
   - Generates a worker name: `provision-v2-<random>`
   - Adds name to `cacheProvisioning.pending`
   - Calls `ProvisionWorkerV2()`: clone → wait for IP → shutdown
   - Removes name from pending cache

#### 6.3.2 V1 Provisioning (`provisioningV1`)

1. Same counting logic as V2 but with `provision-v1` prefix and `WorkerModelPath`
2. Additionally fetches the model from the CDS API to verify it doesn't need registration
3. Distributes models in a round-robin provision queue
4. Executes up to `WorkerProvisioningPoolSize` concurrent provisioning goroutines
5. Each provisioning operation:
   - Generates a worker name: `provision-v1-<random>`
   - Calls `ProvisionWorkerV1()`: clone → wait for IP → shutdown

#### 6.3.3 Provisioned VM Lifecycle

A provisioned VM follows this lifecycle:

```
Template ──clone──► Provisioned VM (powered on)
                    │
                    ├── Wait for IP
                    ├── Shutdown (stays in powered-off state)
                    │
                    └── On job assignment (FindProvisionnedWorker):
                        ├── Rename to worker name
                        ├── Power on
                        ├── Wait for IP
                        └── Launch worker script
```

#### 6.3.4 Finding a Provisioned Worker (`FindProvisionnedWorker`)

When spawning a worker, the hatchery tries to reuse a pre-provisioned VM:

1. Determine expected model path (V2: image name, V1: full CDS model path)
2. Iterate all VMs, filtering by prefix (`provision-v2` or `provision-v1`)
3. Parse annotation, verify `Provisioning` flag and matching model path
4. Skip VMs in `cacheProvisioning.pending` (still being created)
5. Skip VMs in `cacheToDelete` (marked for deletion)
6. Skip VMs without `VmPoweredOffEvent` (not yet fully provisioned)
7. Skip VMs in `cacheProvisioning.using` (already being assigned)
8. Mark selected VM as "using" in cache
9. Return the VM for reuse

### 6.4 Capacity Management and MaxWorker

Before attempting to spawn any worker (for a job or for model registration), the common hatchery
framework calls `checkCapacities()`. This function enforces several limits. All of these checks
happen **before** `CanSpawn` or `CanAllocateResources` are called.

#### 6.4.1 Worker Pool

The capacity check starts by building a **worker pool** via `WorkerPool()`. This pool is the union of:

1. **Registered workers** — workers known to the CDS API (fetched via `WorkerList()`), in statuses:
   `Checking`, `Waiting`, `Building`, `Pending`, `Registering`
2. **Started workers** — workers reported by the hatchery's `WorkersStarted()` but not yet
   registered on the CDS API. These are assigned status `WorkerPending` (or `WorkerRegistering`
   if their name starts with `register-`).

For the vSphere hatchery, `WorkersStarted()` returns the names of all non-template VMs in the
datacenter, **excluding VMs whose name starts with `provision-`**. This means pre-provisioned
(idle) VMs are not counted against the worker limit, but they become counted as soon as they are
renamed to a real worker name during job assignment.

**Consistency check**: If a registered worker exists on the CDS API but is not found in the
hatchery's started list, it is flagged as inconsistent and disabled via `WorkerDisable()`.

#### 6.4.2 MaxWorker Limit

```
Provision.MaxWorker (default: 10)
```

If the size of the worker pool (see above) is **greater than or equal to** `MaxWorker`, no new
worker can be spawned. The check returns `false` and the job is skipped for this scheduling cycle.

```go
if len(workerPool) >= h.Configuration().Provision.MaxWorker {
    return false  // capacity reached
}
```

This limit applies uniformly to:
- **Job V1 processing** (`processJobV1QueueV1`)
- **Job V2 processing** (`processJobV2`)
- **Model registration** (`workerRegister`)

#### 6.4.3 MaxConcurrentProvisioning Limit

```
Provision.MaxConcurrentProvisioning (default: 10)
```

In addition to the MaxWorker limit, the framework checks that the number of workers currently in
`Pending` status does not exceed `MaxConcurrentProvisioning`. This prevents too many VMs from
being created simultaneously.

```go
if nbPending >= maxConcurrentProvisioning {
    return false
}
```

A separate in-memory atomic counter (`nbWorkerToStart`) also tracks goroutines that are about to
call `SpawnWorker()`. If this counter reaches `MaxConcurrentProvisioning`, capacity is exhausted.

#### 6.4.4 MaxConcurrentRegistering Limit

```
Provision.MaxConcurrentRegistering (default: 2, -1 to disable)
```

Controls the maximum number of worker models being registered simultaneously. Checked before
spawning a registration-only worker.

#### 6.4.5 Configuration Validation

At startup, the hatchery validates that:
- `MaxConcurrentProvisioning <= MaxWorker`
- `MaxConcurrentRegistering <= MaxWorker`

#### 6.4.6 Interaction with vSphere Pre-Provisioning

The `MaxWorker` limit and the vSphere-specific `WorkerProvisioning` (pre-provisioning pool) are
**independent mechanisms** that interact as follows:

- Pre-provisioned VMs (named `provision-*`) are **excluded** from `WorkersStarted()` and therefore
  **do not count** against `MaxWorker`.
- When a provisioned VM is assigned to a job, it is renamed (e.g. `provision-v2-xxx` →
  `worker-abc`). From that point, it **counts** against `MaxWorker`.
- The `WorkerProvisioningPoolSize` config controls how many provisioning operations run in
  parallel in the vSphere provisioning loop, but this is separate from
  `MaxConcurrentProvisioning` which governs the common framework's capacity check.
- There is **no global coordination** between the provisioning pool and `MaxWorker`. It is the
  operator's responsibility to ensure that `WorkerProvisioning[].Number` (total pre-provisioned
  VMs) plus expected active workers stays within the infrastructure's capacity.

#### 6.4.7 Monitoring

The `Status()` method reports the current worker count vs. MaxWorker:
```
Workers: <current>/<maxWorker>
```

### 6.5 Spawn Eligibility (`CanSpawn`)

Before spawning, the hatchery checks:

1. **Model type**: Must be `vsphere` (V1 or V2)
2. **Unsupported requirements**: Returns `false` if any requirement is of type:
   - `ServiceRequirement`
   - `MemoryRequirement`
   - `HostnameRequirement`
   - `FlavorRequirement`
3. **Empty Cmd**: Returns `false` if the model has no command defined
4. **Registration checks** (for register jobs):
   - V2 models cannot be registered (returns `false`)
   - Checks no temporary VM (`<model>-tmp`) or registering VM (`register-<model>`) exists
5. **Duplicate job check**: Ensures no existing VM annotation references the same `JobID`
6. **Pending job check**: Ensures the job ID is not in the local `cachePendingJobID`
7. **IP availability**: If IP range is configured, verifies at least one IP is available

### 6.6 Resource Allocation (`CanAllocateResources`)

The current implementation is a no-op stub:

```go
func (h *HatcheryVSphere) CanAllocateResources(...) (bool, error) {
    return true, nil
}
```

No resource limits (CPU, RAM, disk) are verified before spawning.

### 6.7 Model Registration (`NeedRegistration`) — V1 Only

The hatchery checks whether a V1 model needs re-registration by:

1. Looking up the existing VM template by model name
2. Parsing its annotation
3. Comparing `model.UserLastModified` with `annotation.WorkerModelLastModified`
4. Returns `true` if the model is flagged for registration or the template is outdated

### 6.8 Default V2 Model Selection — V1 Jobs on V2 Models

For V1 jobs that need to run on a V2 model (`GetDetaultModelV2Name`):

1. If no binary requirements exist in the job, returns the first configured default V2 model
2. Otherwise, iterates `DefaultWorkerModelsV2` and returns the first model whose `Binaries` list
   contains at least one of the job's binary requirements
3. Returns empty string if no match is found

## 7. IP Address Management

### 7.1 IP Allocation

When `iprange` is configured, the hatchery manages a pool of static IP addresses.

#### Finding an available IP (`findAvailableIP`):

1. Acquire IP mutex
2. List all VMs, collect used IPs from:
   - VM annotations (`annot.IPAddress`)
   - Guest network info (`Guest.Net[].IpAddress`)
3. Additionally track IPs that have been reserved locally but not yet assigned to a VM
4. Return the first IP from the configured range that is neither used nor reserved

#### Reserving an IP (`reserveIPAddress`):

1. Check the IP is not already reserved
2. Add to `reservedIPAddresses` list
3. Start a goroutine that removes the reservation after 5 minutes (safety timeout)

### 7.2 IP-less Mode

When `iprange` is not configured, no static IP assignment occurs. VMs rely on DHCP or
template-defined network settings.

## 8. Cleanup and Garbage Collection

### 8.1 Kill Awol Servers (`killAwolServers`) — Every 2 minutes

For each VM with a CDS annotation belonging to this hatchery:

1. **Marked for deletion**: Delete immediately
2. **Provisioned VMs** (`provision-` prefix): Skip (managed by provisioning loop)
3. **Model templates** (`Model: true`): Skip (never delete)
4. **Event analysis**: Load VM events (`VmStartingEvent`, `VmPoweredOffEvent`, `VmRenamedEvent`)
   - Filter out events related to provisioning (`provision-` in message)
   - Find the most recent start, power-off, and rename events
5. **Renamed but never started**: If `VmRenamedEvent` exists but no `VmStartingEvent`, and the
   rename is older than `WorkerRegistrationTTL` → delete
6. **No start event found**: Skip (VM not yet fully created)
7. **Worker exists on API side**: Delete if `vmStartedTime + WorkerTTL` has expired
8. **Worker does not exist on API side**:
   - If `VmPoweredOffEvent` found (after start): Worker finished → delete
   - If no power-off event: Worker still starting → delete if `vmStartedTime + WorkerRegistrationTTL` has expired

### 8.2 Kill Disabled Workers (`killDisabledWorkers`) — Every 2 minutes

1. Fetch the pool of disabled workers from the CDS API
2. For each disabled worker, find the matching VM by name
3. Mark matching VMs for deletion

### 8.3 Server Deletion (`deleteServer`)

1. Load the VM object
2. If the VM name starts with `register-`, check worker model registration status and report
   spawn errors to the API
3. If the VM is powered on, power it off
4. Remove from `cacheToDelete`
5. Destroy the VM via vSphere API

## 9. vSphere Client Interface

The hatchery interacts with vSphere through the `VSphereClient` interface, which wraps the
govmomi SDK. All vSphere API calls use a 15-second request timeout.

```go
type VSphereClient interface {
    ListVirtualMachines(ctx) ([]mo.VirtualMachine, error)
    LoadVirtualMachine(ctx, name) (*object.VirtualMachine, error)
    LoadVirtualMachineDevices(ctx, vm) (object.VirtualDeviceList, error)
    StartVirtualMachine(ctx, vm) error
    ShutdownVirtualMachine(ctx, vm) error
    DestroyVirtualMachine(ctx, vm) error
    CloneVirtualMachine(ctx, vm, folder, name, config) (*ManagedObjectReference, error)
    GetVirtualMachinePowerState(ctx, vm) (VirtualMachinePowerState, error)
    NewVirtualMachine(ctx, cloneSpec, ref, vmName) (*object.VirtualMachine, error)
    RenameVirtualMachine(ctx, vm, newName) error
    MarkVirtualMachineAsTemplate(ctx, vm) error
    WaitForVirtualMachineShutdown(ctx, vm) error
    WaitForVirtualMachineIP(ctx, vm, IPAddress, vmName) error
    LoadFolder(ctx) (*object.Folder, error)
    SetupEthernetCard(ctx, card, ethernetCardName, network) error
    LoadNetwork(ctx, name) (object.NetworkReference, error)
    LoadResourcePool(ctx) (*object.ResourcePool, error)
    LoadDatastore(ctx, name) (*object.Datastore, error)
    ProcessManager(ctx, vm) (*guest.ProcessManager, error)
    StartProgramInGuest(ctx, procman, req) (*StartProgramInGuestResponse, error)
    LoadVirtualMachineEvents(ctx, vm, eventTypes...) ([]BaseEvent, error)
}
```

The interface is mockable for unit testing (see `mock_vsphere/`).

### 9.1 VM Readiness

After cloning, the hatchery waits for full VM readiness in multiple stages:

1. **Guest operations ready**: Polls `Guest.GuestOperationsReady` (timeout: 3 minutes)
2. **IP address**: Polls `vm.WaitForIP()`, optionally matching an expected static IP (timeout: 3 minutes)
3. **Command execution ready**: Runs `env` in the guest to verify guest operations work (timeout: 1 minute)

### 9.2 VM Listing and Filtering

The hatchery uses govmomi's `ContainerView` to list all VMs in the datacenter. VMs are
fetched with properties: `name`, `summary`, `guest`, `config`.

Filtering helpers:
- `getVirtualMachines()`: Returns non-template VMs only
- `getRawTemplates()`: Returns template VMs only
- `getVirtualMachineTemplates()`: Returns templates with CDS annotation `Model: true`
- `getVirtualMachineTemplateByName()`: Finds a specific CDS model template

## 10. Internal Caches

The hatchery maintains several in-memory caches protected by mutexes:

| Cache | Type | Purpose |
|-------|------|---------|
| `cachePendingJobID` | `[]string` | Job IDs currently being spawned, prevents duplicates |
| `cacheProvisioning.pending` | `[]string` | VM names currently being provisioned |
| `cacheProvisioning.using` | `[]string` | Provisioned VM names being assigned to a job |
| `cacheToDelete` | `[]string` | VM names marked for deletion by spawn logic |
| `availableIPAddresses` | `[]string` | All IPs parsed from the configured range |
| `reservedIPAddresses` | `[]string` | IPs temporarily reserved (5-minute TTL) |

## 11. Limitations

1. **No VM resource customization**: CPU, RAM, and disk size are inherited from the template.
   There is no flavor or sizing mechanism.
2. **No resource limit enforcement**: `CanAllocateResources()` always returns `true`.
   The hatchery has no awareness of available vSphere resources.
3. **Unsupported job requirements**: Service, Memory, Hostname, and Flavor requirements
   cause the hatchery to reject the job.
4. **Linux only**: Customization assumes Linux guests (`CustomizationLinuxPrep`).
5. **Single datacenter**: The hatchery operates on a single vSphere datacenter.
6. **No V2 model registration**: V2 templates must be pre-created in vSphere manually.

---

# Amendment A: Resource-Based Capacity Management

## A.1 Motivation

The current capacity management relies solely on `MaxWorker` — a simple count of simultaneous
workers. This is a coarse-grained limit that does not account for the actual resource consumption
of each VM. A hatchery running 10 workers with 2 vCPUs each consumes very different infrastructure
capacity than 10 workers with 8 vCPUs each, yet both are treated identically.

The vSphere infrastructure already knows its own capacity through the **Resource Pool** abstraction.
Every vSphere hatchery is configured with a `resourcePool` — the hatchery should leverage the
Resource Pool's runtime information as the **primary** capacity management mechanism, rather than
relying on an arbitrary `MaxWorker` number that must be manually kept in sync with reality.

The vSphere API exposes the data required for resource-aware capacity management:

- **Resource Pool capacity** (primary): `ResourcePool.Runtime` provides `ResourcePoolResourceUsage`
  with `MaxUsage`, `OverallUsage`, and `UnreservedForVm` for both CPU and Memory. This is the
  **source of truth** for what the infrastructure can actually handle.
- **Per-VM resources** (supplementary): `summary.Config.NumCpu` and `summary.Config.MemorySizeMB`
  are already fetched by the existing `ListVirtualMachines()` call (properties: `name`, `summary`,
  `guest`, `config`). No additional API call is needed.

This amendment replaces the `CanAllocateResources()` stub with a real implementation that checks
whether the vSphere infrastructure has enough CPU and memory to spawn a new worker. It also makes
`MaxWorker` optional (value `0` = unlimited) so operators can rely purely on resource-based
capacity when desired.

## A.2 Scope

### In Scope

- Query **Resource Pool runtime info** for infrastructure-level capacity (**primary** mechanism,
  always enabled)
- Query resource usage from running VMs (CPU, RAM) using data already available from
  `ListVirtualMachines()` (**supplementary**, used for allocated-resource counting and optional
  static limits)
- Implement `CanAllocateResources()` with Resource Pool awareness and optional CPU/memory limits
- Add optional configuration fields for static resource limits (`maxCpus`, `maxMemoryMB`)
- Make `MaxWorker` optional (`0` = unlimited), requiring a framework change
- Expose resource usage in the monitoring status

### Out of Scope

- Flavor/resize support (see Amendment B)
- Disk capacity management
- Per-model resource tracking

## A.3 Design

### A.3.1 Resource Pool as Primary Capacity Source

The vSphere hatchery is always configured with a `resourcePool`. The Resource Pool provides
real-time capacity information via its `Runtime` property:

```go
type ResourcePoolResourceUsage struct {
    ReservationUsed      int64  // Reservation already consumed
    ReservationUsedForVm int64  // Reservation consumed by VMs
    UnreservedForPool    int64  // Free capacity for sub-pools
    UnreservedForVm      int64  // Free capacity for new VMs
    OverallUsage         int64  // Current total usage
    MaxUsage             int64  // Maximum capacity
}
// Available separately for CPU (MHz) and Memory (bytes) in ResourcePoolRuntimeInfo
```

The key fields for capacity decisions are:
- **Memory**: `Runtime.Memory.UnreservedForVm` (in bytes) — the amount of memory still available
  for new VMs. This is precise and reliable.
- **CPU**: `Runtime.Cpu.UnreservedForVm` (in MHz) — the CPU capacity still available for new VMs.

**Important**: Resource Pool CPU values are expressed in **MHz**, not vCPU count. Converting
between MHz and vCPU count requires knowledge of the host CPU frequency, which makes MHz-based
checks less intuitive. However, the Resource Pool check remains valuable as a **real
infrastructure-level guardrail** — it catches situations where the pool is genuinely exhausted,
regardless of how vCPUs are counted. The hatchery compares `UnreservedForVm` against the
next worker's estimated MHz requirement (derived from its vCPU count × an estimated MHz-per-vCPU,
or simply checking that `UnreservedForVm > 0`).

The Resource Pool check is **always enabled** — every vSphere hatchery has a Resource Pool, and
querying its runtime properties adds minimal overhead (a single property read on an already-loaded
object).

### A.3.2 Supplementary Resource Counting (VM-level)

In addition to the Resource Pool check, the hatchery counts resources currently allocated by
iterating all VMs it manages. This serves two purposes:
1. Enforcing optional static limits (`maxCpus`, `maxMemoryMB`) — useful when the operator wants
   to reserve part of the Resource Pool for non-CDS workloads.
2. Providing monitoring data (current CPU/memory usage).

The hatchery already calls `ListVirtualMachines()` frequently (in `getVirtualMachines()`, used by
cleanup loops, provisioning, CanSpawn, etc.). Each returned `mo.VirtualMachine` includes:

```go
vm.Summary.Config.NumCpu       // int32 — number of vCPUs
vm.Summary.Config.MemorySizeMB // int32 — RAM in MB
```

To count resources currently allocated by this hatchery, we iterate all non-template VMs that
belong to this hatchery (i.e. have a CDS annotation with matching `HatcheryName`), and sum
their CPU and memory values.

```go
func (h *HatcheryVSphere) countAllocatedResources(ctx context.Context) (totalCPUs int32, totalMemoryMB int32) {
    srvs := h.getVirtualMachines(ctx)
    for _, s := range srvs {
        annot := getVirtualMachineCDSAnnotation(ctx, s)
        if annot == nil || annot.HatcheryName != h.Name() {
            continue
        }
        if annot.Model {
            continue // don't count template VMs
        }
        totalCPUs += s.Summary.Config.NumCpu
        totalMemoryMB += s.Summary.Config.MemorySizeMB
    }
    return
}
```

**Key point**: This counts **all** VMs managed by this hatchery, including pre-provisioned VMs
(named `provision-*`). This is intentional — provisioned VMs consume real vSphere resources even
when idle, and must be accounted for in capacity planning. This differs from the `MaxWorker`
check which excludes provisioned VMs.

### A.3.3 Resource of the Next Worker

To decide whether a new worker can be spawned, the hatchery needs to know the resources the new
VM will consume. Without flavors (Amendment B), this is the resource footprint of the source
template.

The hatchery can read the template's resources from the already-loaded VM listing:

```go
func (h *HatcheryVSphere) getTemplateResources(ctx context.Context, templateName string) (cpus int32, memoryMB int32, err error) {
    vms := h.getRawVMs(ctx)
    for _, vm := range vms {
        if vm.Name == templateName {
            return vm.Summary.Config.NumCpu, vm.Summary.Config.MemorySizeMB, nil
        }
    }
    return 0, 0, fmt.Errorf("template %q not found", templateName)
}
```

## A.4 Configuration Changes

### A.4.1 New Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `maxCpus` | `int` | `0` (unlimited) | Optional. Maximum total vCPUs this hatchery may allocate across all VMs. `0` means no static CPU limit (Resource Pool remains the guardrail). |
| `maxMemoryMB` | `int64` | `0` (unlimited) | Optional. Maximum total RAM (MB) this hatchery may allocate across all VMs. `0` means no static memory limit (Resource Pool remains the guardrail). |

No `useResourcePoolLimits` flag is needed — the Resource Pool check is **always enabled**.

### A.4.2 `MaxWorker` Becomes Optional

Currently, `MaxWorker` defaults to `10` and a value of `0` blocks all spawning (because
`len(pool) >= 0` is always true). This amendment changes `MaxWorker` semantics so that `0` means
**unlimited** (no count-based limit), allowing operators to rely purely on resource-based capacity
management.

This requires a change in the **common hatchery framework** (`sdk/hatchery/provisionning.go`):

```go
// Before (current):
if len(workerPool) >= h.Configuration().Provision.MaxWorker {
    return false
}

// After (Amendment A):
if h.Configuration().Provision.MaxWorker > 0 &&
    len(workerPool) >= h.Configuration().Provision.MaxWorker {
    return false
}
```

And in the **configuration validation** (`engine/service/types.go`):

```go
// Before (current):
if hcc.Provision.MaxConcurrentProvisioning > hcc.Provision.MaxWorker {
    return error ...
}

// After (Amendment A): skip check if MaxWorker is 0 (unlimited)
if hcc.Provision.MaxWorker > 0 && hcc.Provision.MaxConcurrentProvisioning > hcc.Provision.MaxWorker {
    return error ...
}
// Same for MaxConcurrentRegistering
```

**Note**: This framework change is backward-compatible. Since the current default for `MaxWorker`
is `10`, existing deployments that do not explicitly set `maxWorker = 0` continue to behave
identically. Only deployments that opt in to `maxWorker = 0` get the new "unlimited" behavior.

### A.4.3 Example Configuration

```toml
[commonConfiguration.provision]
# MaxWorker = 0 means unlimited — rely on resource-based capacity only
maxWorker = 0

# Optional static resource limits (0 = no static limit)
maxCpus = 64
maxMemoryMB = 131072    # 128 GB
```

### A.4.4 Capacity Strategy Matrix

Operators can choose their strategy by combining these settings:

| MaxWorker | maxCpus/maxMemoryMB | Strategy |
|-----------|---------------------|----------|
| `10` (default) | `0` / `0` | **Current behavior** — count-based limit only, Resource Pool is a safety net |
| `0` | `0` / `0` | **Resource Pool only** — rely entirely on vSphere infrastructure limits |
| `0` | set | **Resource-based** — static limits + Resource Pool |
| `20` | set | **Belt and suspenders** — MaxWorker as safety ceiling, resource limits for precision, Resource Pool as infrastructure guardrail |

In all cases, the effective capacity is:
`min(MaxWorker (if >0), static resource limits (if >0), Resource Pool available capacity)`

## A.5 `CanAllocateResources` Implementation

```go
func (h *HatcheryVSphere) CanAllocateResources(ctx context.Context,
    model sdk.WorkerStarterWorkerModel, jobID string,
    requirements []sdk.Requirement) (bool, error) {

    // Determine resource footprint of the next worker (from template)
    templateName := model.GetVSphereImage()
    nextCPUs, nextMemoryMB, err := h.getTemplateResources(ctx, templateName)
    if err != nil {
        log.Warn(ctx, "CanAllocateResources> unable to get template resources for %q: %v", templateName, err)
        // If we can't determine the template size, fall back to allowing the spawn
        // to avoid blocking the hatchery due to a transient error
        return true, nil
    }

    // === Primary check: Resource Pool capacity (always enabled) ===
    canFit, err := h.checkResourcePoolCapacity(ctx, nextCPUs, nextMemoryMB)
    if err != nil {
        log.Warn(ctx, "CanAllocateResources> Resource Pool check failed: %v", err)
        // Don't block on Resource Pool query failure — fall through to static limits
    } else if !canFit {
        log.Info(ctx, "CanAllocateResources> Resource Pool capacity insufficient")
        return false, nil
    }

    // === Supplementary checks: static limits (if configured) ===

    if h.Config.MaxCPUs > 0 || h.Config.MaxMemoryMB > 0 {
        usedCPUs, usedMemoryMB := h.countAllocatedResources(ctx)

        // Check configured CPU limit
        if h.Config.MaxCPUs > 0 {
            if int(usedCPUs) + int(nextCPUs) > h.Config.MaxCPUs {
                log.Info(ctx, "CanAllocateResources> CPU limit reached: %d used + %d needed > %d max",
                    usedCPUs, nextCPUs, h.Config.MaxCPUs)
                return false, nil
            }
        }

        // Check configured memory limit
        if h.Config.MaxMemoryMB > 0 {
            if int64(usedMemoryMB) + int64(nextMemoryMB) > h.Config.MaxMemoryMB {
                log.Info(ctx, "CanAllocateResources> Memory limit reached: %d used + %d needed > %d max",
                    usedMemoryMB, nextMemoryMB, h.Config.MaxMemoryMB)
                return false, nil
            }
        }
    }

    return true, nil
}
```

## A.6 Resource Pool Capacity Check

This check is the **primary** capacity gate and is always performed.

```go
func (h *HatcheryVSphere) checkResourcePoolCapacity(ctx context.Context,
    requiredCPUs int32, requiredMemoryMB int32) (bool, error) {

    pool, err := h.vSphereClient.LoadResourcePool(ctx)
    if err != nil {
        return false, err
    }

    var poolMo mo.ResourcePool
    if err := pool.Properties(ctx, pool.Reference(), []string{"runtime"}, &poolMo); err != nil {
        return false, sdk.WrapError(err, "unable to get resource pool properties")
    }

    // Memory check (Resource Pool reports memory in bytes)
    requiredMemoryBytes := int64(requiredMemoryMB) * 1024 * 1024
    if poolMo.Runtime.Memory.UnreservedForVm < requiredMemoryBytes {
        log.Info(ctx, "checkResourcePoolCapacity> insufficient memory: %d bytes unreserved, %d bytes needed",
            poolMo.Runtime.Memory.UnreservedForVm, requiredMemoryBytes)
        return false, nil
    }

    // CPU check (Resource Pool reports CPU in MHz — this is a coarse check)
    // We verify that the pool has remaining CPU capacity.
    // Since MHz ≠ vCPUs, this is a guardrail rather than a precise limit.
    if poolMo.Runtime.Cpu.MaxUsage > 0 &&
        poolMo.Runtime.Cpu.UnreservedForVm <= 0 {
        log.Info(ctx, "checkResourcePoolCapacity> CPU pool exhausted: unreserved=%d MHz",
            poolMo.Runtime.Cpu.UnreservedForVm)
        return false, nil
    }

    return true, nil
}
```

**Note on CPU MHz vs vCPUs**: The Resource Pool reports CPU capacity in MHz, which does not map
directly to vCPU count. The CPU part of this check is therefore a **coarse safety net** — it
catches situations where the pool is genuinely exhausted, but does not provide precise vCPU-level
control. Precise vCPU control, if needed, comes from the optional `maxCpus` configuration field.
The **memory** check, however, is precise — bytes map directly to MB.

## A.7 VSphereClient Interface Extension

The Resource Pool runtime check requires reading properties from the already-loaded Resource Pool
object. This can be done inline (as shown in A.6) without extending the `VSphereClient` interface,
since `LoadResourcePool()` already returns `*object.ResourcePool`.

Optionally, a convenience method can be added to the interface for testability:

```go
type VSphereClient interface {
    // ... existing methods ...
    GetResourcePoolRuntime(ctx context.Context) (*types.ResourcePoolRuntimeInfo, error)
}
```

## A.8 Backward Compatibility

- If `maxCpus` and `maxMemoryMB` are both at their default values (`0`), no static resource limit
  is enforced. The Resource Pool check is always active but is a soft guardrail — if the Resource
  Pool query fails, the spawn is allowed to proceed (fail-open).
- `MaxWorker` continues to work as before with its default value of `10`. Only when explicitly set
  to `0` does it become unlimited. Existing deployments are unaffected.
- The framework change to `checkCapacities()` and `Check()` is backward-compatible: the guard
  `MaxWorker > 0` only changes behavior when `MaxWorker == 0`, which was previously a
  misconfiguration (it blocked all spawning). This amendment gives it a useful meaning instead.
- No changes to worker models, annotations, SDK, or spawn logic are required.
- The Resource Pool runtime query (`pool.Properties(ctx, ..., []string{"runtime"}, ...)`) is a
  lightweight read on an already-loaded object, adding negligible overhead.

## A.9 Implementation Plan

- [ ] Phase 1 — Framework change: Make `MaxWorker=0` mean "unlimited" in `sdk/hatchery/provisionning.go` and `engine/service/types.go`
- [ ] Phase 2 — Configuration: Add `MaxCPUs`, `MaxMemoryMB` fields to `HatcheryConfiguration`
- [ ] Phase 3 — Resource Pool check: Implement `checkResourcePoolCapacity()` (always-on primary check)
- [ ] Phase 4 — Resource Counting: Implement `countAllocatedResources()` using existing VM listing data
- [ ] Phase 5 — Template Resources: Implement `getTemplateResources()` for next-worker sizing
- [ ] Phase 6 — CanAllocateResources: Replace stub with Resource Pool + optional static limit checks
- [ ] Phase 7 — Tests: Unit tests for resource counting, Resource Pool capacity, limit enforcement, MaxWorker=0

---

# Amendment B: VM Flavor Support (CPU/RAM Resize)

*This amendment builds on Amendment A (resource-based capacity management).*

## B.1 Motivation

Even with Amendment A's resource-based capacity management, all VMs cloned by the vSphere hatchery
inherit their hardware configuration (CPU count, memory) from the source template. Operators must
maintain separate templates for each resource profile they need, which does not scale.

The OpenStack hatchery supports a **flavor** mechanism that maps abstract size names (e.g. `small`,
`medium`, `large`) to concrete provider flavors. This amendment adds an equivalent mechanism for
the vSphere hatchery, where flavors map to explicit CPU/RAM values applied at clone time via
`VirtualMachineConfigSpec.NumCPUs` and `MemoryMB`.

With Amendment A already in place, the capacity management automatically accounts for the resized
VMs since `countAllocatedResources()` reads actual VM hardware from `summary.Config`, not
template defaults.

## B.2 Scope

### In Scope

- Define flavors in the hatchery configuration with explicit CPU and RAM values
- Apply CPU/RAM overrides at VM clone time via `VirtualMachineConfigSpec`
- Add a `Flavor` field to `V2WorkerModelVSphereSpec`
- Support the `FlavorRequirement` job requirement type for vSphere models
- Adapt `CanAllocateResources()` to use flavor-defined resources instead of template resources
- Integrate flavors with the pre-provisioning system
- Flavor starvation prevention (reserve capacity for smaller flavors)

### Out of Scope (deferred)

- Disk resize (requires `VirtualDisk.CapacityInKB` manipulation + guest filesystem resize)
- Hot-add CPU/RAM on running VMs

## B.3 Configuration Changes

### B.3.1 New Types

```go
// VSphereFlavorConfig defines the hardware resources for a flavor.
type VSphereFlavorConfig struct {
    CPUs     int32 `mapstructure:"cpus" toml:"cpus" json:"cpus"`
    MemoryMB int64 `mapstructure:"memoryMB" toml:"memoryMB" json:"memoryMB"`
}
```

### B.3.2 New Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `flavors` | `map[string]VSphereFlavorConfig` | `nil` | Map of flavor name → resource definition |
| `defaultFlavor` | `string` | `""` | Default flavor when none is specified |
| `countSmallerFlavorToKeep` | `int` | `0` (disabled) | Reserve capacity for smaller flavors to prevent starvation |

**Note**: `maxCpus` and `maxMemoryMB` are already provided by Amendment A and reused here.

### B.3.3 Example Configuration

```toml
[commonConfiguration.provision]
maxWorker = 0  # unlimited — rely on resource-based capacity (Amendment A)

# Optional static resource limits (from Amendment A)
maxCpus = 64
maxMemoryMB = 131072

# Flavor definitions (Amendment B)
defaultFlavor = "medium"
countSmallerFlavorToKeep = 2

[flavors.small]
cpus = 2
memoryMB = 4096

[flavors.medium]
cpus = 4
memoryMB = 8192

[flavors.large]
cpus = 8
memoryMB = 16384
```

### B.3.4 Worker Provisioning Config Change

Add an optional `Flavor` field:

```go
type WorkerProvisioningConfig struct {
    ModelPath   string
    ModelVMWare string
    Number      int
    Flavor      string  // NEW: flavor to apply to provisioned VMs
}
```

## B.4 Worker Model Changes

### B.4.1 V2 Spec

Add `Flavor` to `V2WorkerModelVSphereSpec`:

```go
type V2WorkerModelVSphereSpec struct {
    Image    string `json:"image"`
    Flavor   string `json:"flavor,omitempty"`    // NEW
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
}
```

Example YAML:
```yaml
name: my-worker-large
type: vsphere
osarch: linux/amd64
spec:
  image: "debian12"
  flavor: "large"
  username: "admin"
  password: "${{ secrets.VM_PASSWORD }}"
```

### B.4.2 V1 Model

The existing `ModelVirtualMachine.Flavor` field is already present in the V1 model struct but
currently unused by the vSphere hatchery. This amendment activates it.

## B.5 Flavor Resolution

Flavor is resolved with the following priority (highest first):

1. **V1**: `Model.ModelVirtualMachine.Flavor`
2. **V2**: `V2WorkerModelVSphereSpec.Flavor`
3. **Job requirement**: `FlavorRequirement` value from job prerequisites
4. **Default**: `HatcheryConfiguration.DefaultFlavor`
5. **None**: No flavor applied → resources inherited from template (pre-Amendment B behavior)

This requires extending `GetFlavor()` in `sdk/hatchery.go` to read from `VSphereSpec.Flavor`.

## B.6 Spawn Changes

### B.6.1 `CanSpawn` Modification

Remove `FlavorRequirement` from the list of unsupported requirements:

```go
// BEFORE
if r.Type == sdk.ServiceRequirement ||
    r.Type == sdk.MemoryRequirement ||
    r.Type == sdk.HostnameRequirement ||
    r.Type == sdk.FlavorRequirement ||
    model.GetCmd() == "" {
    return false
}

// AFTER
if r.Type == sdk.ServiceRequirement ||
    r.Type == sdk.MemoryRequirement ||
    r.Type == sdk.HostnameRequirement ||
    model.GetCmd() == "" {
    return false
}
```

### B.6.2 `prepareCloneSpec` Modification

Accept an optional flavor and apply CPU/RAM overrides to `VirtualMachineConfigSpec`:

```go
func (h *HatcheryVSphere) prepareCloneSpec(ctx context.Context, vm *object.VirtualMachine,
    annot *annotation, flavor *VSphereFlavorConfig) (*types.VirtualMachineCloneSpec, error) {

    // ... existing logic ...

    cloneSpec := &types.VirtualMachineCloneSpec{
        // ... existing fields ...
        Config: &types.VirtualMachineConfigSpec{
            // ... existing fields ...
        },
    }

    // Apply flavor overrides
    if flavor != nil {
        if flavor.CPUs > 0 {
            cloneSpec.Config.NumCPUs = flavor.CPUs
        }
        if flavor.MemoryMB > 0 {
            cloneSpec.Config.MemoryMB = flavor.MemoryMB
        }
    }

    // ... rest of existing logic ...
}
```

### B.6.3 `SpawnWorker` Modification

Resolve flavor before calling `prepareCloneSpec`:

```go
func (h *HatcheryVSphere) SpawnWorker(ctx context.Context, spawnArgs hatchery.SpawnArguments) error {
    // ... existing template resolution ...

    // Resolve flavor
    var flavor *VSphereFlavorConfig
    flavorName := spawnArgs.Model.GetFlavor(spawnArgs.Requirements, h.Config.DefaultFlavor)
    if flavorName != "" {
        if f, ok := h.Config.Flavors[strings.ToLower(flavorName)]; ok {
            flavor = &f
        } else {
            return fmt.Errorf("flavor %q not found in hatchery configuration", flavorName)
        }
    }

    // ... pass flavor to prepareCloneSpec and provisioning ...
}
```

## B.7 Adaptation of `CanAllocateResources` (from Amendment A)

Amendment A's `CanAllocateResources()` determines the next worker's resource footprint by reading
the source template's hardware. With flavors, the footprint is known ahead of time from the flavor
definition, which is both more accurate and avoids a template lookup.

The adaptation is as follows:

```go
func (h *HatcheryVSphere) CanAllocateResources(ctx context.Context,
    model sdk.WorkerStarterWorkerModel, jobID string,
    requirements []sdk.Requirement) (bool, error) {

    // Determine the resource footprint of the next worker
    var nextCPUs int32
    var nextMemoryMB int32

    flavorName := model.GetFlavor(requirements, h.Config.DefaultFlavor)
    if flavorName != "" {
        // Amendment B: use flavor-defined resources
        flavor, ok := h.Config.Flavors[strings.ToLower(flavorName)]
        if !ok {
            return false, fmt.Errorf("unknown flavor %q", flavorName)
        }
        nextCPUs = flavor.CPUs
        nextMemoryMB = int32(flavor.MemoryMB)
    } else {
        // Amendment A fallback: use template resources
        templateName := model.GetVSphereImage()
        var err error
        nextCPUs, nextMemoryMB, err = h.getTemplateResources(ctx, templateName)
        if err != nil {
            log.Warn(ctx, "CanAllocateResources> unable to determine resource footprint: %v", err)
            return true, nil
        }
    }

    // === Primary check: Resource Pool capacity (always enabled, from Amendment A) ===
    canFit, err := h.checkResourcePoolCapacity(ctx, nextCPUs, nextMemoryMB)
    if err != nil {
        log.Warn(ctx, "CanAllocateResources> Resource Pool check failed: %v", err)
    } else if !canFit {
        log.Info(ctx, "CanAllocateResources> Resource Pool capacity insufficient")
        return false, nil
    }

    // === Supplementary checks: static limits (if configured) ===

    usedCPUs, usedMemoryMB := h.countAllocatedResources(ctx)

    // Check CPU limit
    if h.Config.MaxCPUs > 0 {
        if int(usedCPUs) + int(nextCPUs) > h.Config.MaxCPUs {
            log.Info(ctx, "CanAllocateResources> CPU limit reached: %d + %d > %d",
                usedCPUs, nextCPUs, h.Config.MaxCPUs)
            return false, nil
        }

        // Flavor starvation prevention (Amendment B)
        if flavorName != "" && h.Config.CountSmallerFlavorToKeep > 0 {
            smallerCPUs := h.getSmallerFlavorCPUs(flavorName)
            if smallerCPUs > 0 && smallerCPUs != nextCPUs {
                needed := int(nextCPUs) + h.Config.CountSmallerFlavorToKeep * int(smallerCPUs)
                if needed > h.Config.MaxCPUs - int(usedCPUs) {
                    log.Info(ctx, "CanAllocateResources> starvation prevention: need %d CPUs (%d + %d×%d reserve) but only %d left",
                        needed, nextCPUs, h.Config.CountSmallerFlavorToKeep, smallerCPUs, h.Config.MaxCPUs - int(usedCPUs))
                    return false, nil
                }
            }
        }
    }

    // Check memory limit
    if h.Config.MaxMemoryMB > 0 {
        if int64(usedMemoryMB) + int64(nextMemoryMB) > h.Config.MaxMemoryMB {
            log.Info(ctx, "CanAllocateResources> Memory limit reached: %d + %d > %d",
                usedMemoryMB, nextMemoryMB, h.Config.MaxMemoryMB)
            return false, nil
        }
    }

    return true, nil
}
```

**Key insight**: `countAllocatedResources()` (from Amendment A) reads `summary.Config.NumCpu` and
`summary.Config.MemorySizeMB` from actual VMs. When a VM is cloned with a flavor override, its
`summary.Config` reflects the **overridden** values, not the template defaults. Therefore,
Amendment A's resource counting automatically works correctly with resized VMs — no changes to
the counting logic are needed.

### B.7.1 Smaller Flavor Starvation Prevention

```go
func (h *HatcheryVSphere) getSmallerFlavorCPUs(currentFlavorName string) int32 {
    currentFlavor := h.Config.Flavors[strings.ToLower(currentFlavorName)]
    var smallestCPUs int32
    for name, f := range h.Config.Flavors {
        if strings.ToLower(name) == strings.ToLower(currentFlavorName) {
            continue
        }
        if f.CPUs < currentFlavor.CPUs {
            if smallestCPUs == 0 || f.CPUs < smallestCPUs {
                smallestCPUs = f.CPUs
            }
        }
    }
    return smallestCPUs
}
```

## B.8 Provisioning Integration

### B.8.1 Provisioning with Flavors

When `WorkerProvisioningConfig.Flavor` is set, provisioned VMs are created with the specified
flavor's CPU/RAM configuration applied via `prepareCloneSpec`.

### B.8.2 Provisioned VM Matching

`FindProvisionnedWorker` must be extended to match on flavor in addition to model path:

- If the job requests a flavor, only match provisioned VMs with the same flavor in their annotation
- If no flavor is requested and a default flavor is configured, match on the default flavor
- If no flavor is configured at all, match any provisioned VM for the model (current behavior)

### B.8.3 Flavor Mismatch Handling

If a job requests a flavor that differs from available provisioned VMs, the hatchery falls back to
creating a fresh clone with the requested flavor. The provisioned VMs remain available for future
jobs with matching flavors.

### B.8.4 Annotation Change

Add a `Flavor` field to the annotation struct for provisioning matching:

```go
type annotation struct {
    // ... existing fields ...
    Flavor string `json:"flavor,omitempty"` // NEW: flavor name used for this VM
}
```

## B.9 SDK Changes

### B.9.1 `GetFlavor` Extension

```go
func (w WorkerStarterWorkerModel) GetFlavor(reqs RequirementList, defaultFlavor string) string {
    switch {
    case w.ModelV1 != nil:
        if w.ModelV1.ModelVirtualMachine.Flavor != "" {
            return w.ModelV1.ModelVirtualMachine.Flavor
        }
    case w.ModelV2 != nil:
        if w.Flavor != "" {
            return w.Flavor
        }
        if w.OpenstackSpec.Flavor != "" {
            return w.OpenstackSpec.Flavor
        }
        if w.VSphereSpec.Flavor != "" {   // NEW
            return w.VSphereSpec.Flavor
        }
        for _, r := range reqs {
            if r.Type == FlavorRequirement && r.Value != "" {
                return r.Value
            }
        }
    }
    return defaultFlavor
}
```

### B.9.2 `FlavorRequirement` Filter

In `sdk/hatchery/hatchery.go`, allow `FlavorRequirement` for vSphere models:

```go
// BEFORE
if model.Type != sdk.Openstack && r.Type == sdk.FlavorRequirement {

// AFTER
if model.Type != sdk.Openstack && model.Type != sdk.VSphere && r.Type == sdk.FlavorRequirement {
```

## B.10 Backward Compatibility

- If `Flavors` is not set in the configuration, no resize occurs. VMs inherit template resources.
  Amendment A's capacity management continues to work using template-derived resource values.
- If a worker model does not specify a flavor and no `DefaultFlavor` is configured, no resize occurs.
- `countAllocatedResources()` (Amendment A) does not need modification — it reads actual VM
  hardware, which automatically reflects any flavor overrides applied at clone time.

## B.11 Implementation Plan

- [ ] Phase 1 — SDK: Add `Flavor` to `V2WorkerModelVSphereSpec`, extend `GetFlavor()`, update `FlavorRequirement` filter
- [ ] Phase 2 — Configuration: Add `VSphereFlavorConfig`, `Flavors`, `DefaultFlavor`, `CountSmallerFlavorToKeep`
- [ ] Phase 3 — Clone Resize: Modify `prepareCloneSpec()` signature and apply `NumCPUs` / `MemoryMB`
- [ ] Phase 4 — CanAllocateResources: Adapt Amendment A's implementation to prefer flavor-defined resources
- [ ] Phase 5 — Provisioning: Add `Flavor` to `WorkerProvisioningConfig` and annotation, update `FindProvisionnedWorker` matching
- [ ] Phase 6 — Tests: Unit tests for resize, flavor resolution, starvation prevention, provisioning matching

---

# Amendment C: Prometheus Metrics for vSphere Resource Consumption

*This amendment is independent of Amendments A and B but benefits from the resource-counting
functions introduced by Amendment A.*

## C.1 Motivation

The vSphere hatchery currently exposes only generic hatchery metrics (job counts, worker states)
via the `/mon/metrics` Prometheus endpoint. There is no visibility into vSphere-specific resource
consumption: CPU and memory allocated by managed VMs, Resource Pool capacity, or per-worker
resource usage.

Operators need this data to:
- Monitor infrastructure utilization trends over time
- Set up Prometheus alerts (e.g. "Resource Pool memory > 80%")
- Correlate resource usage with job throughput
- Capacity plan based on historical data rather than guesswork

The Swarm hatchery already implements per-worker CPU/memory metrics (`cds/hatchery/worker_cpu`,
`cds/hatchery/worker_memory`). This amendment adds equivalent and additional vSphere-specific
metrics.

## C.2 Scope

### In Scope

- **Per-worker** resource metrics: individual VM vCPUs and memory
- **Hatchery-level** aggregate resource metrics: total vCPUs/memory allocated by this hatchery's
  workers and templates
- **Global pool-level** resource metrics: total vCPUs/memory consumed by ALL VMs in the datacenter
  (including non-CDS VMs and other hatcheries), plus Resource Pool runtime capacity
- VM count metrics: total managed VMs, provisioned VMs
- All metrics exposed on the existing `/mon/metrics` endpoint via OpenCensus/Prometheus exporter

### Out of Scope

- Disk usage metrics
- Network metrics
- Per-job cost/resource attribution
- New HTTP endpoints (reuse existing `/mon/metrics`)

## C.3 Metrics Inventory

The metrics are organized in three observation levels, from fine-grained to global:

### C.3.1 Per-Worker Resource Gauges (Level 1: Individual VMs)

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/worker_vcpus` | Gauge | vCPUs | Number of vCPUs for a worker VM |
| `cds/hatchery/vsphere/worker_memory_mb` | Gauge | MB | Memory allocated to a worker VM |

**Tags**: `service_name`, `service_type`, `worker_name`, `worker_model`, `flavor`

The `flavor` tag is empty if no flavor is applied (or if Amendment B is not implemented).

### C.3.2 Hatchery-Level Aggregate Gauges (Level 2: This Hatchery)

Resources consumed by VMs managed by **this hatchery instance** only (identified by CDS annotation
with matching `HatcheryName`). Includes workers and pre-provisioned VMs. Excludes template VMs.

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/allocated_vcpus` | Gauge | vCPUs | Total vCPUs allocated by this hatchery's VMs |
| `cds/hatchery/vsphere/allocated_memory_mb` | Gauge | MB | Total memory allocated by this hatchery's VMs |
| `cds/hatchery/vsphere/vm_count` | Gauge | count | Total number of VMs managed by this hatchery (workers + provisioned) |
| `cds/hatchery/vsphere/provisioned_vm_count` | Gauge | count | Number of pre-provisioned (idle) VMs |
| `cds/hatchery/vsphere/template_vcpus` | Gauge | vCPUs | Total vCPUs defined by template VMs (annotation `Model=true`) |
| `cds/hatchery/vsphere/template_memory_mb` | Gauge | MB | Total memory defined by template VMs |
| `cds/hatchery/vsphere/template_count` | Gauge | count | Number of template VMs managed by this hatchery |

**Tags**: `service_name`, `service_type`

### C.3.3 Global Pool-Level Gauges (Level 3: Entire vSphere Pool)

Resources consumed by **ALL VMs** visible in the datacenter, regardless of whether they are
managed by CDS. This gives operators visibility into the total infrastructure load, including
non-CDS workloads and VMs from other hatchery instances sharing the same vSphere infrastructure.

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/pool_total_vcpus` | Gauge | vCPUs | Total vCPUs across ALL VMs in the datacenter |
| `cds/hatchery/vsphere/pool_total_memory_mb` | Gauge | MB | Total memory across ALL VMs in the datacenter |
| `cds/hatchery/vsphere/pool_total_vm_count` | Gauge | count | Total number of VMs in the datacenter |

**Tags**: `service_name`, `service_type`

**Data source**: `ListVirtualMachines()` already returns all VMs from `RootFolder` (recursive).
These metrics simply sum `summary.Config.NumCpu` and `summary.Config.MemorySizeMB` across
**all** returned VMs without filtering on CDS annotations.

### C.3.4 Resource Pool Runtime Gauges (Level 3: Infrastructure Capacity)

Direct readings from the vSphere Resource Pool runtime, representing the infrastructure-level
capacity and usage as reported by vSphere itself.

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/resource_pool_cpu_max_mhz` | Gauge | MHz | Resource Pool maximum CPU capacity |
| `cds/hatchery/vsphere/resource_pool_cpu_usage_mhz` | Gauge | MHz | Resource Pool current CPU usage |
| `cds/hatchery/vsphere/resource_pool_cpu_unreserved_mhz` | Gauge | MHz | Resource Pool CPU unreserved for VMs |
| `cds/hatchery/vsphere/resource_pool_memory_max_bytes` | Gauge | bytes | Resource Pool maximum memory capacity |
| `cds/hatchery/vsphere/resource_pool_memory_usage_bytes` | Gauge | bytes | Resource Pool current memory usage |
| `cds/hatchery/vsphere/resource_pool_memory_unreserved_bytes` | Gauge | bytes | Resource Pool memory unreserved for VMs |

**Tags**: `service_name`, `service_type`

### C.3.5 Static Limit Gauges (Configuration Reference)

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/max_vcpus` | Gauge | vCPUs | Configured `maxCpus` limit (0 if unlimited) |
| `cds/hatchery/vsphere/max_memory_mb` | Gauge | MB | Configured `maxMemoryMB` limit (0 if unlimited) |

**Tags**: `service_name`, `service_type`

## C.4 Implementation

### C.4.1 Metrics Struct

```go
type vsphereMetrics struct {
    // Level 1: Per-worker
    WorkerVCPUs    *stats.Int64Measure
    WorkerMemoryMB *stats.Int64Measure

    // Level 2: Hatchery-level aggregate
    AllocatedVCPUs     *stats.Int64Measure
    AllocatedMemoryMB  *stats.Int64Measure
    VMCount            *stats.Int64Measure
    ProvisionedVMCount *stats.Int64Measure
    TemplateVCPUs      *stats.Int64Measure
    TemplateMemoryMB   *stats.Int64Measure
    TemplateCount      *stats.Int64Measure

    // Level 3: Global pool (all VMs in datacenter)
    PoolTotalVCPUs    *stats.Int64Measure
    PoolTotalMemoryMB *stats.Int64Measure
    PoolTotalVMCount  *stats.Int64Measure

    // Level 3: Resource Pool runtime
    ResourcePoolCPUMax        *stats.Int64Measure
    ResourcePoolCPUUsage      *stats.Int64Measure
    ResourcePoolCPUUnreserved *stats.Int64Measure
    ResourcePoolMemMax        *stats.Int64Measure
    ResourcePoolMemUsage      *stats.Int64Measure
    ResourcePoolMemUnreserved *stats.Int64Measure

    // Configuration reference
    MaxVCPUs    *stats.Int64Measure
    MaxMemoryMB *stats.Int64Measure

    // Views (kept for re-registration of per-worker views)
    workerViews []*view.View
}
```

### C.4.2 Metrics Initialization

```go
func (h *HatcheryVSphere) InitVSphereMetrics(ctx context.Context) error {
    // Level 1: Per-worker
    h.metrics.WorkerVCPUs = stats.Int64(
        "cds/hatchery/vsphere/worker_vcpus",
        "vCPUs for a worker VM", stats.UnitDimensionless)
    h.metrics.WorkerMemoryMB = stats.Int64(
        "cds/hatchery/vsphere/worker_memory_mb",
        "memory (MB) for a worker VM", stats.UnitDimensionless)

    // Level 2: Hatchery-level aggregate
    h.metrics.AllocatedVCPUs = stats.Int64(
        "cds/hatchery/vsphere/allocated_vcpus",
        "total vCPUs allocated by this hatchery's VMs",
        stats.UnitDimensionless)
    h.metrics.AllocatedMemoryMB = stats.Int64(
        "cds/hatchery/vsphere/allocated_memory_mb",
        "total memory (MB) allocated by this hatchery's VMs",
        stats.UnitDimensionless)
    h.metrics.VMCount = stats.Int64(
        "cds/hatchery/vsphere/vm_count",
        "total VMs managed by this hatchery",
        stats.UnitDimensionless)
    h.metrics.ProvisionedVMCount = stats.Int64(
        "cds/hatchery/vsphere/provisioned_vm_count",
        "pre-provisioned idle VMs",
        stats.UnitDimensionless)
    h.metrics.TemplateVCPUs = stats.Int64(
        "cds/hatchery/vsphere/template_vcpus",
        "total vCPUs defined by template VMs",
        stats.UnitDimensionless)
    h.metrics.TemplateMemoryMB = stats.Int64(
        "cds/hatchery/vsphere/template_memory_mb",
        "total memory (MB) defined by template VMs",
        stats.UnitDimensionless)
    h.metrics.TemplateCount = stats.Int64(
        "cds/hatchery/vsphere/template_count",
        "number of template VMs",
        stats.UnitDimensionless)

    // Level 3: Global pool (all VMs in datacenter)
    h.metrics.PoolTotalVCPUs = stats.Int64(
        "cds/hatchery/vsphere/pool_total_vcpus",
        "total vCPUs across all VMs in datacenter",
        stats.UnitDimensionless)
    h.metrics.PoolTotalMemoryMB = stats.Int64(
        "cds/hatchery/vsphere/pool_total_memory_mb",
        "total memory (MB) across all VMs in datacenter",
        stats.UnitDimensionless)
    h.metrics.PoolTotalVMCount = stats.Int64(
        "cds/hatchery/vsphere/pool_total_vm_count",
        "total VMs in datacenter",
        stats.UnitDimensionless)

    // Level 3: Resource Pool runtime
    h.metrics.ResourcePoolCPUMax = stats.Int64(
        "cds/hatchery/vsphere/resource_pool_cpu_max_mhz",
        "Resource Pool max CPU in MHz", stats.UnitDimensionless)
    h.metrics.ResourcePoolCPUUsage = stats.Int64(
        "cds/hatchery/vsphere/resource_pool_cpu_usage_mhz",
        "Resource Pool CPU usage in MHz", stats.UnitDimensionless)
    h.metrics.ResourcePoolCPUUnreserved = stats.Int64(
        "cds/hatchery/vsphere/resource_pool_cpu_unreserved_mhz",
        "Resource Pool CPU unreserved for VMs in MHz", stats.UnitDimensionless)
    h.metrics.ResourcePoolMemMax = stats.Int64(
        "cds/hatchery/vsphere/resource_pool_memory_max_bytes",
        "Resource Pool max memory in bytes", stats.UnitDimensionless)
    h.metrics.ResourcePoolMemUsage = stats.Int64(
        "cds/hatchery/vsphere/resource_pool_memory_usage_bytes",
        "Resource Pool memory usage in bytes", stats.UnitDimensionless)
    h.metrics.ResourcePoolMemUnreserved = stats.Int64(
        "cds/hatchery/vsphere/resource_pool_memory_unreserved_bytes",
        "Resource Pool memory unreserved for VMs in bytes", stats.UnitDimensionless)

    // Configuration reference
    h.metrics.MaxVCPUs = stats.Int64(
        "cds/hatchery/vsphere/max_vcpus",
        "configured maxCpus limit", stats.UnitDimensionless)
    h.metrics.MaxMemoryMB = stats.Int64(
        "cds/hatchery/vsphere/max_memory_mb",
        "configured maxMemoryMB limit", stats.UnitDimensionless)

    // Register views
    baseTags := []tag.Key{
        telemetry.MustNewKey(telemetry.TagServiceName),
        telemetry.MustNewKey(telemetry.TagServiceType),
    }
    workerTags := append(baseTags,
        telemetry.MustNewKey("worker_name"),
        telemetry.MustNewKey("worker_model"),
        telemetry.MustNewKey("flavor"),
    )

    h.metrics.workerViews = []*view.View{
        telemetry.NewViewLast("cds/hatchery/vsphere/worker_vcpus", h.metrics.WorkerVCPUs, workerTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/worker_memory_mb", h.metrics.WorkerMemoryMB, workerTags),
    }

    return telemetry.RegisterView(ctx,
        // Level 1: Per-worker
        h.metrics.workerViews[0],
        h.metrics.workerViews[1],
        // Level 2: Hatchery aggregate
        telemetry.NewViewLast("cds/hatchery/vsphere/allocated_vcpus", h.metrics.AllocatedVCPUs, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/allocated_memory_mb", h.metrics.AllocatedMemoryMB, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/vm_count", h.metrics.VMCount, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/provisioned_vm_count", h.metrics.ProvisionedVMCount, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/template_vcpus", h.metrics.TemplateVCPUs, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/template_memory_mb", h.metrics.TemplateMemoryMB, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/template_count", h.metrics.TemplateCount, baseTags),
        // Level 3: Global pool
        telemetry.NewViewLast("cds/hatchery/vsphere/pool_total_vcpus", h.metrics.PoolTotalVCPUs, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/pool_total_memory_mb", h.metrics.PoolTotalMemoryMB, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/pool_total_vm_count", h.metrics.PoolTotalVMCount, baseTags),
        // Level 3: Resource Pool runtime
        telemetry.NewViewLast("cds/hatchery/vsphere/resource_pool_cpu_max_mhz", h.metrics.ResourcePoolCPUMax, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/resource_pool_cpu_usage_mhz", h.metrics.ResourcePoolCPUUsage, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/resource_pool_cpu_unreserved_mhz", h.metrics.ResourcePoolCPUUnreserved, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/resource_pool_memory_max_bytes", h.metrics.ResourcePoolMemMax, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/resource_pool_memory_usage_bytes", h.metrics.ResourcePoolMemUsage, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/resource_pool_memory_unreserved_bytes", h.metrics.ResourcePoolMemUnreserved, baseTags),
        // Configuration reference
        telemetry.NewViewLast("cds/hatchery/vsphere/max_vcpus", h.metrics.MaxVCPUs, baseTags),
        telemetry.NewViewLast("cds/hatchery/vsphere/max_memory_mb", h.metrics.MaxMemoryMB, baseTags),
    )
}
```

### C.4.3 Collection Routine

Following the same pattern as the Swarm hatchery (`StartWorkerMetricsRoutine`), a periodic
goroutine collects and records all metrics:

```go
func (h *HatcheryVSphere) StartVSphereMetricsRoutine(ctx context.Context, delay int64) {
    ticker := time.NewTicker(time.Duration(delay) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            h.GoRoutines.Exec(ctx, "vsphere-metrics", func(ctx context.Context) {
                h.collectVSphereMetrics(ctx)
            })
        case <-ctx.Done():
            return
        }
    }
}

func (h *HatcheryVSphere) collectVSphereMetrics(ctx context.Context) {
    ctx = telemetry.ContextWithTag(ctx, telemetry.TagServiceName, h.Name())
    ctx = telemetry.ContextWithTag(ctx, telemetry.TagServiceType, h.Type())

    // --- Iterate ALL VMs from ListVirtualMachines (returns entire datacenter) ---
    srvs := h.getVirtualMachines(ctx)

    // Level 2: Hatchery-level counters
    var hatcheryCPUs, hatcheryMemMB int64
    var vmCount, provisionedCount int64
    var templateCPUs, templateMemMB, templateCount int64

    // Level 3: Global pool counters (ALL VMs, no annotation filtering)
    var poolCPUs, poolMemMB, poolVMCount int64

    // Re-register per-worker views to drop stale workers
    view.Unregister(h.metrics.workerViews...)
    view.Register(h.metrics.workerViews...)

    for _, s := range srvs {
        cpus := int64(s.Summary.Config.NumCpu)
        memMB := int64(s.Summary.Config.MemorySizeMB)

        // Level 3: count ALL VMs unconditionally
        poolCPUs += cpus
        poolMemMB += memMB
        poolVMCount++

        // Level 2 + Level 1: only for this hatchery's VMs
        annot := getVirtualMachineCDSAnnotation(ctx, s)
        if annot == nil || annot.HatcheryName != h.Name() {
            continue
        }

        if annot.Model {
            // Template VMs
            templateCPUs += cpus
            templateMemMB += memMB
            templateCount++
            continue
        }

        hatcheryCPUs += cpus
        hatcheryMemMB += memMB
        vmCount++

        if strings.HasPrefix(s.Name, "provision-") {
            provisionedCount++
        }

        // Level 1: Per-worker metrics
        wCtx := telemetry.ContextWithTag(ctx, "worker_name", s.Name)
        wCtx = telemetry.ContextWithTag(wCtx, "worker_model", annot.WorkerModelPath)
        flavor := ""
        if annot.Flavor != "" {
            flavor = annot.Flavor
        }
        wCtx = telemetry.ContextWithTag(wCtx, "flavor", flavor)
        stats.Record(wCtx,
            h.metrics.WorkerVCPUs.M(cpus),
            h.metrics.WorkerMemoryMB.M(memMB),
        )
    }

    // Level 2: Hatchery aggregate
    stats.Record(ctx,
        h.metrics.AllocatedVCPUs.M(hatcheryCPUs),
        h.metrics.AllocatedMemoryMB.M(hatcheryMemMB),
        h.metrics.VMCount.M(vmCount),
        h.metrics.ProvisionedVMCount.M(provisionedCount),
        h.metrics.TemplateVCPUs.M(templateCPUs),
        h.metrics.TemplateMemoryMB.M(templateMemMB),
        h.metrics.TemplateCount.M(templateCount),
    )

    // Level 3: Global pool (all VMs in datacenter)
    stats.Record(ctx,
        h.metrics.PoolTotalVCPUs.M(poolCPUs),
        h.metrics.PoolTotalMemoryMB.M(poolMemMB),
        h.metrics.PoolTotalVMCount.M(poolVMCount),
    )

    // Configuration reference
    stats.Record(ctx,
        h.metrics.MaxVCPUs.M(int64(h.Config.MaxCPUs)),
        h.metrics.MaxMemoryMB.M(h.Config.MaxMemoryMB),
    )

    // --- Level 3: Resource Pool runtime ---
    pool, err := h.vSphereClient.LoadResourcePool(ctx)
    if err != nil {
        log.Warn(ctx, "collectVSphereMetrics> Resource Pool load error: %v", err)
        return
    }
    var poolMo mo.ResourcePool
    if err := pool.Properties(ctx, pool.Reference(), []string{"runtime"}, &poolMo); err != nil {
        log.Warn(ctx, "collectVSphereMetrics> Resource Pool properties error: %v", err)
        return
    }

    stats.Record(ctx,
        h.metrics.ResourcePoolCPUMax.M(poolMo.Runtime.Cpu.MaxUsage),
        h.metrics.ResourcePoolCPUUsage.M(poolMo.Runtime.Cpu.OverallUsage),
        h.metrics.ResourcePoolCPUUnreserved.M(poolMo.Runtime.Cpu.UnreservedForVm),
        h.metrics.ResourcePoolMemMax.M(poolMo.Runtime.Memory.MaxUsage),
        h.metrics.ResourcePoolMemUsage.M(poolMo.Runtime.Memory.OverallUsage),
        h.metrics.ResourcePoolMemUnreserved.M(poolMo.Runtime.Memory.UnreservedForVm),
    )
}
```

### C.4.4 Integration Point

`InitVSphereMetrics` is called from `Init()` (after telemetry is initialized).
`StartVSphereMetricsRoutine` is started as a goroutine from `Serve()` alongside the existing
cleanup and provisioning routines.

```go
// In Init() or Serve():
if err := h.InitVSphereMetrics(ctx); err != nil {
    return err
}
go h.StartVSphereMetricsRoutine(ctx, 30) // collect every 30 seconds
```

## C.5 Example Prometheus Queries

```promql
# --- Level 2: Hatchery utilization ---

# vCPU utilization ratio against static limit (requires maxCpus > 0)
cds_hatchery_vsphere_allocated_vcpus / cds_hatchery_vsphere_max_vcpus

# Number of active (non-provisioned) workers
cds_hatchery_vsphere_vm_count - cds_hatchery_vsphere_provisioned_vm_count

# --- Level 3: Global pool visibility ---

# What fraction of the datacenter's VMs belong to this hatchery?
cds_hatchery_vsphere_vm_count / cds_hatchery_vsphere_pool_total_vm_count

# What fraction of datacenter vCPUs are consumed by this hatchery?
cds_hatchery_vsphere_allocated_vcpus / cds_hatchery_vsphere_pool_total_vcpus

# Non-CDS vCPU consumption (other workloads on same infrastructure)
cds_hatchery_vsphere_pool_total_vcpus - cds_hatchery_vsphere_allocated_vcpus
  - cds_hatchery_vsphere_template_vcpus

# Resource Pool memory utilization percentage
cds_hatchery_vsphere_resource_pool_memory_usage_bytes
  / cds_hatchery_vsphere_resource_pool_memory_max_bytes * 100

# Alert: Resource Pool memory > 80%
cds_hatchery_vsphere_resource_pool_memory_unreserved_bytes
  / cds_hatchery_vsphere_resource_pool_memory_max_bytes < 0.2
```

## C.6 Implementation Plan

- [ ] Phase 1 — Struct & Init: Add `vsphereMetrics` struct, implement `InitVSphereMetrics()`
- [ ] Phase 2 — Collection: Implement `collectVSphereMetrics()` and `StartVSphereMetricsRoutine()`
- [ ] Phase 3 — Integration: Wire init and routine into `Init()`/`Serve()`
- [ ] Phase 4 — Tests: Unit tests for metric recording, verify metric names and tags
