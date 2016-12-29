package main

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/log"
)

// WorkerServerPort is name of environment variable set to local worker HTTP server port
// Used only to export build variables for now
const WorkerServerPort = "CDS_EXPORT_PORT"

// This handler is started by the worker instance waiting for action
func server() (int, error) {

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	t := strings.Split(listener.Addr().String(), ":")
	port, err := strconv.ParseInt(t[1], 10, 64)
	if err != nil {
		return 0, err
	}

	log.Notice("Export variable HTTP server: %s\n", listener.Addr().String())
	r := mux.NewRouter()
	r.HandleFunc("/var", addBuildVarHandler)
	r.HandleFunc("/upload", uploadHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:0",
		WriteTimeout: 6 * time.Minute,
		ReadTimeout:  6 * time.Minute,
	}

	go func() {
		log.Fatalf("Cannot start local http server: %s\n", srv.Serve(listener))
	}()

	return int(port), nil
}
