package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/log"
)

var mapBuiltinActions = map[string]BuiltInActionFunc{}

func init() {
	mapBuiltinActions[sdk.ArtifactUpload] = runArtifactUpload
	mapBuiltinActions[sdk.ArtifactDownload] = runArtifactDownload
	mapBuiltinActions[sdk.ScriptAction] = runScriptAction
	mapBuiltinActions[sdk.JUnitAction] = runParseJunitTestResultAction
	mapBuiltinActions[sdk.GitCloneAction] = runGitClone
	mapBuiltinActions[sdk.GitTagAction] = runGitTag
	mapBuiltinActions[sdk.ReleaseAction] = runRelease
	mapBuiltinActions[sdk.CheckoutApplicationAction] = runCheckoutApplication
	mapBuiltinActions[sdk.DeployApplicationAction] = runDeployApplication
	mapBuiltinActions[sdk.CoverageAction] = runParseCoverageResultAction
}

// BuiltInAction defines builtin action signature
type BuiltInAction func(context.Context, *sdk.Action, int64, *[]sdk.Parameter, []sdk.Variable, LoggerFunc) sdk.Result

// BuiltInActionFunc returns the BuiltInAction given a worker
type BuiltInActionFunc func(*currentWorker) BuiltInAction

// LoggerFunc is the type for the logging function through BuiltInActions
type LoggerFunc func(format string)

func getLogger(w *currentWorker, buildID int64, stepOrder int) LoggerFunc {
	return func(s string) {
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		w.sendLog(buildID, s, stepOrder, false)
	}
}

func (w *currentWorker) runBuiltin(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, stepOrder int) sdk.Result {
	log.Info("runBuiltin> Begin buildID:%d stepOrder:%d", buildID, stepOrder)
	defer func() {
		log.Info("runBuiltin> End buildID:%d stepOrder:%d", buildID, stepOrder)
	}()
	defer w.drainLogsAndCloseLogger(ctx)

	//Define a loggin function
	sendLog := getLogger(w, buildID, stepOrder)

	f, ok := mapBuiltinActions[a.Name]
	if !ok {
		res := sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Unknown builtin step: %s\n", a.Name),
		}
		return res
	}

	return f(w)(ctx, a, buildID, params, secrets, sendLog)
}

