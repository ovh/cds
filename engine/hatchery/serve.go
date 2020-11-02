package hatchery

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/square/go-jose.v2"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

type Common struct {
	service.Common
	Router                        *api.Router
	mapServiceNextLineNumberMutex sync.Mutex
	mapServiceNextLineNumber      map[string]int64
}

const panicDumpDir = "panic_dumps"

func (c *Common) servePanicDumpList() ([]string, error) {
	dir, _ := os.Getwd()
	path := filepath.Join(dir, panicDumpDir)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	res := make([]string, len(files))
	for i, f := range files {
		res[i] = f.Name()
	}
	return res, nil
}

func init() {
	// This go routine deletes panic dumps older than 15 minutes
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			dir, err := os.Getwd()
			if err != nil {
				log.Warning(context.Background(), "unable to get working directory: %v", err)
				continue
			}

			path := filepath.Join(dir, panicDumpDir)
			_ = os.MkdirAll(path, os.FileMode(0755))

			files, err := ioutil.ReadDir(path)
			if err != nil {
				log.Warning(context.Background(), "unable to list files in %s: %v", path, err)
				break
			}

			for _, f := range files {
				filename := filepath.Join(path, f.Name())
				file, err := os.Stat(filename)
				if err != nil {
					log.Warning(context.Background(), "unable to get file %s info: %v", f.Name(), err)
					continue
				}
				if file.ModTime().Before(time.Now().Add(-15 * time.Minute)) {
					if err := os.Remove(filename); err != nil {
						log.Warning(context.Background(), "unable to remove file %s: %v", filename, err)
					}
				}
			}
		}
	}()
}

func (c *Common) servePanicDump(f string) (io.ReadCloser, error) {
	dir, _ := os.Getwd()
	path := filepath.Join(dir, panicDumpDir, f)
	return os.OpenFile(path, os.O_RDONLY, os.FileMode(0644))
}

func (c *Common) PanicDumpDirectory() (string, error) {
	dir, _ := os.Getwd()
	path := filepath.Join(dir, panicDumpDir)
	return path, os.MkdirAll(path, os.FileMode(0755))
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
	log.Debug("%s> Router initialized", c.Name())
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
	r.Handle("/mon/errors", nil, r.GET(c.getPanicDumpListHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/errors/{id}", nil, r.GET(c.getPanicDumpHandler, service.OverrideAuth(service.NoAuthMiddleware)))

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

func (c *Common) getPanicDumpListHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		l, err := c.servePanicDumpList()
		if err != nil {
			return err
		}
		return service.WriteJSON(w, l, http.StatusOK)
	}
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
		logger, _, err := log.New(ctx, graylogCfg)
		if err != nil {
			return sdk.WithStack(err)
		}
		c.ServiceLogger = logger
	} else {
		if err := log.ReplaceAllHooks(context.Background(), c.ServiceLogger, graylogCfg); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

func (c *Common) SendServiceLog(ctx context.Context, servicesLogs []log.Message, terminated bool) {
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
			log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			continue
		}
		lineNumber := c.mapServiceNextLineNumber[s.ServiceKey()]
		c.mapServiceNextLineNumber[s.ServiceKey()]++
		if c.ServiceLogger != nil {
			c.ServiceLogger.
				WithField(log.ExtraFieldSignature, sign).
				WithField(log.ExtraFieldLine, lineNumber).
				WithField(log.ExtraFieldTerminated, terminated).
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

func (c *Common) getPanicDumpHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		id := vars["id"]
		f, err := c.servePanicDump(id)
		if err != nil {
			return err
		}
		defer f.Close() // nolint

		if _, err := io.Copy(w, f); err != nil {
			return err
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		return nil
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
