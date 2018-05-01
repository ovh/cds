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

func startGRPCPlugin(ctx context.Context, w *currentWorker, p sdk.GRPCPluginBinary, opts startGRPCPluginOptions) (string, error) {
	buffOut := new(bytes.Buffer)
	buffErr := new(bytes.Buffer)
	mOut := io.MultiWriter(opts.out, buffOut)
	mErr := io.MultiWriter(opts.err, buffErr)

	log.Info("Starting GRPC Plugin %s", p.Name)
	if err := grpcplugin.StartPlugin(ctx, w.basedir, p.Cmd, p.Args, opts.env, mOut, mErr); err != nil {
		return "", err
	}
	log.Info("GRPC Plugin started")

	//Sleep a while, to let the plugin write on stdout the socket address
	time.Sleep(100 * time.Millisecond)

	buff := new(strings.Builder)
	for {
		b, err := buffOut.ReadByte()
		if err != nil && len(buff.String()) > 0 {
			log.Error("error on ReadByte: %v", err)
			return "", fmt.Errorf("unable to get socket address from started binary")
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

	return socket, nil
}

func registerPluginClient(w *currentWorker, pluginName string, c interface{}) {
	w.mapPluginClient[pluginName] = c
}
