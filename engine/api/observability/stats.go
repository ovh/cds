package observability

import (
	"context"
	"net/http"
	"strings"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// RegisterView begins collecting data for the given views
func RegisterView(views ...*view.View) error {
	return sdk.WithStack(view.Register(views...))
}

// FindAndRegisterViewLast begins collecting data for the given views
func FindAndRegisterViewLast(nameInput string, tags []tag.Key) (*view.View, error) {
	name := strings.ToLower(nameInput)
	viewFind := view.Find(name)
	if viewFind != nil {
		return viewFind, nil
	}
	value := stats.Int64("cds/cds-api/"+name, name, stats.UnitDimensionless)
	newView := NewViewLast(name, value, tags)
	return newView, view.Register(newView)
}

// FindAndRegisterViewLastFloat64 begins collecting data for the given views
func FindAndRegisterViewLastFloat64(nameInput string, tags []tag.Key) (*view.View, error) {
	name := strings.ToLower(nameInput)
	viewFind := view.Find(name)
	if viewFind != nil {
		return viewFind, nil
	}
	value := stats.Float64("cds/cds-api/"+name, name, stats.UnitDimensionless)
	newView := NewViewLastFloat64(name, value, tags)
	return newView, view.Register(newView)
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
