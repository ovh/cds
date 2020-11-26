package api

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
	docSDK "github.com/ovh/cds/sdk/doc"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

const nbPanicsBeforeFail = 50

var (
	onceMetrics         sync.Once
	Errors              *stats.Int64Measure
	Hits                *stats.Int64Measure
	WebSocketClients    *stats.Int64Measure
	WebSocketEvents     *stats.Int64Measure
	ServerRequestCount  *stats.Int64Measure
	ServerRequestBytes  *stats.Int64Measure
	ServerResponseBytes *stats.Int64Measure
	ServerLatency       *stats.Float64Measure
)

// Router is a wrapper around mux.Router
type Router struct {
	Background            context.Context
	Mux                   *mux.Router
	SetHeaderFunc         func() map[string]string
	Prefix                string
	URL                   string
	Middlewares           []service.Middleware
	DefaultAuthMiddleware service.Middleware
	PostAuthMiddlewares   []service.Middleware
	PostMiddlewares       []service.Middleware
	mapRouterConfigs      map[string]*service.RouterConfig
	panicked              bool
	nbPanic               int
	lastPanic             *time.Time
	scopeDetails          []sdk.AuthConsumerScopeDetail
}

// HandlerConfigFunc is a type used in the router configuration fonction "Handle"
type HandlerConfigFunc func(service.Handler, ...service.HandlerConfigParam) *service.HandlerConfig

func (r *Router) pprofLabel(config map[string]*service.HandlerConfig, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var name = sdk.RandomString(12)
		rc := config[req.Method]
		if rc != nil && rc.Handler != nil {
			name = runtime.FuncForPC(reflect.ValueOf(rc.Handler).Pointer()).Name()
			name = strings.Replace(name, ".func1", "", 1)
			name = strings.Replace(name, ".1", "", 1)
		}
		id := fmt.Sprintf("%d", sdk.GoroutineID())

		labels := pprof.Labels(
			"http-path", req.URL.Path,
			"goroutine-id", id,
			"goroutine-name", name+"-"+id,
		)

		ctx := pprof.WithLabels(req.Context(), labels)
		pprof.SetGoroutineLabels(ctx)
		req = req.WithContext(ctx)
		fn(w, req)
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

const requestIDHeader = "Request-ID"

func (r *Router) setRequestID(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestID string
		if existingRequestID := r.Header.Get(requestIDHeader); existingRequestID != "" {
			if _, err := uuid.FromString(existingRequestID); err == nil {
				requestID = existingRequestID
			}
		}
		if requestID == "" {
			requestID = sdk.UUID()
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, log.ContextLoggingRequestIDKey, requestID)
		r = r.WithContext(ctx)

		w.Header().Set(requestIDHeader, requestID)

		h(w, r)
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
				case sdk.Error:
					err = re.(sdk.Error)
				case error:
					err = re.(error)
				default:
					err = sdk.ErrUnknownError
				}

				log.Error(context.TODO(), "[PANIC_RECOVERY] Panic occurred on %s:%s, recover %s", req.Method, req.URL.String(), err)

				trace := make([]byte, 4096)
				count := runtime.Stack(trace, true)
				log.Error(req.Context(), "[PANIC_RECOVERY] Stacktrace of %d bytes\n%s\n", count, trace)

				//Checking if there are two much panics in two minutes
				//If last panic was more than 2 minutes ago, reinit the panic counter
				if r.lastPanic == nil {
					r.nbPanic = 0
				} else {
					dur := time.Since(*r.lastPanic)
					if dur.Minutes() > float64(2) {
						log.Info(req.Context(), "[PANIC_RECOVERY] Last panic was %d seconds ago", int(dur.Seconds()))
						r.nbPanic = 0
					}
				}

				r.nbPanic++
				now := time.Now()
				r.lastPanic = &now
				//If two much panic, change the status of /mon/status with panicked = true
				if r.nbPanic > nbPanicsBeforeFail {
					r.panicked = true
					log.Error(req.Context(), "[PANIC_RECOVERY] RESTART NEEDED")
				}

				service.WriteError(req.Context(), w, req, err)
			}
		}()
		h.ServeHTTP(w, req)
	})
}

