package artifactorypluginslib

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	"github.com/jfrog/jfrog-client-go/utils/distribution"
	jfroglog "github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/ovh/cds/contrib/grpcplugins"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	"github.com/ovh/cds/sdk/artifact_manager/artifactory/edge"
	"github.com/ovh/cds/sdk/artifact_manager/artifactory/xray"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/pkg/errors"
)

const (
	DefaultHighMaturity = "release"
)

type logger struct{}

func (*logger) Debug(a ...interface{}) {}
func (*logger) Error(a ...interface{}) {}
func (*logger) GetLogLevel() jfroglog.LevelType {
	return jfroglog.INFO
}
func (*logger) Info(a ...interface{})   {}
func (*logger) Output(a ...interface{}) {}
func (*logger) Warn(a ...interface{})   {}

var _ jfroglog.Log = new(logger)

func PromoteArtifactoryRunResult(ctx context.Context, c *actionplugin.Common, jobContext sdk.WorkflowRunJobsContext, r sdk.V2WorkflowRunResult, maturity string, props *utils.Properties) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	if jobContext.Integrations == nil && jobContext.Integrations.ArtifactManager.Name == "" {
		return errors.New("unable to find artifactory integration")
	}

	integration := jobContext.Integrations.ArtifactManager

	rtConfig := grpcplugins.ArtifactoryConfig{
		URL:   integration.Get(sdk.ArtifactoryConfigURL),
		Token: integration.Get(sdk.ArtifactoryConfigToken),
	}

	artifactClient, err := artifact_manager.NewClient("artifactory", rtConfig.URL, rtConfig.Token)
	if err != nil {
		return errors.Errorf("Failed to create artifactory client: %v", err)
	}

	jfroglog.SetLogger(new(logger)) // reset the logger set by artifact_manager.NewClient

	if r.DataSync == nil {
		r.DataSync = &sdk.WorkflowRunResultSync{}
	}

	latestPromotion := r.DataSync.LatestPromotionOrRelease()
	currentMaturity := integration.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
	if latestPromotion != nil {
		currentMaturity = latestPromotion.ToMaturity
	}

	if maturity == "" {
		maturity = integration.Get(sdk.ArtifactoryConfigPromotionHighMaturity)
	}

	newPromotion := sdk.WorkflowRunResultPromotion{
		Date:         time.Now(),
		FromMaturity: currentMaturity,
		ToMaturity:   maturity,
	}

	data := art.FileToPromote{
		RepoType: r.ArtifactManagerMetadata.Get("type"),
		RepoName: r.ArtifactManagerMetadata.Get("repository"),
		Name:     r.ArtifactManagerMetadata.Get("name"),
		Path:     strings.TrimPrefix(r.ArtifactManagerMetadata.Get("path"), "/"), // strip the first "/"
	}

	switch r.Type {
	case "docker":
		// remove "/manifest.json"
		data.Path = filepath.Dir(r.ArtifactManagerMetadata.Get("path")) // strip the first "/" and remove "/manifest.json"
		if err := art.PromoteDockerImage(ctx, artifactClient, data, newPromotion.FromMaturity, newPromotion.ToMaturity, props, false); err != nil {
			return errors.Errorf("unable to promote docker image: %s to %s: %v", data.Name, newPromotion.ToMaturity, err)
		}
	default:
		if err := art.PromoteFile(artifactClient, data, newPromotion.FromMaturity, newPromotion.ToMaturity, props, false); err != nil {
			return errors.Errorf("unable to promote file: %s: %v", data.Name, err)
		}
	}

	grpcplugins.Successf(c, "%s Successfully promoted to %s", r.Name(), newPromotion.ToMaturity)

	r.Status = sdk.V2WorkflowRunResultStatusPromoted
	r.DataSync.Promotions = append(r.DataSync.Promotions, newPromotion)

	// Update metadata
	r.ArtifactManagerMetadata.Set("localRepository", r.ArtifactManagerMetadata.Get("repository")+"-"+newPromotion.ToMaturity)
	r.ArtifactManagerMetadata.Set("maturity", newPromotion.ToMaturity)

	if _, err := grpcplugins.UpdateRunResult(ctx, c, &workerruntime.V2RunResultRequest{RunResult: &r}); err != nil {
		return err
	}

	return nil
}

