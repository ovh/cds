package telemetry

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// RegisterView begins collecting data for the given views
func RegisterView(ctx context.Context, views ...*view.View) error {
	e := StatsExporter(ctx)
	if e == nil {
		return nil
	}
	e.exposedViewMutex.Lock()
	defer e.exposedViewMutex.Unlock()

	for _, v := range views {
		if view.Find(v.Name) == nil {
			if err := view.Register(v); err != nil {
				return errors.WithStack(err)
			}
			var ev = ExposedView{
				Name:        v.Name,
				Description: v.Description,
				Dimension:   v.Measure.Unit(),
				Aggregation: v.Aggregation.Type.String(),
			}
			for _, t := range v.TagKeys {
				ev.Tags = append(ev.Tags, t.Name())
			}
			e.ExposedViews = append(e.ExposedViews, ev)
		}
	}

	return nil
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

// NewViewLast creates a new view via aggregation LastValue()
func NewViewLast(name string, s *stats.Int64Measure, tags []tag.Key) *view.View {
	return &view.View{
		Name:        name,
		Description: s.Description(),
		Measure:     s,
		Aggregation: view.LastValue(),
		TagKeys:     tags,
	}
}

// NewViewLastFloat64 creates a new view via aggregation LastValue()
func NewViewLastFloat64(name string, s *stats.Float64Measure, tags []tag.Key) *view.View {
	return &view.View{
		Name:        name,
		Description: s.Description(),
		Measure:     s,
		Aggregation: view.LastValue(),
		TagKeys:     tags,
	}
}

// NewViewCount creates a new view via aggregation Count()
func NewViewCount(name string, s *stats.Int64Measure, tags []tag.Key) *view.View {
	return &view.View{
		Name:        name,
		Description: s.Description(),
		Measure:     s,
		Aggregation: view.Count(),
		TagKeys:     tags,
	}
}
