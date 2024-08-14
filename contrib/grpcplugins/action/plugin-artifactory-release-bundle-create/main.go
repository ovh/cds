package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/rockbears/yaml"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/distribution"
	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	speccore "github.com/jfrog/jfrog-cli-core/v2/common/spec"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli/utils/cliutils"
	distributionServicesUtils "github.com/jfrog/jfrog-client-go/distribution/services/utils"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build plugin-artifactory-release-bundle-create
$ make publish plugin-artifactory-release-bundle-create
*/

type artifactoryReleaseBundleCreatePlugin struct {
	actionplugin.Common
}

func (p *artifactoryReleaseBundleCreatePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-artifactory-release-bundle-create",
		Author:      "François Samin <francois.samin@corp.ovh.com>",
		Description: `This action creates and sign a release bundle from a specification file`,
		Version:     sdk.VERSION,
	}, nil
}

func (p *artifactoryReleaseBundleCreatePlugin) PrepareSpecFiles(ctx context.Context, specification string) (*speccore.SpecFiles, error) {
	var content = []byte(specification)
	var specFiles speccore.SpecFiles
	if err := yaml.Unmarshal(content, &specFiles); err != nil {
		return nil, errors.Wrapf(err, "invalid given spec files")
	}

	if err := spec.ValidateSpec(specFiles.Files, false, true, false); err != nil {
		return nil, err
	}

	return &specFiles, nil
}

func (p *artifactoryReleaseBundleCreatePlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
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

func (actPlugin *artifactoryReleaseBundleCreatePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}
	if err := actPlugin.perform(ctx, q); err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("%v", err)
	}
	return res, nil
}

func (p *artifactoryReleaseBundleCreatePlugin) perform(ctx context.Context, q *actionplugin.ActionQuery) error {
	name := q.GetOptions()["name"]
	version := q.GetOptions()["version"]
	description := q.GetOptions()["description"]
	releaseNotes := q.GetOptions()["release_notes"]
	specification := q.GetOptions()["specification"]
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

	fmt.Printf("Preparing release bundle %q version %q\n", name, version)
	releaseBundleParams := distributionServicesUtils.NewReleaseBundleParams(name, version)
	releaseBundleParams.Description = description
	releaseBundleParams.ReleaseNotes = releaseNotes
	releaseBundleParams.ReleaseNotesSyntax = "markdown"
	releaseBundleParams.SignImmediately = true

	releaseBundleSpecs, err := p.PrepareSpecFiles(ctx, specification)
	if err != nil {
		return err
	}

	rtDetails := new(config.ServerDetails)
	url = strings.TrimSuffix(url, "/") // ensure having '/' at the end
	if strings.HasSuffix(url, "/artifactory") {
		url = strings.TrimSuffix(url, "/artifactory")
		url += "/distribution"
	}
	url += "/"
	rtDetails.DistributionUrl = url
	rtDetails.AccessToken = token

	releaseBundleCreateCmd := distribution.NewReleaseBundleCreateCommand()
	releaseBundleCreateCmd.SetServerDetails(rtDetails).SetReleaseBundleCreateParams(releaseBundleParams).SetSpec(releaseBundleSpecs).SetDetailedSummary(true)

	err = commands.Exec(releaseBundleCreateCmd)

	if summary := releaseBundleCreateCmd.GetSummary(); summary != nil {
		fmt.Printf("Result from artifactory:\n")
		_ = cliutils.PrintBuildInfoSummaryReport(summary.IsSucceeded(), summary.GetSha256(), err)
	}

	if err != nil {
		return err
	}
	return nil
}

func main() {
	actPlugin := artifactoryReleaseBundleCreatePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
