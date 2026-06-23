# CDS Hatchery vSphere — Specification

## 1. Overview

The CDS Hatchery vSphere is a component of the CDS (Continuous Delivery Service) platform responsible for
automatically spawning CDS workers on a VMware vSphere infrastructure. It creates virtual machines by cloning
VMware templates, boots them, and launches a CDS worker process inside the guest OS via VMware Guest Operations.

The hatchery implements the `hatchery.InterfaceWithModels` interface and runs CDS workers exclusively on
Worker Model V2 (defined as-code in repositories). CDS V1 jobs are still supported, but they are executed on
V2 worker models (resolved via `GetDetaultModelV2Name`). Worker Model V1 (CDS models registered as vSphere
templates) is no longer supported.

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
| `hatchery.go` | Hatchery lifecycle (init, config, CanSpawn), worker cleanup, provisioning scheduler |
| `spawn.go` | VM spawning logic, provisioning clone, worker bootstrap |
| `provision.go` | On-demand provisioning trigger, clone/finish goroutines, reconciliation, `finishProvisioning` |
| `client.go` | VM listing/filtering, clone spec preparation, guest operations |
| `vsphere.go` | `VSphereClient` interface and govmomi SDK wrapper implementation |
| `init.go` | Hatchery initialization, govmomi client creation, background goroutines setup |
| `ip.go` | IP address management (name encode/parse, used-IP set, free-IP selection) |
| `networks.go` | Multi-network initialization (parses networks config, builds IP pools per network) |
| `monitoring.go` | Prometheus metrics: vSphere resource consumption at per-worker, hatchery, and pool levels |

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
| `iprange` | `string` | — | No | **Deprecated**: use `networks` instead. IP range for static IP assignment (format: `a.a.a.a/b,c.c.c.c/e`) |
| `gateway` | `string` | — | No | **Deprecated**: use `networks` instead. Gateway IP for spawned workers |
| `dns` | `string` | — | No | DNS server IP |
| `subnetMask` | `string` | `255.255.255.0` | No | **Deprecated**: use `networks` instead. Subnet mask |
| `networks` | `[]NetworkConfig` | — | No | List of network configurations (see section 3.1.2) |
| `workerTTL` | `int` | `120` | No | Worker time-to-live in minutes |
| `workerRegistrationTTL` | `int` | `10` | No | Worker registration timeout in minutes |
| `workerProvisioningInterval` | `int` | `120` (2 min) | No | Provisioning loop interval in seconds |
| `workerProvisioningPoolSize` | `int` | `0` | No | Optional cap on concurrent provisioning clones. `0` = unbounded (clones run fully in parallel, bounded by the deficit and IP budget) |
| `workerProvisioning` | `[]WorkerProvisioningConfig` | — | No | List of models to pre-provision |
| `guestCredentials` | `[]GuestCredential` | — | No | **Deprecated**: use `models` instead. Guest OS credentials per model |
| `models` | `[]ModelConfig` | — | No | Per-model configuration (credentials, pre-start script) |
| `defaultWorkerModelsV2` | `[]DefaultWorkerModelsV2` | — | No | Default V2 models used to run V1 jobs (binary matching) |

### 3.1.2 Networks Configuration

The `networks` field allows configuring multiple IP ranges, each with its own gateway and subnet mask.
When a VM is spawned, the hatchery picks the first available IP across all configured networks and
applies the corresponding gateway and subnet mask.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `iprange` | `string` | — | IP range in CIDR notation (format: `a.a.a.a/b,c.c.c.c/e`) |
| `gateway` | `string` | — | Gateway IP for this network |
| `subnetMask` | `string` | `255.255.255.0` | Subnet mask for this network |

