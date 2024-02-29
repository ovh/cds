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

type rtReleasePlugin struct {
	actionplugin.Common
}

func main() {
	p := rtReleasePlugin{}
	if err := actionplugin.Start(context.Background(), &p); err != nil {
		panic(err)
	}
}

func (p *rtReleasePlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "artifactoryRelease",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "Release artifacts.",
		Version:     sdk.VERSION,
	}, nil
}

// Run implements actionplugin.ActionPluginServer.
func (p *rtReleasePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}

	artifacts := q.GetOptions()["artifacts"]
	maturity := q.GetOptions()["maturity"]
	properties := q.GetOptions()["properties"]
	releaseNotes := q.GetOptions()["releaseNotes"]

	if err := p.perform(ctx, artifacts, maturity, properties, releaseNotes); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return res, err
	}

	return res, nil
}

func (p *rtReleasePlugin) perform(ctx context.Context, artifacts string, maturity string, properties string, releaseNotes string) (err error) {
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

	grpcplugins.Logf("Total number of artifacts that will be released: %d", len(results.RunResults))

	if err := artifactorypluginslib.ReleaseArtifactoryRunResult(ctx, &p.Common, results.RunResults, maturity, props, releaseNotes); err != nil {
		return err
	}

	return nil
}
