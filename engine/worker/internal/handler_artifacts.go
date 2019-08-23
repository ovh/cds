package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/ovh/cds/sdk"
)

func artifactsHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}
		defer r.Body.Close()

		var reqArgs workerruntime.DownloadArtifact
		if err := json.Unmarshal(data, &reqArgs); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
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
}
