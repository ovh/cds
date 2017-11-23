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
	"time"

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
	var currentParam []sdk.Parameter
	if wk.currentJob.wJob != nil {
		currentParam = wk.currentJob.wJob.Parameters
	} else {
		currentParam = wk.currentJob.pbJob.Parameters
	}
	var proj, app, pip, bnS, env string
	for _, p := range currentParam {
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

	var uri string
	if wk.currentJob.wJob != nil {
		uri = fmt.Sprintf("/queue/workflows/%d/variable", wk.currentJob.wJob.ID)
	} else {
		uri = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/variable?envName=%s", proj, app, pip, bnS, url.QueryEscape(env))
	}

	var lasterr error
	var code int
	for try := 1; try <= 10; try++ {
		log.Info("addBuildVarHandler> Sending export variable...")
		_, code, lasterr = sdk.Request("POST", uri, data)
		if lasterr == nil && code < 300 {
			log.Info("addBuildVarHandler> Send step export variable OK")
			return http.StatusOK, nil
		}
		log.Warning("addBuildVarHandler> Cannot send export variable result: HTTP %d err: %s - try: %d - new try in 5s", code, lasterr, try)
		time.Sleep(5 * time.Second)
	}
	return http.StatusServiceUnavailable, fmt.Errorf("addBuildVarHandler> Cannot export variable: %s code: %d", lasterr, code)
}
