package action

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

func RunArtifactDownload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	project := sdk.ParameterValue(wk.Parameters(), "cds.project")
	workflow := sdk.ParameterValue(wk.Parameters(), "cds.workflow")
	number := sdk.ParameterValue(wk.Parameters(), "cds.run.number")
	pattern := sdk.ParameterValue(a.Parameters, "pattern")

	destPath := sdk.ParameterValue(a.Parameters, "path")
	tag := sdk.ParameterValue(a.Parameters, "tag")

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

	n, err := strconv.ParseInt(number, 10, 64)
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

	pluginArtifactManagement := wk.GetPlugin(sdk.GRPCPluginDownloadArtifact)

	// Priority:
	// 1. Integration artifact manager on workflow
	// 2. CDN activated or not
	if pluginArtifactManagement != nil {
		return GetArtifactFromIntegrationPlugin(ctx, wk, res, pattern, reg, destPath, pluginArtifactManagement, project, workflow, n)
	}
	// GET Artifact from CDS API
	if !wk.FeatureEnabled(sdk.FeatureCDNArtifact) {
		return GetArtifactFromAPI(ctx, wk, project, workflow, n, res, pattern, reg, tag, destPath, wkDirFS)
	}

	// GET Artifact from CDS CDN
	cdnItems, err := wk.Client().WorkflowRunArtifactsLinks(project, workflow, n)
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
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifact %s from workflow %s/%s on run %d...", destFile, project, workflow, n))

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

func GetArtifactFromIntegrationPlugin(ctx context.Context, wk workerruntime.Runtime, res sdk.Result, pattern string, regexp *regexp.Regexp, destPath string, plugin *sdk.GRPCPlugin, key, wkfName string, number int64) (sdk.Result, error) {
	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return res, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to retrieve artifact manager integration... Aborting")
	}

	binary := plugin.GetBinary(strings.ToLower(sdk.GOOS), strings.ToLower(sdk.GOARCH))
	if binary == nil {
		return res, fmt.Errorf("unable to retrieve the plugin for artifact download integration %s... Aborting", pfName.Value)
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
			return res, err
		}
		opts := sdk.ParametersToMap(wk.Parameters())
		repoName := opts[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigCdsRepository)]
		if repoName != artData.RepoName {
			wg.Done()
			wk.SendLog(ctx, workerruntime.LevelDebug, fmt.Sprintf("%s does not match configured repo name %s - skipped", repoName, artData.RepoName))
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
			if err := runGRPCIntegrationPlugin(ctx, wk, binary, opts); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
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

func runGRPCIntegrationPlugin(ctx context.Context, wk workerruntime.Runtime, binary *sdk.GRPCPluginBinary, opts map[string]string) error {
	pluginSocket, err := startGRPCPlugin(ctx, binary.PluginName, wk, binary, startGRPCPluginOptions{})
	if err != nil {
		return fmt.Errorf("unable to start GRPCPlugin: %v", err)
	}

	c, err := integrationplugin.Client(context.Background(), pluginSocket.Socket)
	if err != nil {
		return fmt.Errorf("unable to call GRPCPlugin: %v", err)
	}

	qPort := integrationplugin.WorkerHTTPPortQuery{Port: wk.HTTPPort()}
	if _, err := c.WorkerHTTPPort(ctx, &qPort); err != nil {
		return fmt.Errorf("unable to setup plugin with worker port: %v", err)
	}

	pluginSocket.Client = c
	if _, err := c.Manifest(context.Background(), new(empty.Empty)); err != nil {
		return fmt.Errorf("unable to call GRPCPlugin: %v", err)
	}

	pluginClient := pluginSocket.Client
	integrationPluginClient, ok := pluginClient.(integrationplugin.IntegrationPluginClient)
	if !ok {
		return fmt.Errorf("unable to retrieve integration GRPCPlugin: %v", err)
	}

	logCtx, stopLogs := context.WithCancel(ctx)
	done := make(chan struct{})
	go enablePluginLogger(logCtx, done, pluginSocket, wk)

	defer integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)

	manifest, err := integrationPluginClient.Manifest(ctx, &empty.Empty{})
	if err != nil {
		return fmt.Errorf("unable to retrieve retrieve plugin manifest: %v", err)
	}

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Plugin %s v%s is ready", manifest.Name, manifest.Version))

	query := integrationplugin.RunQuery{
		Options: opts,
	}

	result, err := integrationPluginClient.Run(ctx, &query)
	if err != nil {
		return fmt.Errorf("error deploying application: %v", err)
	}

	if !strings.EqualFold(result.Status, sdk.StatusSuccess) {
		return fmt.Errorf("plugin execution failed %s: %s", result.Status, result.Details)
	}
	return nil
}

func GetArtifactFromAPI(ctx context.Context, wk workerruntime.Runtime, project string, workflow string, n int64, res sdk.Result, pattern string, regexp *regexp.Regexp, tag string, destPath string, wkDirFS afero.Fs) (sdk.Result, error) {
	wg := new(sync.WaitGroup)
	artifacts, err := wk.Client().WorkflowRunArtifacts(project, workflow, n)
	if err != nil {
		return res, err
	}
	wg.Add(len(artifacts))
	for i := range artifacts {
		a := &artifacts[i]

		if pattern != "" && !regexp.MatchString(a.Name) {
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%s does not match pattern %s - skipped", a.Name, pattern))
			wg.Done()
			continue
		}

		if tag != "" && a.Tag != tag {
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%s does not match tag %s - skipped", a.Name, tag))
			wg.Done()
			continue
		}

		go func(a *sdk.WorkflowNodeRunArtifact) {
			defer wg.Done()

			destFile := path.Join(destPath, a.Name)
			f, err := wkDirFS.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
			if err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact (OpenFile) %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Downloading artifact %s from workflow %s/%s on run %d...", destFile, project, workflow, n))
			if err := wk.Client().WorkflowNodeRunArtifactDownload(project, workflow, *a, f); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
			if err := f.Close(); err != nil {
				res.Status = sdk.StatusFail
				res.Reason = err.Error()
				log.Warn(ctx, "Cannot download artifact %s: %s", destFile, err)
				wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
				return
			}
		}(a)
		// TODO: write here a reason why we are waiting 3 seconds
		if len(artifacts) > 1 {
			time.Sleep(3 * time.Second)
		}
	}
	wg.Wait()
	return res, nil
}
