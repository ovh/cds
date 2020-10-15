package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type NamedService interface {
	Name() string
	Type() string
}

// GetPrometheustMetricsHandler returns a Handler to exposer prometheus views
func GetPrometheustMetricsHandler(s NamedService) func() Handler {
	return func() Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			if telemetry.StatsExporter(ctx) == nil {
				return sdk.WithStack(sdk.ErrNotFound)
			}
			telemetry.StatsExporter(ctx).ServeHTTP(w, r)
			return nil
		}
	}
}

func GetMetricsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var e = telemetry.StatsExporter(ctx)
		if e == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		return WriteJSON(w, e.ExposedViews, http.StatusOK)
	}
}

func (c *Common) CommonMetricsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		var bToMb = func(b uint64) uint64 {
			return b / 1024 / 1024
		}

		var m runtime.MemStats
		// https://golang.org/pkg/runtime/#MemStats
		runtime.ReadMemStats(&m)

		// Alloc is bytes of allocated heap objects.
		// "Allocated" heap objects include all reachable objects, as
		// well as unreachable objects that the garbage collector has
		// not yet freed. Specifically, HeapAlloc increases as heap
		// objects are allocated and decreases as the heap is swept
		// and unreachable objects are freed. Sweeping occurs
		// incrementally between GC cycles, so these two processes
		// occur simultaneously, and as a result HeapAlloc tends to
		// change smoothly (in contrast with the sawtooth that is
		// typical of stop-the-world garbage collectors).

		// TotalAlloc is cumulative bytes allocated for heap objects.
		// TotalAlloc increases as heap objects are allocated, but
		// unlike Alloc and HeapAlloc, it does not decrease when
		// objects are freed.

		// Sys is the total bytes of memory obtained from the OS.
		// Sys is the sum of the XSys fields below. Sys measures the
		// virtual address space reserved by the Go runtime for the
		// heap, stacks, and other internal data structures. It's
		// likely that not all of the virtual address space is backed
		// by physical memory at any given moment, though in general
		// it all was at some point.

		// NumGC is the number of completed GC cycles.
		return WriteJSON(w, map[string]uint64{
			"alloc":       bToMb(m.Alloc),
			"total_alloc": bToMb(m.TotalAlloc),
			"sys":         bToMb(m.Sys),
			"gc":          uint64(m.NumGC),
		}, http.StatusOK)

	}
}

var onceMetrics sync.Once

func (c *Common) RegisterCommonMetricsView(ctx context.Context) {
	onceMetrics.Do(func() {
		allocStats := stats.Int64(
			"cds/alloc",
			"Alloc is bytes of allocated heap objects",
			stats.UnitBytes,
		)

		totalAllocStats := stats.Int64(
			fmt.Sprintf("cds/total_alloc"),
			"Total Alloc is cumulative bytes allocated for heap objects",
			stats.UnitBytes,
		)

		sysStats := stats.Int64(
			fmt.Sprintf("cds/sys"),
			"Sys is the total bytes of memory obtained from the OS",
			stats.UnitBytes,
		)

		gcStats := stats.Int64(
			fmt.Sprintf("cds/num_gc"),
			"NumGC is the number of completed GC cycles",
			stats.UnitDimensionless,
		)

		tagServiceType := telemetry.MustNewKey(telemetry.TagServiceType)
		tagServiceName := telemetry.MustNewKey(telemetry.TagServiceName)
		tagHostname := telemetry.MustNewKey(telemetry.TagHostname)

		allocView := view.View{
			Name:        "cds/mem/alloc",
			Description: allocStats.Description(),
			Measure:     allocStats,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, tagHostname},
			Aggregation: view.LastValue(),
		}

		totalAllocView := view.View{
			Name:        "cds/mem/total_alloc",
			Description: totalAllocStats.Description(),
			Measure:     totalAllocStats,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, tagHostname},
			Aggregation: view.LastValue(),
		}

		sysView := view.View{
			Name:        "cds/mem/sys",
			Description: sysStats.Description(),
			Measure:     sysStats,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, tagHostname},
			Aggregation: view.LastValue(),
		}

		gcView := view.View{
			Name:        "cds/mem/gc",
			Description: gcStats.Description(),
			Measure:     gcStats,
			TagKeys:     []tag.Key{tagServiceType, tagServiceName, tagHostname},
			Aggregation: view.LastValue(),
		}

		if err := telemetry.RegisterView(ctx, &allocView, &totalAllocView, &sysView, &gcView); err != nil {
			// This should not append
			panic(fmt.Errorf("unable to register service metrics view: %v", err))
		}

		c.GoRoutines.Run(ctx, "service_metrics", func(ctx context.Context) {
			var maxMemoryS = os.Getenv("CDS_MAX_HEAP_SIZE") // in bytes
			var maxMemory uint64
			var onceMaxMemorySignal = new(sync.Once)
			if maxMemoryS != "" {
				maxMemory, _ = strconv.ParseUint(maxMemoryS, 10, 64)
			}
			hostname, _ := os.Hostname()
			ctx = telemetry.ContextWithTag(ctx, telemetry.TagHostname, hostname)

			var tick = time.NewTicker(10 * time.Second)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					stats.Record(ctx, allocStats.M(int64(m.Alloc)))
					stats.Record(ctx, totalAllocStats.M(int64(m.TotalAlloc)))
					stats.Record(ctx, sysStats.M(int64(m.Sys)))
					stats.Record(ctx, gcStats.M(int64(m.NumGC)))

					if maxMemory > 0 && m.Alloc >= maxMemory {
						onceMaxMemorySignal.Do(func() {
							p, err := os.FindProcess(os.Getpid())
							if err != nil {
								log.Error(ctx, "unable to find current process: %v", err)
								return
							}
							if err := p.Signal(sdk.SIGINFO); err != nil {
								log.Error(ctx, "unable to send signal: %v", err)
								return
							}
							log.Info(ctx, "metrics> SIGINFO signal send to %v", os.Getpid())
						})
					}
				}
			}
		})
	})
}

func writeJSON(w http.ResponseWriter, i interface{}, statusCode int) error {
	btes, _ := json.Marshal(i)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(btes)))
	w.WriteHeader(statusCode)
	_, err := w.Write(btes)
	return sdk.WithStack(err)
}

func GetMetricHandler(prefix string) func() Handler {
	return func() Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			exporter := telemetry.StatsExporter(ctx)
			if exporter == nil {
				return sdk.WithStack(sdk.ErrNotFound)
			}
			view := strings.TrimPrefix(r.URL.Path, prefix)
			formValues := r.URL.Query()
			tags := make(map[string]string)
			threshold := formValues.Get("threshold")
			for k := range formValues {
				if k != "threshold" {
					tags[k] = formValues.Get(k)
				}
			}
			log.Debug("GetMetricHandler> path: %s - tags: %v", view, tags)

			if view == "" {
				return writeJSON(w, exporter, http.StatusOK)
			}

			metricsView := exporter.GetView(view, tags)
			if metricsView == nil {
				return sdk.WithStack(sdk.ErrNotFound)
			}

			statusCode := http.StatusOK
			if threshold != "" {
				thresholdF, err := strconv.ParseFloat(threshold, 64)
				if err != nil {
					return sdk.WithStack(sdk.ErrWrongRequest)
				}
				if metricsView.Value >= thresholdF {
					log.Error(context.Background(), "GetMetricHandler> %s threshold (%s) reached or exceeded : %v", metricsView.Name, threshold, metricsView.Value)
					statusCode = 509 // Bandwidth Limit Exceeded
				}
			}

			return writeJSON(w, metricsView, statusCode)
		}
	}
}
