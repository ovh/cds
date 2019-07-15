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
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

type Common struct {
	service.Common
	Router     *api.Router
	metrics    hatchery.Metrics
	privateKey *rsa.PrivateKey
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
				log.Warning("unable to get working directory: %v", err)
				continue
			}

			path := filepath.Join(dir, panicDumpDir)
			_ = os.MkdirAll(path, os.FileMode(0755))

			files, err := ioutil.ReadDir(path)
			if err != nil {
				log.Warning("unable to list files in %s: %v", path, err)
				break
			}

			for _, f := range files {
				filename := filepath.Join(path, f.Name())
				file, err := os.Stat(filename)
				if err != nil {
					log.Warning("unable to get file %s info: %v", f.Name(), err)
					continue
				}
				if file.ModTime().Before(time.Now().Add(-15 * time.Minute)) {
					if err := os.Remove(filename); err != nil {
						log.Warning("unable to remove file %s: %v", filename, err)
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

func (c *Common) PrivateKey() *rsa.PrivateKey {
	return c.privateKey
}

// CommonServe start the HatcheryLocal server
func (c *Common) CommonServe(ctx context.Context, h hatchery.Interface) error {
	log.Info("%s> Starting service %s (%s)...", c.Name, h.Configuration().Name, sdk.VERSION)
	c.StartupTime = time.Now()

	var err error
	c.privateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(h.Configuration().RSAPrivateKey))
	if err != nil {
		return fmt.Errorf("unable to parse RSA private Key: %v", err)
	}

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

	if err := c.initMetrics(h.Configuration().Name); err != nil {
		return err
	}

	if err := hatchery.Create(ctx, h); err != nil {
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
	r.Middlewares = append(r.Middlewares, service.CheckRequestSignatureMiddleware(c.ParsedAPIPublicKey))

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(getStatusHandler(h), api.Auth(false)))
	r.Handle("/mon/workers", nil, r.GET(getWorkersPoolHandler(h), api.Auth(false)))
	r.Handle("/mon/metrics", nil, r.GET(observability.StatsHandler, api.Auth(false)))
	r.Handle("/mon/errors", nil, r.GET(c.getPanicDumpListHandler, api.Auth(false)))
	r.Handle("/mon/errors/{id}", nil, r.GET(c.getPanicDumpHandler, api.Auth(false)))

	r.Mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.Mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.Mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.Mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	// need 2 routes for index, one for index page, another with {action}
	r.Mux.HandleFunc("/debug/pprof/{action}", pprof.Index)
	r.Mux.HandleFunc("/debug/pprof/", pprof.Index)

	r.Mux.NotFoundHandler = http.HandlerFunc(api.NotFoundHandler)
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
			status := srv.Status()
			return service.WriteJSON(w, status, status.HTTPStatusCode())
		}
	}
}
