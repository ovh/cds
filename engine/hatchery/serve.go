package hatchery

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/rockbears/log"
	"gopkg.in/square/go-jose.v2"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

type Common struct {
	service.Common
	Router                        *api.Router
	mapServiceNextLineNumberMutex sync.Mutex
	mapServiceNextLineNumber      map[string]int64
}

func (c *Common) Service() *sdk.Service {
	return c.Common.ServiceInstance
}

func (c *Common) ServiceName() string {
	return c.Common.ServiceName
}

//CDSClient returns cdsclient instance
func (c *Common) CDSClient() cdsclient.Interface {
	return c.Client
}

// GetGoRoutines returns the goRoutines manager
func (c *Common) GetGoRoutines() *sdk.GoRoutines {
	return c.GoRoutines
}

// CommonServe start the HatcheryLocal server
func (c *Common) CommonServe(ctx context.Context, h hatchery.Interface) error {
	log.Info(ctx, "%s> Starting service %s (%s)...", c.Name(), h.Configuration().Name, sdk.VERSION)
	c.StartupTime = time.Now()

	//Init the http server
	c.initRouter(ctx, h)
	if err := api.InitRouterMetrics(ctx, h); err != nil {
		log.Error(ctx, "unable to init router metrics: %v", err)
	}

	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", h.Configuration().HTTP.Addr, h.Configuration().HTTP.Port),
		Handler:        c.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		//Start the http server
		log.Info(ctx, "%s> Starting HTTP Server on port %d", c.Name(), h.Configuration().HTTP.Port)
		if err := server.ListenAndServe(); err != nil {
			log.Error(ctx, "%s> Listen and serve failed: %v", c.Name(), err)
		}

		//Gracefully shutdown the http server
		select {
		case <-ctx.Done():
			log.Info(ctx, "%s> Shutdown HTTP Server", c.Name())
			server.Shutdown(ctx)
		}
	}()

	if err := hatchery.Create(ctx, h); err != nil {
		return err
	}

	return ctx.Err()
}

func (c *Common) initRouter(ctx context.Context, h hatchery.Interface) {
	log.Debug(ctx, "%s> Router initialized", c.Name())
	r := c.Router
	r.Background = ctx
	r.URL = h.Configuration().URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(c.ParsedAPIPublicKey)

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(getStatusHandler(h), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/workers", nil, r.GET(getWorkersPoolHandler(h), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(c), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.Mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.Mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.Mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	// need 2 routes for index, one for index page, another with {action}
	r.Mux.HandleFunc("/debug/pprof/{action}", pprof.Index)
	r.Mux.HandleFunc("/debug/pprof/", pprof.Index)

	r.Mux.NotFoundHandler = http.HandlerFunc(api.NotFoundHandler)
}

func (c *Common) GetPrivateKey() *rsa.PrivateKey {
	return c.Common.PrivateKey
}

func (c *Common) RefreshServiceLogger(ctx context.Context) error {
	cdnConfig, err := c.Client.ConfigCDN()
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			c.CDNLogsURL = ""
			c.ServiceLogger = nil
		}
		return err
	}
	if cdnConfig.TCPURL == c.CDNLogsURL {
		return nil
	}
	c.CDNLogsURL = cdnConfig.TCPURL

	if c.Signer == nil {
		var signer jose.Signer
		signer, err = jws.NewSigner(c.Common.PrivateKey)
		if err != nil {
			return sdk.WithStack(err)
		}
		c.Signer = signer
	}

	var graylogCfg = &hook.Config{
		Addr:     c.CDNLogsURL,
		Protocol: "tcp",
	}

	if c.ServiceLogger == nil {
		logger, _, err := cdslog.New(ctx, graylogCfg)
		if err != nil {
			return sdk.WithStack(err)
		}
		c.ServiceLogger = logger
	} else {
		if err := cdslog.ReplaceAllHooks(context.Background(), c.ServiceLogger, graylogCfg); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

func (c *Common) SendServiceLog(ctx context.Context, servicesLogs []cdslog.Message, terminated bool) {
	if c.ServiceLogger == nil {
		return
	}

	c.mapServiceNextLineNumberMutex.Lock()
	defer c.mapServiceNextLineNumberMutex.Unlock()
	if c.mapServiceNextLineNumber == nil {
		c.mapServiceNextLineNumber = make(map[string]int64)
	}

	// Init missing service line counters
	for _, s := range servicesLogs {
		key := s.ServiceKey()
		if _, ok := c.mapServiceNextLineNumber[key]; !ok {
			c.mapServiceNextLineNumber[key] = 0
		}
	}

	// Iterate over service log and send value
	for _, s := range servicesLogs {
		sign, err := jws.Sign(c.Signer, s.Signature)
		if err != nil {
			err = sdk.WrapError(err, "unable to sign service log message")
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, err.Error())
			continue
		}
		lineNumber := c.mapServiceNextLineNumber[s.ServiceKey()]
		c.mapServiceNextLineNumber[s.ServiceKey()]++
		if c.ServiceLogger != nil {
			c.ServiceLogger.
				WithField(cdslog.ExtraFieldSignature, sign).
				WithField(cdslog.ExtraFieldLine, lineNumber).
				WithField(cdslog.ExtraFieldTerminated, terminated).
				Log(s.Level, s.Value)
		}
	}

	// If log status is terminated for given service, we can remove line counters
	if terminated {
		for _, s := range servicesLogs {
			delete(c.mapServiceNextLineNumber, s.ServiceKey())
		}
	}
}

func getWorkersPoolHandler(h hatchery.Interface) service.HandlerFunc {
	return func() service.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			if h == nil {
				return nil
			}
			pool, err := hatchery.WorkerPool(ctx, h)
			if err != nil {
				return sdk.WrapError(err, "getWorkersPoolHandler")
			}
			return service.WriteJSON(w, pool, http.StatusOK)
		}
	}
}

func getStatusHandler(h hatchery.Interface) service.HandlerFunc {
	return func() service.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			if h == nil {
				return nil
			}
			srv, ok := h.(service.Service)
			if !ok {
				return fmt.Errorf("unable to get status from %s", h.Service().Name)
			}
			status := srv.Status(ctx)
			return service.WriteJSON(w, status, status.HTTPStatusCode())
		}
	}
}
