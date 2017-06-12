package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getArtifactParams(action *sdk.Action) (string, string) {
	var tag, filePattern string

	// Replace step argument in action arguments
	for _, p := range action.Parameters {
		switch p.Name {
		case "path":
			fmt.Printf("runArtifactUpload: path=%s\n", p.Value)
			filePattern = p.Value
			break
		case "tag":
			fmt.Printf("runArtifactUpload: tag=%s\n", p.Value)
			tag = p.Value
			break
		}
	}
	return filePattern, tag
}

func (w *currentWorker) runArtifactUpload(ctx context.Context, filePattern, tag string, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	res := sdk.Result{Status: sdk.StatusSuccess.String()}
	var project, pipeline, application, environment, buildNumberString string

	for _, p := range pbJob.Parameters {
		switch p.Name {
		case "cds.pipeline":
			fmt.Printf("runArtifactUpload: cds.pipeline=%s\n", p.Value)
			pipeline = p.Value
			break
		case "cds.project":
			fmt.Printf("runArtifactUpload: cds.project=%s\n", p.Value)
			project = p.Value
			break
		case "cds.application":
			fmt.Printf("runArtifactUpload: cds.application=%s\n", p.Value)
			application = p.Value
			break
		case "cds.environment":
			fmt.Printf("runArtifactUpload: cds.environment=%s\n", p.Value)
			environment = p.Value
			break
		case "cds.buildNumber":
			fmt.Printf("runArtifactUpload: buildNumber=%s\n", p.Value)
			buildNumberString = p.Value
		}
	}

	if tag == "" {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("tag variable is empty. aborting\n")
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}
	tag = strings.Replace(tag, "/", "-", -1)
	tag = url.QueryEscape(tag)

	// Global all files matching filePath
	filesPath, err := filepath.Glob(filePattern)
	if err != nil {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("cannot perform globbing of pattern '%s': %s\n", filePattern, err)
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	if len(filesPath) == 0 {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("Pattern '%s' matched no file\n", filePattern)
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	buildNumber, errBN := strconv.Atoi(buildNumberString)
	if errBN != nil {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("BuilNumber is not an integer %s\n", errBN)
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	for _, filePath := range filesPath {
		filename := filepath.Base(filePath)
		w.sendLog(pbJob.ID, fmt.Sprintf("Uploading '%s' into %s-%s-%s/%s...\n", filename, project, application, pipeline, tag), pbJob.PipelineBuildID, stepOrder, false)
		if err := sdk.UploadArtifact(project, pipeline, application, tag, filePath, buildNumber, environment); err != nil {
			res.Status = sdk.StatusFail.String()
			res.Reason = fmt.Sprintf("Error while uploading artefact: %s\n", err)
			w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
			return res
		}
	}

	return res
}

func (w *currentWorker) runArtifactDownload(ctx context.Context, a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	res := sdk.Result{Status: sdk.StatusSuccess.String()}
	var project, pipeline, application, environment, tag, filePath string
	enabled := true

	for _, p := range pbJob.Parameters {
		switch p.Name {
		case "cds.pipeline":
			fmt.Printf("runArtifactDownload: cds.pipeline=%s\n", p.Value)
			pipeline = p.Value
			break
		case "cds.project":
			fmt.Printf("runArtifactDownload: cds.project=%s\n", p.Value)
			project = p.Value
			break
		case "cds.application":
			fmt.Printf("runArtifactDownload: cds.application=%s\n", p.Value)
			application = p.Value
			break
		case "cds.environment":
			fmt.Printf("runArtifactDownload: cds.environment=%s\n", p.Value)
			environment = p.Value
			break
		case "enabled":
			fmt.Printf("runArtifactDownload: enabled=%s\n", p.Value)
			enabled = (p.Value != "false")
			break
		}
	}

	// Replace step argument in action arguments
	for _, p := range a.Parameters {
		switch p.Name {
		case "path":
			fmt.Printf("runArtifactDownload: path=%s\n", p.Value)
			filePath = p.Value
			break
		case "tag":
			fmt.Printf("runArtifactDownload: tag=%s\n", p.Value)
			tag = p.Value
			break
		case "pipeline":
			fmt.Printf("runArtifactDownload: pipeline=%s\n", p.Value)
			pipeline = p.Value
		case "application":
			fmt.Printf("runArtifactDownload: application=%s\n", p.Value)
			application = p.Value
		}
	}

	if !enabled {
		w.sendLog(pbJob.ID, fmt.Sprintf("Artifact Download is disabled. return\n"), pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	if tag == "" {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("tag variable is empty. aborting\n")
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}
	tag = strings.Replace(tag, "/", "-", -1)
	tag = url.QueryEscape(tag)

	if pipeline == "" {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("pipeline variable is empty. aborting\n")
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	w.sendLog(pbJob.ID, fmt.Sprintf("Downloading artifacts from %s-%s-%s/%s into '%s'...\n", project, application, pipeline, tag, filePath), pbJob.PipelineBuildID, stepOrder, false)
	err := sdk.DownloadArtifacts(project, application, pipeline, tag, filePath, environment)
	if err != nil {
		res.Status = sdk.StatusFail.String()
		res.Reason = fmt.Sprintf("%s\n", err)
		log.Warning("Cannot download artifacts: %s\n", err)
		w.sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	return res
}
