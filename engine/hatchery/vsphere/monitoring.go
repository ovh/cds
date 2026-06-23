package vsphere

import (
	"context"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/sdk/telemetry"
)

type vsphereMetrics struct {
	// Level 1: Per-worker
	WorkerVCPUs    *stats.Int64Measure
	WorkerMemoryMB *stats.Int64Measure
	WorkerDiskGB   *stats.Int64Measure

	// Level 2: Hatchery-level aggregate
	AllocatedVCPUs     *stats.Int64Measure
	AllocatedMemoryMB  *stats.Int64Measure
	AllocatedDiskGB    *stats.Int64Measure
	VMCount            *stats.Int64Measure
	WorkerVMCount      *stats.Int64Measure
	ProvisionedVMCount *stats.Int64Measure
	// Provisioned VMs broken down by state (sum of ready+starting+dying equals
	// ProvisionedVMCount; in-flight clones are not VMs yet so they are tracked apart).
	ProvisionReadyCount    *stats.Int64Measure
	ProvisionStartingCount *stats.Int64Measure
	ProvisionDyingCount    *stats.Int64Measure
	ProvisionInflightCount *stats.Int64Measure
	TemplateVCPUs          *stats.Int64Measure
	TemplateMemoryMB       *stats.Int64Measure
	TemplateCount          *stats.Int64Measure

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

	// IP address tracking
	IPActiveCount   *stats.Int64Measure
	IPReservedCount *stats.Int64Measure
	IPTotalCount    *stats.Int64Measure

	// Views (kept for re-registration of per-worker views)
	workerViews    []*view.View
	aggregateViews []*view.View
}

func (h *HatcheryVSphere) initVSphereMetrics(ctx context.Context) error {
	h.initVSphereMetricsMeasures()
	return telemetry.RegisterView(ctx, h.allViews()...)
}

