package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

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

func (wk *currentWorker) addBuildVarHandler(w http.ResponseWriter, r *http.Request) {
	// Get body
	data, errra := ioutil.ReadAll(r.Body)
	if errra != nil {
		log.Error("addBuildVarHandler> Cannot ReadAll err: %s", errra)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var v sdk.Variable
	if err := json.Unmarshal(data, &v); err != nil {
		log.Error("addBuildVarHandler> Cannot Unmarshal err: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	v.Name = "cds.build." + v.Name

	errstatus, err := wk.addVariableInPipelineBuild(v, nil)
	if err != nil {
		w.WriteHeader(errstatus)
	}
}

func (wk *currentWorker) addVariableInPipelineBuild(v sdk.Variable, params *[]sdk.Parameter) (int, error) {
	// OK, so now we got our new variable. We need to:
	// - add it as a build var in API
	if strings.HasPrefix(v.Name, "cds.build") {
		wk.currentJob.buildVariables = append(wk.currentJob.buildVariables, v)
	} else if params != nil {
		*params = append(*params, sdk.Parameter{
			Name:  v.Name,
			Type:  v.Type,
			Value: v.Value,
		})
	}

	// - add it in current building Action
	data, errm := json.Marshal(v)
	if errm != nil {
		log.Error("addBuildVarHandler> Cannot Marshal err: %s", errm)
		return http.StatusBadRequest, fmt.Errorf("addBuildVarHandler> Cannot Marshal err: %s", errm)
	}
	// Retrieve build info
	var proj, app, pip, bnS, env string
	for _, p := range wk.currentJob.pbJob.Parameters {
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

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/variable?envName=%s", proj, app, pip, bnS, url.QueryEscape(env))
	_, code, err := sdk.Request("POST", uri, data)
	if err == nil && code > 300 {
		err = fmt.Errorf("HTTP %d", code)
	}
	if err != nil {
		log.Error("addBuildVarHandler> Cannot export variable: %s", err)
		return http.StatusServiceUnavailable, fmt.Errorf("addBuildVarHandler> Cannot export variable: %s", err)
	}
	return http.StatusOK, nil
}
