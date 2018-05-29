package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	muxcontext "github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/tracingutils"
)

const nbPanicsBeforeFail = 50

// Router is a wrapper around mux.Router
type Router struct {
	Background             context.Context
	AuthDriver             auth.Driver
	Mux                    *mux.Router
	SetHeaderFunc          func() map[string]string
	Prefix                 string
	URL                    string
	Middlewares            []Middleware
	PostMiddlewares        []Middleware
	mapRouterConfigs       map[string]*RouterConfig
	mapAsynchronousHandler map[string]HandlerFunc
	panicked               bool
	nbPanic                int
	lastPanic              *time.Time
}

// Handler defines the HTTP handler used in CDS engine
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// AsynchronousHandler defines the HTTP asynchronous handler used in CDS engine
type AsynchronousHandler func(ctx context.Context, r *http.Request) error

// Middleware defines the HTTP Middleware used in CDS engine
type Middleware func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error)

// HandlerFunc defines the way to instanciate a handler
type HandlerFunc func() Handler

// AsynchronousHandlerFunc defines the way to instanciate a handler
type AsynchronousHandlerFunc func() AsynchronousHandler

// RouterConfigParam is the type of anonymous function returned by POST, GET and PUT functions
type RouterConfigParam func(rc *RouterConfig)

// RouterConfig contains a map of handler configuration. Key is the method of the http route
type RouterConfig struct {
	config map[string]*HandlerConfig
}

// HandlerConfig is the configuration for one handler
type HandlerConfig struct {
	Name         string
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
		AuthDriver:             a,
		Mux:                    m,
		Prefix:                 p,
		URL:                    "",
		mapRouterConfigs:       map[string]*RouterConfig{},
		mapAsynchronousHandler: map[string]HandlerFunc{},
		Background:             context.Background(),
	}
}

// HandlerConfigParam is a type used in handler configuration, to set specific config on a route given a method
type HandlerConfigParam func(*HandlerConfig)

// HandlerConfigFunc is a type used in the router configuration fonction "Handle"
type HandlerConfigFunc func(Handler, ...HandlerConfigParam) *HandlerConfig

func (r *Router) pprofLabel(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		labels := pprof.Labels("http-path", r.URL.Path)
		pprof.Do(r.Context(), labels, func(ctx context.Context) {
			fn.ServeHTTP(w, r)
		})
	}
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

var headers = []string{
	http.CanonicalHeaderKey(tracingutils.TraceIDHeader),
	http.CanonicalHeaderKey(tracingutils.SpanIDHeader),
	http.CanonicalHeaderKey(tracingutils.SampledHeader),
	http.CanonicalHeaderKey(sdk.WorkflowAsCodeHeader),
	http.CanonicalHeaderKey(sdk.ResponseWorkflowIDHeader),
	http.CanonicalHeaderKey(sdk.ResponseWorkflowNameHeader),
}

// DefaultHeaders is a set of default header for the router
func DefaultHeaders() map[string]string {
	now := time.Now()
	return map[string]string{
		"Access-Control-Allow-Origin":              "*",
		"Access-Control-Allow-Methods":             "GET,OPTIONS,PUT,POST,DELETE",
		"Access-Control-Allow-Headers":             "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id, If-Modified-Since, Content-Disposition, " + strings.Join(headers, ", "),
		"Access-Control-Expose-Headers":            "Accept, Origin, Referer, User-Agent, Content-Type, Authorization, Session-Token, Last-Event-Id, ETag, Content-Disposition, " + strings.Join(headers, ", "),
		cdsclient.ResponseAPINanosecondsTimeHeader: fmt.Sprintf("%d", now.UnixNano()),
		cdsclient.ResponseAPITimeHeader:            now.Format(time.RFC3339),
		cdsclient.ResponseEtagHeader:               fmt.Sprintf("%d", now.Unix()),
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

		// writeNoContentPostMiddleware is compliant Middleware Interface
		// but no need to check ct, err in return
		writeNoContentPostMiddleware(ctx, w, req, rc)

		for _, m := range r.PostMiddlewares {
			var err error
			ctx, err = m(ctx, w, req, rc)
			if err != nil {
				log.Error("PostMiddlewares > %s", err)
			}
		}
	}

	// The chain is http -> mux -> f -> recover -> wrap -> pprof -> opencensus -> http
	r.Mux.Handle(uri, r.pprofLabel(r.compress(r.recoverWrap(f))))
}

