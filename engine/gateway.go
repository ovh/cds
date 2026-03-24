package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// gatewayConfig maps service types to their URL path prefix on the unified gateway.
var gatewayPrefixes = map[string]string{
	sdk.TypeAPI:           "/cdsapi",
	sdk.TypeCDN:           "/cdscdn",
	sdk.TypeHooks:         "/cdshooks",
	sdk.TypeVCS:           "/vcs",
	sdk.TypeRepositories:  "/repositories",
	sdk.TypeElasticsearch: "/elasticsearch",
	sdk.TypeHatchery:      "/hatchery",
	sdk.TypeUI:            "", // UI at root, must be last
}

// gateway manages a unified HTTP server for all co-located services.
type gateway struct {
	ctx      context.Context
	mux      *mux.Router
	services map[string]service.Service
	addr     string
	port     int
}

// newGateway creates a gateway for co-located services.
func newGateway(ctx context.Context, addr string, port int) *gateway {
	return &gateway{
		ctx:      ctx,
		mux:      mux.NewRouter(),
		services: make(map[string]service.Service),
		addr:     addr,
		port:     port,
	}
}

// register adds a service to the gateway. Call this after the service has
// initialized its router (i.e., after Serve or BeforeStart has been called).
func (g *gateway) register(svc service.Service) {
	g.services[svc.Type()] = svc
}

// build mounts all registered service handlers on the gateway mux.
// Must be called after all services have initialized their routers.
func (g *gateway) build() {
	// Mount services with prefixes (UI last since it's the catch-all "/")
	for _, svcType := range []string{
		sdk.TypeAPI,
		sdk.TypeCDN,
		sdk.TypeHooks,
		sdk.TypeVCS,
		sdk.TypeRepositories,
		sdk.TypeElasticsearch,
		sdk.TypeHatchery,
		sdk.TypeUI,
	} {
		svc, ok := g.services[svcType]
		if !ok {
			continue
		}

		hp, ok := svc.(service.HandlerProvider)
		if !ok {
			continue
		}

		handler := hp.GetHandler()
		if handler == nil {
			continue
		}

		prefix := gatewayPrefixes[svcType]
		if prefix == "" {
			// UI: mount at root as catch-all (last)
			g.mux.PathPrefix("/").Handler(handler)
		} else {
			// Strip the prefix before forwarding to the service handler
			g.mux.PathPrefix(prefix).Handler(
				http.StripPrefix(prefix, handler),
			)
		}

		log.Info(g.ctx, "gateway> mounted %s at %s", svcType, prefix)
	}
}

// serve starts the unified HTTP server.
func (g *gateway) serve() error {
	addr := fmt.Sprintf("%s:%d", g.addr, g.port)
	log.Info(g.ctx, "gateway> Starting unified HTTP server on %s", addr)

	s := &http.Server{
		Addr:           addr,
		Handler:        g.mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	// Graceful shutdown when context is cancelled
	go func() {
		<-g.ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdownCtx)
	}()

	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// setGatewayMode marks all co-located services so they don't start their own
// HTTP listeners. Also updates their HTTPURL to use the gateway prefix.
func setGatewayMode(serviceConfs []serviceConf, gatewayBaseURL string) {
	for i := range serviceConfs {
		svc := serviceConfs[i].service
		if sc, ok := svc.(service.ServiceCommon); ok {
			c := sc.GetCommon()
			c.GatewayServiceMode = true

			prefix := gatewayPrefixes[svc.Type()]
			c.HTTPURL = strings.TrimSuffix(gatewayBaseURL, "/") + prefix
		}
	}
}
