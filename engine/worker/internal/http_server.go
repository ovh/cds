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

// This handler is started by the worker instance waiting for action
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

	r.HandleFunc("/artifacts", artifactsHandler(w))
	r.HandleFunc("/cache/{ref}/pull", cachePullHandler(w))
	r.HandleFunc("/cache/{ref}/push", cachePushHandler(w))
	r.HandleFunc("/download", downloadHandler(w))
	r.HandleFunc("/exit", exitHandler(w))
	r.HandleFunc("/key/{key}/install", keyInstallHandler(w))
	r.HandleFunc("/tag", tagHandler(w))
	r.HandleFunc("/tmpl", tmplHandler(w))
	r.HandleFunc("/upload", uploadHandler(w))
	r.HandleFunc("/checksecret", checkSecretHandler(w))
	r.HandleFunc("/var", addBuildVarHandler(w))
	r.HandleFunc("/vulnerability", vulnerabilityHandler(w))

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
