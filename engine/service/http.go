package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// HandlerFunc defines the way to instanciate a handler
type HandlerFunc func() Handler

// AsynchronousHandlerFunc defines the way to instanciate a handler
type AsynchronousHandlerFunc func() AsynchronousHandler

// RouterConfigParam is the type of anonymous function returned by POST, GET and PUT functions
type RouterConfigParam func(rc *RouterConfig)

// RouterConfig contains a map of handler configuration. Key is the method of the http route
type RouterConfig struct {
	Config map[string]*HandlerConfig
}

// HandlerConfig is the configuration for one handler
type HandlerConfig struct {
	Name         string
	Method       string
	Handler      Handler
	IsDeprecated bool
	Options      map[string]string
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
func Write(w http.ResponseWriter, btes []byte, status int, contentType string) error {
	w.Header().Add("Content-Type", contentType)
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(btes)))
	WriteProcessTime(w)
	w.WriteHeader(status)
	_, err := w.Write(btes)
	return err
}

// WriteJSON is a helper function to marshal json, handle errors and set Content-Type for the best
func WriteJSON(w http.ResponseWriter, data interface{}, status int) error {
	b, e := json.Marshal(data)
	if e != nil {
		return sdk.WrapError(e, "WriteJSON> unable to marshal : %s", e)
	}

	return Write(w, b, status, "application/json")
}

// WriteProcessTime writes the duration of the call in the responsewriter
func WriteProcessTime(w http.ResponseWriter) {
	if h := w.Header().Get(cdsclient.ResponseAPINanosecondsTimeHeader); h != "" {
		start, err := strconv.ParseInt(h, 10, 64)
		if err != nil {
			log.Error("WriteProcessTime> error on ParseInt header ResponseAPINanosecondsTimeHeader: %s", err)
		}
		w.Header().Add(cdsclient.ResponseProcessTimeHeader, fmt.Sprintf("%d", time.Now().UnixNano()-start))
	}
}

// WriteError is a helper function to return error in a language the called understand
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	msg, errProcessed := sdk.ProcessError(err, al)
	sdkErr := sdk.Error{Message: msg}

	// ErrAlreadyTaken and ErrWorkerModelAlreadyBooked are not useful to log in warning
	if sdk.ErrorIs(errProcessed, sdk.ErrAlreadyTaken) ||
		sdk.ErrorIs(errProcessed, sdk.ErrWorkerModelAlreadyBooked) ||
		sdk.ErrorIs(errProcessed, sdk.ErrJobAlreadyBooked) {
		log.Debug("%-7s | %-4d | %s \t %s", r.Method, errProcessed.Status, r.RequestURI, err)
	} else {
		log.Warning("%-7s | %-4d | %s \t %s", r.Method, errProcessed.Status, r.RequestURI, err)
	}

	_ = WriteJSON(w, sdkErr, errProcessed.Status)
}
