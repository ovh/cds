package internal

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func returnHTTPError(ctx context.Context, w http.ResponseWriter, code int, e error) {
	err := sdk.Error{
		Message: e.Error(),
		Status:  code,
	}
	log.Error(ctx, "%v", err)
	writeJSON(w, err, err.Status)
}

func LogMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("[Worker HTTP Server] " + r.Method + " " + r.URL.String())
		h(w, r)
	}
}

func (w *CurrentWorker) Serve(c context.Context) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	t := strings.Split(listener.Addr().String(), ":")
	port, err := strconv.ParseInt(t[1], 10, 64)
	if err != nil {
		return err
	}

	log.Info(c, "Export variable HTTP server: %s", listener.Addr().String())
	r := mux.NewRouter()

	r.HandleFunc("/artifacts", LogMiddleware(artifactsHandler(c, w)))
	r.HandleFunc("/cache/{ref}/pull", LogMiddleware(cachePullHandler(c, w)))
	r.HandleFunc("/cache/push", LogMiddleware(cachePushHandler(c, w)))
	r.HandleFunc("/download", LogMiddleware(downloadHandler(c, w)))
	r.HandleFunc("/exit", LogMiddleware(exitHandler(c, w)))
	r.HandleFunc("/key/{key}/install", LogMiddleware(keyInstallHandler(c, w)))
	r.HandleFunc("/services/{type}", LogMiddleware(serviceHandler(c, w)))
	r.HandleFunc("/tag", LogMiddleware(tagHandler(c, w)))
	r.HandleFunc("/tmpl", LogMiddleware(tmplHandler(c, w)))
	r.HandleFunc("/upload", LogMiddleware(uploadHandler(c, w)))
	r.HandleFunc("/checksecret", LogMiddleware(checkSecretHandler(c, w)))
	r.HandleFunc("/var", LogMiddleware(addBuildVarHandler(c, w)))
	r.HandleFunc("/vulnerability", LogMiddleware(vulnerabilityHandler(c, w)))
	r.HandleFunc("/version", LogMiddleware(setVersionHandler(c, w)))

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:0",
		WriteTimeout: 6 * time.Minute,
		ReadTimeout:  6 * time.Minute,
	}

	//Start the server
	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Error(c, "%v", err)
		}
	}()

	//Handle shutdown
	go func() {
		<-c.Done()
		_ = srv.Shutdown(c)
	}()

	w.httpPort = int32(port)
	return nil
}

func writeByteArray(w http.ResponseWriter, data []byte, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	b, _ := json.Marshal(data)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	sdkErr := sdk.ExtractHTTPError(err, al)
	writeJSON(w, sdkErr, sdkErr.Status)
}
