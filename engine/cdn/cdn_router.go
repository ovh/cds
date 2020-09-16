package cdn

import (
	"context"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey))

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, api.Auth(false)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), api.Auth(false)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, api.Auth(false)))

	r.Handle("/item/delete", nil, r.POST(s.markItemToDeleteHandler))
	r.Handle("/item/{type}/{apiRef}", nil, r.GET(s.getItemLogsHandler))
	r.Handle("/item/{type}/{apiRef}/download", nil, r.GET(s.getItemLogsDownloadHandler, api.Auth(false)))

}
