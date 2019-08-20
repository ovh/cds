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
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/tracingutils"
)

const nbPanicsBeforeFail = 50

// Router is a wrapper around mux.Router
type Router struct {
	Background context.Context
	//	AuthDriver             auth.Driver
	Mux                    *mux.Router
	SetHeaderFunc          func() map[string]string
	Prefix                 string
	URL                    string
	Middlewares            []service.Middleware
	PostMiddlewares        []service.Middleware
	mapRouterConfigs       map[string]*service.RouterConfig
	mapAsynchronousHandler map[string]service.HandlerFunc
	panicked               bool
	nbPanic                int
	lastPanic              *time.Time
	Stats                  struct {
		Errors     *stats.Int64Measure
		Hits       *stats.Int64Measure
		SSEClients *stats.Int64Measure
		SSEEvents  *stats.Int64Measure
	}
}

func newRouter(m *mux.Router, p string) *Router {
	r := &Router{
		Mux:                    m,
		Prefix:                 p,
		URL:                    "",
		mapRouterConfigs:       map[string]*service.RouterConfig{},
		mapAsynchronousHandler: map[string]service.HandlerFunc{},
		Background:             context.Background(),
	}
	return r
}

// HandlerConfigParam is a type used in handler configuration, to set specific config on a route given a method
type HandlerConfigParam func(*service.HandlerConfig)

// HandlerConfigFunc is a type used in the router configuration fonction "Handle"
type HandlerConfigFunc func(service.Handler, ...HandlerConfigParam) *service.HandlerConfig

