package main

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/ovh/cds/contrib/grpcplugins"
	artifactorypluginslib "github.com/ovh/cds/contrib/grpcplugins/action/artifactory-plugins-lib"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/pkg/errors"
)

type rtPromotePlugin struct {
	actionplugin.Common
}

func main() {
	p := rtPromotePlugin{}
	if err := actionplugin.Start(context.Background(), &p); err != nil {
		panic(err)
	}
}

func (p *rtPromotePlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "artifactoryPromote",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "Promote artifacts.",
		Version:     sdk.VERSION,
	}, nil
}

func (p *rtPromotePlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	artifacts := q.GetOptions()["artifacts"]
	maturity := q.GetOptions()["maturity"]
	properties := q.GetOptions()["properties"]

	if err := p.perform(ctx, artifacts, maturity, properties); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
	}
	return stream.Send(res)
}

// Run implements actionplugin.ActionPluginServer.
func (p *rtPromotePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (p *rtPromotePlugin) perform(ctx context.Context, artifacts string, maturity string, properties string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			fmt.Println(string(debug.Stack()))
			err = errors.Errorf("Internal server error: panic")
		}
	}()

	results, err := grpcplugins.GetArtifactoryRunResults(ctx, &p.Common, artifacts)
	if err != nil {
		return err
	}

	if len(results.RunResults) == 0 {
		return errors.Errorf("no artifacts match %q", artifacts)
	}

	var props *utils.Properties
	if properties != "" {
		var err error
		props, err = utils.ParseProperties(properties)
		if err != nil {
			return errors.Errorf("unable to parse given properties: %v", err)
		}
	}

	grpcplugins.Logf(&p.Common, "Total number of artifacts that will be promoted: %d", len(results.RunResults))

	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		return errors.Errorf("unable to parse given properties: %v", err)
	}

	for _, r := range results.RunResults {
		if err := artifactorypluginslib.PromoteArtifactoryRunResult(ctx, &p.Common, *jobCtx, r, maturity, props); err != nil {
			return err
		}
	}

	return nil
}
