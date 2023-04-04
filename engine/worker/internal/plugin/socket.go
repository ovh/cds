package plugin

import (
	"bytes"
	"context"
	"os"
	"path"
	"strings"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin"
)

func createGRPCPluginSocket(ctx context.Context, pluginName string, w workerruntime.Runtime) (*clientSocket, error) {
	currentOS := strings.ToLower(sdk.GOOS)
	currentARCH := strings.ToLower(sdk.GOARCH)

	var pluginBinaryInfos *sdk.GRPCPluginBinary
	currentPlugin := w.GetPlugin(pluginName)
	if currentPlugin != nil {
		pluginBinaryInfos = currentPlugin.GetBinary(currentOS, currentARCH)
	}

	if pluginBinaryInfos == nil {
		log.Debug(ctx, "Retrieve plugin binary info: %s %s/%s", pluginName, currentOS, currentARCH)
		var err error
		pluginBinaryInfos, err = w.Client().PluginGetBinaryInfos(pluginName, currentOS, currentARCH)
		if err != nil {
			return nil, sdk.WrapError(err, "plugin:%s Unable to get plugin ... Aborting", pluginName)
		}
		if pluginBinaryInfos == nil {
			return nil, sdk.WrapError(err, "plugin:%s plugin %s not found ... Aborting", pluginName)
		}
	}

	// Try to download the plugin
	if _, err := w.BaseDir().Stat(pluginBinaryInfos.Name); os.IsNotExist(err) {
		log.Debug(ctx, "Downloading the plugin %s", pluginBinaryInfos.PluginName)
		//If the file doesn't exist. Download it.
		fi, err := w.BaseDir().OpenFile(pluginBinaryInfos.Name, os.O_CREATE|os.O_RDWR, os.FileMode(pluginBinaryInfos.Perm))
		if err != nil {
			return nil, sdk.WrapError(err, "unable to create the file %s", pluginBinaryInfos)
		}

		log.Debug(ctx, "Get the binary plugin %s", pluginBinaryInfos.PluginName)
		if err := w.Client().PluginGetBinary(pluginBinaryInfos.PluginName, currentOS, currentARCH, fi); err != nil {
			_ = fi.Close()
			return nil, sdk.WrapError(err, "unable to get the binary plugin the file %s", pluginBinaryInfos.PluginName)
		}
		_ = fi.Close()
	} else {
		log.Debug(ctx, "plugin binary is in cache %s", pluginBinaryInfos)
	}

	log.Info(ctx, "Starting GRPC Plugin %s", pluginBinaryInfos.Name)
	fileContent, err := afero.ReadFile(w.BaseDir(), pluginBinaryInfos.GetName())
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

	cmd := pluginBinaryInfos.Cmd
	if _, err := sdk.LookPath(w.BaseDir(), cmd); err != nil {
		return nil, sdk.WrapError(err, "plugin:%s unable to find GRPC plugin, binary command not found.", pluginName)
	}
	cmd = path.Join(basedir, cmd)

	for i := range pluginBinaryInfos.Entrypoints {
		pluginBinaryInfos.Entrypoints[i] = path.Join(basedir, pluginBinaryInfos.Entrypoints[i])
	}
	args := append(pluginBinaryInfos.Entrypoints, pluginBinaryInfos.Args...)
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

	c := clientSocket{}
	if c.StdPipe, c.Socket, errstart = grpcplugin.StartPlugin(ctx, pluginName, dir, cmd, args, []string{}); errstart != nil {
		return nil, sdk.WrapError(errstart, "plugin:%s unable to start GRPC plugin... Aborting", pluginName)
	}
	return &c, nil
}
