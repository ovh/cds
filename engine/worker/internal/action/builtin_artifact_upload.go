package action

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func RunArtifactUpload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	path := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
	if path == "" {
		path = "."
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
	} else {
		abs = workdir.Name()
	}

	wkDirFS := afero.NewBasePathFs(afero.NewOsFs(), abs)

	tag := sdk.ParameterFind(a.Parameters, "tag")
	if tag == nil {
		return res, errors.New("tag variable is empty. aborting")
	}

	path = strings.TrimPrefix(path, abs)
	// Global all files matching filePath
	filesPath, err := afero.Glob(wkDirFS, path)
	if err != nil {
		return res, fmt.Errorf("cannot perform globbing of pattern '%s': %s", path, err)
	}

	if len(filesPath) == 0 {
		return res, fmt.Errorf("pattern '%s' matched no file", path)
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

	integrationName := sdk.DefaultIfEmptyStorage(strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination")))
	projectKey := sdk.ParameterValue(wk.Parameters(), "cds.project")

	wg.Add(len(filesPath))
	for _, p := range filesPath {
		go func(path string) {
			absFile := abs + "/" + strings.TrimPrefix(path, abs)
			log.Debug("Uploading %s projectKey:%v integrationName:%v job:%d", absFile, projectKey, integrationName, jobID)
			defer wg.Done()
			throughTempURL, duration, err := wk.Client().QueueArtifactUpload(ctx, projectKey, integrationName, jobID, tag.Value, absFile)
			if err != nil {
				chanError <- sdk.WrapError(err, "Error while uploading artifact %s", absFile)
				wgErrors.Add(1)
				return
			}
			if throughTempURL {
				wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to object store", path, duration.Seconds()))
			} else {
				wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS API", path, duration.Seconds()))
			}
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
		log.Error(ctx, "Error while uploading artifact: %v", globalError.Error())
		return res, fmt.Errorf("error: %v", globalError.Error())
	}

	return res, nil
}
