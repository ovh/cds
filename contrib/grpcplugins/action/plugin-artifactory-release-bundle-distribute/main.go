package main

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	distriUtils "github.com/jfrog/jfrog-client-go/distribution/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/contrib/integrations/artifactory/plugin-artifactory-release/edge"

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

func (actPlugin *artifactoryReleaseBundleDistributePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
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
		url = q.GetOptions()["cds.integration.artifact_manager.url"]
		token = q.GetOptions()["cds.integration.artifact_manager.release.token"]
	}

	if url == "" {
		return actionplugin.Fail("missing Artifactory URL")
	}
	if token == "" {
		return actionplugin.Fail("missing Artifactory Distribution Token")
	}

	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))
	distriClient, err := art.CreateDistributionClient(url, token)
	if err != nil {
		return actionplugin.Fail("unable to create distribution client: %v", err)
	}

	fmt.Printf("Listing Edge nodes to distribute the release\n")
	edges, err := edge.ListEdgeNodes(distriClient, url, token)
	if err != nil {
		return actionplugin.Fail("%v", err)
	}

	if len(edges) == 0 {
		return actionplugin.Fail("No destination available. Please check your credentials", err)
	}

	distributionParams := services.NewDistributeReleaseBundleParams(name, version)
	distributionParams.DistributionRules = make([]*distriUtils.DistributionCommonParams, 0, len(edges))
	for _, e := range edges {
		fmt.Printf("Distribute Release %s %s on %s\n", name, version, e.Name)
		distributionParams.DistributionRules = append(distributionParams.DistributionRules, &distriUtils.DistributionCommonParams{
			SiteName:     e.SiteName,
			CityName:     e.City.Name,
			CountryCodes: []string{e.City.CountryCode},
		})
	}

	if err := distriClient.DistributeReleaseBundleSync(distributionParams, 10); err != nil {
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
