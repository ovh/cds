package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/utils/distribution"
	"github.com/jfrog/jfrog-client-go/utils/log"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/sdk/artifact_manager/artifactory/edge"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build plugin-artifactory-release-bundle-distribute
$ make publish plugin-artifactory-release-bundle-distribute
*/

type artifactoryReleaseBundleDistributePlugin struct {
	actionplugin.Common
}

func (actPlugin *artifactoryReleaseBundleDistributePlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-artifactory-release-bundle-distribute",
		Author:      "François Samin <francois.samin@corp.ovh.com>",
		Description: `This action distributes a release bundle on all the JFrog Platform`,
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *artifactoryReleaseBundleDistributePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	name := q.GetOptions()["name"]
	version := q.GetOptions()["version"]
	url := q.GetOptions()["url"]

	token := q.GetOptions()[q.GetOptions()["token_variable"]]

	if url == "" {
		artiURL := q.GetOptions()["cds.integration.artifact_manager.url"]
		artiToken := q.GetOptions()["cds.integration.artifact_manager.token"]

		token = q.GetOptions()["cds.integration.artifact_manager.release.token"]
		url = q.GetOptions()["cds.integration.artifact_manager.distribution.url"]

		if url == "" {
			url = artiURL
		}
		if token == "" {
			token = artiToken
		}
	}

	if url == "" {
		return actionplugin.Fail("missing Artifactory URL")
	}
	if token == "" {
		return actionplugin.Fail("missing Artifactory Distribution Token")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))
	distriClient, err := art.CreateDistributionClient(ctx, url, token)
	if err != nil {
		return actionplugin.Fail("unable to create distribution client: %v", err)
	}

	fmt.Printf("Listing Edge nodes to distribute the release\n")
	edges, err := edge.ListEdgeNodes(distriClient)
	if err != nil {
		return actionplugin.Fail("%v", err)
	}

	if len(edges) == 0 {
		return actionplugin.Fail("No destination available. Please check your credentials", err)
	}

	distributionParams := distribution.NewDistributeReleaseBundleParams(name, version)
	distributionParams.DistributionRules = make([]*distribution.DistributionCommonParams, 0, len(edges))
	for _, e := range edges {
		fmt.Printf("Distribute Release %s %s on %s\n", name, version, e.Name)
		distributionParams.DistributionRules = append(distributionParams.DistributionRules, &distribution.DistributionCommonParams{
			SiteName:     e.SiteName,
			CityName:     e.City.Name,
			CountryCodes: []string{e.City.CountryCode},
		})
	}

	if err := distriClient.Dsm.DistributeReleaseBundleSync(distributionParams, 10, false); err != nil {
		return actionplugin.Fail("unable to distribute version: %v", err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := artifactoryReleaseBundleDistributePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
