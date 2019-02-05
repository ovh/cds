package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func runArtifactUpload(wk *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, wJobID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		res := sdk.Result{Status: sdk.StatusSuccess.String()}

		path := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
		if path == "" {
			path = "."
		}

		tag := sdk.ParameterFind(&a.Parameters, "tag")
		if tag == nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("tag variable is empty. aborting")
			sendLog(res.Reason)
			return res
		}

		// Global all files matching filePath
		filesPath, err := filepath.Glob(path)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("cannot perform globbing of pattern '%s': %s", path, err)
			sendLog(res.Reason)
			return res
		}

		if len(filesPath) == 0 {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Pattern '%s' matched no file", path)
			sendLog(res.Reason)
			return res
		}

		var globalError = &sdk.MultiError{}
		var chanError = make(chan error)
		var wg = new(sync.WaitGroup)
		var wgErrors = new(sync.WaitGroup)

		go func() {
			for err := range chanError {
				sendLog(err.Error())
				globalError.Append(err)
				wgErrors.Done()
			}
		}()

		integrationName := sdk.DefaultIfEmptyStorage(strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination")))
		projectKey := sdk.ParameterValue(*params, "cds.project")

		wg.Add(len(filesPath))
		for _, p := range filesPath {
			filename := filepath.Base(p)
			go func(path string) {
				log.Debug("Uploading %s", path)
				defer wg.Done()
				throughTempURL, duration, err := wk.client.QueueArtifactUpload(ctx, projectKey, integrationName, wJobID, tag.Value, path)
				if err != nil {
					chanError <- sdk.WrapError(err, "Error while uploading artifact %s", path)
					wgErrors.Add(1)
					return
				}
				if throughTempURL {
					sendLog(fmt.Sprintf("File '%s' uploaded in %.2fs to object store", filename, duration.Seconds()))
				} else {
					sendLog(fmt.Sprintf("File '%s' uploaded in %.2fs to CDS API", filename, duration.Seconds()))
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
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Error: %v", globalError.Error())
			return res
		}

		return res
	}
}
