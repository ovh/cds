package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

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
			pattern := sdk.ParameterValue(a.Parameters, "pattern")

			if pattern != "" {
				sendLog("pattern variable can be only used with CDS Workflow - ignored.")
			}

			if !enabled {
				sendLog("Artifact Download is disabled.")
				return res
			}

			if path == "" {
				path = "."
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
		pattern := sdk.ParameterValue(a.Parameters, "pattern")

		destPath := sdk.ParameterValue(a.Parameters, "path")
		tag := sdk.ParameterValue(a.Parameters, "tag")

		if destPath == "" {
			destPath = "."
		}

		if !enabled {
			sendLog("Artifact Download is disabled.")
			return res
		}

		if err := os.MkdirAll(destPath, os.FileMode(0744)); err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Unable to create %s: %v", destPath, err)
			sendLog(res.Reason)
			return res
		}

		if tag != "" {
			sendLog("tag variable can not be used with CDS Workflow - ignored.")
		}

		sendLog(fmt.Sprintf("Downloading artifacts from workflow into '%s'...", destPath))

		n, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("cds.run.number variable is not valid. aborting")
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

		regexp := regexp.MustCompile(pattern)
		for _, a := range artifacts {
			if pattern != "" && !regexp.MatchString(a.Name) {
				sendLog(fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, pattern))
				continue
			}
			destFile := path.Join(destPath, a.Name)
			f, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE, os.FileMode(a.Perm))
			if err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifact (OpenFile) %s: %s", destFile, err)
				sendLog(res.Reason)
				return res
			}
			sendLog(fmt.Sprintf("downloading artifact %s from workflow %s/%s on run %d...", destFile, project, workflow, n))
			if err := w.client.WorkflowNodeRunArtifactDownload(project, workflow, a.ID, f); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifact %s: %s", destFile, err)
				sendLog(res.Reason)
				return res
			}
			if err := f.Close(); err != nil {
				res.Status = sdk.StatusFail.String()
				res.Reason = err.Error()
				log.Warning("Cannot download artifact %s: %s", destFile, err)
				sendLog(res.Reason)
				return res
			}
		}

		return res
	}
}