func (w *currentWorker) runGRPCPlugin(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, stepOrder int, sendLog LoggerFunc) sdk.Result {
	log.Debug("runGRPCPlugin> Begin buildID:%d stepOrder:%d", buildID, stepOrder)
	defer func() {
		log.Debug("runGRPCPlugin> End buildID:%d stepOrder:%d", buildID, stepOrder)
	}()

	chanRes := make(chan sdk.Result, 1)
	done := make(chan struct{})
	sdk.GoRoutine(ctx, "runGRPCPlugin", func(ctx context.Context) {
		params := *params
		//For the moment we consider that plugin name = action name
		pluginName := a.Name

		envs := make([]string, 0, len(w.currentJob.buildVariables))
		//set up environment variables from job parameters
		for _, p := range params {
			// avoid put private key in environment var as it's a binary value
			if (p.Type == sdk.KeyPGPParameter || p.Type == sdk.KeySSHParameter) && strings.HasSuffix(p.Name, ".priv") {
				continue
			}
			if p.Type == sdk.KeyParameter && !strings.HasSuffix(p.Name, ".pub") {
				continue
			}
			envName := strings.Replace(p.Name, ".", "_", -1)
			envName = strings.ToUpper(envName)
			envs = append(envs, fmt.Sprintf("%s=%s", envName, p.Value))
		}

		for _, p := range w.currentJob.buildVariables {
			envName := strings.Replace(p.Name, ".", "_", -1)
			envName = strings.ToUpper(envName)
			envs = append(envs, fmt.Sprintf("%s=%s", envName, p.Value))
			sdk.AddParameter(&params, p.Name, p.Type, p.Value)
		}

		pluginSocket, err := startGRPCPlugin(context.Background(), pluginName, w, nil, startGRPCPluginOptions{
			envs: envs,
		})
		if err != nil {
			close(done)
			pluginFail(chanRes, sendLog, fmt.Sprintf("Unable to start grpc plugin... Aborting (%v)", err))
			return
		}

		c, err := actionplugin.Client(ctx, pluginSocket.Socket)
		if err != nil {
			close(done)
			pluginFail(chanRes, sendLog, fmt.Sprintf("Unable to call grpc plugin... Aborting (%v)", err))
			return
		}
		qPort := actionplugin.WorkerHTTPPortQuery{Port: w.exportPort}
		if _, err := c.WorkerHTTPPort(ctx, &qPort); err != nil {
			close(done)
			pluginFail(chanRes, sendLog, fmt.Sprintf("Unable to set worker http port for grpc plugin... Aborting (%v)", err))
			return
		}

		pluginSocket.Client = c
		pluginClient := pluginSocket.Client
		actionPluginClient, ok := pluginClient.(actionplugin.ActionPluginClient)
		if !ok {
			close(done)
			pluginFail(chanRes, sendLog, "Unable to retrieve plugin client... Aborting")
			return
		}

		logCtx, stopLogs := context.WithCancel(ctx)
		go enablePluginLogger(logCtx, done, sendLog, pluginSocket)

		manifest, err := actionPluginClient.Manifest(ctx, &empty.Empty{})
		if err != nil {
			pluginFail(chanRes, sendLog, fmt.Sprintf("Unable to retrieve plugin manifest... Aborting (%v)", err))
			actionPluginClientStop(ctx, actionPluginClient, stopLogs)
			return
		}
		log.Debug("plugin successfully initialized: %#v", manifest)

		sendLog(fmt.Sprintf("# Plugin %s v%s is ready", manifest.Name, manifest.Version))
		query := actionplugin.ActionQuery{
			Options: sdk.ParametersMapMerge(sdk.ParametersToMap(params), sdk.ParametersToMap(a.Parameters), sdk.MapMergeOptions.ExcludeGitParams),
			JobID:   buildID,
		}

		result, err := actionPluginClient.Run(ctx, &query)
		pluginDetails := fmt.Sprintf("plugin %s v%s", manifest.Name, manifest.Version)
		if err != nil {
			t := fmt.Sprintf("failure %s err: %v", pluginDetails, err)
			actionPluginClientStop(ctx, actionPluginClient, stopLogs)
			log.Error(t)
			pluginFail(chanRes, sendLog, fmt.Sprintf("Error running action: %v", err))
			return
		}

		actionPluginClientStop(ctx, actionPluginClient, stopLogs)

		chanRes <- sdk.Result{
			Status: result.GetStatus(),
			Reason: result.GetDetails(),
		}
	})

	select {
	case <-ctx.Done():
		log.Error("CDS Worker execution cancelled: %v", ctx.Err())
		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: "CDS Worker execution cancelled",
		}
	case res := <-chanRes:
		// Useful to wait all logs are send before sending final status and log
		<-done
		return res
	}
}

func actionPluginClientStop(ctx context.Context, actionPluginClient actionplugin.ActionPluginClient, stopLogs context.CancelFunc) {
	if _, err := actionPluginClient.Stop(ctx, new(empty.Empty)); err != nil {
		// Transport is closing is a "normal" error, as we requested plugin to stop
		if !strings.Contains(err.Error(), "transport is closing") {
			log.Error("Error on actionPluginClient.Stop: %s", err)
		}
	}
	stopLogs()
}

func pluginFail(chanRes chan<- sdk.Result, sendLog LoggerFunc, reason string) {
	res := sdk.Result{
		Reason: reason,
		Status: sdk.StatusFail.String(),
	}
	sendLog(res.Reason)
	chanRes <- res
}
