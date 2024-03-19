package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

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
	actPlugin := runActionScriptPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}

func (actPlugin *runActionScriptPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "script",
		Author:      "Steven GUIHEUX <steven.guiheux@corp.ovh.com>",
		Description: "This is a plugin to run action of type script",
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *runActionScriptPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	goRoutines := sdk.NewGoRoutines(ctx)

	content := q.GetOptions()["content"]

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &actPlugin.Common)
	if err != nil {
		return nil, fmt.Errorf("unable to get working directory: %v", err)
	}

	chanRes := make(chan *actionplugin.ActionResult)

	goRoutines.Exec(ctx, "runActionScriptPlugin-runScript", func(ctx context.Context) {
		if err := grpcplugins.RunScript(ctx, chanRes, workDirs.WorkingDir, content); err != nil {
			fmt.Printf("%+v\n", err)
		}
	})

	res := &actionplugin.ActionResult{}
	select {
	case <-ctx.Done():
		fmt.Printf("CDS Worker execution canceled: %v", ctx.Err())
		return nil, errors.New("CDS Worker execution canceled")
	case res = <-chanRes:
		if res.Status != sdk.StatusFail {
			res.Status = sdk.StatusSuccess
		}
	}
	return res, nil
}
