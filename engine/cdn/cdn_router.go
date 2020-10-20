package cdn

import (
	"context"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.Middlewares = append(r.Middlewares, s.jwtMiddleware)
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/cache", nil, r.DELETE(s.deleteCacheHandler))
	r.Handle("/cache/status", nil, r.GET(s.getStatusCacheHandler))

	r.Handle("/item/delete", nil, r.POST(s.markItemToDeleteHandler))
	r.Handle("/item/stream", nil, r.GET(s.getItemLogsStreamHandler, service.OverrideAuth(s.validJWTMiddleware)))
	r.Handle("/item/{type}/{apiRef}/download", nil, r.GET(s.getItemDownloadHandler, service.OverrideAuth(s.itemAccessMiddleware)))
	r.Handle("/item/{type}/{apiRef}/lines", nil, r.GET(s.getItemLogsLinesHandler, service.OverrideAuth(s.itemAccessMiddleware)))

	r.Handle("/sync/projects", nil, r.POST(s.syncProjectsHandler))

	r.Handle("/size/item/project/{projectKey}", nil, r.GET(s.getSizeByProjectHandler))
}
