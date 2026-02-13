# Amendment A: Prometheus Metrics for vSphere Resource Consumption

*This amendment is the recommended first step. It is independent of Amendments B and C and
has no prerequisites. The resource-counting functions it introduces (iterating VMs, reading
Resource Pool runtime) will be reused by Amendment B.*

## A.1 Motivation

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

## A.2 Scope

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

## A.3 Metrics Inventory

The metrics are organized in three observation levels, from fine-grained to global:

### A.3.1 Per-Worker Resource Gauges (Level 1: Individual VMs)

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/worker_vcpus` | Gauge | vCPUs | Number of vCPUs for a worker VM |
| `cds/hatchery/vsphere/worker_memory_mb` | Gauge | MB | Memory allocated to a worker VM |

**Tags**: `service_name`, `service_type`, `worker_name`, `worker_model`, `flavor`

The `flavor` tag is empty if no flavor is applied (or if Amendment C is not implemented).

### A.3.2 Hatchery-Level Aggregate Gauges (Level 2: This Hatchery)

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

### A.3.3 Global Pool-Level Gauges (Level 3: Entire vSphere Pool)

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

### A.3.4 Resource Pool Runtime Gauges (Level 3: Infrastructure Capacity)

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

### A.3.5 Static Limit Gauges (Configuration Reference)

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `cds/hatchery/vsphere/max_vcpus` | Gauge | vCPUs | Configured `maxCpus` limit (0 if unlimited) |
| `cds/hatchery/vsphere/max_memory_mb` | Gauge | MB | Configured `maxMemoryMB` limit (0 if unlimited) |

**Tags**: `service_name`, `service_type`

## A.4 Implementation

### A.4.1 Metrics Struct

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

### A.4.2 Metrics Initialization

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

### A.4.3 Collection Routine

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

### A.4.4 Integration Point

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

## A.5 Example Prometheus Queries

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

## A.6 Implementation Plan

- [ ] Phase 1 — Struct & Init: Add `vsphereMetrics` struct, implement `InitVSphereMetrics()`
- [ ] Phase 2 — Collection: Implement `collectVSphereMetrics()` and `StartVSphereMetricsRoutine()`
- [ ] Phase 3 — Integration: Wire init and routine into `Init()`/`Serve()`
- [ ] Phase 4 — Tests: Unit tests for metric recording, verify metric names and tags
