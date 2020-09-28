package elasticsearch

import (
	"context"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("Repositories> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.getStatusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/events", nil, r.GET(s.getEventsHandler), r.POST(s.postEventHandler))
	r.Handle("/metrics", nil, r.GET(s.getMetricsHandler), r.POST(s.postMetricsHandler))
}
