package plugin

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/kardianos/osext"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin"
)

func createGRPCPluginSocket(ctx context.Context, pluginType string, pluginName string, w workerruntime.Runtime, env map[string]string) (*clientSocket, *sdk.GRPCPlugin, error) {
	log.Info(ctx, "create socket for plugin %q", pluginName)
	currentOS := strings.ToLower(sdk.GOOS)
	currentARCH := strings.ToLower(sdk.GOARCH)

	var currentPlugin *sdk.GRPCPlugin
	switch pluginType {
	case TypeAction, TypeStream:
		currentPlugin = w.GetActionPlugin(pluginName)
		if currentPlugin == nil {
			var err error
			currentPlugin, err = w.PluginGet(pluginName)
			if err != nil {
				return nil, nil, sdk.NewErrorFrom(sdk.ErrNotFound, "plugin:%s Unable to get plugin ... Aborting", pluginName)
			}
			w.SetActionPlugin(currentPlugin)
		}
	case TypeIntegration:
		currentPlugin = w.GetIntegrationPlugin(pluginName)
		if currentPlugin == nil {
			return nil, nil, sdk.NewErrorFrom(sdk.ErrNotFound, "plugin:%s Unable to get plugin ... Aborting", pluginName)
		}
	}

	// Download the plugin and check its integrity. The returned descriptor is read from the
	// API: it takes precedence over the one carried by the job payload, which can be older
	// than the binary actually served. Use currentPlugin.Name and not pluginName: for
	// integration plugins the latter is an integration type, not a plugin name.
	pluginBinaryInfos, err := DownloadBinary(ctx, w, currentPlugin.Name, currentOS, currentARCH)
	if err != nil {
		return nil, nil, err
	}

	log.Info(ctx, "Starting GRPC Plugin %s", pluginBinaryInfos.Name)
	fileContent, err := afero.ReadFile(w.BaseDir(), pluginBinaryInfos.GetName())
	if err != nil {
		return nil, nil, sdk.WrapError(err, "plugin:%s unable to get plugin binary file... Aborting", pluginName)
	}

	switch {
	case sdk.IsTar(fileContent):
		if err := sdk.Untar(w.BaseDir(), "", bytes.NewReader(fileContent)); err != nil {
			return nil, nil, sdk.WrapError(err, "plugin:%s unable to untar binary file", pluginName)
		}
	case sdk.IsGz(fileContent):
		if err := sdk.UntarGz(w.BaseDir(), "", bytes.NewReader(fileContent)); err != nil {
			return nil, nil, sdk.WrapError(err, "plugin:%s unable to untarGz binary file", pluginName)
		}
	}

	var basedir string
	if x, ok := w.BaseDir().(*afero.BasePathFs); ok {
		basedir, _ = x.RealPath(".")
	} else {
		basedir = w.BaseDir().Name()
	}

	cmd := pluginBinaryInfos.Cmd
	if _, err := sdk.LookPath(w.BaseDir(), cmd); err != nil {
		return nil, nil, sdk.WrapError(err, "plugin:%s unable to find GRPC plugin, binary command not found.", pluginName)
	}
	cmd = path.Join(basedir, cmd)

	for i := range pluginBinaryInfos.Entrypoints {
		pluginBinaryInfos.Entrypoints[i] = path.Join(basedir, pluginBinaryInfos.Entrypoints[i])
	}
	args := append(pluginBinaryInfos.Entrypoints, pluginBinaryInfos.Args...)
	var errstart error

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return nil, nil, err
	}
	var dir string
	if x, ok := w.BaseDir().(*afero.BasePathFs); ok {
		dir, _ = x.RealPath(workdir.Name())
	} else {
		dir = workdir.Name()
	}

	// Retrieve worker environment variables
	workerEnvs := w.Environ()
	mWorkerEnvs := make(map[string]string, len(workerEnvs))
	for _, e := range workerEnvs {
		splitted := strings.SplitN(e, "=", 2)
		if len(splitted) != 2 {
			continue
		}
		mWorkerEnvs[splitted[0]] = splitted[1]
	}

	// Add env variable from execution context
	for k, v := range env {
		// Set all env ( do not ovveride existing var )
		if _, ok := mWorkerEnvs[k]; !ok && k != "PATH" {
			mWorkerEnvs[k] = v
			continue
		}
	}

	// Manage PATH
	if v, has := env["PATH"]; has {
		existingPath := mWorkerEnvs["PATH"]
		existingPathList := filepath.SplitList(existingPath)
		newPath := v
		newPathList := filepath.SplitList(newPath)
		newPathList = append(newPathList, existingPathList...)
		newPathList = sdk.Unique(newPathList)
		mWorkerEnvs["PATH"] = strings.Join(newPathList, string(filepath.ListSeparator))
	}

	var envs []string
	for k, v := range mWorkerEnvs {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	c := clientSocket{}

	workerpath, err := osext.Executable()
	if err != nil {
		return nil, nil, sdk.WrapError(err, "unable to get current executable path")
	}

	log.Debug(ctx, "runScriptAction> Worker binary path: %s", path.Dir(workerpath))
	for i := range envs {
		if strings.HasPrefix(envs[i], "PATH=") {
			envs[i] = fmt.Sprintf("%s:%s", envs[i], path.Dir(workerpath))
			break
		}
	}

	if c.StdPipe, c.Socket, errstart = grpcplugin.StartPlugin(ctx, pluginName, dir, cmd, args, envs); errstart != nil {
		return nil, nil, sdk.WrapError(errstart, "plugin:%s unable to start GRPC plugin... Aborting", pluginName)
	}
	return &c, currentPlugin, nil
}
