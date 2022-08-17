package art

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/distribution"
	authdistrib "github.com/jfrog/jfrog-client-go/distribution/auth"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/pkg/errors"

	"github.com/ovh/cds/engine/api/integration/artifact_manager"
	"github.com/ovh/cds/sdk"
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

func SetProperties(artiClient artifactory.ArtifactoryServicesManager, repoName string, filePath string, props map[string]string) error {
	fileInfoURL := fmt.Sprintf("%sapi/storage/%s/%s?properties=", artiClient.GetConfig().GetServiceDetails().GetUrl(), repoName, filePath)

	for k, v := range props {
		fileInfoURL += fmt.Sprintf("%s=%s%s", k, url.QueryEscape(v), url.QueryEscape("|"))

	}
	httpDetails := artiClient.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	resp, body, err := artiClient.Client().SendPut(fileInfoURL, nil, &httpDetails)
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory: %v", err)
	}

	if resp.StatusCode >= 400 {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to call artifactory [HTTP: %d] %s: %s", resp.StatusCode, fileInfoURL, string(body))
	}
	return nil
}

func PromoteFile(artiClient artifactory.ArtifactoryServicesManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string) error {
	srcRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := services.NewMoveCopyParams()
	params.Pattern = fmt.Sprintf("%s/%s", srcRepo, data.Path)
	params.Target = fmt.Sprintf("%s/%s", targetRepo, data.Path)
	params.Flat = true

	// Check if artifact already exist on destination
	exist, err := checkArtifactExists(artiClient, targetRepo, data.Path)
	if err != nil {
		return err
	}

	if !exist {
		// If source repository is a release repository, we should not move but copy the artifact
		// Get the properties of the source reposiytory
		maturity, err := GetRepositoryMaturity(artiClient, srcRepo)
		if err != nil {
			fmt.Printf("Warning: unable to get repository maturity: %v\n", err)
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
		return nil
	}
	fmt.Printf("%s has been already promoted\n", data.Name)
	return nil
}

type PropertiesResponse struct {
	Properties map[string][]string
}

func GetRepositoryMaturity(artiClient artifactory.ArtifactoryServicesManager, repoName string) (string, error) {
	httpDetails := artiClient.GetConfig().GetServiceDetails().CreateHttpClientDetails()
	uri := fmt.Sprintf(artiClient.GetConfig().GetServiceDetails().GetUrl()+"api/storage/%s?properties", repoName)
	re, body, _, err := artiClient.Client().SendGet(uri, true, &httpDetails)
	if err != nil {
		return "", errors.Errorf("unable to get properties %s: %v", repoName, err)
	}
	if re.StatusCode == 404 {
		return "", errors.Errorf("repository %s properties not foud", repoName)
	}
	if re.StatusCode >= 400 {
		return "", errors.Errorf("unable to call artifactory [HTTP: %d] %s %s", re.StatusCode, uri, string(body))
	}
	var props PropertiesResponse
	if err := json.Unmarshal(body, &props); err != nil {
		return "", errors.WithStack(err)
	}
	fmt.Printf("Repository %q has properties: %+v\n", repoName, props.Properties)
	for k, p := range props.Properties {
		if k == "ovh.maturity" {
			return p[0], nil
		}
	}
	return "", nil
}

func PromoteDockerImage(artiClient artifactory.ArtifactoryServicesManager, data sdk.WorkflowRunResultArtifactManager, lowMaturity, highMaturity string) error {
	sourceRepo := fmt.Sprintf("%s-%s", data.RepoName, lowMaturity)
	targetRepo := fmt.Sprintf("%s-%s", data.RepoName, highMaturity)
	params := services.NewDockerPromoteParams(data.Path, sourceRepo, targetRepo)

	maturity, err := GetRepositoryMaturity(artiClient, sourceRepo)
	if err != nil {
		fmt.Printf("Warning: unable to get repository maturity: %v\n", err)
	}

	// Check if artifact already exist on destination
	exist, err := checkArtifactExists(artiClient, targetRepo, data.Path)
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
		return artiClient.PromoteDocker(params)
	}
	fmt.Printf("%s has been already promoted\n", data.Name)
	return nil
}

