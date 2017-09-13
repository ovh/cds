package api

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const nbPanicsBeforeFail = 50

// Router is a wrapper around mux.Router
type Router struct {
	Background       context.Context
	AuthDriver       auth.Driver
	Mux              *mux.Router
	SetHeaderFunc    func() map[string]string
	Prefix           string
	URL              string
	Middlewares      []Middleware
	mapRouterConfigs map[string]*RouterConfig
	panicked         bool
	nbPanic          int
	lastPanic        *time.Time
}

// Handler defines the HTTP handler used in CDS engine
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Middleware defines the HTTP Middleware used in CDS engine
type Middleware func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error)

// HandlerFunc defines the way to a handler
type HandlerFunc func() Handler

// RouterConfigParam is the type of anonymous function returned by POST, GET and PUT functions
type RouterConfigParam func(rc *RouterConfig)

// RouterConfig contains a map of handler configuration. Key is the method of the http route
type RouterConfig struct {
	config map[string]*HandlerConfig
}

// HandlerConfig is the configuration for one handler
type HandlerConfig struct {
	Method       string
	Handler      Handler
	IsDeprecated bool
	Options      map[string]string
}

// NewHandlerConfig returns a new HandlerConfig pointer
func NewHandlerConfig() *HandlerConfig {
	return &HandlerConfig{
		Options: map[string]string{},
	}
}

func newRouter(a auth.Driver, m *mux.Router, p string) *Router {
	return &Router{
		AuthDriver:       a,
		Mux:              m,
		Prefix:           p,
		URL:              "",
		mapRouterConfigs: map[string]*RouterConfig{},
		Background:       context.Background(),
	}
}

// HandlerConfigParam is a type used in handler configuration, to set specific config on a route given a method
type HandlerConfigParam func(*HandlerConfig)

// HandlerConfigFunc is a type used in the router configuration fonction "Handle"
type HandlerConfigFunc func(Handler, ...HandlerConfigParam) *HandlerConfig

// ServeAbsoluteFile Serve file to download
func (r *Router) ServeAbsoluteFile(uri, path, filename string) {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=\"%s\"", filename))
		http.ServeFile(w, r, path)
	}
	r.Mux.HandleFunc(r.Prefix+uri, f)
}

func (r *Router) compress(fn http.HandlerFunc) http.HandlerFunc {
	return handlers.CompressHandlerLevel(fn, gzip.DefaultCompression).ServeHTTP
}

func (r *Router) recoverWrap(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var err error
		defer func() {
			if re := recover(); re != nil {
				switch t := re.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = re.(error)
				case sdk.Error:
					err = re.(sdk.Error)
				default:
					err = sdk.ErrUnknownError
				}
				log.Error("[PANIC_RECOVERY] Panic occurred on %s:%s, recover %s", req.Method, req.URL.String(), err)
				trace := make([]byte, 4096)
				count := runtime.Stack(trace, true)
				log.Error("[PANIC_RECOVERY] Stacktrace of %d bytes\n%s\n", count, trace)

				//Checking if there are two much panics in two minutes
				//If last panic was more than 2 minutes ago, reinit the panic counter
				if r.lastPanic == nil {
					r.nbPanic = 0
				} else {
					dur := time.Since(*r.lastPanic)
					if dur.Minutes() > float64(2) {
						log.Info("[PANIC_RECOVERY] Last panic was %d seconds ago", int(dur.Seconds()))
						r.nbPanic = 0
					}
				}

				r.nbPanic++
				now := time.Now()
				r.lastPanic = &now
				//If two much panic, change the status of /mon/status with panicked = true
				if r.nbPanic > nbPanicsBeforeFail {
					r.panicked = true
					log.Error("[PANIC_RECOVERY] RESTART NEEDED")
				}

				WriteError(w, req, err)
			}
		}()
		h.ServeHTTP(w, req)
	})
}

func defaultHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":   "*",
		"Access-Control-Allow-Methods":  "GET,OPTIONS,PUT,POST,DELETE",
		"Access-Control-Allow-Headers":  "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id, If-Modified-Since, Content-Disposition",
		"Access-Control-Expose-Headers": "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id, ETag, Content-Disposition",
		"X-Api-Time":                    time.Now().Format(time.RFC3339),
		"ETag":                          fmt.Sprintf("%d", time.Now().Unix()),
	}
}

