package bootstrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
)

// Bootstrap bootstraps tonic with gin engine.
// This must be called after ALL tonic-enabled handlers have been
// added to the gin engine.
//
// This retrieves the routes from gin engine and populates tonic handler
// with route information (method, path).
func Bootstrap(e *gin.Engine) {
	defer tonic.SetExecHook(tonic.GetExecHook())

	// Define an exec hook that populates our tonic handler
	// with route information (method, path).
	tonic.SetExecHook(func(c *gin.Context, _ gin.HandlerFunc, fname string) {
		if r, ok := tonic.GetRoutes()[fname]; ok {
			r.Path = c.Request.URL.Path
			r.Method = c.Request.Method
		}
	})

	// Call each route defined in gin
	for _, r := range e.Routes() {
		req, err := http.NewRequest(r.Method, r.Path, nil)
		if err != nil {
			panic(err)
		}
		e.ServeHTTP(newDummyResponseWriter(), req)
	}
}

// DummyResponseWriter is a dummy http.ResponseWriter implementation.
type DummyResponseWriter struct{}

// Header makes DummyResponseWriter implement http.ResponseWriter.
func (d *DummyResponseWriter) Header() http.Header {
	h := make(map[string][]string)
	return h
}

// Write makes DummyResponseWriter implement http.ResponseWriter.
func (d *DummyResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

// WriteHeader makes DummyResponseWriter implement http.ResponseWriter.
func (d *DummyResponseWriter) WriteHeader(int) {}

func newDummyResponseWriter() *DummyResponseWriter {
	return &DummyResponseWriter{}
}
