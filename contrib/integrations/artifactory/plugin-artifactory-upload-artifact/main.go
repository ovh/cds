package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

/*
This plugin have to be used as a upload artifact integration plugin

Artifactory upload artifact plugin must configured as following:
	name: artifactory-upload-artifact-plugin
	type: integration
	author: "Steven Guiheux"
	description: "OVH Artifactory Upload Artifact Plugin"

$ cdsctl admin plugins import artifactory-upload-artifact-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add artifactory-upload-artifact-plugin artifactory-upload-artifact-plugin-bin.yml <path-to-binary-file>

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

type artifactoryUploadArtifactPlugin struct {
	integrationplugin.Common
}

func (e *artifactoryUploadArtifactPlugin) Manifest(_ context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Artifactory Upload Artifact Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Artifactory Upload Artifact Plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *artifactoryUploadArtifactPlugin) Run(_ context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	prefix := "cds.integration.artifact_manager"
	cdsRepo := opts.GetOptions()[fmt.Sprintf("%s.%s", prefix, sdk.ArtifactoryConfigCdsRepository)]
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("%s.%s", prefix, sdk.ArtifactoryConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("%s.%s", prefix, sdk.ArtifactoryConfigToken)]
	pathToUpload := opts.GetOptions()[fmt.Sprintf("%s.upload.path", prefix)]
	projectKey := opts.GetOptions()["cds.project"]
	workflowName := opts.GetOptions()["cds.workflow"]
	version := opts.GetOptions()["cds.version"]
	buildInfo := opts.GetOptions()[fmt.Sprintf("%s.%s", prefix, sdk.ArtifactoryConfigBuildInfoPrefix)]

	artiClient, err := art.CreateArtifactoryClient(artifactoryURL, token)
	if err != nil {
		return fail("unable to create artifactory client: %v", err)
	}
	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))

	params := services.NewUploadParams()
	params.Pattern = pathToUpload
	params.Target = fmt.Sprintf("%s/%s/%s/%s/", cdsRepo, projectKey, workflowName, version)
	params.Flat = true
	params.BuildProps = fmt.Sprintf("build.name=%s/%s/%s;build.number=%s;build.timestamp=%d", buildInfo, projectKey, workflowName, url.QueryEscape(version), time.Now().Unix())
	params.Retries = 5

	summary, err := artiClient.UploadFilesWithSummary(params)
	if err != nil || summary.TotalFailed > 0 {
		return fail("unable to upload file %s into artifactory[%s] %s: %v", pathToUpload, artifactoryURL, params.Target, err)
	}
	defer summary.Close()

	result := make(map[string]string)
	for artDetails := new(utils.ArtifactDetails); summary.ArtifactsDetailsReader.NextRecord(artDetails) == nil; artDetails = new(utils.ArtifactDetails) {
		result[sdk.ArtifactUploadPluginOutputPathMD5] = artDetails.Checksums.Md5
		result[sdk.ArtifactUploadPluginOutputPathFilePath] = strings.TrimPrefix(artDetails.ToBuildInfoArtifact().Path, cdsRepo+"/")
		result[sdk.ArtifactUploadPluginOutputPathFileName] = artDetails.ToBuildInfoArtifact().Name
		result[sdk.ArtifactUploadPluginOutputPathRepoType] = "generic"
		result[sdk.ArtifactUploadPluginOutputPathRepoName] = cdsRepo
	}

	fileMode, err := os.Stat(pathToUpload)
	if err != nil {
		return fail("unable to get file stat: %v", err)
	}
	result[sdk.ArtifactUploadPluginOutputPerm] = strconv.FormatUint(uint64(fileMode.Mode().Perm()), 10)
	result[sdk.ArtifactUploadPluginOutputSize] = strconv.FormatInt(fileMode.Size(), 10)

	return &integrationplugin.RunResult{
		Status:  sdk.StatusSuccess,
		Outputs: result,
	}, nil
}

func main() {
	e := artifactoryUploadArtifactPlugin{}
	if err := integrationplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}

func fail(format string, args ...interface{}) (*integrationplugin.RunResult, error) {
	msg := fmt.Sprintf(format, args...)
	return &integrationplugin.RunResult{
		Details: msg,
		Status:  sdk.StatusFail,
	}, nil
}
