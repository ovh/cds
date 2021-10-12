package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

/*
This plugin have to be used as a download artifact integration plugin

Artifactory download artifact plugin must configured as following:
	name: artifactory-download-artifact-plugin
	type: integration
	author: "Steven Guiheux"
	description: "OVH Artifactory Upload Artifact Plugin"

$ cdsctl admin plugins import artifactory-download-artifact-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add artifactory-download-artifact-plugin artifactory-download-artifact-plugin-bin.yml <path-to-binary-file>

Artifactory integration must configured as following
	name: Artifactory
	default_config:
	  artifactory.url:
		type: string
	  artifactory.token:
		type: password
	  artifactory.cds_repository:
		type: string
	artifact_manager: true
*/

type artifactoryDownloadArtifactPlugin struct {
	integrationplugin.Common
}

func (e *artifactoryDownloadArtifactPlugin) Manifest(_ context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Artifactory Download Artifact Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Artifactory Download Artifact Plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *artifactoryDownloadArtifactPlugin) Run(_ context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	cdsRepo := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigCdsRepository)]
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken)]

	filePath := opts.GetOptions()[sdk.ArtifactDownloadPluginInputFilePath]
	path := opts.GetOptions()[sdk.ArtifactDownloadPluginInputDestinationPath]
	md5 := opts.GetOptions()[sdk.ArtifactDownloadPluginInputMd5]
	permS := opts.GetOptions()[sdk.ArtifactDownloadPluginInputPerm]

	perm, err := strconv.ParseUint(permS, 10, 32)
	if err != nil {
		return fail("unable to read file permission %s: %v", permS, err)
	}

	artiClient, err := art.CreateArtifactoryClient(artifactoryURL, token)
	if err != nil {
		return fail("unable to create artifactory client: %v", err)
	}
	log.SetLogger(log.NewLogger(log.ERROR, os.Stdout))
	fileutils.SetTempDirBase(opts.GetOptions()["cds.workspace"])

	params := services.NewDownloadParams()
	params.Pattern = fmt.Sprintf("%s/%s", cdsRepo, filePath)
	params.Target = path
	params.Flat = true
	params.Retries = 5

	summary, err := artiClient.DownloadFilesWithSummary(params)
	if err != nil || summary.TotalFailed > 0 {
		return fail("unable to download files %s from artifactory %s: %v", filePath, params.Target, err)
	}
	defer summary.Close()
	for artDetails := new(utils.ArtifactDetails); summary.ArtifactsDetailsReader.NextRecord(artDetails) == nil; artDetails = new(utils.ArtifactDetails) {
		if md5 != artDetails.Checksums.Md5 {
			return fail("wrong md5 for file %s. Got %s Want %s", filePath, artDetails.Checksums.Md5, md5)
		}
	}

	fileMode, err := os.Stat(path)
	if err != nil {
		return fail("unable to get file stat: %v", err)
	}
	currentperm := uint32(fileMode.Mode().Perm())
	if currentperm != uint32(perm) {
		if err := os.Chmod(path, os.FileMode(uint32(perm))); err != nil {
			return fail("unable to chmod file %s: %v", path, err)
		}
	}
	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	e := artifactoryDownloadArtifactPlugin{}
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
