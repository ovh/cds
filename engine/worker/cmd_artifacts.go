package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdArtifacts(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "artifacts",
		Short: "worker artifacts [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]",
		Long: `
Inside a job, you can list artifacts of a workflow:

	worker artifacts --pattern="files.*.yml"

	#theses two commands have the same result:
	worker artifacts
	worker artifacts --workflow={{.cds.workflow}} --number={{.cds.run.number}}

		`,
		Run: artifactsCmd(w),
	}
	c.Flags().StringVar(&cmdDownloadWorkflowName, "workflow", "", "Workflow name. Optional, default: current workflow")
	c.Flags().StringVar(&cmdDownloadNumber, "number", "", "Workflow Number. Optional, default: current workflow run")
	c.Flags().StringVar(&cmdDownloadArtefactName, "pattern", "", "Pattern matching files to list. Optional, default: *")
	c.Flags().StringVar(&cmdDownloadTag, "tag", "", "Tag matching files to list. Optional")

	return c
}

func artifactsCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		var number int64
		if cmdDownloadNumber != "" {
			var errN error
			number, errN = strconv.ParseInt(cmdDownloadNumber, 10, 64)
			if errN != nil {
				sdk.Exit("number parameter have to be an integer")
			}
		}

		a := workerDownloadArtifact{
			Workflow: cmdDownloadWorkflowName,
			Number:   number,
			Pattern:  cmdDownloadArtefactName,
			Tag:      cmdDownloadTag,
		}

		data, errMarshal := json.Marshal(a)
		if errMarshal != nil {
			sdk.Exit("internal error (%s)\n", errMarshal)
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/artifacts", port), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("cannot post worker artifacts (Request): %s\n", errRequest)
		}

		client := http.DefaultClient

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("cannot post worker artifacts (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cannot list artifacts HTTP %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("artifacts failed: %v\n", cdsError)
		}

		// step: read the response body
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("artifacts failed ReadAll: %v\n", err)
		}
		fmt.Println(string(respBody))
	}
}

func (wk *currentWorker) artifactsHandler(w http.ResponseWriter, r *http.Request) {
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
		writeError(w, r, newError)
		return
	}
	defer r.Body.Close()

	var reqArgs workerDownloadArtifact
	if err := json.Unmarshal(data, &reqArgs); err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}

	if wk.currentJob.wJob == nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("command 'worker artifacts' is only available on CDS Workflows"))
		writeError(w, r, newError)
		return
	}

	if reqArgs.Workflow == "" {
		reqArgs.Workflow = sdk.ParameterValue(wk.currentJob.params, "cds.workflow")
	}

	if reqArgs.Number == 0 {
		var errN error
		buildNumberString := sdk.ParameterValue(wk.currentJob.params, "cds.run.number")
		reqArgs.Number, errN = strconv.ParseInt(buildNumberString, 10, 64)
		if errN != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Cannot parse '%s' as run number: %s", buildNumberString, errN))
			writeError(w, r, newError)
			return
		}
	}

	projectKey := sdk.ParameterValue(wk.currentJob.params, "cds.project")
	artifacts, err := wk.client.WorkflowRunArtifacts(projectKey, reqArgs.Workflow, reqArgs.Number)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Cannot list artifacts with worker artifacts: %s", err))
		writeError(w, r, newError)
		return
	}

	regexp, errp := regexp.Compile(reqArgs.Pattern)
	if errp != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid pattern %s : %s", reqArgs.Pattern, errp))
		writeError(w, r, newError)
		return
	}

	artifactsJSON := []sdk.WorkflowNodeRunArtifact{}
	for i := range artifacts {
		a := &artifacts[i]

		if reqArgs.Pattern != "" && !regexp.MatchString(a.Name) {
			continue
		}

		if reqArgs.Tag != "" && a.Tag != reqArgs.Tag {
			continue
		}
		artifactsJSON = append(artifactsJSON, *a)
	}

	writeJSON(w, artifactsJSON, http.StatusOK)
}
