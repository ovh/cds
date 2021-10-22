package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	"github.com/ovh/cds/contrib/grpcplugins"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
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

func (e *artifactoryBuildInfoPlugin) Run(_ context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken)]
	tokenName := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigTokenName)]
	lowMaturitySuffix := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigPromotionLowMaturity)]
	artifactoryProjectKey := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigProjectKey)]
	buildInfo := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigBuildInfoPrefix)]

	version := opts.GetOptions()["cds.version"]
	projectKey := opts.GetOptions()["cds.project"]
	workflowName := opts.GetOptions()["cds.workflow"]

	artiClient, err := art.CreateArtifactoryClient(artifactoryURL, token)
	if err != nil {
		return fail("unable to create artifactory client: %v", err)
	}
	log.SetLogger(log.NewLogger(log.ERROR, os.Stdout))

	buildInfoName := fmt.Sprintf("%s/%s/%s", buildInfo, projectKey, workflowName)

	// Check existing build info
	if err := e.deleteExistingBuild(artiClient, artifactoryProjectKey, buildInfoName, version); err != nil {
		return fail("unable to clean existing build: %v", err)
	}

	nodeRunURL := opts.GetOptions()["cds.ui.pipeline.run"]
	runURL := nodeRunURL[0:strings.Index(nodeRunURL, "/node/")]

	buildInfoRequest := &buildinfo.BuildInfo{
		Properties: map[string]string{},
		Name:       buildInfoName,
		Agent: &buildinfo.Agent{
			Name:    "artifactory-build-info-plugin",
			Version: sdk.VERSION,
		},
		BuildAgent: &buildinfo.Agent{
			Name:    "CDS",
			Version: sdk.VERSION,
		},
		ArtifactoryPrincipal:     fmt.Sprintf("token:%s", tokenName),
		ArtifactoryPluginVersion: sdk.VERSION,
		Started:                  time.Now().Format("2006-01-02T15:04:05.999-07:00"),
		Number:                   version,
		BuildUrl:                 runURL,
		Modules:                  []buildinfo.Module{},
		VcsList:                  make([]buildinfo.Vcs, 0),
	}
	buildInfoRequest.VcsList = append(buildInfoRequest.VcsList, buildinfo.Vcs{
		Branch:   opts.GetOptions()["git.branch"],
		Message:  opts.GetOptions()["git.message"],
		Url:      opts.GetOptions()["git.http_url"],
		Revision: opts.GetOptions()["git.hash"],
	})

	execContext := executionContext{
		buildInfo:         buildInfo,
		lowMaturitySuffix: lowMaturitySuffix,
		workflowName:      workflowName,
		version:           version,
		projectKey:        projectKey,
	}
	modules, err := e.computeBuildInfoModules(artiClient, execContext)
	if err != nil {
		return fail("unable to compute build info: %v", err)
	}
	buildInfoRequest.Modules = modules
	if _, err := artiClient.PublishBuildInfo(buildInfoRequest, artifactoryProjectKey); err != nil {
		return fail("unable to push build info: %v", err)
	}
	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

type DeleteBuildRequest struct {
	Project         string   `json:"project"`
	BuildName       string   `json:"buildName"`
	BuildNumbers    []string `json:"buildNumbers"`
	DeleteArtifacts bool     `json:"deleteArtifacts"`
	DeleteAll       bool     `json:"deleteAll"`
}

func (e *artifactoryBuildInfoPlugin) deleteExistingBuild(client artifactory.ArtifactoryServicesManager, artifactoryProjectKey string, buildName string, buildVersion string) error {
	httpDetails := client.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	utils.SetContentType("application/json", &httpDetails.Headers)
	request := DeleteBuildRequest{
		Project:      artifactoryProjectKey,
		BuildName:    buildName,
		BuildNumbers: []string{buildVersion},
	}
	bts, _ := json.Marshal(request)
	deleteBuildURL := fmt.Sprintf("%sapi/build/delete", client.GetConfig().GetServiceDetails().GetUrl())
	re, body, err := client.Client().SendPost(deleteBuildURL, bts, &httpDetails)
	if err != nil {
		return err
	}
	if re.StatusCode == http.StatusNotFound || re.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("unable to delete build: %s", string(body))
}

func (e *artifactoryBuildInfoPlugin) computeBuildInfoModules(client artifactory.ArtifactoryServicesManager, execContext executionContext) ([]buildinfo.Module, error) {
	modules := make([]buildinfo.Module, 0)
	runResults, err := grpcplugins.GetRunResults(e.HTTPPort)
	if err != nil {
		return nil, err
	}
	for _, r := range runResults {
		if r.Type != sdk.WorkflowRunResultTypeArtifactManager {
			continue
		}
		data, err := r.GetArtifactManager()
		if err != nil {
			return nil, err
		}

		mod := buildinfo.Module{
			Id:           fmt.Sprintf("%s:%s", data.RepoType, data.Name),
			Artifacts:    make([]buildinfo.Artifact, 0, len(runResults)),
			Dependencies: nil,
		}
		switch data.RepoType {
		case "docker":
			mod.Type = buildinfo.Docker
			props := make(map[string]string)
			parsedUrl, err := url.Parse(client.GetConfig().GetServiceDetails().GetUrl())
			if err != nil {
				return nil, fmt.Errorf("unable to parse artifactory url [%s]: %v", client.GetConfig().GetServiceDetails().GetUrl(), err)
			}
			urlArtifactory := parsedUrl.Host
			if parsedUrl.Port() != "" {
				urlArtifactory += ":" + parsedUrl.Port()
			}
			props["docker.image.tag"] = fmt.Sprintf("%s.%s/%s", data.RepoName, urlArtifactory, data.Name)
			mod.Properties = props
		}

		artifacts, err := e.retrieveModulesArtifacts(client, data.RepoName, data.Path, execContext)
		if err != nil {
			return nil, err
		}
		mod.Artifacts = artifacts
		modules = append(modules, mod)
	}

	return modules, nil
}

func (e *artifactoryBuildInfoPlugin) retrieveModulesArtifacts(client artifactory.ArtifactoryServicesManager, repoName string, path string, execContext executionContext) ([]buildinfo.Artifact, error) {
	fileInfo, err := art.GetFileInfo(client, repoName, path)
	if err != nil {
		return nil, err
	}
	artifacts := make([]buildinfo.Artifact, 0)

	// If no children, it's a file, so we have checksum
	_, objectName := filepath.Split(path)

	if len(fileInfo.Children) == 0 {
		props := make(map[string]string)
		props["build.name"] = fmt.Sprintf("%s/%s/%s", execContext.buildInfo, execContext.projectKey, execContext.workflowName)
		props["build.number"] = execContext.version
		props["build.timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
		repoSrc := repoName
		repoSrc += "-" + execContext.lowMaturitySuffix
		if err := art.SetProperties(client, repoSrc, path, props); err != nil {
			return nil, err
		}

		currentArtifact := buildinfo.Artifact{
			Name: objectName,
			Type: strings.TrimPrefix(filepath.Ext(objectName), "."),
			Checksum: &buildinfo.Checksum{
				Md5: fileInfo.Checksums.Md5,
			},
		}
		artifacts = append(artifacts, currentArtifact)
	} else {
		for _, c := range fileInfo.Children {
			artsChildren, err := e.retrieveModulesArtifacts(client, repoName, fmt.Sprintf("%s%s", path, c.Uri), execContext)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, artsChildren...)
		}
	}
	return artifacts, nil
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
