package hooks

import (
	"context"

	"github.com/ovh/cds/engine/api"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, s.authMiddleware)

	r.Handle("/webhook/{uuid}", r.POST(s.webhookHandler, api.Auth(false)), r.GET(s.webhookHandler, api.Auth(false)), r.DELETE(s.webhookHandler, api.Auth(false)), r.PUT(s.webhookHandler, api.Auth(false)))

	r.Handle("/task", r.POST(s.postTaskHandler))
	r.Handle("/task/{uuid}", r.GET(s.getTaskHandler), r.PUT(s.putTaskHandler), r.DELETE(s.deleteTaskHandler))
	r.Handle("/task/{uuid}/execution", r.GET(s.getTaskExecutionsHandler))
}
