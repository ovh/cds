package repositories

import (
	"context"

	"github.com/ovh/cds/engine/service"
	"github.com/rockbears/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug(ctx, "router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.TracingMiddlewareFunc(s))
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)
	r.PostMiddlewares = append(r.PostMiddlewares, service.TracingPostMiddleware)

	r.Handle("/admin/cache", nil, r.GET(service.GetLocalCacheHandler), r.DELETE(service.ClearLocalCacheHandler))

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.getStatusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/operations", nil, r.POST(s.postOperationHandler))
	r.Handle("/operations/{uuid}", nil, r.GET(s.getOperationsHandler))
}