// computeScopeDetails iterate over declared handlers for routers and populate router scope details.
func (r *Router) computeScopeDetails() {
	// create temporary map of scopes, for each scope we will create a map of routes with methods.
	m := make(map[sdk.AuthConsumerScope]map[string]map[string]struct{})

	for uri, cfg := range r.mapRouterConfigs {
		var err error
		uri, err = docSDK.CleanAndCheckURL(uri)
		if err != nil {
			panic(errors.Wrap(err, "error computing scope detail"))
		}

		if len(cfg.Config) == 0 {
			continue
		}

		methods := make([]string, 0, len(cfg.Config))
		var scopes []sdk.AuthConsumerScope
		for method, handler := range cfg.Config {
			// Take scopes from the first handler as every handlers should have the same scopes
			if scopes == nil {
				scopes = handler.AllowedScopes
			}
			methods = append(methods, method)
		}

		for i := range scopes {
			if _, ok := m[scopes[i]]; !ok {
				m[scopes[i]] = make(map[string]map[string]struct{})
			}
			if _, ok := m[scopes[i]][uri]; !ok {
				m[scopes[i]][uri] = make(map[string]struct{})
			}
			for j := range methods {
				m[scopes[i]][uri][methods[j]] = struct{}{}
			}
		}
	}

	// return scope details
	details := make([]sdk.AuthConsumerScopeDetail, len(sdk.AuthConsumerScopes))
	for i, scope := range sdk.AuthConsumerScopes {
		endpoints := make([]sdk.AuthConsumerScopeEndpoint, 0, len(m[scope]))
		for uri, mMethods := range m[scope] {
			methods := make([]string, 0, len(mMethods))
			for k := range mMethods {
				methods = append(methods, k)
			}
			endpoints = append(endpoints, sdk.AuthConsumerScopeEndpoint{
				Route:   uri,
				Methods: methods,
			})
		}

		details[i].Scope = scope
		details[i].Endpoints = endpoints
	}

	r.scopeDetails = details
}

// Handle adds all handler for their specific verb in gorilla router for given uri
func (r *Router) Handle(uri string, scope HandlerScope, handlers ...*service.HandlerConfig) {
	uri = r.Prefix + uri
	config, f := r.handle(uri, scope, handlers...)
	r.Mux.Handle(uri, r.pprofLabel(config, r.compress(r.setRequestID(r.recoverWrap(f)))))
}

func (r *Router) HandlePrefix(uri string, scope HandlerScope, handlers ...*service.HandlerConfig) {
	uri = r.Prefix + uri
	config, f := r.handle(uri, scope, handlers...)
	r.Mux.PathPrefix(uri).HandlerFunc(r.pprofLabel(config, r.compress(r.setRequestID(r.recoverWrap(f)))))
}