func (r *Router) pprofLabel(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		labels := pprof.Labels("http-path", r.URL.Path)
		ctx := pprof.WithLabels(r.Context(), labels)
		pprof.SetGoroutineLabels(ctx)
		r = r.WithContext(ctx)
		fn(w, r)
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

				// the SSE handler can panic, and it's the way gorilla/mux works :(
				if strings.HasPrefix(req.URL.String(), "/events") {
					msg := fmt.Sprintf("%v", err)
					for _, s := range handledEventErrors {
						if strings.Contains(msg, s) {
							return
						}
					}
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

				service.WriteError(w, req, err)
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
func (r *Router) Handle(uri string, scope HandlerScope, handlers ...*service.HandlerConfig) {
	uri = r.Prefix + uri
	cfg := &service.RouterConfig{
		Config: map[string]*service.HandlerConfig{},
	}
	if r.mapRouterConfigs == nil {
		r.mapRouterConfigs = map[string]*service.RouterConfig{}
	}
	r.mapRouterConfigs[uri] = cfg

	for i := range handlers {
		handlers[i].AllowedScopes = scope
		cfg.Config[handlers[i].Method] = handlers[i]
	}

	f := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		dateRFC5322 := req.Header.Get("Date")
		dateReq, err := sdk.ParseDateRFC5322(dateRFC5322)
		if err == nil {
			ctx = context.WithValue(ctx, contextDate, dateReq)
		}

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
		observability.Record(r.Background, r.Stats.Hits, 1)

		//Get route configuration
		rc := cfg.Config[req.Method]
		if rc == nil || rc.Handler == nil {
			observability.Record(r.Background, r.Stats.Errors, 1)
			service.WriteError(w, req, sdk.ErrNotFound)
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
				observability.Record(r.Background, r.Stats.Errors, 1)
				service.WriteError(w, req, err)
				return
			}
		}

		if err := rc.Handler(ctx, w, req); err != nil {
			observability.Record(r.Background, r.Stats.Errors, 1)
			observability.End(ctx, w, req)
			service.WriteError(w, req, err)
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

func (r *asynchronousRequest) do(ctx context.Context, h service.AsynchronousHandler) error {
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

func processAsyncRequests(ctx context.Context, chanRequest chan asynchronousRequest, handlerFunc service.AsynchronousHandlerFunc, retry int) {
	handler := handlerFunc()
	for {
		select {
		case req := <-chanRequest:
			if err := req.do(ctx, handler); err != nil {
				myError, ok := err.(sdk.Error)
				if ok && myError.Status >= 500 {
					if req.nbErrors > retry {
						log.Error("Asynchronous Request on Error: %v with status:%d", err, myError.Status)
					} else {
						chanRequest <- req
					}
				} else {
					log.Error("Asynchronous Request on Error: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Asynchronous handles an AsynchronousHandlerFunc
func (r *Router) Asynchronous(handler service.AsynchronousHandlerFunc, retry int) service.HandlerFunc {
	chanRequest := make(chan asynchronousRequest, 1000)
	sdk.GoRoutine(r.Background, "", func(ctx context.Context) {
		processAsyncRequests(ctx, chanRequest, handler, retry)
	})
	return func() service.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			async := asynchronousRequest{
				contextValues: ContextValues(ctx),
				request:       *r,
				vars:          mux.Vars(r),
			}
			if btes, err := ioutil.ReadAll(r.Body); err == nil {
				async.body = bytes.NewBuffer(btes)
			}
			log.Debug("Router> Asynchronous call of %s", r.URL.String())
			chanRequest <- async
			return service.Accepted(w)
		}
	}
}

// DEPRECATED marks the handler as deprecated
var DEPRECATED = func(rc *service.HandlerConfig) {
	rc.IsDeprecated = true
}

// GET will set given handler only for GET request
func (r *Router) GET(h service.HandlerFunc, cfg ...HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.NeedAuth = true
	rc.Method = "GET"
	rc.PermissionLevel = sdk.PermissionRead
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// POST will set given handler only for POST request
func (r *Router) POST(h service.HandlerFunc, cfg ...HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.NeedAuth = true
	rc.Method = "POST"
	rc.PermissionLevel = sdk.PermissionReadWriteExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// POSTEXECUTE will set given handler only for POST request and add a flag for execution permission
func (r *Router) POSTEXECUTE(h service.HandlerFunc, cfg ...HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.NeedAuth = true
	rc.Method = "POST"
	rc.PermissionLevel = sdk.PermissionReadExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// PUT will set given handler only for PUT request
func (r *Router) PUT(h service.HandlerFunc, cfg ...HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.NeedAuth = true
	rc.Method = "PUT"
	rc.PermissionLevel = sdk.PermissionReadWriteExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// DELETE will set given handler only for DELETE request
func (r *Router) DELETE(h service.HandlerFunc, cfg ...HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.NeedAuth = true
	rc.Method = "DELETE"
	rc.PermissionLevel = sdk.PermissionReadWriteExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// NeedAdmin set the route for cds admin only (or not)
func NeedAdmin(admin bool) HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.NeedAdmin = admin
	}
	return f
}

// AllowProvider set the route for external providers
func AllowProvider(need bool) HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.AllowProvider = need
	}
	return f
}

// NeedToken set the route for requests that have the given header
func NeedToken(k, v string) HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.AllowedTokens = append(rc.AllowedTokens, fmt.Sprintf("%s:%s", k, v))
	}
	return f
}

// Auth set manually whether authorisation layer should be applied
// Authorization is enabled by default
func Auth(v bool) HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.NeedAuth = v
	}
	return f
}

// MaintenanceAware route need CDS maintenance off
func MaintenanceAware() HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.MaintenanceAware = true
	}
	return f
}

// EnableTracing on a route
func EnableTracing() HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.EnableTracing = true
	}
	return f
}

// NotFoundHandler is called by default by Mux is any matching handler has been found
func NotFoundHandler(w http.ResponseWriter, req *http.Request) {
	service.WriteError(w, req, sdk.ErrNotFound)
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

// InitMetrics initialize prometheus metrics
func (r *Router) InitMetrics(service, name string) error {
	label := fmt.Sprintf("cds/%s/%s/router_errors", service, name)
	r.Stats.Errors = stats.Int64(label, "number of errors", stats.UnitDimensionless)
	label = fmt.Sprintf("cds/%s/%s/router_hits", service, name)
	r.Stats.Hits = stats.Int64(label, "number of hits", stats.UnitDimensionless)
	label = fmt.Sprintf("cds/%s/%s/sse_clients", service, name)
	r.Stats.SSEClients = stats.Int64(label, "number of sse clients", stats.UnitDimensionless)
	label = fmt.Sprintf("cds/%s/%s/sse_events", service, name)
	r.Stats.SSEEvents = stats.Int64(label, "number of sse events", stats.UnitDimensionless)

	tagCDSInstance, _ := tag.NewKey("cds")
	tags := []tag.Key{tagCDSInstance}

	log.Info("Stats initialized")

	return observability.RegisterView(
		observability.NewViewCount("router_errors", r.Stats.Errors, tags),
		observability.NewViewCount("router_hits", r.Stats.Hits, tags),
		observability.NewViewLast("sse_clients", r.Stats.SSEClients, tags),
		observability.NewViewCount("sse_events", r.Stats.SSEEvents, tags),
	)
}
