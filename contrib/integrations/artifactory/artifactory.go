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

func PromoteFile(artiClient artifact_manager.ArtifactManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string, props *utils.Properties, skipExistingArtifacts bool) error {
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
		var skipArtifact bool
		if skipExistingArtifacts {
			exist, err := artiClient.CheckArtifactExists(targetRepo, data.Path)
			if err != nil {
				return err
			}
			skipArtifact = exist
		}
		if !skipArtifact {
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

func PromoteDockerImage(ctx context.Context, artiClient artifact_manager.ArtifactManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string, props *utils.Properties, skipExistingArtifacts bool) error {
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
				return err
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

func computeBuildInfoModules(ctx context.Context, artiClient artifact_manager.ArtifactManager, execContext executionContext, runResults []sdk.WorkflowRunResult) ([]buildinfo.Module, error) {
	ctx, end := telemetry.Span(ctx, "artifactory.computeBuildInfoModules")
	defer end()
	modules := make([]buildinfo.Module, 0)

	runResultDatas := make([]sdk.WorkflowRunResultArtifactManager, 0, len(runResults))
	for _, r := range runResults {
		if r.Type != sdk.WorkflowRunResultTypeArtifactManager {
			continue
		}

		data, err := r.GetArtifactManager()
		if err != nil {
			return nil, err
		}
		runResultDatas = append(runResultDatas, data)

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
		props.AddProperty("build.name", fmt.Sprintf("%s/%s/%s", execContext.buildInfo, execContext.projectKey, execContext.workflowName))
		props.AddProperty("build.number", execContext.version)
		props.AddProperty("build.timestamp", strconv.FormatInt(time.Now().Unix(), 10))

		if err := SetPropertiesRecursive(ctx, artiClient, data.RepoName, currentMaturity, []sdk.FileInfo{{Path: data.Path}}, props); err != nil {
			return nil, err
		}

	}
	return modules, nil
}

func retrieveModulesFiles(ctx context.Context, client artifact_manager.ArtifactManager, repoName string, path string) ([]sdk.FileInfo, error) {
	ctx, end := telemetry.Span(ctx, "workflow.retrieveModulesFiles")
	defer end()
	log.Debug(ctx, "retrieve:ModulesFiles repoName:%s path:%s", repoName, path)
	_, endc := telemetry.Span(ctx, "artifactoryClient.GetFileInfo", telemetry.Tag("path", path), telemetry.Tag("repoName", repoName))
	fileInfo, err := client.GetFileInfo(repoName, path)
	endc()
	if err != nil {
		return nil, err
	}

	// If it can be downloaded, it's a file
	if fileInfo.DownloadURI != "" {
		return []sdk.FileInfo{fileInfo}, nil
	}
	return retrieveModulesFilesFromFolder(ctx, client, repoName, path)
}

func retrieveModulesFilesFromFolder(ctx context.Context, client artifact_manager.ArtifactManager, repoName string, path string) ([]sdk.FileInfo, error) {
	ctx, end := telemetry.Span(ctx, "workflow.retrieveModulesFilesFromFolder")
	defer end()
	log.Debug(ctx, "retrieve:retrieveModulesFilesFromFolder repoName:%s path:%s", repoName, path)
	_, endc := telemetry.Span(ctx, "artifactoryClient.GetFolderInfo", telemetry.Tag("path", path), telemetry.Tag("repoName", repoName))
	folderInfo, err := client.GetFolderInfo(repoName, path)
	endc()
	if err != nil {
		return nil, err
	}

	files := make([]sdk.FileInfo, 0)
	for _, c := range folderInfo.Children {
		if c.Folder {
			childrenFiles, err := retrieveModulesFilesFromFolder(ctx, client, repoName, fmt.Sprintf("%s%s", path, c.Uri))
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
