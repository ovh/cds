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
		SetHttpTimeout(60 * time.Second).
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
		SetHttpTimeout(60 * time.Second).
		SetHttpRetries(5).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create service config: %v", err)
	}
	return artifactory.New(serviceConfig)
}

func PromoteFile(artiClient artifact_manager.ArtifactManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string, props *utils.Properties) error {
	srcRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := services.NewMoveCopyParams()
	params.Pattern = fmt.Sprintf("%s/%s", srcRepo, data.Path)
	params.Target = fmt.Sprintf("%s/%s", targetRepo, data.Path)
	params.Flat = true

	if lowMaturity == highMaturity {
		fmt.Printf("%s has been already promoted\n", data.Name)
	} else {
		// Check if artifact already exist on destination
		exist, err := artiClient.CheckArtifactExists(targetRepo, data.Path)
		if err != nil {
			return err
		}

		if !exist {
			// If source repository is a release repository, we should not move but copy the artifact
			// Get the properties of the source reposiytory
			maturity, err := artiClient.GetRepositoryMaturity(srcRepo)
			if err != nil {
				return fmt.Errorf("unable to get repository maturity: %v\n", err)
			}

			if maturity == "release" {
				fmt.Printf("Copying file %s from %s to %s\n", data.Name, srcRepo, targetRepo)
				nbSuccess, nbFailed, err := artiClient.Copy(params)
				if err != nil {
					return err
				}
				if nbFailed > 0 || nbSuccess == 0 {
					return fmt.Errorf("%s: copy failed with no reason", data.Name)
				}
			} else {
				fmt.Printf("Promoting file %s from %s to %s\n", data.Name, srcRepo, targetRepo)
				nbSuccess, nbFailed, err := artiClient.Move(params)
				if err != nil {
					return err
				}
				if nbFailed > 0 || nbSuccess == 0 {
					return fmt.Errorf("%s: copy failed with no reason", data.Name)
				}
			}
		} else {
			fmt.Printf("%s has been already promoted\n", data.Name)
		}
	}

	if props != nil {
		fmt.Printf("Set properties %+v on file %s at %s\n", props, data.Name, targetRepo)
		if err := artiClient.SetProperties(targetRepo, data.Path, props); err != nil {
			return err
		}
	}

	return nil
}

func PromoteDockerImage(ctx context.Context, artiClient artifact_manager.ArtifactManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string, props *utils.Properties) error {
	sourceRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := services.NewDockerPromoteParams(data.Path, sourceRepo, targetRepo)

	if lowMaturity == highMaturity {
		fmt.Printf("%s has been already promoted\n", data.Name)
	} else {
		maturity, err := artiClient.GetRepositoryMaturity(sourceRepo)
		if err != nil {
			fmt.Printf("Warning: unable to get repository maturity: %v\n", err)
		}

		// Check if artifact already exist on destination
		exist, err := artiClient.CheckArtifactExists(targetRepo, data.Path)
		if err != nil {
			return err
		}
		if !exist {
			if maturity == "release" {
				fmt.Printf("Copying docker image %s from %s to %s\n", data.Name, params.SourceRepo, params.TargetRepo)
				params.Copy = true
			} else {
				fmt.Printf("Promoting docker image %s from %s to %s\n", data.Name, params.SourceRepo, params.TargetRepo)
				params.Copy = false
			}
			if err := artiClient.PromoteDocker(params); err != nil {
				return err
			}
		} else {
			fmt.Printf("%s has been already promoted\n", data.Name)
		}
	}

	if props != nil {
		fmt.Printf("Set properties %+v on file %s at %s\n", props, data.Name, targetRepo)
		files, err := retrieveModulesFiles(ctx, artiClient, targetRepo, data.Path)
		if err != nil {
			return err
		}
		if err := SetPropertiesRecursive(ctx, artiClient, data.RepoName, highMaturity, files, props); err != nil {
			return err
		}
	}

	return nil
}

type executionContext struct {
	buildInfo                string
	projectKey               string
	workflowName             string
	version                  string
	defaultLowMaturitySuffix string
}

type BuildInfoRequest struct {
	BuildInfoPrefix          string
	ProjectKey               string
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
}

func PrepareBuildInfo(ctx context.Context, artiClient artifact_manager.ArtifactManager, r BuildInfoRequest) (*buildinfo.BuildInfo, error) {
	buildInfoName := fmt.Sprintf("%s/%s/%s", r.BuildInfoPrefix, r.ProjectKey, r.WorkflowName)
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
		buildInfo:                r.BuildInfoPrefix,
		defaultLowMaturitySuffix: r.DefaultLowMaturitySuffix,
		workflowName:             r.WorkflowName,
		version:                  r.Version,
		projectKey:               r.ProjectKey,
	}
	modules, err := computeBuildInfoModules(ctx, artiClient, execContext, r.RunResults)
	if err != nil {
		return nil, err
	}
	buildInfoRequest.Modules = modules

	return buildInfoRequest, nil
}

