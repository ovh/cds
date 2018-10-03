package main

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

// WorkerServerPort is name of environment variable set to local worker HTTP server port
const WorkerServerPort = "CDS_EXPORT_PORT"

// This handler is started by the worker instance waiting for action
func (w *currentWorker) serve(c context.Context) (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	t := strings.Split(listener.Addr().String(), ":")
	port, err := strconv.ParseInt(t[1], 10, 64)
	if err != nil {
		return 0, err
	}

	log.Info("Export variable HTTP server: %s", listener.Addr().String())
	r := mux.NewRouter()

	r.HandleFunc("/artifacts", w.artifactsHandler)
	r.HandleFunc("/cache/{ref}/pull", w.cachePullHandler)
	r.HandleFunc("/cache/{ref}/push", w.cachePushHandler)
	r.HandleFunc("/download", w.downloadHandler)
	r.HandleFunc("/exit", w.exitHandler)
	r.HandleFunc("/key/{key}/install", w.keyInstallHandler)
	r.HandleFunc("/services/{type}", w.serviceHandler)
	r.HandleFunc("/tag", w.tagHandler)
	r.HandleFunc("/tmpl", w.tmplHandler)
	r.HandleFunc("/upload", w.uploadHandler)
	r.HandleFunc("/var", w.addBuildVarHandler)
	r.HandleFunc("/vulnerability", w.vulnerabilityHandler)

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

	return int(port), nil
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
	msg, sdkError := sdk.ProcessError(err, al)
	sdkErr := sdk.Error{Message: msg}
	writeJSON(w, sdkErr, sdkError.Status)
}
