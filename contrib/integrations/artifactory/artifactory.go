package art

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/distribution"
	authdistrib "github.com/jfrog/jfrog-client-go/distribution/auth"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	"github.com/ovh/cds/sdk/telemetry"
)

type DistribClient struct {
	Dsm           *distribution.DistributionServicesManager
	ServiceConfig config.Config
}

func CreateDistributionClient(ctx context.Context, url, token string) (DistribClient, error) {
	dtb := authdistrib.NewDistributionDetails()
	dtb.SetUrl(strings.Replace(url, "/artifactory/", "/distribution/", -1))
	dtb.SetAccessToken(token)
	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(dtb).
		SetThreads(1).
		SetDryRun(false).
		SetContext(ctx).
		SetDialTimeout(60 * time.Second).
		SetOverallRequestTimeout(60 * time.Second).
		SetHttpRetries(5).
		Build()
	if err != nil {
		return DistribClient{}, fmt.Errorf("unable to create service config: %v", err)
	}
	dsm, err := distribution.New(serviceConfig)
	if err != nil {
		return DistribClient{}, nil
	}
	return DistribClient{Dsm: dsm, ServiceConfig: serviceConfig}, nil
}

func CreateArtifactoryClient(ctx context.Context, url, token string) (artifactory.ArtifactoryServicesManager, error) {
	rtDetails := auth.NewArtifactoryDetails()
	rtDetails.SetUrl(strings.TrimSuffix(url, "/") + "/") // ensure having '/' at the end
	rtDetails.SetAccessToken(token)
	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetThreads(1).
		SetDryRun(false).
		SetContext(ctx).
		SetDialTimeout(120 * time.Second).
		SetOverallRequestTimeout(120 * time.Second).
		SetHttpRetries(5).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create service config: %v", err)
	}
	return artifactory.New(serviceConfig)
}

type FileToPromote struct {
	RepoType string
	RepoName string
	Name     string
	Path     string
}

func PromoteFile(artiClient artifact_manager.ArtifactManager, data FileToPromote, lowMaturity, highMaturity string, props *utils.Properties, skipExistingArtifacts bool) (bool, error) {
	hasBeenPromoted := false
	// artifactory does not manage virtual cargo repositories
	var srcRepo, targetRepo string
	switch data.RepoType {
	case "cargo":
		repoParts := strings.Split(data.RepoName, "-")
		srcRepo = fmt.Sprintf("%s-%s", strings.Join(repoParts[:len(repoParts)-1], "-"), lowMaturity)
		targetRepo = fmt.Sprintf("%s-%s", strings.Join(repoParts[:len(repoParts)-1], "-"), highMaturity)
	default:
		srcRepo = fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
		targetRepo = fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	}

	params := services.NewMoveCopyParams()
	params.Pattern = fmt.Sprintf("%s/%s", srcRepo, data.Path)
	params.Target = fmt.Sprintf("%s/%s", targetRepo, data.Path)
	params.Flat = true

	if lowMaturity == highMaturity {
		fmt.Printf("%s has been already promoted\n", data.Name)
	} else {
		// Check if artifact already exist on destination
		var skipArtifact bool
		if skipExistingArtifacts {
			exist, err := artiClient.CheckArtifactExists(targetRepo, data.Path)
			if err != nil {
				return hasBeenPromoted, err
			}
			skipArtifact = exist
		}
		if !skipArtifact {
			// If source repository is a release repository, we should not move but copy the artifact
			// Get the properties of the source reposiytory
			maturity, err := artiClient.GetRepositoryMaturity(srcRepo)
			if err != nil {
				return hasBeenPromoted, fmt.Errorf("unable to get repository maturity: %v", err)
			}

			if maturity == "release" {
				fmt.Printf("Copying file %s from %s to %s\n", data.Name, srcRepo, targetRepo)
				nbSuccess, nbFailed, err := artiClient.Copy(params)
				if err != nil {
					return hasBeenPromoted, err
				}
				if nbFailed > 0 || nbSuccess == 0 {
					return hasBeenPromoted, fmt.Errorf("%s: copy failed with no reason", data.Name)
				}
			} else {
				fmt.Printf("Promoting file %s from %s to %s\n", data.Name, srcRepo, targetRepo)
				nbSuccess, nbFailed, err := artiClient.Move(params)
				if err != nil {
					return hasBeenPromoted, err
				}
				if nbFailed > 0 || nbSuccess == 0 {
					return hasBeenPromoted, fmt.Errorf("%s: copy failed with no reason", data.Name)
				}
			}
			hasBeenPromoted = true
		} else {
			fmt.Printf("%s already exists on destination repository\n", data.Name)
		}
	}

	if props != nil && hasBeenPromoted {
		fmt.Printf("Set properties %+v on file %s at %s\n", props, data.Name, targetRepo)
		if err := artiClient.SetProperties(targetRepo, data.Path, props); err != nil {
			return hasBeenPromoted, err
		}
	}

	return hasBeenPromoted, nil
}