func checkArtifactExists(artiClient artifactory.ArtifactoryServicesManager, repoName string, artiName string) (bool, error) {
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

type executionContext struct {
	buildInfo         string
	projectKey        string
	workflowName      string
	version           string
	lowMaturitySuffix string
}

type BuildInfoRequest struct {
	BuildInfoPrefix   string
	ProjectKey        string
	WorkflowName      string
	Version           string
	AgentName         string
	TokenName         string
	RunURL            string
	GitBranch         string
	GitMessage        string
	GitURL            string
	GitHash           string
	LowMaturitySuffix string
	RunResults        []sdk.WorkflowRunResult
}

func PrepareBuildInfo(ctx context.Context, artiClient artifact_manager.ArtifactManager, r BuildInfoRequest) (*buildinfo.BuildInfo, error) {
	buildInfoName := fmt.Sprintf("%s/%s/%s", r.BuildInfoPrefix, r.ProjectKey, r.WorkflowName)
	log.Debug(ctx, "PrepareBuildInfo %q", buildInfoName)

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
		ArtifactoryPrincipal:     fmt.Sprintf("token:%s", r.TokenName),
		ArtifactoryPluginVersion: sdk.VERSION,
		Started:                  time.Now().Format("2006-01-02T15:04:05.999-07:00"),
		Number:                   r.Version,
		BuildUrl:                 r.RunURL,
		Modules:                  []buildinfo.Module{},
		VcsList:                  make([]buildinfo.Vcs, 0),
	}

	buildInfoRequest.VcsList = append(buildInfoRequest.VcsList, buildinfo.Vcs{
		Branch:   r.GitBranch,
		Message:  r.GitMessage,
		Url:      r.GitURL,
		Revision: r.GitHash,
	})

	execContext := executionContext{
		buildInfo:         r.BuildInfoPrefix,
		lowMaturitySuffix: r.LowMaturitySuffix,
		workflowName:      r.WorkflowName,
		version:           r.Version,
		projectKey:        r.ProjectKey,
	}
	modules, err := computeBuildInfoModules(ctx, artiClient, execContext, r.RunResults)
	if err != nil {
		return nil, err
	}
	buildInfoRequest.Modules = modules

	return buildInfoRequest, nil
}

func computeBuildInfoModules(ctx context.Context, client artifact_manager.ArtifactManager, execContext executionContext, runResults []sdk.WorkflowRunResult) ([]buildinfo.Module, error) {
	modules := make([]buildinfo.Module, 0)
runResults:
	for _, r := range runResults {
		if r.Type != sdk.WorkflowRunResultTypeArtifactManager {
			continue
		}
		data, err := r.GetArtifactManager()
		if err != nil {
			return nil, err
		}
		for _, mod := range modules {
			for _, art := range mod.Artifacts {
				if art.Md5 == data.Path && art.Path == data.Path {
					continue runResults
				}
			}
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
			parsedUrl, err := url.Parse(client.GetURL())
			if err != nil {
				return nil, fmt.Errorf("unable to parse artifactory url [%s]: %v", client.GetURL(), err)
			}
			urlArtifactory := parsedUrl.Host
			if parsedUrl.Port() != "" {
				urlArtifactory += ":" + parsedUrl.Port()
			}
			props["docker.image.tag"] = fmt.Sprintf("%s.%s/%s", data.RepoName, urlArtifactory, data.Name)
			mod.Properties = props
		}

		artifacts, err := retrieveModulesArtifacts(ctx, client, data.RepoName, data.Path, execContext)
		if err != nil {
			return nil, err
		}
		mod.Artifacts = artifacts
		modules = append(modules, mod)
	}

	return modules, nil
}

func retrieveModulesArtifacts(ctx context.Context, client artifact_manager.ArtifactManager, repoName string, path string, execContext executionContext) ([]buildinfo.Artifact, error) {
	log.Debug(ctx, "retrieve:ModulesArtifacts repoName:%s path:%s execContext:%+v", repoName, path, execContext)
	fileInfo, err := client.GetFileInfo(repoName, path)
	if err != nil {
		return nil, err
	}
	artifacts := make([]buildinfo.Artifact, 0)

	// If no children, it's a file, so we have checksum
	_, objectName := filepath.Split(path)

	if len(fileInfo.Children) == 0 {
		props := []sdk.KeyValues{
			{
				Key:    "build.name",
				Values: []string{fmt.Sprintf("%s/%s/%s", execContext.buildInfo, execContext.projectKey, execContext.workflowName)},
			}, {
				Key:    "build.number",
				Values: []string{execContext.version},
			}, {
				Key:    "build.timestamp",
				Values: []string{strconv.FormatInt(time.Now().Unix(), 10)},
			},
		}
		repoSrc := repoName
		repoSrc += "-" + execContext.lowMaturitySuffix
		log.Debug(ctx, "setting properties %+v on repoSrc:%s path:%s", props, repoSrc, props)
		if err := client.SetProperties(repoSrc, path, props...); err != nil {
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
			artsChildren, err := retrieveModulesArtifacts(ctx, client, repoName, fmt.Sprintf("%s%s", path, c.Uri), execContext)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, artsChildren...)
		}
	}
	return artifacts, nil
}