If `networks` is set, the legacy `iprange`, `gateway`, and `subnetMask` fields are ignored.
If `networks` is empty and the legacy `iprange` is set, it is treated as a single-entry networks list.

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
    ModelVMWare string  // VMware template name (e.g. "debian12")
    Number      int     // Number of VMs to keep pre-provisioned
}
```

### 3.3 Model Configuration

```go
type ModelConfig struct {
    ModelVMWare    string  // VMware template name (e.g. "debian12")
    Username       string  // Guest OS username
    Password       string  // Guest OS password
    PreStartScript string  // Shell script executed inside the VM before the worker starts
}
```

The `models` configuration replaces the deprecated `guestCredentials` and adds support for a
pre-start script that runs inside the VM after boot but before the CDS worker starts. This can
be used for filesystem resizing, environment setup, or any other initialization.

**Credential resolution order**:
1. `models` config, matched by `ModelVMWare` (VMware template name)
2. Deprecated `guestCredentials` config (fallback)
3. Worker model spec (`Username`/`Password`)

#### 3.3.1 Guest Credentials (Deprecated)

```go
type GuestCredential struct {
    ModelVMWare string  // VMware template name
    Username    string  // Guest OS username
    Password    string  // Guest OS password
}
```

Kept for backward compatibility. Prefer `models` for new configurations.

### 3.4 Default Worker Models V2

```go
type DefaultWorkerModelsV2 struct {
    WorkerModelV2 string    // V2 worker model reference
    Binaries      []string  // If a job binary requirement matches, use this model
}
```

Used to bridge V1 jobs (which select models by binary requirements) to V2 worker models.

## 4. Worker Models

The vSphere hatchery only supports **Worker Model V2**. Worker Model V1 (CDS models that were registered
*into* vSphere as templates) is no longer supported: the hatchery never creates, registers, or manages
templates on behalf of CDS models. Templates must already exist in vSphere.

CDS **V1 jobs** are still supported; they simply run on a V2 worker model resolved via
`GetDetaultModelV2Name` (see section 6.8).

### 4.1 Worker Model V2

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
`models` configuration (which takes precedence over the model spec).

## 5. VM Annotations

Every VM created by the hatchery carries a JSON annotation stored in `VirtualMachineConfigSpec.Annotation`.
This annotation is the primary mechanism for tracking VM state and ownership.

```go
type annotation struct {
    HatcheryName    string    // Name of the hatchery that created this VM
    WorkerName      string    // CDS worker name assigned to this VM
    Provisioning    bool      // True while the VM is a pre-provisioned idle worker (cleared when claimed)
    WorkerModelPath string    // CDS worker model path
    VMwareModelPath string    // VMware template name (V2)
    Model           bool      // True if VM is a model template (do not destroy)
    Created         time.Time // Provision clone timestamp
    WorkerStartTime time.Time // Time the VM was claimed for a job (turned into a worker)
    JobID           string    // CDS job ID assigned to this worker
    IPAddress       string    // Static IP assigned to this VM
}
```

All hatchery operations (cleanup, provisioning lookup, duplicate detection) rely on parsing these
annotations from `VirtualMachine.Config.Annotation`.

The annotation is set at clone time, and **updated in place** (via `ReconfigVM_Task`, the same
mechanism used for flavor reconfiguration) when a provision is claimed for a job. Note the
distinction between the two timestamps:

- `Created` is the **provision clone time** — it can be arbitrarily old (a provision may sit
  pooled for a long time), so it is not a reliable indicator of worker activity.
- `WorkerStartTime` is stamped when the provision is **claimed and turned into a worker**. Being
  persisted in the annotation, it survives a hatchery restart and never ages out like vSphere
  events do. `killAwolServers` uses it to decide when a VM can be reclaimed (see §8.1).

## 6. Lifecycle

### 6.1 Initialization

On startup (`InitHatchery`), the hatchery:

1. Initializes the common hatchery subsystem
2. Creates a govmomi client connected to the vSphere endpoint
3. Instantiates the `VSphereClient` wrapper bound to the configured datacenter
4. Parses the IP range (if configured) into a list of available IP addresses
5. Starts background goroutines:
   - **Provisioning loop** (if `workerProvisioning` is configured): runs `provisioningV2` every `workerProvisioningInterval` seconds **and** on demand via the signal channel; each run computes the deficit and launches clones fire-and-forget (§6.3.1). An optional semaphore (`WorkerProvisioningPoolSize > 0`) caps concurrent clones.
   - **Kill awol servers loop**: runs every `killAwolServersInterval` seconds, removes stale/expired VMs
   - **Kill disabled workers loop**: runs every 2 minutes, removes disabled workers

### 6.2 Spawning a Worker

Workers are **only** started from pre-provisioned VMs: there is no fallback cloning the
template at spawn time. `CanSpawn` only accepts a job when a provisioned VM is available
for the model, and `SpawnWorker` fails if none can be claimed (e.g. when another job claimed
it in the meantime — the job is then rescheduled).

The main spawn flow (`SpawnWorker`) proceeds as follows:

```
SpawnWorker(spawnArgs)
│
├── 1. Claim a pre-provisioned VM (FindProvisionnedWorker)
│   └── None available: FAIL (job will be rescheduled)
│
├── 2. Start the claimed VM
│   ├── If flavor: reconfigure (CPU/RAM/disk) while powered off
│   ├── Rename to the worker name
│   ├── Stamp the annotation with WorkerStartTime (+ JobID, Provisioning=false)
│   │   before power-on, preserving the reserved IP and VMware model path
│   ├── Power on (bounded retry, tolerates a VM not yet startable after reconfigure)
│   ├── Wait for the reserved IP
│   ├── Wait for guest operations readiness, run the pre-start script (if any)
│   └── Launch worker script via guest operations → DONE
│
└── 3. Error handling: a single deferred teardown releases the provision from the
       in-use cache and, on ANY failure above, shuts the VM down and marks it for
       deletion — so a partially-configured worker is never left holding an IP.
