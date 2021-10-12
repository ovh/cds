package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/distribution"
	authdistrib "github.com/jfrog/jfrog-client-go/distribution/auth"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	distriUtils "github.com/jfrog/jfrog-client-go/distribution/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	"github.com/ovh/cds/contrib/grpcplugins"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

/*
This plugin have to be used as a releaseplugin

Artifactory release plugin must configured as following:
	name: artifactory-release-plugin
	type: integration
	author: "Steven Guiheux"
	description: "OVH Artifactory Release Plugin"

$ cdsctl admin plugins import artifactory-release-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add artifactory-release-plugin artifactory-release-plugin-bin.yml <path-to-binary-file>
*/

const (
	DefaultHighMaturity = "release"
)

type artifactoryReleasePlugin struct {
	integrationplugin.Common
}

type EdgeNode struct {
	Name     string `json:"name"`
	SiteName string `json:"site_name"`
	City     struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
	} `json:"city"`
	LicenseType   string `json:"license_type"`
	LicenseStatus string `json:"license_status"`
}

func (e *artifactoryReleasePlugin) Manifest(_ context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Artifactory Release Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Artifactory Release Plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *artifactoryReleasePlugin) Run(_ context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken)]
	releaseToken := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigReleaseToken)]

	buildInfo := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigBuildInfoPrefix)]

	version := opts.GetOptions()["cds.version"]
	projectKey := opts.GetOptions()["cds.project"]
	workflowName := opts.GetOptions()["cds.workflow"]

	artifactList := opts.GetOptions()["artifacts"]
	releaseNote := opts.GetOptions()["releaseNote"]
	srcMaturity := opts.GetOptions()["srcMaturity"]
	destMaturity := opts.GetOptions()["destMaturity"]

	if srcMaturity == "" {
		srcMaturity = "snapshot"
	}
	if destMaturity == "" {
		destMaturity = DefaultHighMaturity
	}

	runResult, err := grpcplugins.GetRunResults(e.HTTPPort)
	if err != nil {
		return fail("unable to list run results: %v", err)
	}

	log.SetLogger(log.NewLogger(log.ERROR, os.Stdout))
	distriClient, err := art.CreateDistributionClient(artifactoryURL, releaseToken)
	if err != nil {
		return fail("unable to create distribution client: %v", err)
	}

	// Promotion
	artiClient, err := art.CreateArtifactoryClient(artifactoryURL, token)
	if err != nil {
		return fail("unable to create artifactory client: %v", err)
	}

	artSplitted := strings.Split(artifactList, ",")
	artRegs := make([]*regexp.Regexp, 0, len(artSplitted))
	for _, arti := range artSplitted {
		r, err := regexp.Compile(arti)
		if err != nil {
			return fail("unable compile regexp in artifact list: %v", err)
		}
		artRegs = append(artRegs, r)
	}

	promotedArtifacts := make([]string, 0)
	for _, r := range runResult {
		// static-file type does not need to be released
		if r.Type == sdk.WorkflowRunResultTypeStaticFile {
			continue
		}
		rData, err := r.GetArtifactManager()
		if err != nil {
			return fail("unable to read result %s: %v", r.ID, err)
		}
		skip := true
		for _, reg := range artRegs {
			if reg.MatchString(rData.Name) {
				skip = false
				break
			}
		}
		if skip {
			continue
		}
		switch rData.RepoType {
		case "docker":
			if err := art.PromoteDockerImage(artiClient, rData, srcMaturity, destMaturity); err != nil {
				return fail("unable to promote docker image: %s: %v", rData.Name+"-"+destMaturity, err)
			}
			promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s/%s/manifest.json", rData.RepoName, destMaturity, rData.Path))
		default:
			if err := art.PromoteFile(artiClient, rData, srcMaturity, destMaturity); err != nil {
				return fail("unable to promote file: %s: %v", rData.Name, err)
			}
			promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s/%s", rData.RepoName, destMaturity, rData.Path))
		}

	}

	// Release bundle
	releaseName, releaseVersion, err := e.createReleaseBundle(distriClient, projectKey, workflowName, version, buildInfo, promotedArtifacts, destMaturity, releaseNote, artifactoryURL, releaseToken)
	if err != nil {
		return fail(err.Error())
	}

	fmt.Printf("Listing Edge nodes to distribute the release \n")
	edges, err := e.listEdgeNodes(distriClient, artifactoryURL, releaseToken)
	if err != nil {
		return fail("%v", err)
	}
	edges = e.removeNonEdge(edges)

	fmt.Printf("Distribute Release %s %s\n", releaseName, releaseVersion)
	distributionParams := services.NewDistributeReleaseBundleParams(releaseName, releaseVersion)
	distributionParams.DistributionRules = make([]*distriUtils.DistributionCommonParams, 0, len(edges))
	for _, e := range edges {
		distributionParams.DistributionRules = append(distributionParams.DistributionRules, &distriUtils.DistributionCommonParams{
			SiteName:     e.SiteName,
			CityName:     e.City.Name,
			CountryCodes: []string{e.City.CountryCode},
		})
	}
	if err := distriClient.DistributeReleaseBundleSync(distributionParams, 10); err != nil {
		return fail("unable to distribute version: %v", err)
	}

	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func (e *artifactoryReleasePlugin) createReleaseBundle(distriClient *distribution.DistributionServicesManager, projectKey, workflowName, version, buildInfo string, artifactPromoted []string, destMaturity, releaseNote string, artifactoryURL, releaseToken string) (string, string, error) {
	buildInfoName := fmt.Sprintf("%s/%s/%s", buildInfo, projectKey, workflowName)

	params := services.NewCreateReleaseBundleParams(strings.Replace(buildInfoName, "/", "-", -1), version)
	if destMaturity != "" && destMaturity != DefaultHighMaturity {
		params.Version += "-" + destMaturity
	}

	exist, err := e.checkReleaseBundleExist(distriClient, artifactoryURL, releaseToken, params.Name, params.Version)
	if err != nil {
		return "", "", err
	}
	if !exist {
		params.ReleaseNotes = releaseNote
		params.ReleaseNotesSyntax = "plain_text"

		paramsBuild := fmt.Sprintf("%s/%s", strings.Replace(buildInfoName, "/", "\\/", -1), version)

		params.SpecFiles = make([]*utils.ArtifactoryCommonParams, 0, len(artifactPromoted))
		for _, arti := range artifactPromoted {
			query := &utils.ArtifactoryCommonParams{
				Recursive: true,
				Build:     paramsBuild,
				Pattern:   arti,
			}
			params.SpecFiles = append(params.SpecFiles, query)
		}

		params.SignImmediately = true
		fmt.Printf("Creating release %s %s\n", params.Name, params.Version)

		if _, err := distriClient.CreateReleaseBundle(params); err != nil {
			return "", "", fmt.Errorf("unable to create release bundle %s/%s: %v", params.Name, params.Version, err)
		}
	} else {
		fmt.Printf("Release bundle %s/%s already exist\n", params.Name, params.Version)
	}
	return params.Name, params.Version, nil
}