func computeBuildInfoModules(ctx context.Context, client artifact_manager.ArtifactManager, execContext executionContext, runResults []sdk.WorkflowRunResult) ([]buildinfo.Module, error) {
	ctx, end := telemetry.Span(ctx, "artifactory.computeBuildInfoModules")
	defer end()
	modules := make([]buildinfo.Module, 0)
	for _, r := range runResults {
		ctx, endc := telemetry.Span(ctx, "artifactory.PrepareBuildInfo", telemetry.Tag("runResult.Type", r.Type))
		if r.Type != sdk.WorkflowRunResultTypeArtifactManager {
			endc()
			continue
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

		data, err := r.GetArtifactManager()
		if err != nil {
			endc()
			return nil, err
		}
		var moduleExists bool
		mod := buildinfo.Module{
			Id:           fmt.Sprintf("%s:%s", data.RepoType, data.Name),
			Artifacts:    make([]buildinfo.Artifact, 0, len(runResults)),
			Dependencies: nil,
		}
		for _, m := range modules {
			if m.Id == mod.Id {
				moduleExists = true
				endc()
				break
			}
		}
		if moduleExists {
			endc()
			continue
		}
		switch data.RepoType {
		case "docker":
			mod.Type = buildinfo.Docker
			props := make(map[string]string)
			parsedUrl, err := url.Parse(client.GetURL())
			if err != nil {
				endc()
				return nil, sdk.WrapError(err, "unable to parse artifactory url [%s]: %v", client.GetURL())
			}
			urlArtifactory := parsedUrl.Host
			if parsedUrl.Port() != "" {
				urlArtifactory += ":" + parsedUrl.Port()
			}
			props["docker.image.tag"] = fmt.Sprintf("%s.%s/%s", data.RepoName, urlArtifactory, data.Name)
			mod.Properties = props
		}

		files, err := retrieveModulesFiles(ctx, client, data.RepoName, data.Path)
		if err != nil {
			endc()
			return nil, err
		}

		props := utils.NewProperties()
		props.AddProperty("build.name", fmt.Sprintf("%s/%s/%s", execContext.buildInfo, execContext.projectKey, execContext.workflowName))
		props.AddProperty("build.number", execContext.version)
		props.AddProperty("build.timestamp", strconv.FormatInt(time.Now().Unix(), 10))

		if err := SetPropertiesRecursive(ctx, client, data.RepoName, currentMaturity, files, props); err != nil {
			endc()
			return nil, err
		}

		artifacts, err := retrieveModulesArtifacts(ctx, client, files)
		if err != nil {
			endc()
			return nil, err
		}
		mod.Artifacts = artifacts
		modules = append(modules, mod)
	}

	return modules, nil
}

func retrieveModulesFiles(ctx context.Context, client artifact_manager.ArtifactManager, repoName string, path string) ([]sdk.FileInfo, error) {
	ctx, end := telemetry.Span(ctx, "workflow.retrieveModulesFiles")
	defer end()
	log.Debug(ctx, "retrieve:ModulesFiles repoName:%s path:%s", repoName, path)
	_, endc := telemetry.Span(ctx, "artifactoryClient.GetFileInfo", telemetry.Tag("path", path), telemetry.Tag("repoName", repoName))
	folderInfo, err := client.GetFolderInfo(repoName, path)
	endc()
	if err != nil {
		return nil, err
	}

	files := make([]sdk.FileInfo, 0)

	for _, c := range folderInfo.Children {
		if c.Folder {
			childrenFiles, err := retrieveModulesFiles(ctx, client, repoName, fmt.Sprintf("%s%s", path, c.Uri))
			if err != nil {
				return nil, err
			}
			files = append(files, childrenFiles...)
		} else {
			_, end := telemetry.Span(ctx, "artifactoryClient.GetFileInfo", telemetry.Tag("path", path), telemetry.Tag("uri", c.Uri))
			fileInfo, err := client.GetFileInfo(repoName, fmt.Sprintf("%s%s", path, c.Uri))
			end()
			if err != nil {
				return nil, err
			}
			files = append(files, fileInfo)
		}
	}

	return files, nil
}

func SetPropertiesRecursive(ctx context.Context, client artifact_manager.ArtifactManager, repoName string, maturity string, files []sdk.FileInfo, props *utils.Properties) error {
	ctx, end := telemetry.Span(ctx, "artifactory.SetPropertiesRecursive")
	defer end()
	if props == nil {
		return nil
	}
	for _, fileInfo := range files {
		repoSrc := repoName
		repoSrc += "-" + maturity
		log.Debug(ctx, "setting properties %+v on repoSrc:%s path:%s", props, repoSrc, fileInfo.Path)
		_, endc := telemetry.Span(ctx, "artifactory.SetProperties", telemetry.Tag("repoSrc", repoSrc))
		if err := client.SetProperties(repoSrc, fileInfo.Path, props); err != nil {
			endc()
			return err
		}
		endc()
	}
	return nil
}

func retrieveModulesArtifacts(ctx context.Context, client artifact_manager.ArtifactManager, files []sdk.FileInfo) ([]buildinfo.Artifact, error) {
	artifacts := make([]buildinfo.Artifact, 0)
	for _, fileInfo := range files {
		// If no children, it's a file, so we have checksum
		_, objectName := filepath.Split(fileInfo.Path)
		currentArtifact := buildinfo.Artifact{
			Name: objectName,
			Type: strings.TrimPrefix(filepath.Ext(objectName), "."),
			Checksum: buildinfo.Checksum{
				Md5: fileInfo.Checksums.Md5,
			},
		}
		artifacts = append(artifacts, currentArtifact)
	}
	return artifacts, nil
}