```

The `WorkerStartTime` stamp is written **before** power-on so that, even if the hatchery crashes
mid-spawn (the deferred teardown would not run), the VM still carries a start time and is cleaned
up by `killAwolServers` (see §8.1).

#### 6.2.1 Clone Specification (`prepareCloneSpec`)

The clone specification defines how a provisioned VM is created (used by the provisioning
scheduler, see §6.3):

- **Network**: Loads the first ethernet card from the template's devices, reconfigures it with the
  configured card type and network backing
- **Resource Pool**: Uses the datacenter's default resource pool
- **Datastore**: Relocates the VM to the configured datastore
- **Disk**: Uses `MoveChildMostDiskBacking` disk move type (linked clone)
- **Customization**: Linux prep with auto-generated hostname. If IP range is configured, assigns
  a static IP with subnet mask, gateway, and DNS
- **Annotation**: Serializes the `annotation` struct as JSON into `VirtualMachineConfigSpec.Annotation`
- **Power On**: `PowerOn: true`, the VM boots immediately to complete its provisioning
- **VM Tools**: Configured to run after power on

**Important**: CPU, RAM, and disk size are **not** specified in the clone spec. All hardware
reconfiguration (CPU, RAM, disk) is handled by `reconfigureVM` on the powered-off provisioned
VM when it is claimed by `SpawnWorker`.

#### 6.2.2 Worker Script Launch (`launchScriptWorker`)

After the VM is cloned and has obtained an IP:

1. Wait for the VM IP address to be available
2. Check VM readiness by running `env` command via guest operations
3. Execute the **pre-start script** (if configured in `models` for this model) — runs inside the
   guest before the worker starts, useful for filesystem resizing or environment preparation
4. Generate worker configuration (API endpoint, tokens, etc.)
5. Build the launch script: `PreCmd + Cmd + PostCmd`, templated with worker config
6. Execute the launch script via `StartProgramInGuest` with guest credentials
7. The script is run as: `/bin/echo -n ;<script>`, with `CDS_CONFIG` passed as environment variable

#### 6.2.3 Guest Operations Authentication

Guest OS credentials are resolved in order:
1. From `models` config, matched by `ModelVMWare` (VMware template name)
2. From deprecated `guestCredentials` config (fallback)
3. If not found in config, from the worker model spec (`Username`/`Password`)

If neither provides valid credentials, spawning fails.

### 6.3 Pre-Provisioning

Pre-provisioning creates idle VMs ahead of time so that job assignment is faster.

#### 6.3.1 Provisioning (`provisioningV2`)

Provisioning is keyed on the VMware template name (`provision-v2` VMs). `provisioningV2` is the
trigger handler: it runs on the provisioning ticker **and** on demand whenever a provision is
consumed — `requestProvisioning()` is called after every spawn claim and after `killAwolServers` deletes
anything (it does a non-blocking send on a size-1 signal channel, coalescing bursts into a single
pending run). Each run is **fire-and-forget**: it computes what is missing, submits
the clones as independent goroutines, and returns immediately so the trigger loop stays responsive.

**Step 1 — Compute the deficit (vSphere is the source of truth):**
1. List `provision-v2` VMs and count per VMware model path, **including all power states**:
   - **READY** = powered off, `Provisioning: true` → a finished, claimable provision
   - **STARTING** = powered on, `Provisioning: true` → a provision still being created
2. Also count **in-flight clones not yet visible** in the inventory — entries in
   `cacheProvisioning.pending` (name → model) whose VM does not yet appear in the list.
3. `deficit(model) = configured Number − READY − STARTING − not-yet-listable pending`.
4. **Deprovisioning**: for each model with more provisioned VMs than configured (or models removed
   from config), mark excess VMs for deletion.

Counting READY + STARTING from vSphere is what makes the deficit correct after a restart with an
empty cache (see §6.3.3); `pending` only additionally covers the brief window where a clone has been
launched but its VM is not yet listable, so a rapid retrigger never double-creates.

**Step 2 — Interleave models using round-robin:**
- Uses `roundRobinInterleave()` to produce a fair ordering where models are picked one task at a time in config order
- Prevents models with larger counts from monopolizing the provisioning queue
- Example: deficits A=3, B=1, C=2 → queue `[A, B, C, A, C, A]`

**Step 3 — Assign IPs and launch clones:**
- **Reconciliation**: resume orphaned in-flight provisions (see §6.3.3).
- When an IP range is configured, build the in-use IP set (`getUsedIPs` over the VM list + the IPs of
  in-flight clones parsed from `pending` names). For each model in the queue, pick a distinct free IP
  with `pickFreeIP` (adding it to the set so the next pick differs); stop when no IP remains — the IP
  range is the budget (see §7.3).
- Launch a clone per model via `startProvisionClone(model, name, ip)` — one goroutine, **no waiting
  for the batch**. `WorkerProvisioningPoolSize > 0` caps concurrent clones (a semaphore); `0`
  maximizes parallelism.

Each clone:
- Uses the caller-chosen name `provision-v2-ip-<dashed-ip>-<random>` (the IP-encoded lock, see §7.1;
  DHCP mode uses a plain `provision-v2-<random>`), recorded in `cacheProvisioning.pending` **before**
  the goroutine starts.
- Calls `ProvisionWorkerV2()` (clone with that IP), then `finishProvisioning()` (wait IP → shutdown).
- On any failure marks the VM for deletion; in all cases removes its name from `pending` when done.

#### 6.3.2 Provisioned VM Lifecycle

A provisioned VM follows this lifecycle:

```
Template ──clone──► Provisioned VM (powered on)
                    │
                    ├── Wait for IP (3 min timeout)
                    ├── Shutdown (stays in powered-off state)
                    │
                    └── On job assignment (FindProvisionnedWorker):
                        ├── Rename to worker name
                        ├── Power on
                        ├── Wait for IP
                        └── Launch worker script
