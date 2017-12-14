package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
)

func runArtifactUpload(w *currentWorker) BuiltInAction {
	if w.currentJob.wJob == nil {
		return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
			res := sdk.Result{Status: sdk.StatusSuccess.String()}

			pipeline := sdk.ParameterValue(*params, "cds.pipeline")
			project := sdk.ParameterValue(*params, "cds.project")
			application := sdk.ParameterValue(*params, "cds.application")
			environment := sdk.ParameterValue(*params, "cds.environment")
			buildNumberString := sdk.ParameterValue(*params, "cds.buildNumber")

			path := sdk.ParameterValue(a.Parameters, "path")
			if path == "" {
				path = "."
			}

			tag := sdk.ParameterFind(a.Parameters, "tag")
			if tag == nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("tag variable is empty. aborting")
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
					res.Reason = fmt.Sprintf("Error while uploading artifact '%s': %v", filename, err)
					sendLog(res.Reason)
					return res
				}
			}

			return res
		}
	}

	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		res := sdk.Result{Status: sdk.StatusSuccess.String()}

		path := sdk.ParameterValue(a.Parameters, "path")
		if path == "" {
			path = "."
		}

		tag := sdk.ParameterFind(a.Parameters, "tag")
		if tag == nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("tag variable is empty. aborting")
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

		for _, filePath := range filesPath {
			filename := filepath.Base(filePath)
			sendLog(fmt.Sprintf("Uploading '%s'\n", filename))
			if err := w.client.QueueArtifactUpload(buildID, tag.Value, filePath); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("Error while uploading artefact: %s\n", err)
				sendLog(res.Reason)
				return res
			}
		}

		return res
	}
}
