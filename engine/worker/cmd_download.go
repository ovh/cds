package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	cmdDownloadWorkflowName string
	cmdDownloadNumber       string
	cmdDownloadArtefactName string
	cmdDownloadTag          string
)

func cmdDownload(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "download",
		Short: "worker download [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]",
		Long: `
Inside a job, there are two ways to download an artifact:

* with a step using action Download Artifacts
* with a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), using the worker command.

Worker Command:

	worker download --tag=<tag> <path>

Example:

	worker download --pattern="files.*.yml"

Theses two commands have the same result:

	worker download
	worker download --workflow={{.cds.workflow}} --number={{.cds.run.number}}

		`,
		Run: downloadCmd(w),
	}
	c.Flags().StringVar(&cmdDownloadWorkflowName, "workflow", "", "Workflow name to download from. Optional, default: current workflow")
	c.Flags().StringVar(&cmdDownloadNumber, "number", "", "Workflow Number to download from. Optional, default: current workflow run")
	c.Flags().StringVar(&cmdDownloadArtefactName, "pattern", "", "Pattern matching files to download. Optional, default: *")
	c.Flags().StringVar(&cmdDownloadTag, "tag", "", "Tag matching files to download. Optional")
	return c
}

type workerDownloadArtifact struct {
	Workflow    string `json:"workflow"`
	Number      int64  `json:"number"`
	Pattern     string `json:"pattern" cli:"pattern"`
	Tag         string `json:"tag" cli:"tag"`
	Destination string `json:"destination"`
}

func downloadCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
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

		wd, _ := os.Getwd()
		a := workerDownloadArtifact{
			Workflow:    cmdDownloadWorkflowName,
			Number:      number,
			Pattern:     cmdDownloadArtefactName,
			Tag:         cmdDownloadTag,
			Destination: wd,
		}

		data, errMarshal := json.Marshal(a)
		if errMarshal != nil {
			sdk.Exit("internal error (%s)\n", errMarshal)
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/download", port), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("cannot post worker download (Request): %s\n", errRequest)
		}

		client := http.DefaultClient

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("cannot post worker download (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cannot artefact download HTTP %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("download failed: %v\n", cdsError)
		}
	}
}

func (wk *currentWorker) downloadHandler(w http.ResponseWriter, r *http.Request) {
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

	sendLog := getLogger(wk, wk.currentJob.wJob.ID, wk.currentJob.currentStep)

	currentProject := sdk.ParameterValue(wk.currentJob.params, "cds.project")
	currentWorkflow := sdk.ParameterValue(wk.currentJob.params, "cds.workflow")
	if reqArgs.Workflow == "" {
		reqArgs.Workflow = currentWorkflow
	}

	// If the reqArgs.Number is empty and if the reqArgs.Workflow is the current workflow, take the current build number
	if reqArgs.Number == 0 {
		if reqArgs.Workflow == currentWorkflow {
			var errN error
			buildNumberString := sdk.ParameterValue(wk.currentJob.params, "cds.run.number")
			reqArgs.Number, errN = strconv.ParseInt(buildNumberString, 10, 64)
			if errN != nil {
				newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Cannot parse '%s' as run number: %s", buildNumberString, errN))
				writeError(w, r, newError)
				return
			}
		} else { // If this is another workflow, check the latest run
			filters := []cdsclient.Filter{
				{
					Name:  "workflow",
					Value: reqArgs.Workflow,
				},
			}
			runs, err := wk.client.WorkflowRunSearch(currentProject, 0, 0, filters...)
			if err != nil {
				writeError(w, r, err)
				return
			}
			if len(runs) < 1 {
				writeError(w, r, fmt.Errorf("workflow run not found"))
				return
			}
			reqArgs.Number = runs[0].Number
		}
	}

	projectKey := sdk.ParameterValue(wk.currentJob.params, "cds.project")
	artifacts, err := wk.client.WorkflowRunArtifacts(projectKey, reqArgs.Workflow, reqArgs.Number)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Cannot download artifacts with worker download: %s", err))
		writeError(w, r, newError)
		return
	}

	regexp, errp := regexp.Compile(reqArgs.Pattern)
	if errp != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid pattern %s : %s", reqArgs.Pattern, errp))
		writeError(w, r, newError)
		return
	}
	wg := new(sync.WaitGroup)
	wg.Add(len(artifacts))

	sendLog("Downloading artifacts from into current directory")

	var isInError bool
	for i := range artifacts {
		a := &artifacts[i]

		if reqArgs.Pattern != "" && !regexp.MatchString(a.Name) {
			sendLog(fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, reqArgs.Pattern))
			wg.Done()
			continue
		}

		if reqArgs.Tag != "" && a.Tag != reqArgs.Tag {
			sendLog(fmt.Sprintf("%s does not match tag %s - skipped", a.Name, reqArgs.Tag))
			wg.Done()
			continue
		}

		go func(a *sdk.WorkflowNodeRunArtifact) {
			defer wg.Done()

			path := path.Join(reqArgs.Destination, a.Name)
			f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.FileMode(a.Perm))
			if err != nil {
				sendLog(fmt.Sprintf("Cannot download artifact (OpenFile) %s: %s", a.Name, err))
				isInError = true
				return
			}
			sendLog(fmt.Sprintf("downloading artifact %s with tag %s from workflow %s/%s on run %d (%s)...", a.Name, a.Tag, projectKey, reqArgs.Workflow, reqArgs.Number, path))
			if err := wk.client.WorkflowNodeRunArtifactDownload(projectKey, reqArgs.Workflow, *a, f); err != nil {
				sendLog(fmt.Sprintf("Cannot download artifact %s: %s", a.Name, err))
				isInError = true
				return
			}
			if err := f.Close(); err != nil {
				sendLog(fmt.Sprintf("Cannot download artifact %s: %s", a.Name, err))
				isInError = true
				return
			}
		}(a)

		// there is one error, do not try to load all artifacts
		if isInError {
			break
		}
		if len(artifacts) > 1 {
			time.Sleep(3 * time.Second)
		}
	}

	wg.Wait()
	if isInError {
		newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("Error while downloading artefacts - see previous logs"))
		writeError(w, r, newError)
	}

}
