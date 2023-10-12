package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	"github.com/jfrog/jfrog-client-go/utils/distribution"
	"github.com/rockbears/log"

	"github.com/ovh/cds/contrib/grpcplugins"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/contrib/integrations/artifactory/plugin-artifactory-release/edge"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
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

func (e *artifactoryReleasePlugin) Manifest(_ context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Artifactory Release Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Artifactory Release Plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *artifactoryReleasePlugin) Run(ctx context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	log.Factory = log.NewStdWrapper(log.StdWrapperOptions{DisableTimestamp: true, Level: log.LevelInfo})
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine, log.FieldStackTrace)

	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigURL)]
	distributionURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigDistributionURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken)]
	releaseToken := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigReleaseToken)]

	buildInfo := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigBuildInfoPrefix)]

	version := opts.GetOptions()["cds.version"]
	projectKey := opts.GetOptions()["cds.project"]
	workflowName := opts.GetOptions()["cds.workflow"]

	artifactList := opts.GetOptions()["artifacts"]
	releaseNote := opts.GetOptions()["releaseNote"]
	destMaturity := opts.GetOptions()["destMaturity"]
	if destMaturity == "" {
		destMaturity = DefaultHighMaturity
	}

	var props *utils.Properties
	var err error
	setProperties := opts.GetOptions()["setProperties"]
	if setProperties != "" {
		props, err = utils.ParseProperties(setProperties)
		if err != nil {
			return fail("unable to parse given properties: %v", err)
		}
	}

	runResult, err := grpcplugins.GetRunResults(e.HTTPPort)
	if err != nil {
		return fail("unable to list run results: %v", err)
	}

	fmt.Printf("Found %d run results\n", len(runResult))

	if distributionURL == "" {
		fmt.Printf("Using %s to release\n", artifactoryURL)
		distributionURL = artifactoryURL
	}
	if releaseToken == "" {
		fmt.Println("Using artifactory token to release")
		releaseToken = token
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	distriClient, err := art.CreateDistributionClient(ctx, distributionURL, releaseToken)
	if err != nil {
		return fail("unable to create distribution client: %v", err)
	}

	artifactClient, err := artifact_manager.NewClient("artifactory", artifactoryURL, token)
	if err != nil {
		return fail("Failed to create artifactory client: %s", err)
	}

	artSplit := strings.Split(artifactList, ",")
	artRegs := make([]*regexp.Regexp, 0, len(artSplit))
	for _, arti := range artSplit {
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
		name, err := r.ComputeName()
		if err != nil {
			return fail("unable to read result %s: %v", r.ID, err)
		}
		if skip {
			fmt.Printf("Result %q skipped\n", name)
			continue
		} else {
			fmt.Printf("Result %q to promote\n", name)
		}
		if r.DataSync == nil {
			return fail("unable to find an existing release for result %s", r.ID)
		}
		latestPromotion := r.DataSync.LatestPromotionOrRelease()
		if latestPromotion == nil {
			return fail("unable to find an existing release for result %s", r.ID)
		}
		switch rData.RepoType {
		case "docker":
			if err := art.PromoteDockerImage(ctx, artifactClient, rData, latestPromotion.FromMaturity, latestPromotion.ToMaturity, props, true); err != nil {
				return fail("unable to promote docker image: %s: %v", rData.Name+"-"+latestPromotion.ToMaturity, err)
			}
			promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s/%s/manifest.json", rData.RepoName, latestPromotion.ToMaturity, rData.Path))
		default:
			if err := art.PromoteFile(artifactClient, rData, latestPromotion.FromMaturity, latestPromotion.ToMaturity, props, true); err != nil {
				return fail("unable to promote file: %s: %v", rData.Name, err)
			}
			// artifactory does not manage virtual cargo repositories
			if rData.RepoType == "cargo" {
				repoParts := strings.Split(rData.RepoName, "-")
				promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s/%s", strings.Join(repoParts[:len(repoParts)-1], "-"), latestPromotion.ToMaturity, rData.Path))
			} else {
				promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s/%s", rData.RepoName, latestPromotion.ToMaturity, rData.Path))
			}
		}
	}

	if len(promotedArtifacts) == 0 {
		return fail("There is no artifact to release.")
	}

	// Release bundle
	releaseName, releaseVersion, err := e.createReleaseBundle(distriClient, projectKey, workflowName, version, buildInfo, promotedArtifacts, destMaturity, releaseNote)
	if err != nil {
		return fail(err.Error())
	}

	fmt.Printf("Listing Edge nodes to distribute the release \n")
	edges, err := edge.ListEdgeNodes(distriClient)
	if err != nil {
		return fail("%v", err)
	}

	fmt.Printf("Distribute Release %s %s\n", releaseName, releaseVersion)

	distributionParams := distribution.NewDistributeReleaseBundleParams(releaseName, releaseVersion)
	distributionParams.DistributionRules = make([]*distribution.DistributionCommonParams, 0, len(edges))
	for _, e := range edges {
		fmt.Printf("Will be distributed to edge %s\n", e.Name)
		distributionParams.DistributionRules = append(distributionParams.DistributionRules, &distribution.DistributionCommonParams{
			SiteName:     e.SiteName,
			CityName:     e.City.Name,
			CountryCodes: []string{e.City.CountryCode},
		})
	}
	if err := distriClient.Dsm.DistributeReleaseBundleSync(distributionParams, 10, false); err != nil {
		return fail("unable to distribute version: %v", err)
	}

	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func (e *artifactoryReleasePlugin) createReleaseBundle(distriClient art.DistribClient, projectKey, workflowName, version, buildInfo string, artifactPromoted []string, destMaturity, releaseNote string) (string, string, error) {
	buildInfoName := fmt.Sprintf("%s/%s/%s", buildInfo, projectKey, workflowName)

	params := services.NewCreateReleaseBundleParams(strings.Replace(buildInfoName, "/", "-", -1), version)
	if destMaturity != "" && destMaturity != DefaultHighMaturity {
		params.Version += "-" + destMaturity
	}

	exist, err := e.checkReleaseBundleExist(distriClient, params.Name, params.Version)
	if err != nil {
		return "", "", err
	}
	if !exist {
		params.ReleaseNotes = releaseNote
		params.ReleaseNotesSyntax = "markdown"

		paramsBuild := fmt.Sprintf("%s/%s", strings.Replace(buildInfoName, "/", "\\/", -1), version)

		newReleaseProperties := utils.NewProperties()
		newReleaseProperties.AddProperty("release.name", params.Name)
		newReleaseProperties.AddProperty("release.version", params.Version)
		newReleaseProperties.AddProperty("release.timestamp", strconv.FormatInt(time.Now().Unix(), 10))

		params.SpecFiles = make([]*utils.CommonParams, 0, len(artifactPromoted))
		for _, arti := range artifactPromoted {
			query := &utils.CommonParams{
				Recursive:   true,
				Build:       paramsBuild,
				Pattern:     arti,
				TargetProps: newReleaseProperties,
			}
			params.SpecFiles = append(params.SpecFiles, query)
		}

		params.SignImmediately = true
		fmt.Printf("Creating release %s %s\n", params.Name, params.Version)

		if _, err := distriClient.Dsm.CreateReleaseBundle(params); err != nil {
			return "", "", fmt.Errorf("unable to create release bundle %s/%s: %v", params.Name, params.Version, err)
		}
	} else {
		fmt.Printf("Release bundle %s/%s already exist\n", params.Name, params.Version)
	}
	return params.Name, params.Version, nil
}

func (e *artifactoryReleasePlugin) checkReleaseBundleExist(client art.DistribClient, name string, version string) (bool, error) {
	getReleasePath := fmt.Sprintf("api/v1/release_bundle/%s/%s?format=json", name, version)

	fakeService := services.NewCreateReleaseBundleService(client.Dsm.Client())
	fakeService.DistDetails = client.ServiceConfig.GetServiceDetails()
	clientDetail := fakeService.DistDetails.CreateHttpClientDetails()
	getReleaseURL := fmt.Sprintf("%s%s", fakeService.DistDetails.GetUrl(), getReleasePath)
	utils.SetContentType("application/json", &clientDetail.Headers)

	resp, body, _, err := client.Dsm.Client().SendGet(getReleaseURL, true, &clientDetail)
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
