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
	"time"

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

func enablePluginLogger(ctx context.Context, done chan struct{}, sendLog LoggerFunc, c *pluginClientSocket) {
	var shouldExit bool
	stdoutreader := bufio.NewReader(c.StdoutPipe)
	stderrreader := bufio.NewReader(c.StderrPipe)

	defer close(done)

	go func() {
		for {
			if ctx.Err() != nil {
				shouldExit = true
			}
			line, errs := stdoutreader.ReadString('\n')
			if errs != nil || shouldExit {
				c.StdoutPipe.Close()
				return
			} else if errs == io.EOF {
				continue
			}
			sendLog(line)
		}
	}()

	go func() {
		for {
			if ctx.Err() != nil {
				shouldExit = true
			}
			line, errs := stderrreader.ReadString('\n')
			if errs != nil || shouldExit {
				c.StderrPipe.Close()
				return
			} else if errs == io.EOF {
				continue
			}
			sendLog(line)
		}
	}()
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
		if errBi != nil || binary == nil {
			return nil, fmt.Errorf("plugin:%s Unable to get plugin binary infos... Aborting (%v)", pluginName, errBi)
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
		return nil, fmt.Errorf("plugin:%s unable to get plugin binary file... Aborting (%v)", pluginName, err)
	}

	switch {
	case sdk.IsTar(fileContent):
		if err := sdk.Untar(w.basedir, bytes.NewReader(fileContent)); err != nil {
			return nil, fmt.Errorf("plugin:%s unable to untar binary file (%v)", pluginName, err)
		}
	case sdk.IsGz(fileContent):
		if err := sdk.UntarGz(w.basedir, bytes.NewReader(fileContent)); err != nil {
			return nil, fmt.Errorf("plugin:%s unable to untarGz binary file (%v)", pluginName, err)
		}
	}

	for i := range binary.Entrypoints {
		binary.Entrypoints[i] = path.Join(w.basedir, binary.Entrypoints[i])
	}

	cmd := binary.Cmd
	if _, err := exec.LookPath(cmd); err != nil {
		cmd = path.Join(w.basedir, cmd)
		if _, err := exec.LookPath(cmd); err != nil {
			return nil, fmt.Errorf("plugin:%s unable to start GRPC plugin, binary command not found (%v)", pluginName, err)
		}
	}
	args := append(binary.Entrypoints, binary.Args...)
	var errstart error
	if c.StdoutPipe, c.StderrPipe, errstart = grpcplugin.StartPlugin(ctx, dir, cmd, args, envs); errstart != nil {
		return nil, fmt.Errorf("plugin:%s unable to start GRPC plugin... Aborting. err:%v", pluginName, errstart)
	}
	log.Info("GRPC Plugin %s started", binary.Name)

	//Sleep a while, to let the plugin write on stdout the socket address
	time.Sleep(500 * time.Millisecond)
	tsStart := time.Now()

	stdoutreader := bufio.NewReader(c.StdoutPipe)

	for {
		line, errs := stdoutreader.ReadString('\n')
		if errs == io.EOF {
			continue
		}
		if errs != nil {
			if time.Now().Before(tsStart.Add(5 * time.Second)) {
				log.Warning("plugin:%s error on ReadString, retry in 500ms...", pluginName)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			log.Error("plugin:%s error on ReadString(len buff %d, content: %s): %v", pluginName, len(line), line, err)
			continue
		}
		if strings.HasSuffix(line, "is ready to accept new connection\n") {
			socket := strings.TrimSpace(strings.Replace(line, " is ready to accept new connection\n", "", 1))
			log.Info("socket %s ready", socket)
			c.Socket = socket
			break
		}
	}

	registerPluginClient(w, pluginName, &c)

	return &c, nil
}

func registerPluginClient(w *currentWorker, pluginName string, c *pluginClientSocket) {
	w.mapPluginClient[pluginName] = c
}
