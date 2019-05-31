package elasticsearch

import (
	"context"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("Repositories> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(s.getStatusHandler))
	r.Handle("/events", nil, r.GET(s.getEventsHandler), r.POST(s.postEventHandler))
	r.Handle("/metrics", nil, r.GET(s.getMetricsHandler), r.POST(s.postMetricsHandler))
}
