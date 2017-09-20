package hooks

import "context"

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.Handle("/webhook/{uuid}", r.POST(s.webhookHandler), r.GET(s.webhookHandler), r.DELETE(s.webhookHandler), r.PUT(s.webhookHandler))
}