func (h *HatcheryVSphere) initVSphereMetricsMeasures() {
	// Level 1: Per-worker
	h.metrics.WorkerVCPUs = stats.Int64(
		"cds/hatchery/vsphere/worker_vcpus",
		"vCPUs for a worker VM", stats.UnitDimensionless)
	h.metrics.WorkerMemoryMB = stats.Int64(
		"cds/hatchery/vsphere/worker_memory_mb",
		"memory (MB) for a worker VM", stats.UnitDimensionless)
	h.metrics.WorkerDiskGB = stats.Int64(
		"cds/hatchery/vsphere/worker_disk_gb",
		"disk capacity (GB) for a worker VM", stats.UnitDimensionless)

	// Level 2: Hatchery-level aggregate
	h.metrics.AllocatedVCPUs = stats.Int64(
		"cds/hatchery/vsphere/allocated_vcpus",
		"total vCPUs allocated by this hatchery's VMs", stats.UnitDimensionless)
	h.metrics.AllocatedMemoryMB = stats.Int64(
		"cds/hatchery/vsphere/allocated_memory_mb",
		"total memory (MB) allocated by this hatchery's VMs", stats.UnitDimensionless)
	h.metrics.AllocatedDiskGB = stats.Int64(
		"cds/hatchery/vsphere/allocated_disk_gb",
		"total disk capacity (GB) allocated by this hatchery's VMs", stats.UnitDimensionless)
	h.metrics.VMCount = stats.Int64(
		"cds/hatchery/vsphere/vm_count",
		"total VMs managed by this hatchery (workers + provisions)", stats.UnitDimensionless)
	h.metrics.WorkerVMCount = stats.Int64(
		"cds/hatchery/vsphere/worker_vm_count",
		"VMs running as workers (claimed, no longer in the provision pool)", stats.UnitDimensionless)
	h.metrics.ProvisionedVMCount = stats.Int64(
		"cds/hatchery/vsphere/provisioned_vm_count",
		"pre-provisioned VMs (ready + starting + dying)", stats.UnitDimensionless)
	h.metrics.ProvisionReadyCount = stats.Int64(
		"cds/hatchery/vsphere/provision_ready_count",
		"provisioned VMs powered off and ready to be claimed", stats.UnitDimensionless)
	h.metrics.ProvisionStartingCount = stats.Int64(
		"cds/hatchery/vsphere/provision_starting_count",
		"provisioned VMs powered on, still being created/finished", stats.UnitDimensionless)
	h.metrics.ProvisionDyingCount = stats.Int64(
		"cds/hatchery/vsphere/provision_dying_count",
		"provisioned VMs marked for deletion, not yet reaped", stats.UnitDimensionless)
	h.metrics.ProvisionInflightCount = stats.Int64(
		"cds/hatchery/vsphere/provision_inflight_count",
		"in-flight provision clones not yet visible in the inventory", stats.UnitDimensionless)
	h.metrics.TemplateVCPUs = stats.Int64(
		"cds/hatchery/vsphere/template_vcpus",
		"total vCPUs defined by template VMs", stats.UnitDimensionless)
	h.metrics.TemplateMemoryMB = stats.Int64(
		"cds/hatchery/vsphere/template_memory_mb",
		"total memory (MB) defined by template VMs", stats.UnitDimensionless)
	h.metrics.TemplateCount = stats.Int64(
		"cds/hatchery/vsphere/template_count",
		"number of template VMs", stats.UnitDimensionless)

	// Level 3: Global pool
	h.metrics.PoolTotalVCPUs = stats.Int64(
		"cds/hatchery/vsphere/pool_total_vcpus",
		"total vCPUs across all VMs in datacenter", stats.UnitDimensionless)
	h.metrics.PoolTotalMemoryMB = stats.Int64(
		"cds/hatchery/vsphere/pool_total_memory_mb",
		"total memory (MB) across all VMs in datacenter", stats.UnitDimensionless)
	h.metrics.PoolTotalVMCount = stats.Int64(
		"cds/hatchery/vsphere/pool_total_vm_count",
		"total VMs in datacenter", stats.UnitDimensionless)

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

	// IP address tracking
	h.metrics.IPActiveCount = stats.Int64(
		"cds/hatchery/vsphere/ip_active_count",
		"Number of IP addresses active on running VMs", stats.UnitDimensionless)
	h.metrics.IPReservedCount = stats.Int64(
		"cds/hatchery/vsphere/ip_reserved_count",
		"Number of IP addresses held by VMs (any power state)", stats.UnitDimensionless)
	h.metrics.IPTotalCount = stats.Int64(
		"cds/hatchery/vsphere/ip_total_count",
		"Total number of IP addresses in configured range", stats.UnitDimensionless)

	// Build views
	baseTags := []tag.Key{
		telemetry.MustNewKey(telemetry.TagServiceName),
		telemetry.MustNewKey(telemetry.TagServiceType),
	}
	workerTags := []tag.Key{
		telemetry.MustNewKey(telemetry.TagServiceName),
		telemetry.MustNewKey(telemetry.TagServiceType),
		telemetry.MustNewKey("worker_name"),
		telemetry.MustNewKey("worker_model"),
	}

	h.metrics.workerViews = []*view.View{
		telemetry.NewViewLast("cds/hatchery/vsphere/worker_vcpus", h.metrics.WorkerVCPUs, workerTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/worker_memory_mb", h.metrics.WorkerMemoryMB, workerTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/worker_disk_gb", h.metrics.WorkerDiskGB, workerTags),
	}

	h.metrics.aggregateViews = []*view.View{
		// Level 2: Hatchery aggregate
		telemetry.NewViewLast("cds/hatchery/vsphere/allocated_vcpus", h.metrics.AllocatedVCPUs, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/allocated_memory_mb", h.metrics.AllocatedMemoryMB, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/allocated_disk_gb", h.metrics.AllocatedDiskGB, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/vm_count", h.metrics.VMCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/worker_vm_count", h.metrics.WorkerVMCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/provisioned_vm_count", h.metrics.ProvisionedVMCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/provision_ready_count", h.metrics.ProvisionReadyCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/provision_starting_count", h.metrics.ProvisionStartingCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/provision_dying_count", h.metrics.ProvisionDyingCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/provision_inflight_count", h.metrics.ProvisionInflightCount, baseTags),
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
		// IP address tracking
		telemetry.NewViewLast("cds/hatchery/vsphere/ip_active_count", h.metrics.IPActiveCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/ip_reserved_count", h.metrics.IPReservedCount, baseTags),
		telemetry.NewViewLast("cds/hatchery/vsphere/ip_total_count", h.metrics.IPTotalCount, baseTags),
	}
}

func (h *HatcheryVSphere) allViews() []*view.View {
	var views []*view.View
	views = append(views, h.metrics.workerViews...)
	views = append(views, h.metrics.aggregateViews...)
	return views
}

func (h *HatcheryVSphere) startVSphereMetricsRoutine(ctx context.Context, delay int64) {
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

	// Iterate ALL VMs from ListVirtualMachines (returns entire datacenter)
	srvs := h.getRawVMs(ctx)

	// Level 2: Hatchery-level counters
	var hatcheryCPUs, hatcheryMemMB, hatcheryDiskGB int64
	var vmCount, workerCount, provisionedCount int64
	var provisionReady, provisionStarting, provisionDying int64
	var templateCPUs, templateMemMB, templateCount int64
	// Names of provisions currently listed, used to count in-flight clones (in the
	// pending cache but not yet visible in the inventory) without double counting.
	provisionNames := make(map[string]struct{})

	// Level 3: Global pool counters (ALL VMs, no annotation filtering)
	var poolCPUs, poolMemMB, poolVMCount int64

	// Re-register per-worker views to drop stale workers
	view.Unregister(h.metrics.workerViews...)
	if err := view.Register(h.metrics.workerViews...); err != nil {
		log.Warn(ctx, "collectVSphereMetrics> unable to re-register worker views: %v", err)
	}

	for _, s := range srvs {
		cpus := int64(s.Summary.Config.NumCpu)
		memMB := int64(s.Summary.Config.MemorySizeMB)
		diskGB := vmDiskCapacityGB(s)

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
			templateCPUs += cpus
			templateMemMB += memMB
			templateCount++
			continue
		}

		hatcheryCPUs += cpus
		hatcheryMemMB += memMB
		hatcheryDiskGB += diskGB
		vmCount++

		// Classify the VM. A pooled provision still carries the Provisioning flag
		// and the provision-v2 prefix; once claimed it is renamed and the flag is
		// cleared, so it counts as a worker. Provisions are broken down by state
		// (same rule as provisioningV2): dying first, then ready (powered off) vs
		// starting (powered on).
		if strings.HasPrefix(s.Name, "provision-v2") && annot.Provisioning {
			provisionedCount++
			provisionNames[s.Name] = struct{}{}
			switch {
			case h.isMarkedToDelete(s):
				provisionDying++
			case s.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOff:
				provisionReady++
			default:
				provisionStarting++
			}
		} else {
			workerCount++
		}

		// Level 1: Per-worker metrics
		wCtx := telemetry.ContextWithTag(ctx, "worker_name", s.Name)
		wCtx = telemetry.ContextWithTag(wCtx, "worker_model", annot.WorkerModelPath)
		stats.Record(wCtx,
			h.metrics.WorkerVCPUs.M(cpus),
			h.metrics.WorkerMemoryMB.M(memMB),
			h.metrics.WorkerDiskGB.M(diskGB),
		)
	}

	// In-flight clones: tracked in the pending cache but not yet visible in the
	// inventory above. Counted apart from the VM-based provision states.
	var provisionInflight int64
	h.cacheProvisioning.mu.Lock()
	for name := range h.cacheProvisioning.pending {
		if _, listed := provisionNames[name]; !listed {
			provisionInflight++
		}
	}
	h.cacheProvisioning.mu.Unlock()

	// Level 2: Hatchery aggregate
	stats.Record(ctx,
		h.metrics.AllocatedVCPUs.M(hatcheryCPUs),
		h.metrics.AllocatedMemoryMB.M(hatcheryMemMB),
		h.metrics.AllocatedDiskGB.M(hatcheryDiskGB),
		h.metrics.VMCount.M(vmCount),
		h.metrics.WorkerVMCount.M(workerCount),
		h.metrics.ProvisionedVMCount.M(provisionedCount),
		h.metrics.ProvisionReadyCount.M(provisionReady),
		h.metrics.ProvisionStartingCount.M(provisionStarting),
		h.metrics.ProvisionDyingCount.M(provisionDying),
		h.metrics.ProvisionInflightCount.M(provisionInflight),
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

	// IP address tracking
	if len(h.availableIPAddresses) > 0 {
		// Reserved: IPs held by any VM via annotation (includes powered-off provisioned VMs)
		reservedIPs := make(map[string]struct{})
		// Active: IPs visible on running VMs via guest network info
		activeIPs := make(map[string]struct{})
		for _, s := range srvs {
			annot := getVirtualMachineCDSAnnotation(ctx, s)
			if annot != nil && annot.IPAddress != "" {
				reservedIPs[annot.IPAddress] = struct{}{}
			}
			if s.Guest != nil {
				for _, n := range s.Guest.Net {
					for _, ip := range n.IpAddress {
						activeIPs[ip] = struct{}{}
					}
				}
			}
		}
		var ipReserved, ipActive int64
		for _, ip := range h.availableIPAddresses {
			if _, ok := reservedIPs[ip]; ok {
				ipReserved++
			}
			if _, ok := activeIPs[ip]; ok {
				ipActive++
			}
		}
		stats.Record(ctx,
			h.metrics.IPActiveCount.M(ipActive),
			h.metrics.IPReservedCount.M(ipReserved),
			h.metrics.IPTotalCount.M(int64(len(h.availableIPAddresses))),
		)
	}

	// Level 3: Resource Pool runtime
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

// vmDiskCapacityGB returns the total disk capacity in GB for a VM by summing
// all VirtualDisk devices found in Config.Hardware.Device.
func vmDiskCapacityGB(vm mo.VirtualMachine) int64 {
	if vm.Config == nil {
		return 0
	}
	var totalKB int64
	for _, dev := range vm.Config.Hardware.Device {
		if disk, ok := dev.(*types.VirtualDisk); ok {
			totalKB += disk.CapacityInKB
		}
	}
	return totalKB / (1024 * 1024)
}
