package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
				res.Reason = fmt.Sprintf("BuilNumber is not an integer %s\n", errBN)
				sendLog(res.Reason)
				return res
			}

			for _, filePath := range filesPath {
				filename := filepath.Base(filePath)
				sendLog(fmt.Sprintf("Uploading '%s'\n", filename))
				//TODO send artifact on workflow
				if err := sdk.UploadArtifact(project, pipeline, application, tag.Value, filePath, buildNumber, environment); err != nil {
					res.Status = sdk.StatusFail.String()
					res.Reason = fmt.Sprintf("Error while uploading artefact: %s\n", err)
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
			//TODO send artifact on workflow
			if err := w.client.QueueArtifactUpload(buildID, tag.Value, filename); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("Error while uploading artefact: %s\n", err)
				sendLog(res.Reason)
				return res
			}
		}

		return res
	}
}

func runArtifactDownload(w *currentWorker) BuiltInAction {
	if w.currentJob.wJob == nil {
		return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
			res := sdk.Result{Status: sdk.StatusSuccess.String()}

			project := sdk.ParameterValue(*params, "cds.project")
			environment := sdk.ParameterValue(*params, "cds.environment")
			enabled := sdk.ParameterValue(*params, "enabled") != "false"

			application := sdk.ParameterValue(a.Parameters, "application")
			pipeline := sdk.ParameterValue(a.Parameters, "pipeline")
			path := sdk.ParameterValue(a.Parameters, "path")
			tag := sdk.ParameterValue(a.Parameters, "tag")

			if !enabled {
				sendLog("Artifact Download is disabled.")
				return res
			}

			if tag == "" {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("tag variable is empty. aborting")
				sendLog(res.Reason)
				return res
			}
			tag = strings.Replace(tag, "/", "-", -1)
			tag = url.QueryEscape(tag)

			if pipeline == "" {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("pipeline variable is empty. aborting\n")
				sendLog(res.Reason)
				return res
			}

			sendLog(fmt.Sprintf("Downloading artifacts from into '%s'...", path))
			if err := sdk.DownloadArtifacts(project, application, pipeline, tag, path, environment); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifacts: %s", err)
				sendLog(res.Reason)
				return res
			}

			return res
		}
	}

	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		res := sdk.Result{Status: sdk.StatusSuccess.String()}

		project := sdk.ParameterValue(*params, "cds.project")
		workflow := sdk.ParameterValue(*params, "cds.workflow")
		number := sdk.ParameterValue(*params, "cds.run.number")
		enabled := sdk.ParameterValue(*params, "enabled") != "false"

		path := sdk.ParameterValue(a.Parameters, "path")
		tag := sdk.ParameterValue(a.Parameters, "tag")

		if !enabled {
			sendLog("Artifact Download is disabled.")
			return res
		}

		if tag == "" {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("tag variable is empty. aborting")
			sendLog(res.Reason)
			return res
		}
		tag = strings.Replace(tag, "/", "-", -1)
		tag = url.QueryEscape(tag)

		sendLog(fmt.Sprintf("Downloading artifacts from into '%s'...", path))

		n, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("cds.run.nubmer variable is not valid. aborting\n")
			sendLog(res.Reason)
			return res
		}
		artifacts, err := w.client.WorkflowRunArtifacts(project, workflow, n)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = err.Error()
			log.Warning("Cannot download artifacts: %s", err)
			sendLog(res.Reason)
			return res
		}

		for _, a := range artifacts {
			f, err := os.OpenFile(a.Name, os.O_RDWR|os.O_CREATE, os.FileMode(a.Perm))
			if err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifacts: %s", err)
				sendLog(res.Reason)
				return res
			}
			if err := w.client.WorkflowNodeRunArtifactDownload(project, workflow, a.ID, f); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifacts: %s", err)
				sendLog(res.Reason)
				return res
			}
			if err := f.Close(); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifacts: %s", err)
				sendLog(res.Reason)
				return res
			}
		}

		return res
	}
}