// Handle adds all handler for their specific verb in gorilla router for given uri
func (r *Router) handle(uri string, scope HandlerScope, handlers ...*service.HandlerConfig) (map[string]*service.HandlerConfig, http.HandlerFunc) {
	cfg := &service.RouterConfig{
		Config: map[string]*service.HandlerConfig{},
	}
	if r.mapRouterConfigs == nil {
		r.mapRouterConfigs = map[string]*service.RouterConfig{}
	}
	r.mapRouterConfigs[uri] = cfg

	cleanURL := doc.CleanURL(uri)
	for i := range handlers {
		handlers[i].CleanURL = cleanURL
		handlers[i].AllowedScopes = scope
		name := runtime.FuncForPC(reflect.ValueOf(handlers[i].Handler).Pointer()).Name()
		name = strings.Replace(name, ".func1", "", 1)
		name = strings.Replace(name, ".1", "", 1)
		name = strings.Replace(name, "github.com/ovh/cds/engine/", "", 1)
		handlers[i].Name = name
		cfg.Config[handlers[i].Method] = handlers[i]
	}

	f := func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()

		ctx = telemetry.ContextWithTelemetry(r.Background, ctx)

		var requestID string
		iRequestID := ctx.Value(log.ContextLoggingRequestIDKey)
		if iRequestID != nil {
			if id, ok := iRequestID.(string); ok {
				requestID = id
			}
		}

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
			telemetry.Record(ctx, Errors, 1)
			service.WriteError(ctx, w, req, sdk.ErrNotFound)
			return
		}

		// Make the request context inherit from the context of the router
		tags := telemetry.ContextGetTags(r.Background, telemetry.TagServiceType, telemetry.TagServiceName)
		ctx, err = tag.New(ctx, tags...)
		if err != nil {
			log.Error(ctx, "telemetry.ContextGetTags> %v", err)
		}
		ctx = telemetry.ContextWithTag(ctx,
			telemetry.RequestID, requestID,
			telemetry.Handler, rc.Name,
			telemetry.Host, req.Host,
			telemetry.Path, req.URL.Path,
			telemetry.Method, req.Method)

		// Prepare logging fields
		ctx = context.WithValue(ctx, log.ContextLoggingFuncKey, func(ctx context.Context) logrus.Fields {
			fields := make(logrus.Fields)

			// Add consumer info if exists
			iConsumer := ctx.Value(contextAPIConsumer)
			if iConsumer != nil {
				if consumer, ok := iConsumer.(*sdk.AuthConsumer); ok {
					fields["auth_user_id"] = consumer.AuthentifiedUserID
					fields["auth_consumer_id"] = consumer.ID
				}
			}

			// Add session info if exists
			iSession := ctx.Value(contextSession)
			if iSession != nil {
				if session, ok := iSession.(*sdk.AuthSession); ok {
					fields["auth_session_id"] = session.ID
				}
			}

			return fields
		})

		// Log request start
		start := time.Now()
		log.InfoWithFields(ctx, log.Fields{
			"method":      req.Method,
			"route":       cleanURL,
			"request_uri": req.RequestURI,
			"deprecated":  rc.IsDeprecated,
			"handler":     rc.Name,
		}, "%s | BEGIN | %s [%s]", req.Method, req.URL, rc.Name)

		// Defer log request end
		deferFunc := func(ctx context.Context) {
			if responseWriter.statusCode == 0 {
				responseWriter.statusCode = 200
			}
			ctx = telemetry.ContextWithTag(ctx, telemetry.StatusCode, responseWriter.statusCode)
			end := time.Now()
			latency := end.Sub(start)

			log.InfoWithFields(ctx, log.Fields{
				"method":      req.Method,
				"latency_num": latency.Nanoseconds(),
				"latency":     latency,
				"status_num":  responseWriter.statusCode,
				"status":      responseWriter.statusCode,
				"route":       cleanURL,
				"request_uri": req.RequestURI,
				"deprecated":  rc.IsDeprecated,
				"handler":     rc.Name,
			}, "%s | END   | %s [%s] | [%d]", req.Method, req.URL, rc.Name, responseWriter.statusCode)

			telemetry.RecordFloat64(ctx, ServerLatency, float64(latency)/float64(time.Millisecond))
			telemetry.Record(ctx, ServerRequestBytes, responseWriter.reqSize)
			telemetry.Record(ctx, ServerResponseBytes, responseWriter.respSize)
		}

		telemetry.Record(r.Background, Hits, 1)
		telemetry.Record(ctx, ServerRequestCount, 1)

		for _, m := range r.Middlewares {
			var err error
			ctx, err = m(ctx, responseWriter, req, rc)
			if err != nil {
				telemetry.Record(r.Background, Errors, 1)
				service.WriteError(ctx, responseWriter, req, err)
				deferFunc(ctx)
				return
			}
		}

		authMiddleware := r.DefaultAuthMiddleware
		if rc.OverrideAuthMiddleware != nil {
			authMiddleware = rc.OverrideAuthMiddleware
		}
		if authMiddleware != nil {
			var err error
			ctx, err = authMiddleware(ctx, responseWriter, req, rc)
			if err != nil {
				telemetry.Record(r.Background, Errors, 1)
				service.WriteError(ctx, responseWriter, req, err)
				deferFunc(ctx)
				return
			}
		}

		for _, m := range r.PostAuthMiddlewares {
			var err error
			ctx, err = m(ctx, responseWriter, req, rc)
			if err != nil {
				telemetry.Record(r.Background, Errors, 1)
				service.WriteError(ctx, responseWriter, req, err)
				deferFunc(ctx)
				return
			}
		}

		var end func()
		ctx, end = telemetry.SpanFromMain(ctx, "router.handle")

		if err := rc.Handler(ctx, responseWriter.wrappedResponseWriter(), req); err != nil {
			telemetry.Record(r.Background, Errors, 1)
			telemetry.End(ctx, responseWriter, req) // nolint
			service.WriteError(ctx, responseWriter, req, err)
			end()
			deferFunc(ctx)
			return
		}
		end()

		// writeNoContentPostMiddleware is compliant Middleware Interface
		// but no need to check ct, err in return
		writeNoContentPostMiddleware(ctx, responseWriter, req, rc) // nolint

		for _, m := range r.PostMiddlewares {
			var err error
			ctx, err = m(ctx, responseWriter, req, rc)
			if err != nil {
				log.Error(ctx, "PostMiddlewares > %s", err)
			}
		}

		deferFunc(ctx)
	}

	return cfg.Config, f
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
			if iRequestID, ok := req.contextValues[log.ContextLoggingRequestIDKey]; ok {
				if requestID, ok := iRequestID.(string); ok {
					ctx = context.WithValue(ctx, log.ContextLoggingRequestIDKey, requestID)
				}
			}
			if err := req.do(ctx, handler); err != nil {
				isErrWithStack := sdk.IsErrorWithStack(err)
				fields := log.Fields{}
				if isErrWithStack {
					fields["stack_trace"] = fmt.Sprintf("%+v", err)
				}
				myError, ok := err.(sdk.Error)
				if ok && myError.Status >= 500 {
					if req.nbErrors > retry {
						log.ErrorWithFields(ctx, fields, "Asynchronous Request on Error: %v with status: %d", err, myError.Status)
					} else {
						chanRequest <- req
					}
				} else {
					log.ErrorWithFields(ctx, fields, "Asynchronous Request on Error: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Asynchronous handles an AsynchronousHandlerFunc
func (r *Router) Asynchronous(handler service.AsynchronousHandlerFunc, retry int, goRoutines *sdk.GoRoutines) service.HandlerFunc {
	chanRequest := make(chan asynchronousRequest, 1000)
	goRoutines.Exec(r.Background, "", func(ctx context.Context) {
		processAsyncRequests(ctx, chanRequest, handler, retry)
	})
	return func() service.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			async := asynchronousRequest{
				contextValues: map[interface{}]interface{}{
					log.ContextLoggingRequestIDKey: ctx.Value(log.ContextLoggingRequestIDKey),
					contextSession:                 ctx.Value(contextSession),
					contextAPIConsumer:             ctx.Value(contextAPIConsumer),
					service.ContextJWT:             ctx.Value(service.ContextJWT),
					service.ContextJWTRaw:          ctx.Value(service.ContextJWTRaw),
					service.ContextJWTFromCookie:   ctx.Value(service.ContextJWTFromCookie),
				},
				request: *r,
				vars:    mux.Vars(r),
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
func (r *Router) GET(h service.HandlerFunc, cfg ...service.HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.Method = "GET"
	rc.PermissionLevel = sdk.PermissionRead
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// POST will set given handler only for POST request
func (r *Router) POST(h service.HandlerFunc, cfg ...service.HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.Method = "POST"
	rc.PermissionLevel = sdk.PermissionReadWriteExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// POSTEXECUTE will set given handler only for POST request and add a flag for execution permission
func (r *Router) POSTEXECUTE(h service.HandlerFunc, cfg ...service.HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.Method = "POST"
	rc.PermissionLevel = sdk.PermissionReadExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// PUT will set given handler only for PUT request
func (r *Router) PUT(h service.HandlerFunc, cfg ...service.HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.Method = "PUT"
	rc.PermissionLevel = sdk.PermissionReadWriteExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// DELETE will set given handler only for DELETE request
func (r *Router) DELETE(h service.HandlerFunc, cfg ...service.HandlerConfigParam) *service.HandlerConfig {
	var rc service.HandlerConfig
	rc.Handler = h()
	rc.Method = "DELETE"
	rc.PermissionLevel = sdk.PermissionReadWriteExecute
	for _, c := range cfg {
		c(&rc)
	}
	return &rc
}

// MaintenanceAware route need CDS maintenance off
func MaintenanceAware() service.HandlerConfigParam {
	f := func(rc *service.HandlerConfig) {
		rc.MaintenanceAware = true
	}
	return f
}

// NotFoundHandler is called by default by Mux is any matching handler has been found
func NotFoundHandler(w http.ResponseWriter, req *http.Request) {
	service.WriteError(context.Background(), w, req, sdk.NewError(sdk.ErrNotFound, fmt.Errorf("%s not found", req.URL.Path)))
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