func PromoteDockerImage(ctx context.Context, artiClient artifact_manager.ArtifactManager, data FileToPromote, lowMaturity, highMaturity string, props *utils.Properties, skipExistingArtifacts bool) (bool, error) {
	hasBeenPromoted := false
	sourceRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := services.NewDockerPromoteParams(data.Path, sourceRepo, targetRepo)

	if lowMaturity == highMaturity {
		fmt.Printf("%s has been already promoted\n", data.Name)
	} else {
		// Check if artifact already exist on destination
		var skipArtifact bool
		if skipExistingArtifacts {
			exist, err := artiClient.CheckArtifactExists(targetRepo, data.Path)
			if err != nil {
				return hasBeenPromoted, err
			}
			skipArtifact = exist
		}
		if !skipArtifact {
			maturity, err := artiClient.GetRepositoryMaturity(sourceRepo)
			if err != nil {
				fmt.Printf("Warning: unable to get repository maturity: %v\n", err)
			}

			if maturity == "release" {
				fmt.Printf("Copying docker image %s from %s to %s\n", data.Name, params.SourceRepo, params.TargetRepo)
				params.Copy = true
			} else {
				fmt.Printf("Promoting docker image %s from %s to %s\n", data.Name, params.SourceRepo, params.TargetRepo)
				params.Copy = false
			}
			if err := artiClient.PromoteDocker(params); err != nil {
				return hasBeenPromoted, err
			}
			hasBeenPromoted = true
		} else {
			fmt.Printf("%s already exists on destination repository\n", data.Name)
		}
	}

	if props != nil {
		if err := SetPropertiesRecursive(ctx, artiClient, data.RepoType, data.RepoName, highMaturity, data.Path, props); err != nil {
			return hasBeenPromoted, err
		}
	}

	return hasBeenPromoted, nil
}

type executionContext struct {
	buildInfoName            string
	projectKey               string
	workflowName             string
	version                  string
	defaultLowMaturitySuffix string
}

type BuildInfoRequest struct {
	BuildInfoPrefix          string
	ProjectKey               string
	VCS                      string
	Repository               string
	WorkflowName             string
	Version                  string
	AgentName                string
	TokenName                string
	RunURL                   string
	GitBranch                string
	GitMessage               string
	GitURL                   string
	GitHash                  string
	DefaultLowMaturitySuffix string
	RunResults               []sdk.WorkflowRunResult
	RunResultsV2             []sdk.V2WorkflowRunResult
}