```

#### 6.3.3 Provisioning State Reconciliation

`cacheProvisioning.pending` is in-memory only and **must be safe to lose at any time** (restart or
crash). Correctness never relies on it surviving: the deficit (§6.3.1) is computed from the vSphere
VM list, which is the source of truth. On restart the cache is empty, but every real provision is
still classifiable by power state — **READY** (powered off) vs **STARTING** (powered on) — so the
deficit is reconstructed exactly and no duplicates are created.

The open question on restart is what to do with the **STARTING** provisions (powered on, mid-creation
when the old process died). The hatchery **reuses** them: each provisioning run calls
`reconcileProvisionedVMs()` to resume their provisioning rather than discarding the in-progress clone:

1. List powered-on VMs with the `provision-` prefix, `Provisioning: true`, and matching `HatcheryName`
2. Skip VMs already in `cacheProvisioning.pending` (already being handled this process)
3. For each remaining orphan, resume it via `startProvisionFinish()` (a fire-and-forget goroutine):
   - Record it in `cacheProvisioning.pending`
   - Run `finishProvisioning`: `WaitForVirtualMachineIP` → `ShutdownVirtualMachine`
   - If it already has its IP, this returns quickly and the VM is shut down → becomes READY
   - If it never gets an IP, the IP-wait timeout fires and the VM is marked for deletion (and
     recreated by a later deficit) — so a genuinely stuck provision is never reused forever
   - On completion (success or failure) the VM is removed from `cacheProvisioning.pending`

Because the deficit counts STARTING provisions, the in-flight ones being reconciled are not also
cloned fresh.

This mechanism also covers VMs where `ShutdownVirtualMachine` failed during normal operation
(not just restarts), since the provisioning scheduler periodically re-runs reconciliation.

#### 6.3.4 Finding a Provisioned Worker (`FindProvisionnedWorker`)

When spawning a worker, the hatchery tries to reuse a pre-provisioned VM:

1. Determine expected model path (the VMware template / image name)
2. Iterate all VMs, filtering by the `provision-v2` prefix
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

The `MaxWorker` configuration sets a ceiling on the number of concurrent workers. Its behavior is:

- **`MaxWorker > 0`**: Hard limit. If the worker pool size (registered + pending) is **greater than or equal**
  to `MaxWorker`, no new worker can be spawned. The check returns `false` and the job is skipped.
  
- **`MaxWorker = 0`**: **Unlimited**. No worker count limit is enforced, and capacity management
  relies entirely on resource-based checks (see Section 12).

```go
// Framework logic (sdk/hatchery/provisionning.go)
if h.Configuration().Provision.MaxWorker > 0 && len(workerPool) >= h.Configuration().Provision.MaxWorker {
    return false  // capacity reached
}
```

**Status Display**: When `MaxWorker = 0`, the hatchery status displays "N/unlimited" instead of "N/0".

This limit applies uniformly to:
- **Job V1 processing** (`processJobV1QueueV1`)
- **Job V2 processing** (`processJobV2`)

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

A common-framework limit controlling the maximum number of worker models being registered
simultaneously. The vSphere hatchery no longer registers worker models (see §6.7), so this
limit has no practical effect for this hatchery; it is kept for framework compatibility.

#### 6.4.5 Configuration Validation

At startup, the hatchery validates that (if `MaxWorker > 0`):
- `MaxConcurrentProvisioning <= MaxWorker`
- `MaxConcurrentRegistering <= MaxWorker`

When `MaxWorker = 0`, these validations are skipped (unlimited mode).

#### 6.4.6 Interaction with vSphere Pre-Provisioning

The `MaxWorker` limit and the vSphere-specific `WorkerProvisioning` (pre-provisioning pool) are
**independent mechanisms** that interact as follows:

- Pre-provisioned VMs (named `provision-*`) are **excluded** from `WorkersStarted()` and therefore
  **do not count** against `MaxWorker`.
- When a provisioned VM is assigned to a job, it is renamed (e.g. `provision-v2-xxx` →
  `worker-abc`). From that point, it **counts** against `MaxWorker`.
- The `WorkerProvisioningPoolSize` config optionally caps how many provision clones run
  concurrently (`0` = unbounded — clones run fully in parallel, bounded only by the deficit and IP
  budget). This is separate from `MaxConcurrentProvisioning`, which governs the common framework's
  capacity check.
- There is **no global coordination** between provisioning and `MaxWorker`. It is the
  operator's responsibility to ensure that `WorkerProvisioning[].Number` (total pre-provisioned
  VMs) plus expected active workers stays within the infrastructure's capacity.

#### 6.4.7 Monitoring

The `Status()` method reports the current worker count vs. MaxWorker:
```
Workers: <current>/<maxWorker>   (or "<current>/unlimited" if MaxWorker=0)
```

#### 6.4.8 Resource-Based Capacity (`CanAllocateResources`)

In addition to the count-based checks (MaxWorker, MaxConcurrentProvisioning, MaxConcurrentRegistering),
the hatchery also performs **resource-based capacity checks** to ensure the vSphere infrastructure can
handle the next worker.

See **Section 12 (Resource-Based Capacity Management)** for full details.

Quick summary:
- Queries Resource Pool runtime (`UnreservedForVm`) to check available CPU and memory
- Optionally enforces static `maxCpus` and `maxMemoryMB` configuration limits
- Graceful degradation if Resource Pool query fails (continues with static limits only)

This resource-aware check happens **before** every `SpawnWorker()` call.

### 6.5 Spawn Eligibility (`CanSpawn`)

Before spawning, the hatchery checks:

1. **Model type**: Must be a `vsphere` V2 model. Worker model V1 is rejected outright (returns `false`).
2. **Unsupported requirements**: Returns `false` if any requirement is of type:
   - `ServiceRequirement`
   - `MemoryRequirement`
   - `HostnameRequirement`
   - `FlavorRequirement`
3. **Empty Cmd**: Returns `false` if the model has no command defined
4. **Duplicate job check**: Ensures no existing VM annotation references the same `JobID`
5. **Pending job check**: Ensures the job ID is not in the local `cachePendingJobID`
6. **Provisioned worker availability** (`hasAvailableProvisionedWorker`): a provisioned VM
   matching the model must be ready to be claimed (powered off, not pending/used/marked for
   deletion). Workers are only started from provisioned VMs, and a provisioned VM already
   holds its own IP (reserved at clone time) — so no free-IP check is performed: it would
   wrongly refuse to spawn when the whole IP range is held by provisioned machines, which is
   the nominal situation.

### 6.6 Resource Allocation (`CanAllocateResources`)

The current implementation is a no-op stub:

```go
func (h *HatcheryVSphere) CanAllocateResources(...) (bool, error) {
    return true, nil
}
```

No resource limits (CPU, RAM, disk) are verified before spawning.

### 6.7 Model Registration (`NeedRegistration`)

Worker model V1 is no longer supported, so the hatchery never registers CDS models as vSphere
templates. `NeedRegistration` always returns `false`.

### 6.8 Default V2 Model Selection — V1 Jobs on V2 Models

For V1 jobs that need to run on a V2 model (`GetDetaultModelV2Name`):

1. If no binary requirements exist in the job, returns the first configured default V2 model
2. Otherwise, iterates `DefaultWorkerModelsV2` and returns the first model whose `Binaries` list
   contains at least one of the job's binary requirements
3. Returns empty string if no match is found

## 7. IP Address Management

### 7.1 IP Lifecycle

When IP ranges are configured (via `networks` or legacy `iprange`), each provision is assigned a
static IP at clone time through vSphere guest customization. The IP is:

1. **Chosen** by `provisioningV2` (the single provisioning goroutine) via `pickFreeIP`, which returns
   the first IP not currently in use (see §7.2). Choosing in one goroutine means parallel clones can
   never pick the same IP.
2. **Locked by the VM name**: the provision is named `provision-v2-ip-<o1>-<o2>-<o3>-<o4>-<rand>`. The
   cloning VM is listed in the vSphere inventory **with its name as soon as the clone starts** —
   before its annotation is populated — so the IP is immediately visible as "in use" to every other
   pass, and this lock survives a hatchery restart. There is no in-memory reservation and no timeout.
3. **Stored** in the VM annotation (`annot.IPAddress`) as part of the clone spec — the compatibility
   anchor (see §7.5), used by the spawn claim's `WaitForVirtualMachineIP`.
4. **Applied** by vSphere guest customization when the VM first boots.

The IP persists for the VM's lifetime: a provisioned VM powered off and later restarted for a job
comes back with the **same IP** (baked into the guest via customization).

### 7.2 IP Ownership and Counting (`getUsedIPs`)

An IP is "in use" if it appears in **any** non-template VM, from any of three sources:

- **Provision name** (`provision-v2-ip-...`): covers a provision *during its clone*, before the
  annotation exists, and after a restart (parsed straight from the live VM list).
- **VM annotation** (`annot.IPAddress`): covers claimed workers (renamed away from the IP-encoded
  name), old-style provisions created before this scheme, and any VM whose guest tools have not yet
  reported the IP.
- **Guest network info** (`Guest.Net[].IpAddress`): covers running VMs where the OS has the IP active.

Powered-off provisioned VMs therefore **hold their IP** (counted as in use), correctly preventing
reallocation.

### 7.3 IP Budget in Provisioning

`provisioningV2` builds the in-use set with `getUsedIPs` over the current VM list, plus the IPs of
in-flight clones not yet listed (parsed from the `cacheProvisioning.pending` names), then picks a
distinct free IP per clone with `pickFreeIP`, adding each pick to the set as it goes. When no free IP
remains it stops launching clones — the IP range is the natural budget, no separate counter needed.

### 7.4 Why the name, not a reservation

The destination VM is not in the inventory until partway through the clone, and its annotation only
appears when the clone completes — but its **name** is visible almost immediately. Encoding the IP in
the name makes the IP lock readable from vSphere during the whole clone and across restarts, which an
in-memory reservation (lost on restart, and needing a fragile TTL longer than the clone) could not
provide. The annotation still carries the IP for backward/forward compatibility (§7.5).

### 7.5 Compatibility

- **Old-style provisions** (named `provision-v2-<rand>`, no IP token): their IP is read from the
  annotation, so they are counted and handled exactly as before — name parsing is purely additive.
- **Rollback**: new provisions also store the IP in `annot.IPAddress` and keep the `provision-v2`
  prefix, so a previous (reservation-based) binary handles them transparently.

### 7.6 IP-less Mode (DHCP)

When no IP range is configured, no static IP is assigned and provisions keep the plain
`provision-v2-<rand>` name. `pickFreeIP` is not consulted and provisioning is not IP-bounded; VMs
rely on DHCP or template-defined network settings.

## 8. Cleanup and Garbage Collection

### 8.1 Kill Awol Servers (`killAwolServers`) — Every 2 minutes

This routine relies on the `WorkerStartTime` annotation stamp (see §5) and the VM's **live power
state**, not on vSphere events. Events were previously used to find a VM's start time, but they age
out of vCenter's event retention; a VM whose start event had expired (or never existed, e.g. a spawn
that crashed after rename) was then kept forever, accumulating powered-off VMs that each hold a
reserved IP until the pool exhausts its IP range. Using the persisted start time avoids that.

For each VM with a CDS annotation belonging to this hatchery:

1. **Marked for deletion**: Delete immediately
2. **Provisioned VMs** (`provision-` prefix): Skip (still pooled, holding their reserved IP on
   purpose; managed by provisioning loop reconciliation, see §6.3.3)
3. **Model templates** (`Model: true`): Skip (never delete)
4. **Determine the start time**: `WorkerStartTime` from the annotation, or — for VMs created before
   this field existed (upgrade/transition) — the VM's vSphere `CreateDate` as a fallback. If neither
   is available (should not happen), the VM is kept and a warning is logged.
5. **Decide expiry** from the live power state:
   - **Powered off**: a claimed worker is never restarted, so it has finished or failed. Delete once
     `startTime + WorkerRegistrationTTL` has passed. The short grace frees the IP quickly while still
     covering the brief rename→power-on window of an in-flight spawn (whose `WorkerStartTime` is "now").
   - **Powered on and registered on the API**: running worker → delete if `startTime + WorkerTTL` has expired.
   - **Powered on but not on the API**: still booting/registering → delete if `startTime + WorkerRegistrationTTL` has expired.

#### Backward compatibility (VMs created before this change)

No migration step is required — VMs that predate the `WorkerStartTime` stamp are handled
transparently:

- **Existing provisions** (`provision-` prefix, no `WorkerStartTime`): unaffected. They are still
  skipped by `killAwolServers` (rule 2), and the first time the new code claims one it stamps the
  annotation in place (`ReconfigVM_Task`), so it transitions to the new scheme automatically.
- **Existing workers** (already claimed/renamed before the change, so no `WorkerStartTime`):
  `killAwolServers` falls back to the VM's vSphere `CreateDate` (rule 4). This means they are still
  evaluated and eventually cleaned up instead of being kept forever — including the orphaned
  powered-off VMs this change was introduced to reclaim. Because `CreateDate` is the provision clone
  time (older than the real start), such a legacy VM may be reaped slightly earlier than its
  `WorkerStartTime`-based expiry would dictate; this only affects VMs spawned before the upgrade and
  is a one-time transitional effect.

### 8.2 Kill Disabled Workers (`killDisabledWorkers`) — Every 2 minutes

1. Fetch the pool of disabled workers from the CDS API
2. For each disabled worker, find the matching VM by name
3. Mark matching VMs for deletion

### 8.3 Server Deletion (`deleteServer`)

1. Load the VM object
2. If the VM is powered on, power it off
3. Remove from `cacheToDelete`
4. Destroy the VM via vSphere API

## 9. vSphere Client Interface

The hatchery interacts with vSphere through the `VSphereClient` interface, which wraps the
govmomi SDK. Most API calls use a 15-second request timeout; the long-running task waits — clone,
power-off, destroy — use a `vmTaskTimeout` of 10 minutes so a stuck vCenter task fails (and is
recovered) instead of hanging the caller forever.

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
    SetVirtualMachineAnnotation(ctx, vm, annotation) error
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
| `cacheProvisioning.pending` | `map[string]string` | In-flight provisions (VM name → VMware model); accelerator only, safe to lose on restart |
| `cacheProvisioning.using` | `[]string` | Provisioned VM names being assigned to a job |
| `cacheToDelete` | `[]string` | VM names marked for deletion by spawn logic |
| `availableIPAddresses` | `[]string` | All IPs parsed from the configured range |

## 11. Limitations

1. **Unsupported job requirements**: Service, Memory, Hostname requirements
   cause the hatchery to reject the job.
2. **Linux only**: Customization assumes Linux guests (`CustomizationLinuxPrep`).
3. **Single datacenter**: The hatchery operates on a single vSphere datacenter.
4. **No V2 model registration**: V2 templates must be pre-created in vSphere manually.

## 12. Resource-Based Capacity Management

The hatchery implements **resource-aware capacity management** that goes beyond simple worker counts.
Instead of relying solely on `MaxWorker` (a fixed count limit), the hatchery can check actual
vSphere infrastructure capacity before spawning workers.

### 14.1 Capacity Management Mechanisms

The hatchery uses a **layered approach** with three capacity checks (in order of priority):

1. **MaxWorker count** (optional) — Simple worker count ceiling. Set to `0` for unlimited.
2. **Resource Pool runtime** (primary, always enabled) — Queries vSphere Resource Pool's
   `UnreservedForVm` capacity for CPU and memory to ensure infrastructure can handle the next worker.
3. **Static resource limits** (supplementary, optional) — `maxCpus` and `maxMemoryMB` configuration
   fields provide explicit hatchery-level ceilings.

The effective capacity is: `min(MaxWorker (if >0), Resource Pool capacity, static limits (if >0))`

### 14.2 Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `maxWorker` | `int` | `10` | Maximum worker count. Set to `0` for unlimited (rely on resource-based capacity only) |
| `maxCpus` | `int` | `0` (unlimited) | Optional. Maximum total vCPUs this hatchery may allocate. `0` means no static CPU limit |
| `maxMemoryMB` | `int64` | `0` (unlimited) | Optional. Maximum total RAM (MB) this hatchery may allocate. `0` means no static memory limit |

**Example configuration:**

```toml
[hatchery.vsphere]
  user = "admin@vsphere.local"
  endpoint = "pcc-11-222-333-444.ovh.com"
  password = "secret"
  datacenterString = "pcc-11-222-333-444_datacenter1234"
  
  # Optional static resource limits (0 = no static limit, vSphere-specific)
  maxCpus = 64
  maxMemoryMB = 131072    # 128 GB

  [hatchery.vsphere.commonConfiguration.provision]
    # MaxWorker = 0 means unlimited — rely on resource-based capacity
    maxWorker = 0
