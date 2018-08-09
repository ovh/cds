package observability

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
	"go.opencensus.io/stats/view"
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
		statsExporter.ServeHTTP(w, r)
		return nil
	}
}
