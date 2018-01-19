package repositories

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
	//	r.Middlewares = append(r.Middlewares, s.authMiddleware)

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))

}