```

### 14.3 Capacity Strategy Matrix

Operators can choose their strategy:

| MaxWorker | maxCpus/maxMemoryMB | Strategy |
|-----------|---------------------|----------|
| `10` (default) | `0` / `0` | **Count-based only** — Resource Pool is a safety net |
| `0` | `0` / `0` | **Resource Pool only** — rely entirely on vSphere infrastructure limits |
| `0` | set | **Resource-based** — static limits + Resource Pool checks |
| `20` | set | **Belt and suspenders** — count ceiling + resource precision + infrastructure guardrail |

### 14.4 Implementation Details

- `countAllocatedResources()` — Iterates all VMs owned by this hatchery (annotation filter),
  sums `summary.Config.NumCpu` and `summary.Config.MemorySizeMB`. Excludes template VMs and
  **powered-off VMs** (provisioned workers waiting for a job do not consume CPU/RAM in vSphere
  Resource Pools, so they are not counted toward static limits).
- `getTemplateResources()` — Reads CPU/RAM from a vSphere template to estimate the footprint
  of the next worker. Used as fallback when no flavor is specified.
- `checkResourcePoolCapacity()` — Queries `ResourcePool.Runtime.Cpu.UnreservedForVm` and
  `Memory.UnreservedForVm` to verify infrastructure can handle the next worker.
- `CanAllocateResources()` — Combines all three checks with graceful degradation (if Resource Pool
  query fails, falls back to static limits only). When a flavor is requested, **flavor resources
  are used** (not template resources) for an accurate capacity estimate.

## 13. Prometheus Metrics for vSphere Resource Consumption

The hatchery exposes vSphere-specific Prometheus metrics on the existing `/mon/metrics` endpoint
via the OpenCensus/Prometheus exporter. These metrics provide visibility into resource consumption
at three observation levels: per-worker, hatchery-aggregate, and global pool.

Operators can use this data to:
- Monitor infrastructure utilization trends over time
- Set up Prometheus alerts (e.g. "Resource Pool memory > 80%")
- Correlate resource usage with job throughput
- Capacity plan based on historical data rather than guesswork

A background goroutine (`startVSphereMetricsRoutine`) collects metrics every 30 seconds by
iterating all VMs returned by `ListVirtualMachines()` (entire datacenter) and reading the
Resource Pool runtime properties.

### 14.1 Per-Worker Resource Gauges (Level 1: Individual VMs)

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/worker_vcpus` | Gauge | vCPUs | Number of vCPUs for a worker VM |
| `cds/hatchery/vsphere/worker_memory_mb` | Gauge | MB | Memory allocated to a worker VM |
| `cds/hatchery/vsphere/worker_disk_gb` | Gauge | GB | Total disk capacity for a worker VM |

