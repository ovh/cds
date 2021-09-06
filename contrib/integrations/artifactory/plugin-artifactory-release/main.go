package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory"
	artService "github.com/jfrog/jfrog-client-go/artifactory/services"
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

type promotedArtifact struct {
	Pattern string
	Target  string
}

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
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigToken)]
	releaseToken := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigReleaseToken)]
	lowMaturitySuffix := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigPromotionLowMaturity)]
	highMaturitySuffix := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigPromotionHighMaturity)]

	buildInfo := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigBuildInfoPrefix)]
	if buildInfo == "" {
		buildInfo = opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactManagerConfigBuildInfoPath)]
	}

	version := opts.GetOptions()["cds.version"]
	projectKey := opts.GetOptions()["cds.project"]
	workflowName := opts.GetOptions()["cds.workflow"]

	artifactList := opts.GetOptions()["artifacts"]
	releaseNote := opts.GetOptions()["releaseNote"]
	releaseNameSuffix := opts.GetOptions()["releaseNameSuffix"]

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

	artifactPromoted := make([]promotedArtifact, 0)
	patternUsed := make(map[string]struct{})
	for _, r := range runResult {
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
			if err := e.promoteDockerImage(artiClient, rData, lowMaturitySuffix, highMaturitySuffix); err != nil {
				return fail("unable to promote docker image: %s: %v", rData.Name+"-"+highMaturitySuffix, err)
			}

			// Pattern must be like: "<repo_src>/<path>/(*)"
			// Target must be like: "<repo_target>/<path>/$1"
			pattern := fmt.Sprintf("%s/%s/(*)", rData.RepoName+"-"+highMaturitySuffix, rData.Path)
			if _, has := patternUsed[pattern]; !has {
				artifactPromoted = append(artifactPromoted, promotedArtifact{
					Pattern: pattern,
					Target:  fmt.Sprintf("%s/%s/{1}", rData.RepoName, rData.Path),
				})
				patternUsed[pattern] = struct{}{}
			}
		default:
			if err := e.promoteFile(artiClient, rData, lowMaturitySuffix, highMaturitySuffix); err != nil {
				return fail("unable to promote file: %s: %v", rData.Name, err)
			}
			dir, _ := filepath.Split(rData.Path)

			// Pattern must be like: "<repo_src>/<path>/(*)"
			// Target must be like: "<repo_target>/<path>/$1"
			pattern := fmt.Sprintf("%s/%s(*)", rData.RepoName+"-"+highMaturitySuffix, dir)
			if _, has := patternUsed[pattern]; !has {
				artifactPromoted = append(artifactPromoted, promotedArtifact{
					Pattern: pattern,
					Target:  fmt.Sprintf("%s/%s{1}", rData.RepoName, dir),
				})
				patternUsed[pattern] = struct{}{}
			}
		}

	}

	// Release bundle
	releaseName, releaseVersion, err := e.createReleaseBundle(distriClient, projectKey, workflowName, version, buildInfo, artifactList, releaseNameSuffix, releaseNote, artifactPromoted, artifactoryURL, releaseToken)
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

func (e *artifactoryReleasePlugin) createReleaseBundle(distriClient *distribution.DistributionServicesManager, projectKey, workflowName, version, buildInfo string, artifactList, releaseNameSuffix, releaseNote string, artifactPromoted []promotedArtifact, artifactoryURL, releaseToken string) (string, string, error) {
	buildInfoName := fmt.Sprintf("%s/%s/%s", buildInfo, projectKey, workflowName)

	params := services.NewCreateReleaseBundleParams(strings.Replace(buildInfoName, "/", "-", -1), version)
	if releaseNameSuffix != "" {
		params.Name += releaseNameSuffix
	}

	exist, err := e.checkReleaseBundleExist(distriClient, artifactoryURL, releaseToken, params.Name, params.Version)
	if err != nil {
		return "", "", err
	}
	if !exist {
		params.ReleaseNotes = releaseNote
		params.ReleaseNotesSyntax = "plain_text"

		paramsBuild := fmt.Sprintf("%s/%s", strings.Replace(buildInfoName, "/", "\\/", -1), version)
		if artifactList == "" {
			params.SpecFiles = []*utils.ArtifactoryCommonParams{
				{
					Recursive: true,
					Build:     paramsBuild,
				},
			}
		} else {
			params.SpecFiles = make([]*utils.ArtifactoryCommonParams, 0, len(artifactPromoted))
			for _, arti := range artifactPromoted {
				query := &utils.ArtifactoryCommonParams{
					Recursive: true,
					Build:     paramsBuild,
					Pattern:   arti.Pattern,
					Target:    arti.Target,
				}
				params.SpecFiles = append(params.SpecFiles, query)
			}
		}
		params.SignImmediately = true
		fmt.Printf("Creating release %s %s\n", params.Name, params.Version)

		if _, err := distriClient.CreateReleaseBundle(params); err != nil {
			return "", "", fmt.Errorf("unable to create release bundle %s/%s: %v", params.Name, params.Version, err)
		}
	} else {
		fmt.Printf("Release bundle %s/%s already exist", params.Name, params.Version)
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

func (e *artifactoryReleasePlugin) promoteFile(artiClient artifactory.ArtifactoryServicesManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string) error {
	srcRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := artService.NewMoveCopyParams()
	params.Pattern = fmt.Sprintf("%s/%s", srcRepo, data.Path)
	params.Target = fmt.Sprintf("%s/%s", targetRepo, data.Path)
	params.Flat = true

	// Check if artifact already exist on destination
	exist, err := e.checkArtifactExists(artiClient, targetRepo, data.Path)
	if err != nil {
		return err
	}
	if !exist {
		fmt.Printf("Promoting file %s from %s to %s\n", data.Name, srcRepo, targetRepo)
		nbSuccess, nbFailed, err := artiClient.Move(params)
		if err != nil {
			return err
		}
		if nbFailed > 0 || nbSuccess == 0 {
			return fmt.Errorf("%s: copy failed with no reason", data.Name)
		}
		return nil
	}
	fmt.Printf("%s has been already promoted", data.Name)
	return nil
}

func (e *artifactoryReleasePlugin) promoteDockerImage(artiClient artifactory.ArtifactoryServicesManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string) error {
	sourceRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := artService.NewDockerPromoteParams(data.Path, sourceRepo, targetRepo)
	params.Copy = false

	// Check if artifact already exist on destination
	exist, err := e.checkArtifactExists(artiClient, targetRepo, data.Path)
	if err != nil {
		return err
	}
	if !exist {
		fmt.Printf("Promoting docker image %s from %s to %s\n", data.Name, params.SourceRepo, params.TargetRepo)
		return artiClient.PromoteDocker(params)
	}
	fmt.Printf("%s has been already promoted", data.Name)
	return nil
}

func (e *artifactoryReleasePlugin) checkArtifactExists(artiClient artifactory.ArtifactoryServicesManager, repoName string, artiName string) (bool, error) {
	httpDetails := artiClient.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s", artiClient.GetConfig().GetServiceDetails().GetUrl(), repoName, artiName)
	re, body, _, err := artiClient.Client().SendGet(fileInfoURL, true, &httpDetails)
	if err != nil {
		return false, fmt.Errorf("unable to get file info %s/%s: %v", repoName, artiName, err)
	}
	if re.StatusCode == 404 {
		return false, nil
	}
	if re.StatusCode >= 400 {
		return false, fmt.Errorf("unable to call artifactory [HTTP: %d] %s %s", re.StatusCode, fileInfoURL, string(body))
	}
	return true, nil
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
