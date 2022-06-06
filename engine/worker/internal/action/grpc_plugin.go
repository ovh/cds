package action

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type startGRPCPluginOptions struct {
	envs []string
}

type pluginClientSocket struct {
	Socket  string
	StdPipe io.Reader
	Client  interface{}
}

func enablePluginLogger(ctx context.Context, done chan struct{}, c *pluginClientSocket, w workerruntime.Runtime) {
	reader := bufio.NewReader(c.StdPipe)
	var accumulator string
	var shouldExit bool
	defer func() {
		if accumulator != "" {
			w.SendLog(ctx, workerruntime.LevelInfo, accumulator)
		}
		close(done)
	}()

	for {
		if ctx.Err() != nil {
			shouldExit = true
		}

		if reader.Buffered() == 0 && shouldExit {
			return
		}
		b, err := reader.ReadByte()
		if err == io.EOF {
			if shouldExit {
				return
			}
			continue
		}

		content := string(b)
		switch content {
		case "":
			continue
		case "\n":
			accumulator += content
			w.SendLog(ctx, workerruntime.LevelInfo, accumulator)
			accumulator = ""
			continue
		default:
			accumulator += content
			continue
		}
	}
}

func RunGRPCPlugin(ctx context.Context, actionName string, params []sdk.Parameter, action sdk.Action, w workerruntime.Runtime, chanRes chan sdk.Result, done chan struct{}) {
	//For the moment we consider that plugin name = action name
	pluginName := actionName

	var envs []string
	//set up environment variables from job parameters
	for _, p := range params {
		// avoid put private key in environment var as it's a binary value
		if (p.Type == sdk.KeyPGPParameter || p.Type == sdk.KeySSHParameter) && strings.HasSuffix(p.Name, ".priv") {
			continue
		}
		envs = append(envs, sdk.EnvVartoENV(p)...)
		envName := strings.Replace(p.Name, ".", "_", -1)
		envName = strings.ToUpper(envName)
		envs = append(envs, fmt.Sprintf("%s=%s", envName, p.Value))
	}

	pluginSocket, err := startGRPCPlugin(ctx, pluginName, w, nil, startGRPCPluginOptions{
		envs: envs,
	})
	if err != nil {
		close(done)
		pluginFail(ctx, w, chanRes, fmt.Sprintf("Unable to start grpc plugin... Aborting (%v)", err))
		return
	}

	log.Info(ctx, "running plugin through socket %q", pluginSocket.Socket)

	c, err := actionplugin.Client(ctx, pluginSocket.Socket)
	if err != nil {
		close(done)
		pluginFail(ctx, w, chanRes, fmt.Sprintf("Unable to call grpc plugin... Aborting (%v)", err))
		return
	}
	qPort := actionplugin.WorkerHTTPPortQuery{Port: w.HTTPPort()}
	if _, err := c.WorkerHTTPPort(ctx, &qPort); err != nil {
		close(done)
		pluginFail(ctx, w, chanRes, fmt.Sprintf("Unable to set worker http port for grpc plugin... Aborting (%v)", err))
		return
	}

	pluginSocket.Client = c
	pluginClient := pluginSocket.Client
	actionPluginClient, ok := pluginClient.(actionplugin.ActionPluginClient)
	if !ok {
		close(done)
		pluginFail(ctx, w, chanRes, "Unable to retrieve plugin client... Aborting")
		return
	}

	logCtx, stopLogs := context.WithCancel(ctx)
	go enablePluginLogger(logCtx, done, pluginSocket, w)

	manifest, err := actionPluginClient.Manifest(ctx, &empty.Empty{})
	if err != nil {
		pluginFail(ctx, w, chanRes, fmt.Sprintf("Unable to retrieve plugin manifest... Aborting (%v)", err))
		actionPluginClientStop(ctx, actionPluginClient, stopLogs)
		return
	}
	log.Info(ctx, "plugin successfully initialized: %#v", manifest)

	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Plugin %s version %s is ready", manifest.Name, manifest.Version))

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		pluginFail(ctx, w, chanRes, fmt.Sprintf("Unable to retrieve job ID... Aborting (%v)", err))
		actionPluginClientStop(ctx, actionPluginClient, stopLogs)
		return
	}
	query := actionplugin.ActionQuery{
		Options: sdk.ParametersMapMerge(sdk.ParametersToMap(params), sdk.ParametersToMap(action.Parameters), sdk.MapMergeOptions.ExcludeGitParams),
		JobID:   jobID,
	}

	result, err := actionPluginClient.Run(ctx, &query)
	pluginDetails := fmt.Sprintf("plugin %s v%s", manifest.Name, manifest.Version)
	if err != nil {
		t := fmt.Sprintf("failure %s err: %v", pluginDetails, err)
		actionPluginClientStop(ctx, actionPluginClient, stopLogs)
		log.Error(ctx, t)
		pluginFail(ctx, w, chanRes, fmt.Sprintf("Error running action: %v", err))
		return
	}

	actionPluginClientStop(ctx, actionPluginClient, stopLogs)

	chanRes <- sdk.Result{
		Status: result.GetStatus(),
		Reason: result.GetDetails(),
	}
}

