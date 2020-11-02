package internal

import (
	"context"
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

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func downloadHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
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
				runs, err := wk.client.WorkflowRunList(currentProject, reqArgs.Workflow, 0, 0)
				if err != nil {
					writeError(w, r, sdk.WrapError(err, "cannot search run for project %s and workflow: %s", currentProject, reqArgs.Workflow))
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

		wk.SendLog(ctx, workerruntime.LevelInfo, "Downloading artifacts from into current directory")

		var isInError bool
		for i := range artifacts {
			a := &artifacts[i]

			if reqArgs.Pattern != "" && !regexp.MatchString(a.Name) {
				wg.Done()
				continue
			}

			if reqArgs.Tag != "" && a.Tag != reqArgs.Tag {
				wg.Done()
				continue
			}

			go func(a *sdk.WorkflowNodeRunArtifact) {
				defer wg.Done()

				path := path.Join(reqArgs.Destination, a.Name)
				f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
				if err != nil {
					wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Cannot download artifact (OpenFile) %s: %s", a.Name, err))
					isInError = true
					return
				}
				wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("downloading artifact %s with tag %s from workflow %s/%s on run %d (%s)...", a.Name, a.Tag, projectKey, reqArgs.Workflow, reqArgs.Number, path))
				if err := wk.client.WorkflowNodeRunArtifactDownload(projectKey, reqArgs.Workflow, *a, f); err != nil {
					wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Cannot download artifact %s: %s", a.Name, err))
					isInError = true
					return
				}
				if err := f.Close(); err != nil {
					wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Cannot download artifact %s: %s", a.Name, err))
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
			newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("Error while downloading artifacts - see previous logs"))
			writeError(w, r, newError)
		}
	}
}