func PrepareBuildInfo(ctx context.Context, artiClient artifact_manager.ArtifactManager, r BuildInfoRequest) (*buildinfo.BuildInfo, error) {
	var buildInfoName string
	if r.VCS != "" && r.Repository != "" {
		buildInfoName = fmt.Sprintf("%s/%s/%s/%s/%s", r.BuildInfoPrefix, r.ProjectKey, r.VCS, r.Repository, r.WorkflowName)
	} else {
		buildInfoName = fmt.Sprintf("%s/%s/%s", r.BuildInfoPrefix, r.ProjectKey, r.WorkflowName)
	}

	ctx, end := telemetry.Span(ctx, "artifactory.PrepareBuildInfo", telemetry.Tag("buildInfoName", buildInfoName))
	defer end()

	log.Debug(ctx, "PrepareBuildInfo %q maturity:%q", buildInfoName, r.DefaultLowMaturitySuffix)

	buildInfoRequest := &buildinfo.BuildInfo{
		Properties: map[string]string{},
		Name:       buildInfoName,
		Agent: &buildinfo.Agent{
			Name:    "artifactory-build-info-plugin",
			Version: sdk.VERSION,
		},
		BuildAgent: &buildinfo.Agent{
			Name:    r.AgentName,
			Version: sdk.VERSION,
		},
		Principal:     fmt.Sprintf("token:%s", r.TokenName),
		PluginVersion: sdk.VERSION,
		Started:       time.Now().Format("2006-01-02T15:04:05.999-07:00"),
		Number:        r.Version,
		BuildUrl:      r.RunURL,
		Modules:       []buildinfo.Module{},
		VcsList:       make([]buildinfo.Vcs, 0),
	}

	buildInfoRequest.VcsList = append(buildInfoRequest.VcsList, buildinfo.Vcs{
		Branch:   r.GitBranch,
		Message:  r.GitMessage,
		Url:      r.GitURL,
		Revision: r.GitHash,
	})

	execContext := executionContext{
		buildInfoName:            buildInfoName,
		defaultLowMaturitySuffix: r.DefaultLowMaturitySuffix,
		workflowName:             r.WorkflowName,
		version:                  r.Version,
		projectKey:               r.ProjectKey,
	}

	if len(r.RunResults) > 0 {
		modules, err := computeBuildInfoModules(ctx, artiClient, execContext, r.RunResults)
		if err != nil {
			return nil, err
		}
		buildInfoRequest.Modules = modules
	} else if len(r.RunResultsV2) > 0 {
		modules, err := computeBuildInfoModulesV2(ctx, artiClient, execContext, r.RunResultsV2)
		if err != nil {
			return nil, err
		}
		buildInfoRequest.Modules = modules
	}

	return buildInfoRequest, nil
}

func computeBuildInfoModules(ctx context.Context, artiClient artifact_manager.ArtifactManager, execContext executionContext, runResults []sdk.WorkflowRunResult) ([]buildinfo.Module, error) {
	ctx, end := telemetry.Span(ctx, "artifactory.computeBuildInfoModules")
	defer end()
	modules := make([]buildinfo.Module, 0)

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
			Artifacts:    make([]buildinfo.Artifact, 0),
			Dependencies: nil,
		}

		switch data.FileType {
		case "docker":
			mod.Type = buildinfo.Docker
			modProps := make(map[string]string)
			parsedUrl, err := url.Parse(artiClient.GetURL())
			if err != nil {
				return nil, sdk.WrapError(err, "unable to parse artifactory url [%s]", artiClient.GetURL())
			}
			urlArtifactory := parsedUrl.Host
			if parsedUrl.Port() != "" {
				urlArtifactory += ":" + parsedUrl.Port()
			}
			modProps["docker.image.tag"] = fmt.Sprintf("%s.%s/%s", data.RepoName, urlArtifactory, data.Name)
			mod.Properties = modProps

			query := fmt.Sprintf(`items.find({"name" : {"$match": "**"}, "repo":"%s", "path":"%s"}).include("repo","path","name","virtual_repos","actual_md5")`, data.RepoName, strings.TrimSuffix(data.Path, "/"))
			searchResults, err := artiClient.Search(ctx, query)
			if err != nil {
				return nil, err
			}

			for _, sr := range searchResults {
				currentArtifact := buildinfo.Artifact{
					Name: sr.Name,
					Type: strings.TrimPrefix(filepath.Ext(sr.Name), "."),
					Checksum: buildinfo.Checksum{
						Md5: sr.ActualMD5,
					},
				}
				mod.Artifacts = append(mod.Artifacts, currentArtifact)
			}
		default:
			_, objectName := filepath.Split(data.Path)
			currentArtifact := buildinfo.Artifact{
				Name: objectName,
				Type: strings.TrimPrefix(filepath.Ext(objectName), "."),
				Checksum: buildinfo.Checksum{
					Md5: data.MD5,
				},
			}
			mod.Artifacts = append(mod.Artifacts, currentArtifact)
		}
		modules = append(modules, mod)

		var currentMaturity string
		if r.DataSync != nil {
			latestPromotion := r.DataSync.LatestPromotionOrRelease()
			if latestPromotion != nil {
				currentMaturity = latestPromotion.ToMaturity
			}
		}
		if currentMaturity == "" {
			currentMaturity = execContext.defaultLowMaturitySuffix
		}
		props := utils.NewProperties()
		props.AddProperty("build.name", execContext.buildInfoName)
		props.AddProperty("build.number", execContext.version)
		props.AddProperty("build.timestamp", strconv.FormatInt(time.Now().Unix(), 10))

		if err := SetPropertiesRecursive(ctx, artiClient, data.RepoType, data.RepoName, currentMaturity, data.Path, props); err != nil {
			return nil, err
		}

	}
	return modules, nil
}

