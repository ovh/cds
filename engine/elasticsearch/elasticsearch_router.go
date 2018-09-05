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

	for _, r := range s.Cfg.ElasticSearch.Indexes {
		switch r.Role {
		case "events":
			indexEvent = r.Index
		case "metrics":
			indexMetric = r.Index
		}
	}

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", r.GET(s.getStatusHandler))
	r.Handle("/events", r.GET(s.getEventsHandler), r.POST(s.postEventHandler))
	r.Handle("/metrics", r.GET(s.getMetricsHandler), r.POST(s.postMetricsHandler))
}
