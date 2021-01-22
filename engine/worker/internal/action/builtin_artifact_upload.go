package action

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunArtifactUpload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	artifactPath := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
	if artifactPath == "" {
		artifactPath = "."
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return res, err
	}

	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
	} else {
		abs = workdir.Name()
	}

	if !sdk.PathIsAbs(artifactPath) {
		artifactPath = filepath.Join(abs, artifactPath)
	}

	tag := sdk.ParameterFind(a.Parameters, "tag")
	if tag == nil {
		return res, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("tag variable is empty. aborting"))
	}

	// Global all files matching filePath
	filesPath, err := afero.Glob(afero.NewOsFs(), artifactPath)
	if err != nil {
		return res, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("cannot perform globbing of pattern '%s': %s", artifactPath, err))
	}

	if len(filesPath) == 0 {
		return res, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("pattern '%s' matched no file", artifactPath))
	}

	var globalError = &sdk.MultiError{}
	var chanError = make(chan error)
	var wg = new(sync.WaitGroup)
	var wgErrors = new(sync.WaitGroup)

	go func() {
		for err := range chanError {
			wk.SendLog(ctx, workerruntime.LevelInfo, err.Error())
			globalError.Append(err)
			wgErrors.Done()
		}
	}()

	cdnArtifactEnabled := wk.FeatureEnabled(sdk.FeatureCDNArtifact)

	integrationName := sdk.DefaultIfEmptyStorage(strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination")))
	projectKey := sdk.ParameterValue(wk.Parameters(), "cds.project")

	wg.Add(len(filesPath))
	for _, p := range filesPath {
		go func(path string) {
			log.Debug(ctx, "uploading %s projectKey:%v integrationName:%v job:%d", path, projectKey, integrationName, jobID)
			defer wg.Done()

			if !cdnArtifactEnabled {
				throughTempURL, duration, err := wk.Client().QueueArtifactUpload(ctx, projectKey, integrationName, jobID, tag.Value, path)
				if err != nil {
					log.Warn(ctx, "queueArtifactUpload(%s, %s, %d, %s, %s) failed: %v", projectKey, integrationName, jobID, tag.Value, path, err)
					chanError <- sdk.WrapError(err, "Error while uploading artifact %s", path)
					wgErrors.Add(1)
					return
				}
				if throughTempURL {
					wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to object store", path, duration.Seconds()))
				} else {
					wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS API", path, duration.Seconds()))
				}
				return
			}

			_, name := filepath.Split(path)
			signature, err := wk.ArtifactSignature(name)
			if err != nil {
				log.Error(ctx, "unable to sign artifact: %v", err)
				return
			}

			duration, err := wk.Client().CDNArtifactUpdload(ctx, wk.CDNHttpURL(), signature, path)
			if err != nil {
				log.Error(ctx, "upable to upload artifact %q: %v", path, err)
				chanError <- sdk.WrapError(err, "Error while uploading artifact %s", path)
				wgErrors.Add(1)
			}
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS CDN", path, duration.Seconds()))

		}(p)
		if len(filesPath) > 1 {
			//Wait 3 second to get the object storage to set up all the things
			time.Sleep(3 * time.Second)
		}
	}
	wg.Wait()
	close(chanError)
	<-chanError
	wgErrors.Wait()

	if !globalError.IsEmpty() {
		return res, sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("error: %v", globalError.Error()))
	}

	return res, nil
}