**Tags**: `service_name`, `service_type`, `worker_name`, `worker_model`

Per-worker views are unregistered and re-registered each collection cycle to drop stale workers
that no longer exist (same pattern as the Swarm hatchery).

### 14.2 Hatchery-Level Aggregate Gauges (Level 2: This Hatchery)

Resources consumed by VMs managed by **this hatchery instance** only (identified by CDS annotation
with matching `HatcheryName`). Includes workers and pre-provisioned VMs. Excludes template VMs.

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/allocated_vcpus` | Gauge | vCPUs | Total vCPUs allocated by this hatchery's VMs |
| `cds/hatchery/vsphere/allocated_memory_mb` | Gauge | MB | Total memory allocated by this hatchery's VMs |
| `cds/hatchery/vsphere/vm_count` | Gauge | count | Total number of VMs managed by this hatchery (workers + provisioned) |
| `cds/hatchery/vsphere/worker_vm_count` | Gauge | count | VMs running as workers (claimed, no longer in the provision pool) |
| `cds/hatchery/vsphere/provisioned_vm_count` | Gauge | count | Pre-provisioned VMs (sum of ready + starting + dying) |
| `cds/hatchery/vsphere/provision_ready_count` | Gauge | count | Provisioned VMs powered off and ready to be claimed |
| `cds/hatchery/vsphere/provision_starting_count` | Gauge | count | Provisioned VMs powered on, still being created/finished (not yet claimable) |
| `cds/hatchery/vsphere/provision_dying_count` | Gauge | count | Provisioned VMs marked for deletion, not yet reaped |
| `cds/hatchery/vsphere/provision_inflight_count` | Gauge | count | In-flight provision clones not yet visible in the inventory |
| `cds/hatchery/vsphere/template_vcpus` | Gauge | vCPUs | Total vCPUs defined by template VMs (annotation `Model=true`) |
| `cds/hatchery/vsphere/template_memory_mb` | Gauge | MB | Total memory defined by template VMs |
| `cds/hatchery/vsphere/template_count` | Gauge | count | Number of template VMs managed by this hatchery |

**Tags**: `service_name`, `service_type`

### 14.3 Global Pool-Level Gauges (Level 3: Entire vSphere Pool)

Resources consumed by **ALL VMs** visible in the datacenter, regardless of whether they are
managed by CDS. This gives operators visibility into the total infrastructure load, including
non-CDS workloads and VMs from other hatchery instances sharing the same vSphere infrastructure.

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/pool_total_vcpus` | Gauge | vCPUs | Total vCPUs across ALL VMs in the datacenter |
| `cds/hatchery/vsphere/pool_total_memory_mb` | Gauge | MB | Total memory across ALL VMs in the datacenter |
| `cds/hatchery/vsphere/pool_total_vm_count` | Gauge | count | Total number of VMs in the datacenter |

