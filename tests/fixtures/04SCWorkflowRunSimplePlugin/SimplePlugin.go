package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type simplePlugin struct {
	actionplugin.Common
}

func (actPlugin *simplePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-simple",
		Author:      "Steven GUIHEUX <foo.bar@foobar.com>",
		Description: `This plugin do nothing.`,
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *simplePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	fmt.Println("Hello, I'm the simple plugin")
	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func (actPlugin *simplePlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := simplePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
