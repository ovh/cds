package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ovh/cds/sdk"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func CommonMetricsView(ctx context.Context) []*view.View {
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

	tagHostname, _ := tag.NewKey("hostname")

	allocView := view.View{
		Name:        "cds/mem/alloc",
		Description: allocStats.Description(),
		Measure:     allocStats,
		TagKeys:     []tag.Key{tagHostname},
		Aggregation: view.LastValue(),
	}

	totalAllocView := view.View{
		Name:        "cds/mem/total_alloc",
		Description: totalAllocStats.Description(),
		Measure:     totalAllocStats,
		TagKeys:     []tag.Key{tagHostname},
		Aggregation: view.LastValue(),
	}

	sysView := view.View{
		Name:        "cds/mem/sys",
		Description: sysStats.Description(),
		Measure:     sysStats,
		TagKeys:     []tag.Key{tagHostname},
		Aggregation: view.LastValue(),
	}

	gcView := view.View{
		Name:        "cds/mem/gc",
		Description: gcStats.Description(),
		Measure:     gcStats,
		TagKeys:     []tag.Key{tagHostname},
		Aggregation: view.LastValue(),
	}

	sdk.GoRoutine(ctx, "service_metrics", func(ctx context.Context) {
		hostname, _ := os.Hostname()
		ctx, _ = tag.New(ctx, tag.Upsert(tagHostname, hostname))

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
			}
		}
	})

	return []*view.View{&allocView, &totalAllocView, &sysView, &gcView}
}
