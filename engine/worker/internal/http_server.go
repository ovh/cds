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

func returnHTTPError(w http.ResponseWriter, code int, e error) {
	err := sdk.Error{
		Message: e.Error(),
		Status:  code,
	}
	log.Error("%v", err)
	writeJSON(w, err, err.Status)
}

func LogMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("[Worker HTTP Server] %s %s", r.Method, r.URL.String())
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

	log.Info("Export variable HTTP server: %s", listener.Addr().String())
	r := mux.NewRouter()

	r.HandleFunc("/artifacts", LogMiddleware(artifactsHandler(w)))
	r.HandleFunc("/cache/{ref}/pull", LogMiddleware(cachePullHandler(w)))
	r.HandleFunc("/cache/{ref}/push", LogMiddleware(cachePushHandler(w)))
	r.HandleFunc("/download", LogMiddleware(downloadHandler(w)))
	r.HandleFunc("/exit", LogMiddleware(exitHandler(w)))
	r.HandleFunc("/key/{key}/install", LogMiddleware(keyInstallHandler(w)))
	r.HandleFunc("/tag", LogMiddleware(tagHandler(w)))
	r.HandleFunc("/tmpl", LogMiddleware(tmplHandler(w)))
	r.HandleFunc("/upload", LogMiddleware(uploadHandler(w)))
	r.HandleFunc("/checksecret", LogMiddleware(checkSecretHandler(w)))
	r.HandleFunc("/var", LogMiddleware(addBuildVarHandler(w)))
	r.HandleFunc("/vulnerability", LogMiddleware(vulnerabilityHandler(w)))

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:0",
		WriteTimeout: 6 * time.Minute,
		ReadTimeout:  6 * time.Minute,
	}

	//Start the server
	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Error("%v", err)
		}
	}()

	//Handle shutdown
	go func() {
		<-c.Done()
		srv.Shutdown(c)
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
	w.Write(b)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	al := r.Header.Get("Accept-Language")
	sdkErr := sdk.ExtractHTTPError(err, al)
	writeJSON(w, sdkErr, sdkErr.Status)
}
