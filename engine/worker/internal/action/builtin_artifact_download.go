package action

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunArtifactDownload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	project := sdk.ParameterValue(wk.Parameters(), "cds.project")
	workflow := sdk.ParameterValue(wk.Parameters(), "cds.workflow")
	number := sdk.ParameterValue(wk.Parameters(), "cds.run.number")
	pattern := sdk.ParameterValue(a.Parameters, "pattern")

	destPath := sdk.ParameterValue(a.Parameters, "path")

	actionWorkflow := sdk.ParameterValue(a.Parameters, "workflow")
	actionRunNumber := sdk.ParameterValue(a.Parameters, "number")

	destinationWorkflow := workflow
	if actionWorkflow != "" {
		destinationWorkflow = actionWorkflow
	}
	destinationWorkflowNum := number
	if actionRunNumber != "0" && actionRunNumber != "" {
		destinationWorkflowNum = actionRunNumber
	}

	if destPath == "" {
		destPath = "."
	}
	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		res.Status = sdk.StatusFail
		return res, err
	}

	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
	} else {
		abs = workdir.Name()
	}

	if !sdk.PathIsAbs(destPath) {
		destPath = filepath.Join(abs, destPath)
	}

	wkDirFS := afero.NewOsFs()
	if err := wkDirFS.MkdirAll(destPath, os.FileMode(0744)); err != nil {
		res.Status = sdk.StatusFail
		return res, fmt.Errorf("unable to create %s: %v", destPath, err)
	}

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifacts from workflow into '%s'...", destPath))

	n, err := strconv.ParseInt(destinationWorkflowNum, 10, 64)
	if err != nil {
		res.Status = sdk.StatusFail
		return res, fmt.Errorf("cds.run.number variable is not valid. aborting")
	}

	reg, err := regexp.Compile(pattern)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Reason = fmt.Sprintf("Invalid pattern %s, must be a regex : %v", pattern, err)
		wk.SendLog(ctx, workerruntime.LevelInfo, res.Reason)
		return res, err
	}

	// Priority:
	// 1. Integration artifact manager on workflow
	// 2. CDN
	if wk.GetIntegrationPlugin(sdk.GRPCPluginDownloadArtifact) != nil {
		return GetArtifactFromIntegrationPlugin(ctx, wk, res, pattern, reg, destPath, sdk.GRPCPluginDownloadArtifact, project, destinationWorkflow, n)
	}
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("CDN '%s'...", destPath))

	// GET Artifact from CDS CDN
	cdnItems, err := wk.Client().WorkflowRunArtifactsLinks(project, destinationWorkflow, n)
	if err != nil {
		res.Status = sdk.StatusFail
		return res, err
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(cdnItems.Items))
	for i := range cdnItems.Items {
		item := cdnItems.Items[i]
		apiRef, is := item.GetCDNRunResultApiRef()
		if !is {
			res.Status = sdk.StatusFail
			res.Reason = fmt.Sprintf("item %s is not an artifact", item.ID)
			return res, sdk.WrapError(sdk.ErrInvalidData, "item %s is not an artifact", item.ID)
		}
		if pattern != "" && !reg.MatchString(apiRef.ToFilename()) {
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, pattern))
			wg.Done()
			continue
		}

		go func(a sdk.CDNItem) {
			defer wg.Done()
			destFile := path.Join(destPath, a.APIRef.ToFilename())
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifact %s from workflow %s/%s on run %d...", destFile, project, destinationWorkflow, n))

			f, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(apiRef.Perm))
			if err != nil {
				res.Status = sdk.StatusFail
				res.Reason = sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("cannot create file (OpenFile) %s: %s", destFile, err)).Error()
				log.Warn(ctx, "%s", res.Reason)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
			if err := wk.Client().CDNItemDownload(ctx, wk.CDNHttpURL(), item.APIRefHash, sdk.CDNTypeItemRunResult, a.MD5, f); err != nil {
				_ = f.Close()
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
			if err := f.Close(); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to close file %s: %v", destFile, err).Error()
				log.Warn(ctx, "%s", res.Reason)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
		}(item)
	}
	wg.Wait()
	return res, nil
}

func GetArtifactFromIntegrationPlugin(ctx context.Context, wk workerruntime.Runtime, res sdk.Result, pattern string, regexp *regexp.Regexp, destPath string, pluginName string, key, wkfName string, number int64) (sdk.Result, error) {
	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return res, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to retrieve artifact manager integration... Aborting")
	}

	wg := new(sync.WaitGroup)
	runResults, err := wk.Client().WorkflowRunResultsList(ctx, key, wkfName, number)
	if err != nil {
		return res, err
	}

	wg.Add(len(runResults))
	for _, runResult := range runResults {
		if runResult.Type != sdk.WorkflowRunResultTypeArtifactManager {
			wg.Done()
			continue
		}

		artData, err := runResult.GetArtifactManager()
		if err != nil {
			wk.SendLog(ctx, workerruntime.LevelInfo, "Can read run result data: "+err.Error())
			wg.Done()
			continue
		}
		opts := sdk.ParametersToMap(wk.Parameters())
		repoName := opts[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigCdsRepository)]
		if repoName != artData.RepoName {
			wk.SendLog(ctx, workerruntime.LevelDebug, fmt.Sprintf("%s does not match configured repo name %s - skipped", repoName, artData.RepoName))
			wg.Done()
			continue
		}

		if pattern != "" && !regexp.MatchString(artData.Name) {
			wk.SendLog(ctx, workerruntime.LevelDebug, fmt.Sprintf("%s does not match pattern %s - skipped", artData.Name, pattern))
			wg.Done()
			continue
		}
		destFile := path.Join(destPath, artData.Name)
		opts[sdk.ArtifactDownloadPluginInputDestinationPath] = destFile
		opts[sdk.ArtifactDownloadPluginInputFilePath] = artData.Path
		opts[sdk.ArtifactDownloadPluginInputMd5] = artData.MD5
		opts[sdk.ArtifactDownloadPluginInputPerm] = strconv.FormatUint(uint64(artData.Perm), 10)

		go func(opts map[string]string) {
			defer wg.Done()
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifact %s from %s...", artData.Name, repoName))
			resultPart := runGRPCIntegrationPlugin(ctx, wk, pluginName, opts)
			if resultPart.Status == sdk.StatusFail {
				res.Status = resultPart.Status
				res.Reason = resultPart.Reason
			}
		}(opts)
		// Be kind with the artifact manager
		if len(runResults) > 1 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	wg.Wait()
	return res, nil
}

func runGRPCIntegrationPlugin(ctx context.Context, wk workerruntime.Runtime, pluginName string, opts map[string]string) sdk.Result {
	pluginClient, err := plugin.NewClient(ctx, wk, plugin.TypeIntegration, pluginName, plugin.InputManagementDefault)
	if err != nil {
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("unable to start GRPCPlugin: %v", err),
		}
	}
	defer pluginClient.Close(ctx)
	pluginResult := pluginClient.Run(ctx, opts)
	return sdk.Result{Status: pluginResult.Status, Reason: pluginResult.Details}
}
