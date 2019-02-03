package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin"
	"github.com/ovh/cds/sdk/log"
)

type startGRPCPluginOptions struct {
	envs []string
}

type pluginClientSocket struct {
	Socket  string
	StdPipe io.Reader
	Client  interface{}
}

func enablePluginLogger(ctx context.Context, done chan struct{}, sendLog LoggerFunc, c *pluginClientSocket) {
	reader := bufio.NewReader(c.StdPipe)
	var accumulator string
	var shouldExit bool
	defer func() {
		if accumulator != "" {
			sendLog(accumulator)
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
			sendLog(accumulator)
			accumulator = ""
			continue
		default:
			accumulator += content
			continue
		}
	}
}

func startGRPCPlugin(ctx context.Context, pluginName string, w *currentWorker, p *sdk.GRPCPluginBinary, opts startGRPCPluginOptions) (*pluginClientSocket, error) {
	currentOS := strings.ToLower(sdk.GOOS)
	currentARCH := strings.ToLower(sdk.GOARCH)

	binary := p
	if binary == nil {
		var errBi error
		binary, errBi = w.client.PluginGetBinaryInfos(pluginName, currentOS, currentARCH)
		if errBi != nil {
			return nil, sdk.WrapError(errBi, "plugin:%s Unable to get plugin binary infos... Aborting", pluginName)
		} else if binary == nil {
			return nil, fmt.Errorf("plugin:%s Unable to get plugin binary infos - binary is nil... Aborting", pluginName)
		}
	}

	// then try to download the plugin
	pluginBinary := path.Join(w.basedir, binary.Name)
	if _, err := os.Stat(pluginBinary); os.IsNotExist(err) {
		log.Debug("Downloading the plugin %s", binary.PluginName)
		//If the file doesn't exist. Download it.
		fi, err := os.OpenFile(pluginBinary, os.O_CREATE|os.O_RDWR, os.FileMode(binary.Perm))
		if err != nil {
			return nil, sdk.WrapError(err, "unable to create the file %s", pluginBinary)
		}

		log.Debug("Get the binary plugin %s", binary.PluginName)
		if err := w.client.PluginGetBinary(binary.PluginName, currentOS, currentARCH, fi); err != nil {
			_ = fi.Close()
			return nil, sdk.WrapError(err, "unable to get the binary plugin the file %s", binary.PluginName)
		}
		//It's downloaded. Close the file
		_ = fi.Close()
	} else {
		log.Debug("plugin binary is in cache %s", pluginBinary)
	}

	c := pluginClientSocket{}

	dir := w.currentJob.workingDirectory
	if dir == "" {
		dir = w.basedir
	}

	envs := make([]string, 0, len(opts.envs))
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CDS_") {
			continue
		}
		envs = append(envs, env)
	}
	envs = append(envs, opts.envs...)

	log.Info("Starting GRPC Plugin %s in dir %s", binary.Name, dir)
	fileContent, err := ioutil.ReadFile(path.Join(w.basedir, binary.GetName()))
	if err != nil {
		return nil, sdk.WrapError(err, "plugin:%s unable to get plugin binary file... Aborting", pluginName)
	}

	switch {
	case sdk.IsTar(fileContent):
		if err := sdk.Untar(w.basedir, bytes.NewReader(fileContent)); err != nil {
			return nil, sdk.WrapError(err, "plugin:%s unable to untar binary file", pluginName)
		}
	case sdk.IsGz(fileContent):
		if err := sdk.UntarGz(w.basedir, bytes.NewReader(fileContent)); err != nil {
			return nil, sdk.WrapError(err, "plugin:%s unable to untarGz binary file", pluginName)
		}
	}

	for i := range binary.Entrypoints {
		binary.Entrypoints[i] = path.Join(w.basedir, binary.Entrypoints[i])
	}

	cmd := binary.Cmd
	if _, err := exec.LookPath(cmd); err != nil {
		cmd = path.Join(w.basedir, cmd)
		if _, err := exec.LookPath(cmd); err != nil {
			return nil, sdk.WrapError(err, "plugin:%s unable to start GRPC plugin, binary command not found.", pluginName)
		}
	}
	args := append(binary.Entrypoints, binary.Args...)
	var errstart error
	if c.StdPipe, c.Socket, errstart = grpcplugin.StartPlugin(ctx, pluginName, dir, cmd, args, envs); errstart != nil {
		return nil, sdk.WrapError(errstart, "plugin:%s unable to start GRPC plugin... Aborting", pluginName)
	}

	return &c, nil
}
