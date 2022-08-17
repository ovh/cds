package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory/services"

	"github.com/ovh/cds/contrib/grpcplugins"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/engine/api/integration/artifact_manager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

/*
This plugin have to be used as a build info plugin

Artifactory build info plugin must configured as following:
	name: artifactory-build-info-plugin
	type: integration
	author: "Steven Guiheux"
	description: "OVH Artifactory Build Info Plugin"

$ cdsctl admin plugins import artifactory-build-info-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add artifactory-build-info-plugin artifactory-build-info-plugin-bin.yml <path-to-binary-file>
*/

type artifactoryBuildInfoPlugin struct {
	integrationplugin.Common
}

type executionContext struct {
	buildInfo         string
	projectKey        string
	workflowName      string
	version           string
	lowMaturitySuffix string
}

func (e *artifactoryBuildInfoPlugin) Manifest(_ context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Artifactory Build Info Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Artifactory Build Info Plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *artifactoryBuildInfoPlugin) Run(ctx context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken)]
	tokenName := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigTokenName)]
	lowMaturitySuffix := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigPromotionLowMaturity)]
	artifactoryProjectKey := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigProjectKey)]
	buildInfo := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigBuildInfoPrefix)]

	artifactClient, err := artifact_manager.NewClient("artifactory", artifactoryURL, token)
	if err != nil {
		return fail("Failed to create artifactory client: %s", err)
	}

	nodeRunURL := opts.GetOptions()["cds.ui.pipeline.run"]
	runURL := nodeRunURL[0:strings.Index(nodeRunURL, "/node/")]

	// Get the build agent from env variable set by worker
	workerName := os.Getenv("CDS_WORKER")
	if workerName == "" {
		workerName = "CDS"
	}

	// Compute git url
	gitUrl := opts.GetOptions()["git.url"]
	if gitUrl == "" {
		gitUrl = opts.GetOptions()["git.http_url"]
	}

	// Get run results
	runResults, err := grpcplugins.GetRunResults(e.HTTPPort)
	if err != nil {
		return fail("unable to get run results: %v", err)
	}

	buildInfoRequest, err := art.PrepareBuildInfo(ctx, artifactClient, art.BuildInfoRequest{
		BuildInfoPrefix:   buildInfo,
		ProjectKey:        opts.GetOptions()["cds.project"],
		WorkflowName:      opts.GetOptions()["cds.workflow"],
		Version:           opts.GetOptions()["cds.version"],
		AgentName:         workerName,
		TokenName:         tokenName,
		RunURL:            runURL,
		GitBranch:         opts.GetOptions()["git.branch"],
		GitMessage:        opts.GetOptions()["git.message"],
		GitURL:            gitUrl,
		GitHash:           opts.GetOptions()["git.hash"],
		RunResults:        runResults,
		LowMaturitySuffix: lowMaturitySuffix,
	})
	if err != nil {
		return fail("unable to prepare build info: %v", err)
	}
	fmt.Printf("Creating Artifactory Build %s %s on project %s...\n", buildInfoRequest.Name, buildInfoRequest.Number, artifactoryProjectKey)

	if err := artifactClient.DeleteBuild(artifactoryProjectKey, buildInfoRequest.Name, buildInfoRequest.Number); err != nil {
		return fail("unable to clean existing build: %v", err)
	}

	var nbAttempts int
	for {
		nbAttempts++
		err := artifactClient.PublishBuildInfo(artifactoryProjectKey, buildInfoRequest)
		if err == nil {
			break
		} else if nbAttempts >= 3 {
			return fail("unable to push build info: %v", err)
		} else {
			fmt.Printf("Error while pushing buildinfo %s %s. Retrying...\n", buildInfoRequest.Name, buildInfoRequest.Number)
		}
	}

	// Temporary code
	if opts.GetOptions()["cds.proj.xray.enabled"] == "true" {
		fmt.Printf("Triggering XRay Build %s %s scan...\n", buildInfoRequest.Name, buildInfoRequest.Number)

		// Scan build info
		scanBuildRequest := services.NewXrayScanParams()
		scanBuildRequest.BuildName = buildInfoRequest.Name
		scanBuildRequest.BuildNumber = buildInfoRequest.Number
		scanBuildRequest.ProjectKey = artifactoryProjectKey
		scanBuildResponseBtes, err := artifactClient.XrayScanBuild(scanBuildRequest)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(scanBuildResponseBtes))
		}
	}

	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	e := artifactoryBuildInfoPlugin{}
	if err := integrationplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}

func fail(format string, args ...interface{}) (*integrationplugin.RunResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &integrationplugin.RunResult{
		Details: msg,
		Status:  sdk.StatusFail,
	}, nil
}