**Tags**: `service_name`, `service_type`

### 14.4 Resource Pool Runtime Gauges (Level 3: Infrastructure Capacity)

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

If the Resource Pool cannot be loaded (e.g. permissions issue), a warning is logged and these
metrics are simply not recorded for that cycle. Other metrics are unaffected.

### 14.5 IP Address Tracking Gauges (Level 2: This Hatchery)

When an IP range is configured (`iprange`), these metrics track IP address pool utilization.

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/ip_used_count` | Gauge | count | Number of IP addresses from the configured range currently in use |
| `cds/hatchery/vsphere/ip_total_count` | Gauge | count | Total number of IP addresses in the configured range |

**Tags**: `service_name`, `service_type`

An IP is considered "in use" if it appears in a provision VM's IP-encoded name
(`provision-v2-ip-...`), a VM's CDS annotation (`IPAddress` field), or the VM's guest network info
(`Guest.Net[].IpAddress`) — see §7.2. These metrics are only emitted when `iprange` is configured
(i.e. `availableIPAddresses` is non-empty).

### 14.7 Source Files

| File | Responsibility |
|------|----------------|
| `monitoring.go` | `vsphereMetrics` struct, `initVSphereMetrics()`, `collectVSphereMetrics()`, `startVSphereMetricsRoutine()` |
| `monitoring_test.go` | Unit tests for metrics collection |

### 14.8 Example Prometheus Queries

```promql
# --- Level 2: Hatchery utilization ---

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

# --- IP address utilization ---

# IP address utilization percentage
cds_hatchery_vsphere_ip_used_count / cds_hatchery_vsphere_ip_total_count * 100

