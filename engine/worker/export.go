package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Name of environment variable set to local worker HTTP server port
// Used only to export build variables for now
const WorkerServerPort = "CDS_EXPORT_PORT"

// This handler is started by the worker instance waiting for action
func exportHandler() (int, error) {

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

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:0",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Fatalf("Cannot start local http server: %s\n", srv.Serve(listener))
	}()

	return int(port), nil
}

func addBuildVarHandler(w http.ResponseWriter, r *http.Request) {
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var v sdk.Variable
	err = json.Unmarshal(data, &v)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// OK, so now we got our new variable. We need to:
	// - add it as a build var in API
	buildVariables = append(buildVariables, v)
	// - add it in current building Action
	data, err = json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Retrieve build info
	var proj, app, pip, bnS string
	for _, p := range ab.Args {
		switch p.Name {
		case "cds.pipeline":
			pip = p.Value
			break
		case "cds.project":
			proj = p.Value
			break
		case "cds.application":
			app = p.Value
			break
		case "cds.buildNumber":
			bnS = p.Value
			break
		}
	}

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/variable", proj, app, pip, bnS)
	_, code, err := sdk.Request("POST", uri, data)
	if err == nil && code > 300 {
		err = fmt.Errorf("HTTP %d", code)
	}
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
}

func exportCmd(cmd *cobra.Command, args []string) {
	portS := os.Getenv(WorkerServerPort)
	if portS == "" {
		sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
	}

	port, err := strconv.Atoi(portS)
	if err != nil {
		sdk.Exit("cannot parse '%s' as a port number", portS)
	}

	if len(args) != 2 {
		sdk.Exit("Wrong usage: See '%s'\n", cmd.Short)
	}

	v := sdk.Variable{
		Name:  args[0],
		Type:  sdk.StringVariable,
		Value: args[1],
	}

	data, err := json.Marshal(v)
	if err != nil {
		sdk.Exit("internal error (%s)\n", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/var", port), bytes.NewReader(data))
	if err != nil {
		sdk.Exit("cannot add variable: %s\n", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		sdk.Exit("cannot add variable: %s\n", err)
	}

	if resp.StatusCode >= 300 {
		sdk.Exit("cannot add variable: HTTP %d\n", resp.StatusCode)
	}
}
