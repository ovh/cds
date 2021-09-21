package service

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"
)

var headers = []string{
	http.CanonicalHeaderKey(telemetry.TraceIDHeader),
	http.CanonicalHeaderKey(telemetry.SpanIDHeader),
	http.CanonicalHeaderKey(telemetry.SampledHeader),
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

// HandlerConfigParam is a type used in handler configuration, to set specific config on a route given a method
type HandlerConfigParam func(*HandlerConfig)

func TracingMiddlewareFunc(s Service) Middleware {
	return func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
		name := runtime.FuncForPC(reflect.ValueOf(rc.Handler).Pointer()).Name()
		name = strings.Replace(name, ".func1", "", 1)

		splittedName := strings.Split(name, ".")
		name = splittedName[len(splittedName)-1]

		opts := telemetry.Options{
			Name: name,
		}

		ctx, err := telemetry.Start(ctx, s, w, req, opts)
		newReq := req.WithContext(ctx)
		*req = *newReq

		return ctx, err
	}
}
