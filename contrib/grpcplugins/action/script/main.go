package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
 */

type runActionScriptPlugin struct {
	actionplugin.Common
}

func main() {
	p := runActionScriptPlugin{}
	if err := actionplugin.Start(context.Background(), &p); err != nil {
		panic(err)
	}
}

func (plug *runActionScriptPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "script",
		Author:      "Steven GUIHEUX <steven.guiheux@corp.ovh.com>",
		Description: "This is a plugin to run action of type script",
		Version:     sdk.VERSION,
	}, nil
}

func (plug *runActionScriptPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (plug *runActionScriptPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	plug.StreamServer = stream
	goRoutines := sdk.NewGoRoutines(ctx)

	content := q.GetOptions()["content"]

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &plug.Common)
	if err != nil {
		return fmt.Errorf("unable to get working directory: %v", err)
	}

	chanRes := make(chan *actionplugin.ActionResult)

	goRoutines.Exec(ctx, "runActionScriptPlugin-runScript", func(ctx context.Context) {
		grpcplugins.RunScript(ctx, &plug.Common, chanRes, workDirs.WorkingDir, content)
	})

	res := &actionplugin.StreamResult{}
	select {
	case <-ctx.Done():
		res.Status = sdk.StatusFail
		res.Details = "CDS Worker execution canceled: " + ctx.Err().Error()
	case result := <-chanRes:
		res.Status = result.Status
		res.Details = result.Details
	}
	if err := stream.Send(res); err != nil {
		return err
	}
	return nil
}
