package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func downloadHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}
		defer r.Body.Close() // nolint

		var reqArgs workerruntime.DownloadArtifact
		if err := sdk.JSONUnmarshal(data, &reqArgs); err != nil {
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
					newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot parse '%s' as run number: %s", buildNumberString, errN))
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

		// GET Artifact from CDS API
		if !wk.FeatureEnabled(sdk.FeatureCDNArtifact) {
			if err := GetArtifactFromAPI(ctx, wk, projectKey, reqArgs); err != nil {
				writeError(w, r, err)
				return
			}
			return
		}

		// GET Artifact from CDS CDN
		reg, err := regexp.Compile(reqArgs.Pattern)
		if err != nil {
			newError := sdk.NewError(sdk.ErrInvalidData, fmt.Errorf("unable to compile pattern: %v", err))
			writeError(w, r, newError)
			return
		}
		if reqArgs.Destination == "" {
			reqArgs.Destination = "."
		}

		workdir, err := workerruntime.WorkingDirectory(wk.currentJob.context)
		if err != nil {
			newError := sdk.NewError(sdk.ErrInvalidData, fmt.Errorf("unable to get working directory: %v", err))
			writeError(w, r, newError)
			return
		}
		ctx = workerruntime.SetWorkingDirectory(ctx, workdir)

		var abs string
		if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
			abs, _ = x.RealPath(workdir.Name())
		} else {
			abs = workdir.Name()
		}

		if !sdk.PathIsAbs(reqArgs.Destination) {
			reqArgs.Destination = filepath.Join(abs, reqArgs.Destination)
		}
		wkDirFS := afero.NewOsFs()
		if err := wkDirFS.MkdirAll(reqArgs.Destination, os.FileMode(0744)); err != nil {
			newError := sdk.NewError(sdk.ErrInvalidData, fmt.Errorf("unable to create destination directory: %v", err))
			writeError(w, r, newError)
			return
		}

		cdnItems, err := wk.Client().WorkflowRunArtifactsLinks(projectKey, reqArgs.Workflow, reqArgs.Number)
		if err != nil {
			newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("unable to list artifacts: %v", err))
			writeError(w, r, newError)
			return
		}

		wg := new(sync.WaitGroup)
		wg.Add(len(cdnItems.Items))
		for i := range cdnItems.Items {
			item := cdnItems.Items[i]
			apiRef, is := item.GetCDNRunResultApiRef()
			if !is {
				newError := sdk.NewError(sdk.ErrInvalidData, fmt.Errorf("item is not an artifact: %v", err))
				writeError(w, r, newError)
				return
			}
			if reqArgs.Pattern != "" && !reg.MatchString(apiRef.ToFilename()) {
				wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%s does not match pattern %s - skipped", apiRef.ArtifactName, reqArgs.Pattern))
				wg.Done()
				continue
			}

			go func(a sdk.CDNItem) {
				defer wg.Done()
				destFile := path.Join(reqArgs.Destination, a.APIRef.ToFilename())
				wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifact %s from workflow %s/%s on run %d...", destFile, projectKey, reqArgs.Workflow, reqArgs.Number))

				f, err := wkDirFS.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(apiRef.Perm))
				if err != nil {
					newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("cannot create file (OpenFile) %s: %s", destFile, err))
					writeError(w, r, newError)
					return
				}
				defer f.Close() //nolint
				if err := wk.Client().CDNItemDownload(ctx, wk.CDNHttpURL(), item.APIRefHash, sdk.CDNTypeItemRunResult, a.MD5, f); err != nil {
					newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("cannot download artifact %s: %s", destFile, err))
					writeError(w, r, newError)
					return
				}
			}(item)
		}
		wg.Wait()
		return

	}
}

func GetArtifactFromAPI(ctx context.Context, wk *CurrentWorker, projectKey string, reqArgs workerruntime.DownloadArtifact) error {
	artifacts, err := wk.client.WorkflowRunArtifacts(projectKey, reqArgs.Workflow, reqArgs.Number)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot download artifacts with worker download: %s", err))
		return newError
	}

	reg, err := regexp.Compile(reqArgs.Pattern)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid pattern %s : %v", reqArgs.Pattern, err))
		return newError
	}
	wg := new(sync.WaitGroup)
	wg.Add(len(artifacts))

	wk.SendLog(ctx, workerruntime.LevelInfo, "Downloading artifacts into current directory")

	var isInError bool
	for i := range artifacts {
		a := &artifacts[i]

		if reqArgs.Pattern != "" && !reg.MatchString(a.Name) {
			wg.Done()
			continue
		}

		if reqArgs.Tag != "" && a.Tag != reqArgs.Tag {
			wg.Done()
			continue
		}

		go func(a *sdk.WorkflowNodeRunArtifact) {
			defer wg.Done()

			filePath := path.Join(reqArgs.Destination, a.Name)
			f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
			if err != nil {
				wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Cannot download artifact (OpenFile) %s: %s", a.Name, err))
				isInError = true
				return
			}
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("downloading artifact %s with tag %s from workflow %s/%s on run %d (%s)...", a.Name, a.Tag, projectKey, reqArgs.Workflow, reqArgs.Number, filePath))
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
		newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("error while downloading artifacts - see previous logs"))
		return newError
	}
	return nil
}
