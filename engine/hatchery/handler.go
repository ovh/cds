package hatchery

import "net/http"

// GetHandler implements service.HandlerProvider for the gateway.
func (c *Common) GetHandler() http.Handler {
	if c.Router == nil || c.Router.Mux == nil {
		return nil
	}
	return c.Router.Mux
}
