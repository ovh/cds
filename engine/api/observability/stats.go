package observability

import (
	"context"
	"net/http"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
)

// RegisterView begins collecting data for the given views
func RegisterView(views ...*view.View) error {
	if statsExporter == nil {
		log.Info("observability> stats are disabled")
		return nil
	}
	return view.Register(views...)
}

// StatsHandler returns a Handler to exposer prometheus views
func StatsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if statsExporter == nil {
			return nil
		}
		statsExporter.ServeHTTP(w, r)
		return nil
	}
}

// Record an int64 measure
func Record(ctx context.Context, m stats.Measure, v int64) {
	if m == nil {
		return
	}
	mInt64, ok := m.(*stats.Int64Measure)
	if !ok {
		return
	}
	if mInt64 == nil {
		return
	}
	stats.Record(ctx, mInt64.M(v))
}

// RecordFloat64 a float64 measure
func RecordFloat64(ctx context.Context, m stats.Measure, v float64) {
	if m == nil {
		return
	}
	mFloat64, ok := m.(*stats.Float64Measure)
	if !ok {
		return
	}
	if mFloat64 == nil {
		return
	}
	stats.Record(ctx, mFloat64.M(v))
}
