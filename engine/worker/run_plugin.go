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
	Socket     string
	StdoutPipe io.ReadCloser
	StderrPipe io.ReadCloser
	Client     interface{}
}

func readFromPlugin(ctx context.Context, readCloser io.ReadCloser, sendLog LoggerFunc) {
	var shouldExit bool
	stdreader := bufio.NewReader(readCloser)
	go func() {
		for {
			if ctx.Err() != nil {
				shouldExit = true
			}
			line, errs := stdreader.ReadString('\n')
			if errs != nil || shouldExit {
				readCloser.Close()
				return
			} else if errs == io.EOF {
				continue
			}
			sendLog(line)
		}
	}()
}

func enablePluginLogger(ctx context.Context, done chan struct{}, sendLog LoggerFunc, c *pluginClientSocket) {
	defer close(done)
	readFromPlugin(ctx, c.StdoutPipe, sendLog)
	readFromPlugin(ctx, c.StderrPipe, sendLog)
}

func startGRPCPlugin(ctx context.Context, pluginName string, w *currentWorker, p *sdk.GRPCPluginBinary, opts startGRPCPluginOptions) (*pluginClientSocket, error) {
	currentOS := strings.ToLower(sdk.GOOS)
	currentARCH := strings.ToLower(sdk.GOARCH)
	pluginSocket, has := w.mapPluginClient[pluginName]
	if has {
		return pluginSocket, nil
	}

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
	if c.StdoutPipe, c.StderrPipe, c.Socket, errstart = grpcplugin.StartPlugin(ctx, pluginName, dir, cmd, args, envs); errstart != nil {
		return nil, sdk.WrapError(errstart, "plugin:%s unable to start GRPC plugin... Aborting", pluginName)
	}

	registerPluginClient(w, pluginName, &c)

	return &c, nil
}

func registerPluginClient(w *currentWorker, pluginName string, c *pluginClientSocket) {
	w.mapPluginClient[pluginName] = c
}
