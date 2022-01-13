package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/distribution"
	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	speccore "github.com/jfrog/jfrog-cli-core/v2/common/spec"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli/utils/cliutils"
	distributionServicesUtils "github.com/jfrog/jfrog-client-go/distribution/services/utils"

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

func (actPlugin *artifactoryReleaseBundleCreatePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-artifactory-release-bundle-create",
		Author:      "François Samin <francois.samin@corp.ovh.com>",
		Description: `This action creates and sign a release bundle from a specification file`,
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *artifactoryReleaseBundleCreatePlugin) PrepareSpecFiles(ctx context.Context, specification string) (*speccore.SpecFiles, error) {
	var content = []byte(specification)
	var specFiles speccore.SpecFiles
	if err := yaml.Unmarshal(content, &specFiles); err != nil {
		return nil, errors.Errorf("invalid given spec files: %v", err)
	}

	err := spec.ValidateSpec(specFiles.Files, true, true, false)
	if err != nil {
		return nil, err
	}

	return &specFiles, nil
}

func (actPlugin *artifactoryReleaseBundleCreatePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
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

	fmt.Printf("Preparing release bundle %q version %q\n", name, version)
	releaseBundleParams := distributionServicesUtils.NewReleaseBundleParams(name, version)
	releaseBundleParams.Description = description
	releaseBundleParams.ReleaseNotes = releaseNotes
	releaseBundleParams.ReleaseNotesSyntax = "markdown"
	releaseBundleParams.SignImmediately = true

	releaseBundleSpecs, err := actPlugin.PrepareSpecFiles(ctx, specification)
	if err != nil {
		return actionplugin.Fail(err.Error())
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
		fmt.Printf("Error: %v\n", err)
		return &actionplugin.ActionResult{
			Status: sdk.StatusFail,
		}, nil
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := artifactoryReleaseBundleCreatePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