func ReleaseArtifactoryRunResult(ctx context.Context, c *actionplugin.Common, results []sdk.V2WorkflowRunResult, maturity string, props *utils.Properties, releaseNotes string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	t0 := time.Now()

	var (
		promotedArtifacts []string
	)

	jobContext, err := grpcplugins.GetJobContext(ctx, c)
	if err != nil {
		return err
	}

	if jobContext.Integrations == nil || jobContext.Integrations.ArtifactManager.Name == "" {
		return errors.New("unable to find artifactory integration")
	}

	integration := jobContext.Integrations.ArtifactManager

	rtConfig := &grpcplugins.ArtifactoryConfig{
		URL:             integration.Get(sdk.ArtifactoryConfigURL),
		Token:           integration.Get(sdk.ArtifactoryConfigToken),
		DistributionURL: integration.Get(sdk.ArtifactoryConfigDistributionURL),
		ReleaseToken:    integration.Get(sdk.ArtifactoryConfigReleaseToken),
	}

	if rtConfig.DistributionURL == "" {
		grpcplugins.Logf(c, "Using %s to release\n", rtConfig.URL)
		rtConfig.DistributionURL = rtConfig.URL
	}
	if rtConfig.ReleaseToken == "" {
		grpcplugins.Log(c, "Using artifactory token to release")
		rtConfig.ReleaseToken = rtConfig.Token
	}

	if maturity == "" {
		maturity = integration.Get(sdk.ArtifactoryConfigPromotionHighMaturity)
	}

	artifactClient, err := artifact_manager.NewClient("artifactory", rtConfig.URL, rtConfig.Token)
	if err != nil {
		return errors.Errorf("Failed to create artifactory client: %v", err)
	}

	for i := range results {
		r := results[i]

		if r.DataSync == nil {
			r.DataSync = &sdk.WorkflowRunResultSync{}
		}

		latestPromotion := r.DataSync.LatestPromotionOrRelease()
		currentMaturity := integration.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
		if latestPromotion != nil {
			currentMaturity = latestPromotion.ToMaturity
		}

		newPromotion := sdk.WorkflowRunResultPromotion{
			Date:         time.Now(),
			FromMaturity: currentMaturity,
			ToMaturity:   maturity,
		}

		switch r.Type {
		case "docker":
			data := art.FileToPromote{
				RepoType: r.ArtifactManagerMetadata.Get("type"),
				RepoName: r.ArtifactManagerMetadata.Get("repository"),
				Name:     r.ArtifactManagerMetadata.Get("name"),
				Path:     strings.TrimPrefix(filepath.Dir(r.ArtifactManagerMetadata.Get("path")), "/"),
			}
			if err := art.PromoteDockerImage(ctx, artifactClient, data, newPromotion.FromMaturity, newPromotion.ToMaturity, props, true); err != nil {
				return errors.Errorf("unable to promote docker image: %s to %s: %v", data.Name, newPromotion.ToMaturity, err)
			}

			promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s%s", data.RepoName, newPromotion.ToMaturity, r.ArtifactManagerMetadata.Get("path")))
		default:
			data := art.FileToPromote{
				RepoType: r.ArtifactManagerMetadata.Get("type"),
				RepoName: r.ArtifactManagerMetadata.Get("repository"),
				Name:     r.ArtifactManagerMetadata.Get("name"),
				Path:     strings.TrimPrefix(r.ArtifactManagerMetadata.Get("path"), "/"),
			}
			if err := art.PromoteFile(artifactClient, data, newPromotion.FromMaturity, newPromotion.ToMaturity, props, true); err != nil {
				return errors.Errorf("unable to promote file: %s: %v", data.Name, err)
			}

			if data.RepoType == "cargo" { // artifactory does not manage virtual cargo repositories
				repoParts := strings.Split(data.RepoName, "-")
				promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s%s", strings.Join(repoParts[:len(repoParts)-1], "-"), newPromotion.ToMaturity, r.ArtifactManagerMetadata.Get("path")))
			} else {
				promotedArtifacts = append(promotedArtifacts, fmt.Sprintf("%s-%s%s", data.RepoName, newPromotion.ToMaturity, r.ArtifactManagerMetadata.Get("path")))
			}
		}

		r.Status = sdk.V2WorkflowRunResultStatusReleased
		r.DataSync.Releases = append(r.DataSync.Releases, newPromotion)

		// Update metadata
		r.ArtifactManagerMetadata.Set("localRepository", r.ArtifactManagerMetadata.Get("repository")+"-"+newPromotion.ToMaturity)
		r.ArtifactManagerMetadata.Set("maturity", newPromotion.ToMaturity)

		if _, err := grpcplugins.UpdateRunResult(ctx, c, &workerruntime.V2RunResultRequest{RunResult: &r}); err != nil {
			return err
		}
	}

	distriClient, err := art.CreateDistributionClient(ctx, rtConfig.DistributionURL, rtConfig.ReleaseToken)
	if err != nil {
		return errors.Errorf("Failed to create distribution client: %v", err)
	}

	xrayClient, err := xray.NewClient(strings.Replace(rtConfig.URL, "/artifactory/", "/xray", -1), rtConfig.Token)
	if err != nil {
		return errors.Errorf("Failed to create xray client: %v", err)
	}

	if len(promotedArtifacts) == 0 {
		return errors.Errorf("There is no artifact to release.")
	}

	grpcplugins.Success(c, "Promoted artifacts: ")
	for _, s := range promotedArtifacts {
		grpcplugins.Successf(c, "  * %s", s)
	}

	releaseVersion := strings.ReplaceAll(jobContext.Git.SemverCurrent, "+", "-")

	releaseName, releaseVersion, err := createReleaseBundle(ctx, c, distriClient,
		jobContext.CDS.ProjectKey, jobContext.CDS.Workflow, releaseVersion, integration.Get(sdk.ArtifactoryConfigBuildInfoPrefix),
		promotedArtifacts, maturity, releaseNotes)
	if err != nil {
		grpcplugins.Error(c, "Unable to create Release Bundle")
		return err
	}

	grpcplugins.Logf(c, "Listing Edge nodes to distribute the Release Bundle...")
	edges, err := edge.ListEdgeNodes(distriClient)
	if err != nil {
		grpcplugins.Error(c, "Unable to list release bundle")
		return err
	}

	grpcplugins.Logf(c, "Distributing Release Bundle %s %s...", releaseName, releaseVersion)

	distributionParams := distribution.NewDistributeReleaseBundleParams(releaseName, releaseVersion)
	distributionParams.DistributionRules = make([]*distribution.DistributionCommonParams, 0, len(edges))
	for _, e := range edges {
		grpcplugins.Logf(c, "  * Edge %s", e.Name)
		distributionParams.DistributionRules = append(distributionParams.DistributionRules, &distribution.DistributionCommonParams{
			SiteName:     e.SiteName,
			CityName:     e.City.Name,
			CountryCodes: []string{e.City.CountryCode},
		})
	}
	if err := distriClient.Dsm.DistributeReleaseBundleSync(distributionParams, 10, false); err != nil {
		return errors.Errorf("unable to distribute version: %v", err)
	}
	grpcplugins.Successf(c, "Release Bundle %s %s successfully distributed on all edges...", releaseName, releaseVersion)

	// Get the SBOM
	grpcplugins.Logf(c, "Getting SBOM...")
	var sbom json.RawMessage
	until := time.Now().Add(3 * time.Minute)
	for time.Now().Before(until) {
		sbom, err = xrayClient.GetReleaseBundleSBOMRaw(ctx, releaseName, releaseVersion)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return errors.Wrapf(err, "unable to get Release Bundle SBOM %s %s", releaseName, releaseVersion)
	}

	grpcplugins.Success(c, "SBOM successfully downloaded")

	_, err = grpcplugins.CreateRunResult(ctx, c, &workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: t0,
			Status:   sdk.V2WorkflowRunResultStatusCompleted,
			Type:     sdk.V2WorkflowRunResultTypeRelease,
			Detail: sdk.V2WorkflowRunResultDetail{
				Data: sdk.V2WorkflowRunResultReleaseDetail{
					Name:    releaseName,
					Version: releaseVersion,
					SBOM:    sbom,
				},
			},
			ArtifactManagerMetadata: &sdk.V2WorkflowRunResultArtifactManagerMetadata{
				"releaseName":    releaseName,
				"releaseVersion": releaseVersion,
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "unable to create run result for SBOM %s %s", releaseName, releaseVersion)
	}

	return nil
}

