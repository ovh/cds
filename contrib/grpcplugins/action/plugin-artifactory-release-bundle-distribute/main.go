package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	distriUtils "github.com/jfrog/jfrog-client-go/distribution/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/ovh/cds/contrib/grpcplugins"
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
	client, err := art.CreateClient(ctx, url, token)
	if err != nil {
		return actionplugin.Fail("unable to create distribution client: %v", err)
	}

	fmt.Printf("Listing Edge nodes to distribute the release\n")
	edges, err := edge.ListEdgeNodes(*client.Dsm)
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

	if err := client.Dsm.DistributeReleaseBundleSync(distributionParams, 10, false); err != nil {
		return actionplugin.Fail("unable to distribute version: %v", err)
	}

	runResult, err := grpcplugins.GetRunResults(actPlugin.HTTPPort)
	if err != nil {
		return actionplugin.Fail("unable to list run results: %v", err)
	}

	fmt.Printf("Found %d run results\n", len(runResult))

	repoToReindex := make(map[string]string, 0)
	for _, r := range runResult {
		// static-file type does not need to be released
		if r.Type == sdk.WorkflowRunResultTypeStaticFile {
			continue
		}
		rData, err := r.GetArtifactManager()
		if err != nil {
			return actionplugin.Fail("unable to read result %s: %v", r.ID, err)
		}

		repoToReindex[fmt.Sprintf("%s-release", rData.RepoName)] = rData.RepoType
	}

	for k, v := range repoToReindex {
		// Filter on Helm repo
		if v == "helm" {
			fmt.Printf("%s reindex will start soon\n", k)
			if err := art.Reindex(*client.Asm, v, k); err != nil {
				return actionplugin.Fail("unable to reindex: %v", err)
			}
		}
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