func startGRPCPlugin(ctx context.Context, pluginName string, w workerruntime.Runtime, p *sdk.GRPCPluginBinary, opts startGRPCPluginOptions) (*pluginClientSocket, error) {
	currentOS := strings.ToLower(sdk.GOOS)
	currentARCH := strings.ToLower(sdk.GOARCH)

	binary := p
	if binary == nil {
		var errBi error
		binary, errBi = w.Client().PluginGetBinaryInfos(pluginName, currentOS, currentARCH)
		if errBi != nil {
			return nil, sdk.WrapError(errBi, "plugin:%s Unable to get plugin binary infos... Aborting", pluginName)
		} else if binary == nil {
			return nil, fmt.Errorf("plugin:%s Unable to get plugin binary infos - binary is nil... Aborting", pluginName)
		}
	}

	// then try to download the plugin
	pluginBinary := binary.Name
	if _, err := w.BaseDir().Stat(pluginBinary); os.IsNotExist(err) {
		log.Debug(ctx, "Downloading the plugin %s", binary.PluginName)
		//If the file doesn't exist. Download it.
		fi, err := w.BaseDir().OpenFile(pluginBinary, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
		if err != nil {
			return nil, sdk.WrapError(err, "unable to create the file %s", pluginBinary)
		}

		log.Debug(ctx, "Get the binary plugin %s", binary.PluginName)
		//TODO: put afero in the client
		if err := w.Client().PluginGetBinary(binary.PluginName, currentOS, currentARCH, fi); err != nil {
			_ = fi.Close()
			return nil, sdk.WrapError(err, "unable to get the binary plugin the file %s", binary.PluginName)
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug(ctx, "plugin binary is in cache %s", pluginBinary)
	}

	c := pluginClientSocket{}

	envs := make([]string, 0, len(opts.envs))
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CDS_") {
			continue
		}
		envs = append(envs, env)
	}
	envs = append(envs, opts.envs...)

	log.Info(ctx, "Starting GRPC Plugin %s", binary.Name)
	fileContent, err := afero.ReadFile(w.BaseDir(), binary.GetName())
	if err != nil {
		return nil, sdk.WrapError(err, "plugin:%s unable to get plugin binary file... Aborting", pluginName)
	}

	switch {
	case sdk.IsTar(fileContent):
		if err := sdk.Untar(w.BaseDir(), "", bytes.NewReader(fileContent)); err != nil {
			return nil, sdk.WrapError(err, "plugin:%s unable to untar binary file", pluginName)
		}
	case sdk.IsGz(fileContent):
		if err := sdk.UntarGz(w.BaseDir(), "", bytes.NewReader(fileContent)); err != nil {
			return nil, sdk.WrapError(err, "plugin:%s unable to untarGz binary file", pluginName)
		}
	}

	var basedir string
	if x, ok := w.BaseDir().(*afero.BasePathFs); ok {
		basedir, _ = x.RealPath(".")
	} else {
		basedir = w.BaseDir().Name()
	}

	cmd := binary.Cmd
	if _, err := sdk.LookPath(w.BaseDir(), cmd); err != nil {
		return nil, sdk.WrapError(err, "plugin:%s unable to find GRPC plugin, binary command not found.", pluginName)
	}
	cmd = path.Join(basedir, cmd)

	for i := range binary.Entrypoints {
		binary.Entrypoints[i] = path.Join(basedir, binary.Entrypoints[i])
	}
	args := append(binary.Entrypoints, binary.Args...)
	var errstart error

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return nil, err
	}
	var dir string
	if x, ok := w.BaseDir().(*afero.BasePathFs); ok {
		dir, _ = x.RealPath(workdir.Name())
	} else {
		dir = workdir.Name()
	}

	if c.StdPipe, c.Socket, errstart = grpcplugin.StartPlugin(ctx, pluginName, dir, cmd, args, envs); errstart != nil {
		return nil, sdk.WrapError(errstart, "plugin:%s unable to start GRPC plugin... Aborting", pluginName)
	}
	return &c, nil
}

func pluginFail(ctx context.Context, w workerruntime.Runtime, chanRes chan<- sdk.Result, reason string) {
	res := sdk.Result{
		Reason: reason,
		Status: sdk.StatusFail,
	}
	w.SendLog(ctx, workerruntime.LevelError, res.Reason)
	chanRes <- res
}

func actionPluginClientStop(ctx context.Context, actionPluginClient actionplugin.ActionPluginClient, stopLogs context.CancelFunc) {
	if _, err := actionPluginClient.Stop(ctx, new(empty.Empty)); err != nil {
		// Transport is closing is a "normal" error, as we requested plugin to stop
		if !strings.Contains(err.Error(), "transport is closing") {
			log.Error(ctx, "Error on actionPluginClient.Stop: %s", err)
		}
	}
	stopLogs()
}
