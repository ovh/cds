package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/utils/distribution"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/ovh/cds/contrib/grpcplugins"
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

func (p *artifactoryReleaseBundleDistributePlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}
	if err := p.perform(ctx, q); err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("%v", err)
	}
	return stream.Send(res)
}

func (p *artifactoryReleaseBundleDistributePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}
	if err := p.perform(ctx, q); err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("%v", err)
	}
	return res, nil
}

func (p *artifactoryReleaseBundleDistributePlugin) perform(ctx context.Context, q *actionplugin.ActionQuery) error {
	name := q.GetOptions()["name"]
	version := q.GetOptions()["version"]
	url := q.GetOptions()["url"]
	token := q.GetOptions()[q.GetOptions()["token_variable"]]

	if url == "" {
		artiURL := q.GetOptions()["cds.integration.artifact_manager.url"]
		artiToken := q.GetOptions()["cds.integration.artifact_manager.token"]

		token = q.GetOptions()["cds.integration.artifact_manager.release.token"]
		url = q.GetOptions()["cds.integration.artifact_manager.distribution.url"]

		jobContext, err := grpcplugins.GetJobContext(ctx, &p.Common)
		if err != nil {
			return err
		}
		if jobContext.Integrations != nil && jobContext.Integrations.ArtifactManager.Name != "" {
			integration := jobContext.Integrations.ArtifactManager

			artiURL = integration.Get(sdk.ArtifactoryConfigURL)
			artiToken = integration.Get(sdk.ArtifactoryConfigToken)

			token = integration.Get(sdk.ArtifactoryConfigReleaseToken)
			url = integration.Get(sdk.ArtifactoryConfigDistributionURL)
		}

		if url == "" {
			url = artiURL
			if url == "" {
				url = os.Getenv("CDS_INTEGRATION_ARTIFACT_MANAGER_URL")
			}
		}
		if token == "" {
			token = artiToken
			if token == "" {
				token = os.Getenv("CDS_INTEGRATION_ARTIFACT_MANAGER_TOKEN")
			}
		}
	}

	if url == "" {
		return fmt.Errorf("missing Artifactory URL")
	}
	if token == "" {
		return fmt.Errorf("missing Artifactory Distribution Token")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))
	distriClient, err := art.CreateDistributionClient(ctx, url, token)
	if err != nil {
		return fmt.Errorf("unable to create distribution client: %v", err)
	}

	fmt.Printf("Listing Edge nodes to distribute the release\n")
	edges, err := edge.ListEdgeNodes(distriClient)
	if err != nil {
		return err
	}

	if len(edges) == 0 {
		return fmt.Errorf("No destination available. Please check your credentials")
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
		return fmt.Errorf("unable to distribute version: %v", err)
	}
	return nil
}

func main() {
	actPlugin := artifactoryReleaseBundleDistributePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
