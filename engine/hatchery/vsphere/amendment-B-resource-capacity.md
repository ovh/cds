# Amendment B: Resource-Based Capacity Management

## B.1 Motivation

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

## B.2 Scope

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

- Flavor/resize support (see Amendment C)
- Disk capacity management
- Per-model resource tracking

## B.3 Design

### B.3.1 Resource Pool as Primary Capacity Source

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

### B.3.2 Supplementary Resource Counting (VM-level)

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

### B.3.3 Resource of the Next Worker

To decide whether a new worker can be spawned, the hatchery needs to know the resources the new
VM will consume. Without flavors (Amendment C), this is the resource footprint of the source
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

## B.4 Configuration Changes

### B.4.1 New Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `maxCpus` | `int` | `0` (unlimited) | Optional. Maximum total vCPUs this hatchery may allocate across all VMs. `0` means no static CPU limit (Resource Pool remains the guardrail). |
| `maxMemoryMB` | `int64` | `0` (unlimited) | Optional. Maximum total RAM (MB) this hatchery may allocate across all VMs. `0` means no static memory limit (Resource Pool remains the guardrail). |

No `useResourcePoolLimits` flag is needed — the Resource Pool check is **always enabled**.

### B.4.2 `MaxWorker` Becomes Optional

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

// After (Amendment B):
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

// After (Amendment B): skip check if MaxWorker is 0 (unlimited)
if hcc.Provision.MaxWorker > 0 && hcc.Provision.MaxConcurrentProvisioning > hcc.Provision.MaxWorker {
    return error ...
}
// Same for MaxConcurrentRegistering
```

**Note**: This framework change is backward-compatible. Since the current default for `MaxWorker`
is `10`, existing deployments that do not explicitly set `maxWorker = 0` continue to behave
identically. Only deployments that opt in to `maxWorker = 0` get the new "unlimited" behavior.

### B.4.3 Example Configuration

```toml
[commonConfiguration.provision]
# MaxWorker = 0 means unlimited — rely on resource-based capacity only
maxWorker = 0

# Optional static resource limits (0 = no static limit)
maxCpus = 64
maxMemoryMB = 131072    # 128 GB
```

### B.4.4 Capacity Strategy Matrix

Operators can choose their strategy by combining these settings:

| MaxWorker | maxCpus/maxMemoryMB | Strategy |
|-----------|---------------------|----------|
| `10` (default) | `0` / `0` | **Current behavior** — count-based limit only, Resource Pool is a safety net |
| `0` | `0` / `0` | **Resource Pool only** — rely entirely on vSphere infrastructure limits |
| `0` | set | **Resource-based** — static limits + Resource Pool |
| `20` | set | **Belt and suspenders** — MaxWorker as safety ceiling, resource limits for precision, Resource Pool as infrastructure guardrail |

In all cases, the effective capacity is:
`min(MaxWorker (if >0), static resource limits (if >0), Resource Pool available capacity)`

## B.5 `CanAllocateResources` Implementation

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

## B.6 Resource Pool Capacity Check

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

## B.7 VSphereClient Interface Extension

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

## B.8 Backward Compatibility

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

## B.9 Implementation Plan

- [ ] Phase 1 — Framework change: Make `MaxWorker=0` mean "unlimited" in `sdk/hatchery/provisionning.go` and `engine/service/types.go`
- [ ] Phase 2 — Configuration: Add `MaxCPUs`, `MaxMemoryMB` fields to `HatcheryConfiguration`
- [ ] Phase 3 — Resource Pool check: Implement `checkResourcePoolCapacity()` (always-on primary check)
- [ ] Phase 4 — Resource Counting: Implement `countAllocatedResources()` using existing VM listing data
- [ ] Phase 5 — Template Resources: Implement `getTemplateResources()` for next-worker sizing
- [ ] Phase 6 — CanAllocateResources: Replace stub with Resource Pool + optional static limit checks
- [ ] Phase 7 — Tests: Unit tests for resource counting, Resource Pool capacity, limit enforcement, MaxWorker=0

