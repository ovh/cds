package observability

import (
	"context"
	"strings"
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	DefaultSizeDistribution    = view.Distribution(1024, 2048, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216, 67108864, 268435456, 1073741824, 4294967296)
	DefaultLatencyDistribution = view.Distribution(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)
)

const (
	Host           = "http.host"
	StatusCode     = "http.status"
	Path           = "http.path"
	Method         = "http.method"
	KeyServerRoute = "http_server_route"
	Handler        = "http.handler"
)

type ExposedView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Dimension   string   `json:"dimension"`
	Aggregation string   `json:"aggregagtion"`
}

var (
	ExposedViews     []ExposedView
	exposedViewMutex sync.Mutex
)

// RegisterView begins collecting data for the given views
func RegisterView(views ...*view.View) error {
	exposedViewMutex.Lock()
	defer exposedViewMutex.Unlock()

	for _, v := range views {
		if view.Find(v.Name) == nil {
			log.Debug("obserbability.RegisterView> Registering view %s with tags %v on measure %p", v.Name, v.TagKeys, v.Measure)
			if err := view.Register(v); err != nil {
				return sdk.WithStack(err)
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
			ExposedViews = append(ExposedViews, ev)
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