// Handle adds all handler for their specific verb in gorilla router for given uri
func (r *Router) Handle(uri string, handlers ...*HandlerConfig) {
	uri = r.Prefix + uri
	cfg := &RouterConfig{
		config: map[string]*HandlerConfig{},
	}
	if r.mapRouterConfigs == nil {
		r.mapRouterConfigs = map[string]*RouterConfig{}
	}
	r.mapRouterConfigs[uri] = cfg

	for i := range handlers {
		cfg.config[handlers[i].Method] = handlers[i]
	}

	f := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// Close indicates  to close the connection after replying to this request
		req.Close = true

		// Set default headers
		if r.SetHeaderFunc != nil {
			headers := r.SetHeaderFunc()
			for k, v := range headers {
				w.Header().Add(k, v)
			}
		}

		//Always returns OK on Options method
		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		//Get route configuration
		rc := cfg.config[req.Method]
		if rc == nil || rc.Handler == nil {
			WriteError(w, req, sdk.ErrNotFound)
			return
		}

		//Log request
		start := time.Now()
		defer func() {
			end := time.Now()
			latency := end.Sub(start)
			if rc.IsDeprecated {
				log.Error("%-7s | %13v | DEPRECATED ROUTE | %v", req.Method, latency, req.URL)
				w.Header().Add("X-CDS-WARNING", "deprecated route")
			} else {
				log.Debug("%-7s | %13v | %v", req.Method, latency, req.URL)
			}
		}()

		for _, m := range r.Middlewares {
			var err error
			ctx, err = m(ctx, w, req, rc)
			if err != nil {
				WriteError(w, req, err)
				return
			}
		}

		if err := rc.Handler(ctx, w, req); err != nil {
			WriteError(w, req, err)
			return
		}

	}

	r.Mux.HandleFunc(uri, r.compress(r.recoverWrap(f)))
}

// DEPRECATED marks the handler as deprecated
var DEPRECATED = func(rc *HandlerConfig) {
	rc.Options["isDeprecated"] = "true"
}

// GET will set given handler only for GET request
func (r *Router) GET(h HandlerFunc, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := NewHandlerConfig()
	rc.Handler = h()
	rc.Options["auth"] = "true"
	rc.Method = "GET"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// POST will set given handler only for POST request
func (r *Router) POST(h HandlerFunc, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := NewHandlerConfig()
	rc.Handler = h()
	rc.Options["auth"] = "true"
	rc.Method = "POST"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// POSTEXECUTE will set given handler only for POST request and add a flag for execution permission
func (r *Router) POSTEXECUTE(h HandlerFunc, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := NewHandlerConfig()
	rc.Handler = h()
	rc.Options["auth"] = "true"
	rc.Method = "POST"
	rc.Options["isExecution"] = "true"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// PUT will set given handler only for PUT request
func (r *Router) PUT(h HandlerFunc, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := NewHandlerConfig()
	rc.Handler = h()
	rc.Options["auth"] = "true"
	rc.Method = "PUT"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// DELETE will set given handler only for DELETE request
func (r *Router) DELETE(h HandlerFunc, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := NewHandlerConfig()
	rc.Handler = h()
	rc.Options["auth"] = "true"
	rc.Method = "DELETE"
	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// NeedAdmin set the route for cds admin only (or not)
func NeedAdmin(admin bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["needAdmin"] = fmt.Sprintf("%v", admin)
	}
	return f
}

// NeedUsernameOrAdmin set the route for cds admin or current user = username called on route
func NeedUsernameOrAdmin(need bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["needUsernameOrAdmin"] = fmt.Sprintf("%v", need)
	}
	return f
}

// NeedHatchery set the route for hatchery only
func NeedHatchery() HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["needHatchery"] = "true"
	}
	return f
}

// NeedWorker set the route for worker only
func NeedWorker() HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["needWorker"] = "true"
	}
	return f
}

// Auth set manually whether authorisation layer should be applied
// Authorization is enabled by default
func Auth(v bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["auth"] = fmt.Sprintf("%v", v)
	}
	return f
}

func notFoundHandler(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	defer func() {
		end := time.Now()
		latency := end.Sub(start)
		log.Warning("%-7s | %13v | %v", req.Method, latency, req.URL)
	}()
	WriteError(w, req, sdk.ErrNotFound)
}
