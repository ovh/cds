package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
)

func runServeStaticFiles(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		res := sdk.Result{Status: sdk.StatusSuccess.String()}

		pipeline := sdk.ParameterValue(*params, "cds.pipeline")
		project := sdk.ParameterValue(*params, "cds.project")
		node := sdk.ParameterValue(*params, "cds.node")
		run := sdk.ParameterValue(*params, "cds.run")

		path := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
		if path == "" {
			path = "."
		}

		entrypoint := sdk.ParameterFind(&a.Parameters, "entrypoint")
		if entrypoint == nil || entrypoint.Value == "" {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("entrypoint parameter is empty. aborting")
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

		file, err := sdk.CreateTarFromPaths(w.currentJob.workingDirectory, filesPath)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Cannot tar files: %v", err)
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
