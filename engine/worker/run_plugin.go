package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin"
	"github.com/ovh/cds/sdk/log"
)

type startGRPCPluginOptions struct {
	out io.Writer
	err io.Writer
	env []string
}

type pluginClientSocket struct {
	Socket  string
	BuffOut bytes.Buffer
	Client  interface{}
}

func enablePluginLogger(ctx context.Context, sendLog LoggerFunc, c *pluginClientSocket) {
	var accumulator string
	var shouldExit bool

	for {
		if ctx.Err() != nil {
			shouldExit = true
		}

		b, err := c.BuffOut.ReadByte()
		if err == io.EOF && shouldExit {
			sendLog(accumulator)
			return
		}

		switch string(b) {
		case "":
			continue
		case "\n":
			accumulator += string(b)
			sendLog(accumulator)
			accumulator = ""
			continue
		default:
			accumulator += string(b)
			continue
		}
	}
}

func startGRPCPlugin(ctx context.Context, w *currentWorker, p sdk.GRPCPluginBinary, opts startGRPCPluginOptions) (*pluginClientSocket, error) {
	c := pluginClientSocket{}

	mOut := io.MultiWriter(opts.out, &c.BuffOut)
	mErr := io.MultiWriter(opts.err, &c.BuffOut)

	log.Info("Starting GRPC Plugin %s", p.Name)
	if err := grpcplugin.StartPlugin(ctx, w.basedir, p.Cmd, p.Args, opts.env, mOut, mErr); err != nil {
		return nil, err
	}
	log.Info("GRPC Plugin started")

	//Sleep a while, to let the plugin write on stdout the socket address
	time.Sleep(100 * time.Millisecond)

	buff := new(strings.Builder)
	for {
		b, err := c.BuffOut.ReadByte()
		if err != nil && len(buff.String()) > 0 {
			log.Error("error on ReadByte: %v", err)
			return nil, fmt.Errorf("unable to get socket address from started binary")
		}
		if err := buff.WriteByte(b); err != nil {
			log.Error("error on write byte: %v", err)
			break
		}
		if strings.HasSuffix(buff.String(), "is ready to accept new connection\n") {
			break
		}
	}

	socket := strings.Replace(buff.String(), " is ready to accept new connection\n", "", 1)
	log.Info("socket %s ready", socket)

	c.Socket = socket

	return &c, nil
}

func registerPluginClient(w *currentWorker, pluginName string, c *pluginClientSocket) {
	w.mapPluginClient[pluginName] = c
}