func computeBuildInfoModulesV2(ctx context.Context, artiClient artifact_manager.ArtifactManager, execContext executionContext, runResults []sdk.V2WorkflowRunResult) ([]buildinfo.Module, error) {
	ctx, end := telemetry.Span(ctx, "artifactory.computeBuildInfoModulesV2")
	defer end()
	modules := make([]buildinfo.Module, 0)

	for _, r := range runResults {
		if r.ArtifactManagerIntegrationName == nil {
			continue
		}

		var (
			repoName = r.ArtifactManagerMetadata.Get("repository")
			name     = r.ArtifactManagerMetadata.Get("name")
			path     = r.ArtifactManagerMetadata.Get("path")
			repoType = r.ArtifactManagerMetadata.Get("type")
			md5      = r.ArtifactManagerMetadata.Get("md5")
			dir      = r.ArtifactManagerMetadata.Get("dir")
		)

		mod := buildinfo.Module{
			Id:           fmt.Sprintf("%s:%s", repoType, name),
			Artifacts:    make([]buildinfo.Artifact, 0),
			Dependencies: nil,
		}

		var currentMaturity string
		if r.DataSync != nil {
			latestPromotion := r.DataSync.LatestPromotionOrRelease()
			if latestPromotion != nil {
				currentMaturity = latestPromotion.ToMaturity
			}
		}
		if currentMaturity == "" {
			currentMaturity = execContext.defaultLowMaturitySuffix
		}

		switch repoType {
		case "docker":
			mod.Type = buildinfo.Docker
			modProps := make(map[string]string)
			modProps["docker.image.tag"] = name
			mod.Properties = modProps

			path = strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/")

			query := fmt.Sprintf(`items.find({"name" : {"$match": "**"}, "repo":"%s", "path":"%s"}).include("repo","path","name","virtual_repos","actual_md5")`, repoName+"-"+currentMaturity, path)
			searchResults, err := artiClient.Search(ctx, query)
			if err != nil {
				return nil, err
			}

			for _, sr := range searchResults {
				currentArtifact := buildinfo.Artifact{
					Name: sr.Name,
					Type: strings.TrimPrefix(filepath.Ext(sr.Name), "."),
					Checksum: buildinfo.Checksum{
						Md5: sr.ActualMD5,
					},
				}
				mod.Artifacts = append(mod.Artifacts, currentArtifact)
			}
		default:
			_, objectName := filepath.Split(path)
			currentArtifact := buildinfo.Artifact{
				Name: objectName,
				Type: strings.TrimPrefix(filepath.Ext(objectName), "."),
				Checksum: buildinfo.Checksum{
					Md5: md5,
				},
			}
			mod.Artifacts = append(mod.Artifacts, currentArtifact)
		}
		modules = append(modules, mod)

		props := utils.NewProperties()
		props.AddProperty("build.name", execContext.buildInfoName)
		props.AddProperty("build.number", execContext.version)
		props.AddProperty("build.timestamp", strconv.FormatInt(time.Now().Unix(), 10))

		if err := SetPropertiesRecursive(ctx, artiClient, repoType, repoName, currentMaturity, path, props); err != nil {
			return nil, err
		}

	}
	return modules, nil
}

func SetPropertiesRecursive(ctx context.Context, client artifact_manager.ArtifactManager, repoType string, repoName string, maturity string, path string, props *utils.Properties) error {
	ctx, end := telemetry.Span(ctx, "artifactory.SetPropertiesRecursive")
	defer end()
	if props == nil {
		return nil
	}

	repoSrc := repoName
	if repoType == "cargo" {
		repoParts := strings.Split(repoName, "-")
		repoSrc = strings.Join(repoParts[:len(repoParts)-1], "-")
	}
	repoSrc += "-" + maturity

	log.Debug(ctx, "setting properties %+v on repoSrc:%s path:%s", props, repoSrc, path)
	_, endc := telemetry.Span(ctx, "artifactory.SetProperties", telemetry.Tag("repoSrc", repoSrc))
	if err := client.SetProperties(repoSrc, path, props); err != nil {
		endc()
		return err
	}
	endc()

	return nil
}
