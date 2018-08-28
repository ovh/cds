package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func runArtifactUpload(w *currentWorker) BuiltInAction {
	if w.currentJob.wJob == nil {
		return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
			res := sdk.Result{Status: sdk.StatusSuccess.String()}

			pipeline := sdk.ParameterValue(*params, "cds.pipeline")
			project := sdk.ParameterValue(*params, "cds.project")
			application := sdk.ParameterValue(*params, "cds.application")
			environment := sdk.ParameterValue(*params, "cds.environment")
			buildNumberString := sdk.ParameterValue(*params, "cds.buildNumber")

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
			if strings.Contains(tag.Value, "{") || strings.Contains(tag.Value, "}") || strings.Contains(tag.Value, " ") {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("tag variable invalid: %s", tag.Value)
				sendLog(res.Reason)
				return res
			}
			tag.Value = strings.Replace(tag.Value, "/", "-", -1)
			tag.Value = url.QueryEscape(tag.Value)

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

			buildNumber, errBN := strconv.Atoi(buildNumberString)
			if errBN != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("BuilNumber is not an integer %s", errBN)
				sendLog(res.Reason)
				return res
			}

			for _, filePath := range filesPath {
				filename := filepath.Base(filePath)
				throughTempURL, duration, err := sdk.UploadArtifact(project, pipeline, application, tag.Value, filePath, buildNumber, environment)
				if throughTempURL {
					sendLog(fmt.Sprintf("File '%s' uploaded in %.2fs to object store", filename, duration.Seconds()))
				} else {
					sendLog(fmt.Sprintf("File '%s' uploaded in %.2fs to CDS API", filename, duration.Seconds()))
				}
				if err != nil {
					res.Status = sdk.StatusFail.String()
					if throughTempURL {
						res.Reason = fmt.Sprintf("Error while uploading artifact '%s' to object store: %v", filename, err)
					} else {
						res.Reason = fmt.Sprintf("Error while uploading artifact '%s' to CDS API: %v", filename, err)
					}
					sendLog(res.Reason)
					return res
				}
			}

			return res
		}
	}

	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
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

		wg.Add(len(filesPath))
		for _, p := range filesPath {
			filename := filepath.Base(p)
			go func(path string) {
				log.Debug("Uploading %s", path)
				defer wg.Done()
				throughTempURL, duration, err := w.client.QueueArtifactUpload(buildID, tag.Value, path)
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
