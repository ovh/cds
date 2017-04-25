package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var cmdExport = &cobra.Command{
	Use:   "export",
	Short: "worker export <varname> <value>",
	Run:   exportCmd,
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
		w.WriteHeader(http.StatusBadRequest)
	}
	// Retrieve build info
	var proj, app, pip, bnS, env string
	for _, p := range pbJob.Parameters {
		switch p.Name {
		case "cds.pipeline":
			pip = p.Value
		case "cds.project":
			proj = p.Value
		case "cds.application":
			app = p.Value
		case "cds.buildNumber":
			bnS = p.Value
		case "cds.environment":
			env = p.Value
		}
	}

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/variable?envName=%s", proj, app, pip, bnS, env)
	_, code, err := sdk.Request("POST", uri, data)
	if err == nil && code > 300 {
		err = fmt.Errorf("HTTP %d", code)
	}
	if err != nil {
		log.Error("addBuildVarHandler> Cannot export variable: %s", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
}
