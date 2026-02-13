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

# Amendments

The following amendments extend this specification. Each is in a separate file:

- **[Amendment A: Prometheus Metrics for vSphere Resource Consumption](amendment-A-prometheus-metrics.md)** — Observability first. Adds vSphere-specific Prometheus metrics at three levels: per-worker, hatchery-aggregate, and global pool. No behavioral change. Recommended first step.
- **[Amendment B: Resource-Based Capacity Management](amendment-B-resource-capacity.md)** — Replaces the `CanAllocateResources()` stub with real Resource Pool and CPU/memory checks. Makes `MaxWorker` optional (`0` = unlimited).
- **[Amendment C: VM Flavor Support (CPU/RAM Resize)](amendment-C-flavor-resize.md)** — Adds flavor-based VM resize at clone time (CPU/RAM overrides). Builds on Amendment B.
