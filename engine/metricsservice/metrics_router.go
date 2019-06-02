package metricsservice

import (
	"context"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("Metrics> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", r.GET(s.getStatusHandler, api.Auth(false)))

	r.Handle("/events", r.GET(s.getEventsHandler), r.POST(s.postEventHandler))
	r.Handle("/metrics", r.GET(s.getMetricsHandler), r.POST(s.postMetricsHandler))
}
