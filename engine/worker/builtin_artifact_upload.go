package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/storageplugin"
	"github.com/ovh/cds/sdk/log"
)

func runArtifactUpload(wk *currentWorker) BuiltInAction {
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

		// destination attribute is the integrationName
		integrationName := strings.TrimSpace(sdk.ParameterValue(a.Parameters, "destination"))
		if integrationName != "" {
			return runArtifactUploadIntegration(ctx, wk, buildID, params, sendLog, filesPath, path, tag, integrationName)
		}

		return runArtifactUploadSharedInfra(ctx, wk, buildID, sendLog, filesPath, path, tag)
	}
}

func runArtifactUploadIntegration(ctx context.Context, wk *currentWorker, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc, filesPath []string, path string, tag *sdk.Parameter, integration string) sdk.Result {
	//First check OS and Architecture
	var currentOS = strings.ToLower(sdk.GOOS)
	var currentARCH = strings.ToLower(sdk.GOARCH)
	var binary *sdk.GRPCPluginBinary

	project := sdk.ParameterValue(*params, "cds.project")
	prjIntegration, err := wk.client.ProjectIntegrationGet(project, integration, false)
	if err != nil {
		res := sdk.Result{
			Reason: "Unable to get integration data... Aborting",
			Status: sdk.StatusFail.String(),
		}
		sendLog(err.Error())
		return res
	}

	for _, plugin := range prjIntegration.GRPCPlugins {
		for _, b := range plugin.Binaries {
			if b.OS == currentOS && b.Arch == currentARCH {
				binary = &b
				break
			}
		}
	}

	if binary == nil {
		res := sdk.Result{
			Reason: fmt.Sprintf("Unable to retrieve the plugin for storage integration %s... Aborting", integration),
			Status: sdk.StatusFail.String(),
		}
		sendLog(res.Reason)
		return res
	}

	pluginSocket, err := startGRPCPlugin(context.Background(), binary.PluginName, wk, binary, startGRPCPluginOptions{})
	if err != nil {
		res := sdk.Result{
			Reason: "Unable to startGRPCPlugin... Aborting",
			Status: sdk.StatusFail.String(),
		}
		sendLog(err.Error())
		return res
	}

	c, err := storageplugin.Client(context.Background(), pluginSocket.Socket)
	if err != nil {
		res := sdk.Result{
			Reason: "Unable to call grpc plugin... Aborting",
			Status: sdk.StatusFail.String(),
		}
		sendLog(err.Error())
		return res
	}

	pluginSocket.Client = c
	if _, err := c.Manifest(context.Background(), new(empty.Empty)); err != nil {
		res := sdk.Result{
			Reason: "Unable to call grpc plugin manifest... Aborting",
			Status: sdk.StatusFail.String(),
		}
		sendLog(err.Error())
		return res
	}

	pluginClient := pluginSocket.Client
	storagePluginClient, ok := pluginClient.(storageplugin.StoragePluginClient)
	if !ok {
		res := sdk.Result{
			Reason: "Unable to retrieve plugin client... Aborting",
			Status: sdk.StatusFail.String(),
		}
		sendLog(res.Reason)
		return res
	}

	logCtx, stopLogs := context.WithCancel(ctx)
	done := make(chan struct{})
	go enablePluginLogger(logCtx, done, sendLog, pluginSocket)

	manifest, err := storagePluginClient.Manifest(ctx, &empty.Empty{})
	if err != nil {
		res := sdk.Result{
			Reason: "Unable to retrieve plugin manifest... Aborting",
			Status: sdk.StatusFail.String(),
		}
		storagePluginClientStop(ctx, storagePluginClient, done, stopLogs)
		sendLog(err.Error())
		return res
	}

	sendLog(fmt.Sprintf("# Plugin %s v%s is ready", manifest.Name, manifest.Version))

	opts := storageplugin.Options{
		Options: sdk.ParametersToMap(*params),
	}

	res, err := storagePluginClient.ArtifactUpload(ctx, &opts)
	if err != nil {
		res := sdk.Result{
			Reason: fmt.Sprintf("Error deploying application: %v", err),
			Status: sdk.StatusFail.String(),
		}
		storagePluginClientStop(ctx, storagePluginClient, done, stopLogs)
		return res
	}

	sendLog(fmt.Sprintf("# Details: %s", res.Details))
	sendLog(fmt.Sprintf("# Status: %s", res.Status))

	if strings.EqualFold(res.Status, sdk.StatusSuccess.String()) {
		storagePluginClientStop(ctx, storagePluginClient, done, stopLogs)
		return sdk.Result{
			Status: sdk.StatusSuccess.String(),
		}
	}

	storagePluginClientStop(ctx, storagePluginClient, done, stopLogs)

	return sdk.Result{
		Status: sdk.StatusFail.String(),
		Reason: res.Details,
	}
}

func storagePluginClientStop(ctx context.Context, storagePluginClient storageplugin.StoragePluginClient, done chan struct{}, stopLogs context.CancelFunc) {
	if _, err := storagePluginClient.Stop(ctx, new(empty.Empty)); err != nil {
		// Transport is closing is a "normal" error, as we requested plugin to stop
		if !strings.Contains(err.Error(), "transport is closing") {
			log.Error("Error on storagePluginClient.Stop: %s", err)
		}
	}
	stopLogs()
	<-done
}

func runArtifactUploadSharedInfra(ctx context.Context, wk *currentWorker, buildID int64, sendLog LoggerFunc, filesPath []string, path string, tag *sdk.Parameter) sdk.Result {
	var globalError = &sdk.MultiError{}
	var chanError = make(chan error)
	var wg = new(sync.WaitGroup)
	var wgErrors = new(sync.WaitGroup)
	res := sdk.Result{Status: sdk.StatusSuccess.String()}

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
			throughTempURL, duration, err := wk.client.QueueArtifactUpload(ctx, buildID, tag.Value, path)
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