func createReleaseBundle(ctx context.Context, c *actionplugin.Common, distriClient art.DistribClient, projectKey, workflowName, version, buildInfo string, artifactPromoted []string, destMaturity, releaseNote string) (string, string, error) {
	buildInfoName := fmt.Sprintf("%s/%s/%s", buildInfo, projectKey, workflowName)

	params := services.NewCreateReleaseBundleParams(strings.Replace(buildInfoName, "/", "-", -1), version)
	if destMaturity != "" && destMaturity != DefaultHighMaturity {
		params.Version += "-" + destMaturity
	}

	exist, err := checkReleaseBundleExist(ctx, distriClient, params.Name, params.Version)
	if err != nil {
		return "", "", err
	}
	if !exist {
		if releaseNote == "" {
			releaseNote = "Release " + version
		}

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
		grpcplugins.Logf(c, "Creating Release Bundle %s %s...", params.Name, params.Version)

		if _, err := distriClient.Dsm.CreateReleaseBundle(params); err != nil {
			return "", "", fmt.Errorf("unable to create Release Bundle %s/%s: %v", params.Name, params.Version, err)
		}

		grpcplugins.Successf(c, "Release Bundle %s %s created", params.Name, params.Version)
	} else {
		grpcplugins.Logf(c, "Release Bundle %s/%s already exist\n", params.Name, params.Version)
	}
	return params.Name, params.Version, nil
}

func checkReleaseBundleExist(_ context.Context, client art.DistribClient, name string, version string) (bool, error) {
	getReleasePath := fmt.Sprintf("api/v1/release_bundle/%s/%s?format=json", name, version)

	fakeService := services.NewCreateReleaseBundleService(client.Dsm.Client())
	fakeService.DistDetails = client.ServiceConfig.GetServiceDetails()
	clientDetail := fakeService.DistDetails.CreateHttpClientDetails()
	getReleaseURL := fmt.Sprintf("%s%s", fakeService.DistDetails.GetUrl(), getReleasePath)
	utils.SetContentType("application/json", &clientDetail.Headers)

	resp, body, _, err := client.Dsm.Client().SendGet(getReleaseURL, true, &clientDetail)
	if err != nil {
		return false, fmt.Errorf("unable to get Release Bundle %s/%s from distribution (%s): %v", name, version, getReleaseURL, err)
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	if resp.StatusCode >= 400 {
		err := fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
		return false, fmt.Errorf("unable to get Release Bundle %s/%s from distribution (%s): %v", name, version, getReleaseURL, err)
	}
	return true, nil
}
