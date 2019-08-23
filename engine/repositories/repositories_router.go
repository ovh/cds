package repositories

import (
	"context"

	"github.com/ovh/cds/engine/service"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("Repositories> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey))

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(s.getStatusHandler))
	r.Handle("/operations", nil, r.POST(s.postOperationHandler))
	r.Handle("/operations/{uuid}", nil, r.GET(s.getOperationsHandler))
}
