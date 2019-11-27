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
	"reflect"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

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

var (
	onceMetrics         sync.Once
	Errors              *stats.Int64Measure
	Hits                *stats.Int64Measure
	SSEClients          *stats.Int64Measure
	SSEEvents           *stats.Int64Measure
	ServerRequestCount  *stats.Int64Measure
	ServerRequestBytes  *stats.Int64Measure
	ServerResponseBytes *stats.Int64Measure
	ServerLatency       *stats.Float64Measure
)

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
}

// HandlerConfigParam is a type used in handler configuration, to set specific config on a route given a method
type HandlerConfigParam func(*service.HandlerConfig)

// HandlerConfigFunc is a type used in the router configuration fonction "Handle"
type HandlerConfigFunc func(service.Handler, ...HandlerConfigParam) *service.HandlerConfig

func (r *Router) pprofLabel(config map[string]*service.HandlerConfig, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var name = sdk.RandomString(12)
		rc := config[r.Method]
		if rc != nil && rc.Handler != nil {
			name = runtime.FuncForPC(reflect.ValueOf(rc.Handler).Pointer()).Name()
			name = strings.Replace(name, ".func1", "", 1)
			name = strings.Replace(name, ".1", "", 1)
		}
		id := fmt.Sprintf("%d", sdk.GoroutineID())

		labels := pprof.Labels(
			"http-path", r.URL.Path,
			"goroutine-id", id,
			"goroutine-name", name+"-"+id,
		)
		ctx := pprof.WithLabels(r.Context(), labels)
		pprof.SetGoroutineLabels(ctx)
		r = r.WithContext(ctx)
		fn(w, r)
	}
}

func (r *Router) compress(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Disable GZIP compression on prometheus call
		if !strings.Contains(r.Header.Get("User-Agent"), "Prometheus") {
			handlers.CompressHandlerLevel(fn, gzip.DefaultCompression).ServeHTTP(w, r)
		} else {
			fn(w, r)
		}
	}
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

				log.Error(context.TODO(), "[PANIC_RECOVERY] Panic occurred on %s:%s, recover %s", req.Method, req.URL.String(), err)
				trace := make([]byte, 4096)
				count := runtime.Stack(trace, true)
				log.Error(context.TODO(), "[PANIC_RECOVERY] Stacktrace of %d bytes\n%s\n", count, trace)

				//Checking if there are two much panics in two minutes
				//If last panic was more than 2 minutes ago, reinit the panic counter
				if r.lastPanic == nil {
					r.nbPanic = 0
				} else {
					dur := time.Since(*r.lastPanic)
					if dur.Minutes() > float64(2) {
						log.Info(context.Background(), "[PANIC_RECOVERY] Last panic was %d seconds ago", int(dur.Seconds()))
						r.nbPanic = 0
					}
				}

				r.nbPanic++
				now := time.Now()
				r.lastPanic = &now
				//If two much panic, change the status of /mon/status with panicked = true
				if r.nbPanic > nbPanicsBeforeFail {
					r.panicked = true
					log.Error(context.TODO(), "[PANIC_RECOVERY] RESTART NEEDED")
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
		name := runtime.FuncForPC(reflect.ValueOf(handlers[i].Handler).Pointer()).Name()
		name = strings.Replace(name, ".func1", "", 1)
		name = strings.Replace(name, ".1", "", 1)
		name = strings.Replace(name, "github.com/ovh/cds/engine/", "", 1)
		log.Debug("Registering handler %s on %s %s", name, handlers[i].Method, uri)
		handlers[i].Name = name
	}

	f := func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()

		dateRFC5322 := req.Header.Get("Date")
		dateReq, err := sdk.ParseDateRFC5322(dateRFC5322)
		if err == nil {
			ctx = context.WithValue(ctx, contextDate, dateReq)
		}

		responseWriter := &trackingResponseWriter{
			writer: w,
		}
		if req.Body == nil {
			responseWriter.reqSize = -1
		} else if req.ContentLength > 0 {
			responseWriter.reqSize = req.ContentLength
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

		//Get route configuration
		rc := cfg.Config[req.Method]
		if rc == nil || rc.Handler == nil {
			observability.Record(ctx, Errors, 1)
			service.WriteError(w, req, sdk.ErrNotFound)
			return
		}

		// Make the request context inherit from the context of the router
		tags := observability.ContextGetTags(r.Background, observability.TagServiceType, observability.TagServiceName)
		ctx, err = tag.New(ctx, tags...)
		if err != nil {
			log.Error(ctx, "observability.ContextGetTags> %v", err)
		}
		ctx = observability.ContextWithTag(ctx,
			observability.Handler, rc.Name,
			observability.Host, req.Host,
			observability.Path, req.URL.Path,
			observability.Method, req.Method)

		//Log request
		start := time.Now()
		defer func(ctx context.Context) {
			if responseWriter.statusCode == 0 {
				responseWriter.statusCode = 200
			}
			ctx = observability.ContextWithTag(ctx, observability.StatusCode, responseWriter.statusCode)

			end := time.Now()
			latency := end.Sub(start)
			if rc.IsDeprecated {
				log.Error(ctx, "[%-3d] | %-7s | %13v | DEPRECATED ROUTE | %v [%s]", responseWriter.statusCode, req.Method, latency, req.URL, rc.Name)
			} else {
				log.Info(ctx, "[%-3d] | %-7s | %13v | %v [%s]", responseWriter.statusCode, req.Method, latency, req.URL, rc.Name)
			}

			observability.RecordFloat64(ctx, ServerLatency, float64(latency)/float64(time.Millisecond))
			observability.Record(ctx, ServerRequestBytes, responseWriter.reqSize)
			observability.Record(ctx, ServerResponseBytes, responseWriter.respSize)
		}(ctx)

		observability.Record(r.Background, Hits, 1)
		observability.Record(ctx, ServerRequestCount, 1)

		for _, m := range r.Middlewares {
			var err error
			ctx, err = m(ctx, responseWriter, req, rc)
			if err != nil {
				observability.Record(r.Background, Errors, 1)
				service.WriteError(w, req, err)
				return
			}
		}

		if err := rc.Handler(ctx, responseWriter.wrappedResponseWriter(), req); err != nil {
			observability.Record(r.Background, Errors, 1)
			observability.End(ctx, responseWriter, req)
			service.WriteError(responseWriter, req, err)
			return
		}

		// writeNoContentPostMiddleware is compliant Middleware Interface
		// but no need to check ct, err in return
		writeNoContentPostMiddleware(ctx, responseWriter, req, rc)

		for _, m := range r.PostMiddlewares {
			var err error
			ctx, err = m(ctx, responseWriter, req, rc)
			if err != nil {
				log.Error(ctx, "PostMiddlewares > %s", err)
			}
		}
	}

	// The chain is http -> mux -> f -> recover -> wrap -> pprof -> opencensus -> http
	r.Mux.Handle(uri, r.pprofLabel(cfg.Config, r.compress(r.recoverWrap(f))))
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
	//Recreate a new buffer from the bytes stores in memory
	req.Body = ioutil.NopCloser(tee)
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
						log.Error(ctx, "Asynchronous Request on Error: %v with status:%d", err, myError.Status)
					} else {
						chanRequest <- req
					}
				} else {
					log.Error(ctx, "Asynchronous Request on Error: %v", err)
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
