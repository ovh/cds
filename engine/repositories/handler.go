package repositories

import "net/http"

// GetHandler implements service.HandlerProvider for the gateway.
func (s *Service) GetHandler() http.Handler {
	if s.Router == nil || s.Router.Mux == nil {
		return nil
	}
	return s.Router.Mux
}
