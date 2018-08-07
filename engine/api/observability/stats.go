package observability

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"go.opencensus.io/stats/view"
)

// RegisterView begins collecting data for the given views
func RegisterView(views ...*view.View) error {
	if statsExporter != nil {
		return nil
	}
	return view.Register(views...)
}

// StatsHandler returns a Handler to exposer prometheus views
func StatsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		statsExporter.ServeHTTP(w, r)
		return nil
	}
}
