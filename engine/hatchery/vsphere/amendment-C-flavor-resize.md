# Amendment C: VM Flavor Support (CPU/RAM Resize)

*This amendment builds on Amendment B (resource-based capacity management).*

## C.1 Motivation

Even with Amendment B's resource-based capacity management, all VMs cloned by the vSphere hatchery
inherit their hardware configuration (CPU count, memory) from the source template. Operators must
maintain separate templates for each resource profile they need, which does not scale.

The OpenStack hatchery supports a **flavor** mechanism that maps abstract size names (e.g. `small`,
`medium`, `large`) to concrete provider flavors. This amendment adds an equivalent mechanism for
the vSphere hatchery, where flavors map to explicit CPU/RAM values applied at clone time via
`VirtualMachineConfigSpec.NumCPUs` and `MemoryMB`.

With Amendment B already in place, the capacity management automatically accounts for the resized
VMs since `countAllocatedResources()` reads actual VM hardware from `summary.Config`, not
template defaults.

## C.2 Scope

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

## C.3 Configuration Changes

### C.3.1 New Types

```go
// VSphereFlavorConfig defines the hardware resources for a flavor.
type VSphereFlavorConfig struct {
    CPUs     int32 `mapstructure:"cpus" toml:"cpus" json:"cpus"`
    MemoryMB int64 `mapstructure:"memoryMB" toml:"memoryMB" json:"memoryMB"`
}
```

### C.3.2 New Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `flavors` | `map[string]VSphereFlavorConfig` | `nil` | Map of flavor name → resource definition |
| `defaultFlavor` | `string` | `""` | Default flavor when none is specified |
| `countSmallerFlavorToKeep` | `int` | `0` (disabled) | Reserve capacity for smaller flavors to prevent starvation |

**Note**: `maxCpus` and `maxMemoryMB` are already provided by Amendment B and reused here.

### C.3.3 Example Configuration

