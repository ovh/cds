package action

import (
	"context"
	"fmt"

	"os"
	"path/filepath"
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

func RunArtifactUpload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	res := sdk.Result{Status: sdk.StatusSuccess}

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	artifactPath := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "path"))
	if artifactPath == "" {
		artifactPath = "."
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return res, err
	}

	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
	} else {
		abs = workdir.Name()
	}

	if !sdk.PathIsAbs(artifactPath) {
		artifactPath = filepath.Join(abs, artifactPath)
	}

	tag := sdk.ParameterFind(a.Parameters, "tag")
	if tag == nil {
		return res, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("tag variable is empty. aborting"))
	}

	// Global all files matching filePath
	filesPath, err := afero.Glob(afero.NewOsFs(), artifactPath)
	if err != nil {
		return res, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("cannot perform globbing of pattern '%s': %s", artifactPath, err))
	}

	if len(filesPath) == 0 {
		return res, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("pattern '%s' matched no file", artifactPath))
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

	cdnArtifactEnabled := wk.FeatureEnabled(sdk.FeatureCDNArtifact)

	integrationName := sdk.DefaultIfEmptyStorage(strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination")))
	projectKey := sdk.ParameterValue(wk.Parameters(), "cds.project")
	pluginArtifactManagement := wk.GetPlugin(sdk.GRPCPluginUploadArtifact)

	wg.Add(len(filesPath))
	for _, p := range filesPath {
		go func(path string) {
			log.Debug(ctx, "uploading %s projectKey:%v integrationName:%v job:%d", path, projectKey, integrationName, jobID)
			defer wg.Done()

			// Priority:
			// 1. Integration specified on artifact upload action ( advanced parameter )
			// 2. Integration artifact manager on workflow
			// 3. CDN activated or nor
			if integrationName != "" {
				if err := uploadArtifactByApiCall(path, wk, ctx, projectKey, integrationName, jobID, tag); err != nil {
					log.Warn(ctx, "queueArtifactUpload(%s, %s, %d, %s, %s) failed: %v", projectKey, integrationName, jobID, tag.Value, path, err)
					chanError <- sdk.WrapError(err, "Error while uploading artifact by api call %s", path)
					wgErrors.Add(1)
				}
				return
			} else if pluginArtifactManagement != nil {
				if err := uploadArtifactByIntegrationPlugin(path, ctx, wk, pluginArtifactManagement); err != nil {
					log.Warn(ctx, "queueArtifactUpload(%s, %s, %d, %s, %s) failed: %v", projectKey, integrationName, jobID, tag.Value, path, err)
					chanError <- sdk.WrapError(err, "Error while uploading artifact by plugin %s", path)
					wgErrors.Add(1)
				}
			} else if !cdnArtifactEnabled {
				if err := uploadArtifactByApiCall(path, wk, ctx, projectKey, integrationName, jobID, tag); err != nil {
					log.Warn(ctx, "queueArtifactUpload(%s, %s, %d, %s, %s) failed: %v", projectKey, integrationName, jobID, tag.Value, path, err)
					chanError <- sdk.WrapError(err, "Error while uploading artifact by api call %s", path)
					wgErrors.Add(1)
				}
				return
			} else {
				if err := uploadArtifactIntoCDN(path, ctx, wk); err != nil {
					log.Error(ctx, "unable to upload artifact into cdn %q: %v", path, err)
					chanError <- err
					wgErrors.Add(1)
				}
				return
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
		return res, sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("error: %v", globalError.Error()))
	}

	return res, nil
}

func uploadArtifactByIntegrationPlugin(path string, ctx context.Context, wk workerruntime.Runtime, artiManager *sdk.GRPCPlugin) error {
	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to retrieve artifact manager integration... Aborting")
	}

	binary := artiManager.GetBinary(strings.ToLower(sdk.GOOS), strings.ToLower(sdk.GOARCH))
	if binary == nil {
		return fmt.Errorf("unable to retrieve the plugin for artifact upload integration %s... Aborting", pfName.Value)
	}

	pluginSocket, err := startGRPCPlugin(ctx, binary.PluginName, wk, binary, startGRPCPluginOptions{})
	if err != nil {
		return fmt.Errorf("unable to start GRPCPlugin: %v", err)
	}

	c, err := integrationplugin.Client(context.Background(), pluginSocket.Socket)
	if err != nil {
		return fmt.Errorf("unable to call GRPCPlugin: %v", err)
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

	opts := sdk.ParametersToMap(wk.Parameters())
	opts["cds.integration.artifact_manager.upload.path"] = path
	query := integrationplugin.RunQuery{
		Options: opts,
	}

	res, err := integrationPluginClient.Run(ctx, &query)
	if err != nil {
		return fmt.Errorf("error deploying application: %v", err)
	}

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Details: %s", res.Details))
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Status: %s", res.Status))

	if strings.ToUpper(res.Status) != strings.ToUpper(sdk.StatusSuccess) {
		return fmt.Errorf("plugin execution failed %s: %s", res.Status, res.Details)
	}
	return nil
}

func uploadArtifactIntoCDN(path string, ctx context.Context, wk workerruntime.Runtime) error {
	_, name := filepath.Split(path)

	fileMode, err := os.Stat(path)
	if err != nil {
		return sdk.WrapError(err, "unable to get file stat %s", path)
	}
	signature, err := wk.RunResultSignature(name, uint32(fileMode.Mode().Perm()), sdk.WorkflowRunResultTypeArtifact)
	if err != nil {
		return sdk.WrapError(err, "unable to sign artifact")
	}

	duration, err := wk.Client().CDNItemUpload(ctx, wk.CDNHttpURL(), signature, afero.NewOsFs(), path)
	if err != nil {
		return sdk.WrapError(err, "Error while uploading artifact %s", path)
	}
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS CDN", path, duration.Seconds()))
	return nil
}

func uploadArtifactByApiCall(path string, wk workerruntime.Runtime, ctx context.Context, projectKey string, integrationName string, jobID int64, tag *sdk.Parameter) error {
	throughTempURL, duration, err := wk.Client().QueueArtifactUpload(ctx, projectKey, integrationName, jobID, tag.Value, path)
	if err != nil {
		return err
	}
	if throughTempURL {
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to object store", path, duration.Seconds()))
	} else {
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS API", path, duration.Seconds()))
	}
	return nil
}
