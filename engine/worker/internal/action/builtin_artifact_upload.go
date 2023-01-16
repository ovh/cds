package action

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func RunArtifactUpload(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	log.Info(ctx, "runningRunArtifactUpload")
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

	fileTypeParam := sdk.ParameterFind(a.Parameters, "type")
	var fileType string
	if fileTypeParam != nil {
		fileType = fileTypeParam.Value
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

	projectKey := sdk.ParameterValue(wk.Parameters(), "cds.project")
	pluginArtifactManagement := wk.GetPlugin(sdk.GRPCPluginUploadArtifact)

	wg.Add(len(filesPath))
	for _, p := range filesPath {
		go func(path string) {
			log.Info(ctx, "uploading %s projectKey:%v job:%d", path, projectKey, jobID)
			defer wg.Done()

			// Priority:
			// 1. Integration artifact manager on workflow
			// 2. CDN
			if pluginArtifactManagement != nil {
				if err := uploadArtifactByIntegrationPlugin(path, ctx, wk, pluginArtifactManagement, fileType); err != nil {
					chanError <- sdk.WrapError(err, "Error while uploading artifact by plugin %s", path)
					wgErrors.Add(1)
				}
			} else {
				if err := uploadArtifactIntoCDN(path, ctx, wk); err != nil {
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
		return res, fmt.Errorf("%s", globalError.Error())
	}

	return res, nil
}

func uploadArtifactByIntegrationPlugin(path string, ctx context.Context, wk workerruntime.Runtime, artiManager *sdk.GRPCPlugin, fileType string) error {
	_, fileName := filepath.Split(path)

	// Check run result
	code, err := checkArtifactUpload(ctx, wk, fileName, sdk.WorkflowRunResultTypeArtifactManager)
	if err != nil {
		if code == 409 {
			return fmt.Errorf("unable to upload the same file twice: %s", fileName)
		}
		return fmt.Errorf("unable to check artifact upload authorization: %v", err)
	}

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

	opts := sdk.ParametersToMap(wk.Parameters())
	opts[sdk.ArtifactUploadPluginInputPath] = path
	query := integrationplugin.RunQuery{
		Options: opts,
	}

	res, err := integrationPluginClient.Run(ctx, &query)
	if err != nil {
		return fmt.Errorf("error uploading artifact: %v", err)
	}

	if !strings.EqualFold(res.Status, sdk.StatusSuccess) {
		return fmt.Errorf("plugin execution failed %s: %s", res.Status, res.Details)
	}
	res.Outputs[sdk.ArtifactUploadPluginOutputFileType] = fileType

	// Add run result
	if err := addWorkflowRunResult(ctx, wk, path, sdk.WorkflowRunResultTypeArtifactManager, *res); err != nil {
		return fmt.Errorf("unable to add workflow run result for artifact %s: %v", path, err)
	}

	return nil
}

func uploadArtifactIntoCDN(path string, ctx context.Context, wk workerruntime.Runtime) error {
	log.Info(ctx, "uploadArtifactIntoCDN - begin")
	defer log.Info(ctx, "uploadArtifactIntoCDN - end")
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
		return err
	}
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("File '%s' uploaded in %.2fs to CDS CDN", path, duration.Seconds()))
	return nil
}

func checkArtifactUpload(ctx context.Context, wk workerruntime.Runtime, fileName string, runResultType sdk.WorkflowRunResultType) (int, error) {
	runID, runNodeID, runJobID := wk.GetJobIdentifiers()
	runResultCheck := sdk.WorkflowRunResultCheck{
		RunJobID:   runJobID,
		RunNodeID:  runNodeID,
		RunID:      runID,
		Name:       fileName,
		ResultType: runResultType,
	}
	return wk.Client().QueueWorkflowRunResultCheck(ctx, runJobID, runResultCheck)
}

func addWorkflowRunResult(ctx context.Context, wk workerruntime.Runtime, filePath string, runResultType sdk.WorkflowRunResultType, uploadResult integrationplugin.RunResult) error {
	runID, runNodeID, runJobID := wk.GetJobIdentifiers()

	fileMode, err := os.Stat(filePath)
	if err != nil {
		return sdk.WrapError(err, "unable to get file stat %s", fileMode)
	}
	perm, err := strconv.ParseUint(uploadResult.Outputs[sdk.ArtifactUploadPluginOutputPerm], 10, 32)
	if err != nil {
		return sdk.WrapError(err, "unable to retrieve file perm")
	}

	data := sdk.WorkflowRunResultArtifactManager{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: uploadResult.Outputs[sdk.ArtifactUploadPluginOutputPathFileName],
		},
		Perm:     uint32(perm),
		RepoName: uploadResult.Outputs[sdk.ArtifactUploadPluginOutputPathRepoName],
		Path:     uploadResult.Outputs[sdk.ArtifactUploadPluginOutputPathFilePath],
		FileType: uploadResult.Outputs[sdk.ArtifactUploadPluginOutputFileType],
	}

	bts, err := json.Marshal(data)
	if err != nil {
		return sdk.WithStack(err)
	}

	runResult := sdk.WorkflowRunResult{
		WorkflowNodeRunID: runNodeID,
		Type:              runResultType,
		WorkflowRunID:     runID,
		WorkflowRunJobID:  runJobID,
		DataRaw:           json.RawMessage(bts),
	}

	return wk.Client().QueueWorkflowRunResultsAdd(ctx, runJobID, runResult)
}