# Remaining available IPs
cds_hatchery_vsphere_ip_total_count - cds_hatchery_vsphere_ip_used_count

# Alert: IP pool > 90% used
cds_hatchery_vsphere_ip_used_count / cds_hatchery_vsphere_ip_total_count > 0.9
```

---

# 13. VM Flavor Support (CPU/RAM/Disk Sizing)

The hatchery supports **flavors** for flexible CPU, RAM, and disk sizing without requiring multiple vSphere templates. This feature allows worker models to specify resource profiles that are applied dynamically via `reconfigureVM`.

## 13.1 Overview

Flavors map abstract size names (e.g., `small`, `medium`, `large`) to explicit CPU, RAM, and disk values. When a worker is spawned with a flavor, the claimed pre-provisioned VM is reconfigured (while still powered off) to match the flavor before power-on.

`reconfigureVM` is the single entry point for all hardware configuration (CPU, RAM, disk).

## 13.2 Configuration

### Flavor Definition Example

```toml
# Flavor definitions
defaultFlavor = "medium"
countSmallerFlavorToKeep = 2

[[hatchery.vsphere.flavors]]
  name = "small"
  cpus = 2
  memoryMB = 4096

[[hatchery.vsphere.flavors]]
  name = "medium"
  cpus = 4
  memoryMB = 8192
  diskSizeGB = 50

[[hatchery.vsphere.flavors]]
  name = "large"
  cpus = 8
  memoryMB = 16384
  diskSizeGB = 100

[[hatchery.vsphere.flavors]]
  name = "xlarge"
  cpus = 16
  memoryMB = 32768
  diskSizeGB = 200
```

**Note**: Flavors work seamlessly with resource limits (`maxCpus`, `maxMemoryMB`). When a job requests a flavor, capacity validation uses the flavor's resources instead of the template's.

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `flavors` | `map[string]` | `{}` | Map of flavor name → CPU/RAM/disk config |
| `flavors[].diskSizeGB` | `int` | `0` | Disk size in GB. `0` means no disk resize (inherit template disk) |
| `defaultFlavor` | `string` | `""` | Default flavor when none specified |
| `countSmallerFlavorToKeep` | `int` | `0` | Reserve capacity for N smaller flavor workers (starvation prevention) |

## 13.3 Flavor Usage

Use a **generic worker model** and specify the flavor per job:

```yaml
# Worker model (generic, no flavor)
name: debian12-generic
type: vsphere
osarch: linux/amd64
spec:
  image: "debian12"
```

```yaml
# Workflow: different jobs use different flavors
jobs:
  build-small:
    runs-on: 
      model: vsphere/debian12-generic
      flavor: small

  build-large:
    runs-on: 
      model: vsphere/debian12-generic
      flavor: large
```

**Advantage**: Single worker model serves multiple resource profiles, reducing model duplication.

### Flavor Resolution Priority

1. **Job requirement**: `FlavorRequirement` value (from `runs-on` requirements)
2. **Default**: `HatcheryConfiguration.DefaultFlavor`
3. **None**: Template resources used (no resize)

## 13.4 Capacity Management Integration

`CanAllocateResources()` (from Section 12) automatically uses flavor resources instead of template resources when a flavor is requested:

```
Job requests flavor "large" (8 vCPUs, 16GB RAM)
  ↓
CanAllocateResources validates:
  - Resource Pool has 8 vCPUs + 16GB unreserved
  - MaxCPUs: current (powered-on workers only) + 8 ≤ limit
  - MaxMemoryMB: current (powered-on workers only) + 16GB ≤ limit
  ↓
If capacity available → spawn proceeds
```

This ensures the Resource Pool and static limits are checked **before** attempting to spawn/reconfigure VMs.

> **Note**: Powered-off provisioned VMs are **not counted** in the static limits check.
> They do not consume CPU/RAM in vSphere while powered off, so they never block a flavor job.

## 13.5 Starvation Prevention

The `countSmallerFlavorToKeep` setting reserves capacity for smaller flavors when spawning large ones:

```toml
countSmallerFlavorToKeep = 2
```

**Example**: When spawning a `large` worker (8 vCPUs) with this setting:

- Required capacity = 8 vCPUs (large) + 2 × 2 vCPUs (small reserve) = 12 vCPUs
- The spawn is rejected if less than 12 vCPUs are available
- This ensures at least 2 `small` workers can still be spawned after the `large` one

The reserved flavor is always the **smallest** defined flavor (by CPU count).

## 13.6 Pre-Provisioning with Flavors

Pre-provisioned VMs are created **without flavor applied** — they inherit template resources and remain in a "neutral" state. When assigned to a job:

1. `FindProvisionnedWorker()` returns any available provisioned VM (no flavor matching)
2. If flavor requested → `reconfigureVM` adjusts CPU, RAM, and disk while VM is powered off
3. VM is renamed and powered on with target resources

**Flow**:

```
Provisioned VM (2 vCPUs, 4GB, 20GB disk — from template)
  ↓
Job requests "large" flavor
  ↓
reconfigureVM(vm, "large")  →  VM now has 8 vCPUs, 16GB RAM, 100GB disk (powered off)
  ↓
Power on VM
  ↓
Pre-start script (e.g. filesystem resize) runs inside guest
  ↓
Worker starts with 8 vCPUs, 16GB RAM, 100GB disk
```

## 13.7 Backward Compatibility

- If `flavors` map is empty/not configured → no resizing occurs, VMs inherit template resources (legacy behavior)
- If worker model has no flavor and no `defaultFlavor` configured → template resources used
- If `diskSizeGB` is `0` or unset → disk is not resized, template disk size is inherited
- Resource counting (Section 12) reads actual VM hardware → automatically handles resized VMs