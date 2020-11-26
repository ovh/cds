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
	r.Handle("/mon/profile", nil, r.GET(service.GetAllProfilesHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/profile/{name}", nil, r.GET(service.GetProfileHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/cache", nil, r.DELETE(s.deleteCacheHandler))
	r.Handle("/cache/status", nil, r.GET(s.getStatusCacheHandler))

	r.Handle("/bulk/item/delete", nil, r.POST(s.bulkDeleteItemsHandler))

	r.Handle("/item/stream", nil, r.GET(s.getItemLogsStreamHandler, service.OverrideAuth(s.validJWTMiddleware)))
	r.Handle("/item/{type}/{apiRef}", nil, r.GET(s.getItemHandler, service.OverrideAuth(s.itemAccessMiddleware)), r.DELETE(s.deleteItemHandler))
	r.Handle("/item/{type}/{apiRef}/download", nil, r.GET(s.getItemDownloadHandler, service.OverrideAuth(s.itemAccessMiddleware)))
	r.Handle("/item/{type}/{apiRef}/lines", nil, r.GET(s.getItemLogsLinesHandler, service.OverrideAuth(s.itemAccessMiddleware)))

	r.Handle("/sync/projects", nil, r.POST(s.syncProjectsHandler))

	r.Handle("/size/item/project/{projectKey}", nil, r.GET(s.getSizeByProjectHandler))
}
