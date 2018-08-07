package hatchery

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

type Common struct {
	service.Common
	Router      *api.Router
	initialized bool
}

func (c *Common) ServiceName() string {
	return c.Common.ServiceName
}

func (c *Common) AuthMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *api.HandlerConfig) (context.Context, error) {
	if rc.Options["auth"] != "true" {
		return ctx, nil
	}

	hash, err := base64.StdEncoding.DecodeString(req.Header.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad header syntax: %s", err)
	}

	if c.Hash == string(hash) {
		return ctx, nil
	}

	return ctx, sdk.ErrUnauthorized
}

//CDSClient returns cdsclient instance
func (c *Common) CDSClient() cdsclient.Interface {
	return c.Client
}

// IsInitialized returns true if hatchery is fully initialized
func (c *Common) IsInitialized() bool {
	return c.initialized
}

// SetInitialized set initialized = true for this hatchery
func (c *Common) SetInitialized() {
	c.initialized = true
}

// CommonServe start the HatcheryLocal server
func (c *Common) CommonServe(ctx context.Context, h hatchery.Interface) error {
	log.Info("%s> Starting service %s...", c.Name, sdk.VERSION)
	c.StartupTime = time.Now()

	//Init the http server
	c.initRouter(ctx, h)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", h.Configuration().HTTP.Addr, h.Configuration().HTTP.Port),
		Handler:        c.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		//Start the http server
		log.Info("%s> Starting HTTP Server on port %d", c.Name, h.Configuration().HTTP.Port)
		if err := server.ListenAndServe(); err != nil {
			log.Error("%s> Listen and serve failed: %s", c.Name, err)
		}

		//Gracefully shutdown the http server
		select {
		case <-ctx.Done():
			log.Info("%s> Shutdown HTTP Server", c.Name)
			server.Shutdown(ctx)
		}
	}()

	if err := hatchery.Create(h); err != nil {
		return err
	}

	return ctx.Err()
}

func (c *Common) initRouter(ctx context.Context, h hatchery.Interface) {
	log.Debug("%s> Router initialized", c.Name)
	r := c.Router
	r.Background = ctx
	r.URL = h.Configuration().URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, c.AuthMiddleware)

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/workers", r.GET(getWorkersPoolHandler(h), api.Auth(false)))
}

func getWorkersPoolHandler(h hatchery.Interface) api.HandlerFunc {
	return func() api.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			if h == nil {
				return nil
			}
			pool, err := hatchery.WorkerPool(h)
			if err != nil {
				return sdk.WrapError(err, "getWorkersPoolHandler")
			}
			return api.WriteJSON(w, pool, http.StatusOK)
		}
	}
}
