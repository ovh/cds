package service

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// Handler defines the HTTP handler used in CDS engine
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// AsynchronousHandler defines the HTTP asynchronous handler used in CDS engine
type AsynchronousHandler func(ctx context.Context, r *http.Request) error

// Middleware defines the HTTP Middleware used in CDS engine
type Middleware func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error)

// HandlerFunc defines the way to instantiate a handler
type HandlerFunc func() Handler

// AsynchronousHandlerFunc defines the way to instantiate a handler
type AsynchronousHandlerFunc func() AsynchronousHandler

// RouterConfigParam is the type of anonymous function returned by POST, GET and PUT functions
type RouterConfigParam func(rc *RouterConfig)

// RouterConfig contains a map of handler configuration. Key is the method of the http route
type RouterConfig struct {
	Config map[string]*HandlerConfig
}

// HandlerConfig is the configuration for one handler
type HandlerConfig struct {
	Name                   string
	Method                 string
	Handler                Handler
	IsDeprecated           bool
	OverrideAuthMiddleware Middleware
	MaintenanceAware       bool
	AllowedScopes          []sdk.AuthConsumerScope
	PermissionLevel        int
	CleanURL               string
}

// Accepted is a helper function used by asynchronous handlers
func Accepted(w http.ResponseWriter) error {
	const msg = "request accepted"
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(msg)))
	w.WriteHeader(http.StatusAccepted)
	_, err := w.Write([]byte(msg))
	return err
}

// Write is a helper function
func Write(w http.ResponseWriter, r io.Reader, status int, contentType string) error {
	w.Header().Add("Content-Type", contentType)

	WriteProcessTime(context.TODO(), w)
	w.WriteHeader(status)

	n, err := io.Copy(w, r)
	if err != nil {
		return sdk.WithStack(err)
	}

	w.Header().Add("Content-Length", fmt.Sprintf("%d", n))

	return nil
}

// WriteJSON is a helper function to marshal json, handle errors and set Content-Type for the best
func WriteJSON(w http.ResponseWriter, data interface{}, status int) error {
	b, err := json.Marshal(data)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal json data")
	}
	return sdk.WithStack(Write(w, bytes.NewReader(b), status, "application/json"))
}

// WriteProcessTime writes the duration of the call in the responsewriter
func WriteProcessTime(ctx context.Context, w http.ResponseWriter) {
	if h := w.Header().Get(cdsclient.ResponseAPINanosecondsTimeHeader); h != "" {
		start, err := strconv.ParseInt(h, 10, 64)
		if err != nil {
			log.Error(ctx, "WriteProcessTime> error on ParseInt header ResponseAPINanosecondsTimeHeader: %s", err)
		}
		w.Header().Add(cdsclient.ResponseProcessTimeHeader, fmt.Sprintf("%d", time.Now().UnixNano()-start))
	}
}

type ErrorResponse struct {
	sdk.Error
	RequestID string `json:"request_id"`
}

// WriteError is a helper function to return error in a language the called understand
func WriteError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	httpErr := sdk.ExtractHTTPError(err, al)
	isErrWithStack := sdk.IsErrorWithStack(err)

	fields := log.Fields{}
	if isErrWithStack {
		fields["stack_trace"] = fmt.Sprintf("%+v", err)
	}

	if httpErr.Status < 500 {
		log.InfoWithFields(ctx, fields, "%s", err)
	} else {
		log.ErrorWithFields(ctx, fields, "%s", err)
	}

	// Add request info if exists
	iRequestID := ctx.Value(log.ContextLoggingRequestIDKey)
	if iRequestID != nil {
		if requestID, ok := iRequestID.(string); ok {
			httpErr.RequestID = requestID
		}
	}

	// safely ignore error returned by WriteJSON
	_ = WriteJSON(w, httpErr, httpErr.Status)
}

// UnmarshalBody read the request body and tries to json.unmarshal it. It returns sdk.ErrWrongRequest in case of error.
func UnmarshalBody(r *http.Request, i interface{}) error {
	if r == nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "request is null")
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.NewError(sdk.ErrWrongRequest, err)
	}
	defer r.Body.Close()
	if err := json.Unmarshal(data, i); err != nil {
		return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "unable to unmarshal %s", string(data)))
	}
	return nil
}

type httpVerifier struct {
	sync.Mutex
	pubKey *rsa.PublicKey
}

func (v *httpVerifier) SetKey(pubKey *rsa.PublicKey) {
	v.Lock()
	defer v.Unlock()
	v.pubKey = pubKey
}

func (v *httpVerifier) GetKey(id string) interface{} {
	v.Lock()
	defer v.Unlock()
	return v.pubKey
}

var (
	_                  httpsig.KeyGetter = new(httpVerifier)
	globalHTTPVerifier *httpVerifier
)

func CheckRequestSignatureMiddleware(pubKey *rsa.PublicKey) Middleware {
	globalHTTPVerifier = new(httpVerifier)
	globalHTTPVerifier.SetKey(pubKey)

	verifier := httpsig.NewVerifier(globalHTTPVerifier)
	verifier.SetRequiredHeaders([]string{"(request-target)", "host", "date"})

	return func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
		if err := verifier.Verify(req); err != nil {
			return ctx, sdk.NewError(sdk.ErrUnauthorized, err)
		}

		log.Debug("Request has been successfully verified")
		return ctx, nil
	}
}

// FormInt64 return a int64.
func FormInt64(r *http.Request, s string) int64 {
	i, _ := strconv.ParseInt(r.FormValue(s), 10, 64)
	return i
}

// FormInt return a int.
func FormInt(r *http.Request, s string) int {
	i, _ := strconv.Atoi(r.FormValue(s))
	return i
}

// FormUInt return a uint.
func FormUInt(r *http.Request, s string) uint {
	i := FormInt(r, s)
	if i < 0 {
		return 0
	}
	return uint(i)
}

// FormBool return true if the form value is set to true|TRUE|yes|YES|1
func FormBool(r *http.Request, s string) bool {
	v := r.FormValue(s)
	switch v {
	case "true", "TRUE", "yes", "YES", "1":
		return true
	default:
		return false
	}
}