func (e *artifactoryReleasePlugin) listEdgeNodes(distriClient *distribution.DistributionServicesManager, url, token string) ([]EdgeNode, error) {
	// action=x distribute
	listEdgeNodePath := fmt.Sprintf("api/ui/distribution/edge_nodes?action=x")
	dtb := authdistrib.NewDistributionDetails()
	dtb.SetUrl(strings.Replace(url, "/artifactory/", "/distribution/", -1))
	dtb.SetAccessToken(token)

	fakeService := services.NewCreateReleaseBundleService(distriClient.Client())
	fakeService.DistDetails = dtb
	clientDetail := fakeService.DistDetails.CreateHttpClientDetails()
	listEdgeURL := fmt.Sprintf("%s%s", fakeService.DistDetails.GetUrl(), listEdgeNodePath)
	utils.SetContentType("application/json", &clientDetail.Headers)

	resp, body, _, err := distriClient.Client().SendGet(listEdgeURL, true, &clientDetail)
	if err != nil {
		return nil, fmt.Errorf("unable to list edge node from distribution: %v", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	var edges []EdgeNode
	if err := sdk.JSONUnmarshal(body, &edges); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response %s: %v", string(body), err)
	}
	return edges, nil
}

func (e *artifactoryReleasePlugin) removeNonEdge(edges []EdgeNode) []EdgeNode {
	edgeFiltered := make([]EdgeNode, 0, len(edges))
	for _, e := range edges {
		if e.LicenseType != "EDGE" {
			continue
		}
		edgeFiltered = append(edgeFiltered, e)
	}
	return edgeFiltered
}

func (e *artifactoryReleasePlugin) checkReleaseBundleExist(client *distribution.DistributionServicesManager, url string, token string, name string, version string) (bool, error) {
	getReleasePath := fmt.Sprintf("api/v1/release_bundle/%s/%s?format=json", name, version)
	dtb := authdistrib.NewDistributionDetails()
	dtb.SetUrl(strings.Replace(url, "/artifactory/", "/distribution/", -1))
	dtb.SetAccessToken(token)

	fakeService := services.NewCreateReleaseBundleService(client.Client())
	fakeService.DistDetails = dtb
	clientDetail := fakeService.DistDetails.CreateHttpClientDetails()
	getReleaseURL := fmt.Sprintf("%s%s", fakeService.DistDetails.GetUrl(), getReleasePath)
	utils.SetContentType("application/json", &clientDetail.Headers)

	resp, body, _, err := client.Client().SendGet(getReleaseURL, true, &clientDetail)
	if err != nil {
		return false, fmt.Errorf("unable to get release bundle %s/%s from distribution: %v", name, version, err)
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}
	return true, nil
}

func main() {
	e := artifactoryReleasePlugin{}
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
