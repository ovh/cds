package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type cacheRestorePlugin struct {
	actionplugin.Common
}

func (actPlugin *cacheRestorePlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-cacheRestore",
		Author:      "Steven GUIHEUX <steven.guiheux@ovhcloud.com>",
		Description: `This action allow you to retrieve a cache `,
		Version:     sdk.VERSION,
	}, nil
}

func (p *cacheRestorePlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	cacheKey := q.GetOptions()["key"]
	path := q.GetOptions()["path"]
	failOnMiss := q.GetOptions()["fail-on-cache-miss"]

	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to retrieve job context: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	if err := grpcplugins.PerformGetCache(ctx, &p.Common, *jobCtx, cacheKey, workDirs, path, (failOnMiss == "true")); err != nil {
		err := fmt.Errorf("unable to retrieve cache: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)

}

func (actPlugin *cacheRestorePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func main() {
	actPlugin := cacheRestorePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