type asynchronousRequest struct {
	nbErrors      int
	err           error
	contextValues map[interface{}]interface{}
	vars          map[string]string
	request       http.Request
	body          io.Reader
}

func (r *asynchronousRequest) do(ctx context.Context, h AsynchronousHandler) error {
	for k, v := range r.contextValues {
		ctx = context.WithValue(ctx, k, v)
	}
	req := &r.request

	var buf bytes.Buffer
	tee := io.TeeReader(r.body, &buf)
	r.body = &buf
	req.Body = ioutil.NopCloser(tee)
	//Recreate a new buffer from the bytes stores in memory
	for k, v := range r.vars {
		muxcontext.Set(req, k, v)
	}
	r.err = h(ctx, req)
	if r.err != nil {
		r.nbErrors++
	}
	return r.err
}

func processAsyncRequests(ctx context.Context, chanRequest chan asynchronousRequest, handlerFunc AsynchronousHandlerFunc, retry int) {
	handler := handlerFunc()
	for {
		select {
		case req := <-chanRequest:
			if err := req.do(ctx, handler); err != nil {
				if req.nbErrors > retry {
					log.Error("Asynchronous Request on Error : %v", err)
				} else {
					chanRequest <- req
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Asynchronous handles an AsynchronousHandlerFunc
func (r *Router) Asynchronous(handler AsynchronousHandlerFunc, retry int) HandlerFunc {
	chanRequest := make(chan asynchronousRequest, runtime.GOMAXPROCS(0))
	go processAsyncRequests(r.Background, chanRequest, handler, retry)

	return func() Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			async := asynchronousRequest{
				contextValues: auth.ContextValues(ctx),
				request:       *r,
				vars:          mux.Vars(r),
			}
			if btes, err := ioutil.ReadAll(r.Body); err == nil {
				async.body = bytes.NewBuffer(btes)
			}
			log.Debug("Router> Asynchronous call of %s", r.URL.String())
			chanRequest <- async
			return Accepted(w)
		}
	}
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
	rc.Options["allowServices"] = "false"
	rc.Options["allowProvider"] = "false"

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
	rc.Options["allowServices"] = "false"
	rc.Options["allowProvider"] = "false"

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
	rc.Options["allowServices"] = "false"
	rc.Method = "POST"
	rc.Options["isExecution"] = "true"
	rc.Options["allowProvider"] = "false"

	for _, c := range cfg {
		c(rc)
	}
	return rc
}

// PUT will set given handler only for PUT request
func (r *Router) PUT(h HandlerFunc, cfg ...HandlerConfigParam) *HandlerConfig {
	rc := NewHandlerConfig()
	rc.Handler = h()
	rc.Options["allowServices"] = "false"
	rc.Options["auth"] = "true"
	rc.Options["allowProvider"] = "false"

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
	rc.Options["allowServices"] = "false"
	rc.Options["auth"] = "true"
	rc.Options["allowProvider"] = "false"

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

// AllowProvider set the route for external providers
func AllowProvider(need bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["allowProvider"] = fmt.Sprintf("%v", need)
	}
	return f
}

// NeedToken set the route for requests that have the given header
func NeedToken(k, v string) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["token"] = fmt.Sprintf("%s:%s", k, v)
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

// NeedService set the route for hatchery only
func NeedService() HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["needService"] = "true"
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

// AllowServices allows CDS service to use this route
func AllowServices(s bool) HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["allowServices"] = fmt.Sprintf("%v", s)
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

// EnableTracing on a route
func EnableTracing() HandlerConfigParam {
	f := func(rc *HandlerConfig) {
		rc.Options["trace_enable"] = "true"
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

// StatusPanic returns router status. If nbPanic > 30 -> Alert, if nbPanic > 0 -> Warn
func (r *Router) StatusPanic() sdk.MonitoringStatusLine {
	statusPanic := sdk.MonitoringStatusOK
	if r.nbPanic > 30 {
		statusPanic = sdk.MonitoringStatusAlert
	} else if r.nbPanic > 0 {
		statusPanic = sdk.MonitoringStatusWarn
	}
	return sdk.MonitoringStatusLine{Component: "Nb of Panics", Value: fmt.Sprintf("%d", r.nbPanic), Status: statusPanic}
}
