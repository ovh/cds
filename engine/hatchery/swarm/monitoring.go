package swarm

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	TagNodeName     string = "node_name"
	TagJobID        string = "job_id"
	TagWorkerName   string = "worker_name"
	TagResourceName string = "resource_name"
)

func (h *HatcherySwarm) InitWorkersMetrics(ctx context.Context) error {
	h.workerMetrics.CPU = stats.Float64("cds/hatchery/worker_cpu", "number of cpu for a worker resource", stats.UnitDimensionless)
	h.workerMetrics.CPURequest = stats.Float64("cds/hatchery/worker_cpu_request", "number of cpu requested for a worker resource", stats.UnitDimensionless)
	h.workerMetrics.Memory = stats.Int64("cds/hatchery/worker_memory", "number of memory for a worker resource", stats.UnitDimensionless)
	h.workerMetrics.MemoryRequest = stats.Int64("cds/hatchery/worker_memory_request", "number of memory requested for a worker resource", stats.UnitDimensionless)

	tags := []tag.Key{
		telemetry.MustNewKey(telemetry.TagServiceName),
		telemetry.MustNewKey(telemetry.TagServiceType),
		telemetry.MustNewKey(TagNodeName),
		telemetry.MustNewKey(TagJobID),
		telemetry.MustNewKey(TagWorkerName),
		telemetry.MustNewKey(TagResourceName),
	}

	return telemetry.RegisterView(ctx,
		telemetry.NewViewLastFloat64("cds/hatchery/worker_cpu", h.workerMetrics.CPU, tags),
		telemetry.NewViewLastFloat64("cds/hatchery/worker_cpu_request", h.workerMetrics.CPURequest, tags),
		telemetry.NewViewLast("cds/hatchery/worker_memory", h.workerMetrics.Memory, tags),
		telemetry.NewViewLast("cds/hatchery/worker_memory_request", h.workerMetrics.MemoryRequest, tags),
	)
}

func (h *HatcherySwarm) StartWorkerMetricsRoutine(ctx context.Context, delay int64) {
	ticker := time.NewTicker(time.Duration(delay) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.GoRoutines.Exec(ctx, "compute-worker-metrics", func(ctx context.Context) {
				ms, err := h.WorkersMetrics(ctx)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				ctx = telemetry.ContextWithTag(ctx, telemetry.TagServiceName, h.Name())
				ctx = telemetry.ContextWithTag(ctx, telemetry.TagServiceType, h.Type())
				for _, m := range ms {
					ctx = telemetry.ContextWithTag(ctx, TagNodeName, m.Node)
					ctx = telemetry.ContextWithTag(ctx, TagJobID, m.JobID)
					ctx = telemetry.ContextWithTag(ctx, TagResourceName, m.Name)
					ctx = telemetry.ContextWithTag(ctx, TagWorkerName, m.WorkerName)
					stats.Record(ctx,
						h.workerMetrics.CPU.M(m.CPU),
						h.workerMetrics.CPURequest.M(m.CPURequest),
						h.workerMetrics.Memory.M(m.Memory),
						h.workerMetrics.MemoryRequest.M(m.MemoryRequest),
					)
				}
			})
		case <-ctx.Done():
			return
		}
	}
}

func (h *HatcherySwarm) WorkersMetrics(ctx context.Context) ([]WorkerMetricsResource, error) {
	ctx, end := telemetry.Span(ctx, "hatchery.Workers")
	defer end()

	var data []WorkerMetricsResource

	for host, dockerClient := range h.dockerClients {
		cs, err := h.getContainers(ctx, dockerClient, types.ContainerListOptions{All: true})
		if err != nil {
			return nil, sdk.WrapError(err, "unable to list containers")
		}

		chanData := make(chan WorkerMetricsResource, len(cs))

		var wg sync.WaitGroup
		wg.Add(len(cs))

		for i := range cs {
			func(id string) {
				h.GoRoutines.Exec(ctx, "container-get-stats-"+id, func(ctx context.Context) {
					defer wg.Done()

					c, err := dockerClient.ContainerInspect(ctx, id)
					if err != nil {
						log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to get stats for container %s/%s", host, id))
						return
					}

					if c.State == nil || c.State.Status != "running" {
						return
					}

					s, err := dockerClient.ContainerStats(ctx, c.ID, false)
					if err != nil {
						log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to get stats for container %s/%s", host, c.ID))
						return
					}
					v, err := io.ReadAll(s.Body)
					if err != nil {
						log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to get read stats response for container %s/%s", host, c.ID))
						return
					}
					var stats types.Stats
					if err := json.Unmarshal(v, &stats); err != nil {
						log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to get unmarshal stats for container %s/%s", host, c.ID))
						return
					}

					cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
					systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
					onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
					if onlineCPUs == 0.0 {
						onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
					}
					var cpuCoresUsage float64
					if systemDelta > 0.0 && cpuDelta > 0.0 {
						cpuCoresUsage = (cpuDelta / systemDelta) * onlineCPUs
					}

					var workerName string
					if v, ok := c.Config.Labels[LabelServiceWorker]; ok {
						workerName = v
					} else {
						workerName = c.Config.Labels[LabelWorkerName]
					}
					jobID, _ := strconv.ParseInt(c.Config.Labels[LabelJobID], 10, 64)

					chanData <- WorkerMetricsResource{
						Node:          host,
						JobID:         jobID,
						WorkerName:    workerName,
						Name:          strings.TrimPrefix(c.Name, "/"),
						Memory:        int64(stats.MemoryStats.Usage),
						MemoryRequest: c.HostConfig.Memory,
						CPU:           cpuCoresUsage,
						CPURequest:    1,
					}
				})
			}(cs[i].ID)
		}

		wg.Wait()
		close(chanData)

		for v := range chanData {
			data = append(data, v)
		}
	}

	return data, nil
}
