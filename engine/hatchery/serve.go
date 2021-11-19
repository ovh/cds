package hatchery

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rockbears/log"
	"gopkg.in/square/go-jose.v2"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
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

	// Gracefully shutdown the http server
	h.GetGoRoutines().Exec(ctx, "hatchery.httpserver-shutdown", func(ctx context.Context) {
		<-ctx.Done()
		log.Info(ctx, "%s> Shutdown HTTP Server", c.Name())
		_ = server.Shutdown(ctx)
	})

	// Start the http server
	log.Info(ctx, "%s> Starting HTTP Server on port %d", c.Name(), h.Configuration().HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		return sdk.WrapError(err, "listen and serve failed: %s", c.Name())
	}

	return nil
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

	r.Mux.NotFoundHandler = http.HandlerFunc(r.NotFoundHandler)
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

	if cdnConfig.TCPURLEnableTLS {
		tcpCDNUrl := c.CDNLogsURL
		// Check if the url has a scheme
		// We have to remove if to retrieve the hostname
		if i := strings.Index(tcpCDNUrl, "://"); i > -1 {
			tcpCDNUrl = tcpCDNUrl[i+3:]
		}
		tcpCDNHostname, _, err := net.SplitHostPort(tcpCDNUrl)
		if err != nil {
			return sdk.WithStack(err)
		}

		graylogCfg.TLSConfig = &tls.Config{ServerName: tcpCDNHostname}
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

func (c *Common) GenerateWorkerConfig(ctx context.Context, h hatchery.Interface, spawnArgs hatchery.SpawnArguments) workerruntime.WorkerConfig {
	apiURL := h.Configuration().Provision.WorkerAPIHTTP.URL
	httpInsecure := h.Configuration().Provision.WorkerAPIHTTP.Insecure
	if apiURL == "" {
		apiURL = h.Configuration().API.HTTP.URL
		httpInsecure = h.Configuration().API.HTTP.Insecure
	}

	envvars := make(map[string]string, len(h.Configuration().Provision.InjectEnvVars))

	for _, e := range h.Configuration().Provision.InjectEnvVars {
		tuple := strings.SplitN(e, "=", 2)
		if len(tuple) != 2 {
			log.Error(ctx, "invalid env variable to inject: %q", e)
			continue
		}
		envvars[tuple[0]] = tuple[1]
	}

	cfg := workerruntime.WorkerConfig{
		Name:                spawnArgs.WorkerName,
		BookedJobID:         spawnArgs.JobID,
		HatcheryName:        h.Name(),
		Model:               spawnArgs.ModelName(),
		APIToken:            spawnArgs.WorkerToken,
		APIEndpoint:         apiURL,
		APIEndpointInsecure: httpInsecure,
		InjectEnvVars:       envvars,
		Region:              h.Configuration().Provision.Region,
		Log: cdslog.Conf{
			GraylogHost:       h.Configuration().Provision.WorkerLogsOptions.Graylog.Host,
			GraylogPort:       strconv.Itoa(h.Configuration().Provision.WorkerLogsOptions.Graylog.Port),
			GraylogExtraKey:   h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraKey,
			GraylogExtraValue: h.Configuration().Provision.WorkerLogsOptions.Graylog.ExtraValue,
			Level:             h.Configuration().Provision.WorkerLogsOptions.Level,
		},
	}

	log.Debug(ctx, "worker config: %v", cfg.EncodeBase64())
	return cfg
}
