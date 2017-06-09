package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk/log"
)

// WorkerServerPort is name of environment variable set to local worker HTTP server port
// Used only to export build variables for now
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
	r.HandleFunc("/var", w.addBuildVarHandler)
	r.HandleFunc("/upload", w.uploadHandler)

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