```toml
[commonConfiguration.provision]
maxWorker = 0  # unlimited — rely on resource-based capacity (Amendment B)

# Optional static resource limits (from Amendment B)
maxCpus = 64
maxMemoryMB = 131072

# Flavor definitions (Amendment C)
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

### C.3.4 Worker Provisioning Config Change

Add an optional `Flavor` field:

```go
type WorkerProvisioningConfig struct {
    ModelPath   string
    ModelVMWare string
    Number      int
    Flavor      string  // NEW: flavor to apply to provisioned VMs
}
```

## C.4 Worker Model Changes

### C.4.1 V2 Spec

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

### C.4.2 V1 Model

The existing `ModelVirtualMachine.Flavor` field is already present in the V1 model struct but
currently unused by the vSphere hatchery. This amendment activates it.

## C.5 Flavor Resolution

Flavor is resolved with the following priority (highest first):

1. **V1**: `Model.ModelVirtualMachine.Flavor`
2. **V2**: `V2WorkerModelVSphereSpec.Flavor`
3. **Job requirement**: `FlavorRequirement` value from job prerequisites
4. **Default**: `HatcheryConfiguration.DefaultFlavor`
5. **None**: No flavor applied → resources inherited from template (pre-Amendment B behavior)

This requires extending `GetFlavor()` in `sdk/hatchery.go` to read from `VSphereSpec.Flavor`.

## C.6 Spawn Changes

### C.6.1 `CanSpawn` Modification

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

### C.6.2 `prepareCloneSpec` Modification

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

### C.6.3 `SpawnWorker` Modification

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

## C.7 Adaptation of `CanAllocateResources` (from Amendment B)

Amendment B's `CanAllocateResources()` determines the next worker's resource footprint by reading
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
        // Amendment C: use flavor-defined resources
        flavor, ok := h.Config.Flavors[strings.ToLower(flavorName)]
        if !ok {
            return false, fmt.Errorf("unknown flavor %q", flavorName)
        }
        nextCPUs = flavor.CPUs
        nextMemoryMB = int32(flavor.MemoryMB)
    } else {
        // Amendment B fallback: use template resources
        templateName := model.GetVSphereImage()
        var err error
        nextCPUs, nextMemoryMB, err = h.getTemplateResources(ctx, templateName)
        if err != nil {
            log.Warn(ctx, "CanAllocateResources> unable to determine resource footprint: %v", err)
            return true, nil
        }
    }

    // === Primary check: Resource Pool capacity (always enabled, from Amendment B) ===
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

        // Flavor starvation prevention (Amendment C)
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

**Key insight**: `countAllocatedResources()` (from Amendment B) reads `summary.Config.NumCpu` and
`summary.Config.MemorySizeMB` from actual VMs. When a VM is cloned with a flavor override, its
`summary.Config` reflects the **overridden** values, not the template defaults. Therefore,
Amendment B's resource counting automatically works correctly with resized VMs — no changes to
the counting logic are needed.

### C.7.1 Smaller Flavor Starvation Prevention

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

## C.8 Provisioning Integration

### C.8.1 Provisioning with Flavors

When `WorkerProvisioningConfig.Flavor` is set, provisioned VMs are created with the specified
flavor's CPU/RAM configuration applied via `prepareCloneSpec`.

### C.8.2 Provisioned VM Matching

`FindProvisionnedWorker` must be extended to match on flavor in addition to model path:

- If the job requests a flavor, only match provisioned VMs with the same flavor in their annotation
- If no flavor is requested and a default flavor is configured, match on the default flavor
- If no flavor is configured at all, match any provisioned VM for the model (current behavior)

### C.8.3 Flavor Mismatch Handling

If a job requests a flavor that differs from available provisioned VMs, the hatchery falls back to
creating a fresh clone with the requested flavor. The provisioned VMs remain available for future
jobs with matching flavors.

### C.8.4 Annotation Change

Add a `Flavor` field to the annotation struct for provisioning matching:

```go
type annotation struct {
    // ... existing fields ...
    Flavor string `json:"flavor,omitempty"` // NEW: flavor name used for this VM
}
```

## C.9 SDK Changes

### C.9.1 `GetFlavor` Extension

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

### C.9.2 `FlavorRequirement` Filter

In `sdk/hatchery/hatchery.go`, allow `FlavorRequirement` for vSphere models:

```go
// BEFORE
if model.Type != sdk.Openstack && r.Type == sdk.FlavorRequirement {

// AFTER
if model.Type != sdk.Openstack && model.Type != sdk.VSphere && r.Type == sdk.FlavorRequirement {
```

## C.10 Backward Compatibility

- If `Flavors` is not set in the configuration, no resize occurs. VMs inherit template resources.
  Amendment B's capacity management continues to work using template-derived resource values.
- If a worker model does not specify a flavor and no `DefaultFlavor` is configured, no resize occurs.
- `countAllocatedResources()` (Amendment B) does not need modification — it reads actual VM
  hardware, which automatically reflects any flavor overrides applied at clone time.

## C.11 Implementation Plan

- [ ] Phase 1 — SDK: Add `Flavor` to `V2WorkerModelVSphereSpec`, extend `GetFlavor()`, update `FlavorRequirement` filter
- [ ] Phase 2 — Configuration: Add `VSphereFlavorConfig`, `Flavors`, `DefaultFlavor`, `CountSmallerFlavorToKeep`
- [ ] Phase 3 — Clone Resize: Modify `prepareCloneSpec()` signature and apply `NumCPUs` / `MemoryMB`
- [ ] Phase 4 — CanAllocateResources: Adapt Amendment B's implementation to prefer flavor-defined resources
- [ ] Phase 5 — Provisioning: Add `Flavor` to `WorkerProvisioningConfig` and annotation, update `FindProvisionnedWorker` matching
- [ ] Phase 6 — Tests: Unit tests for resize, flavor resolution, starvation prevention, provisioning matching
