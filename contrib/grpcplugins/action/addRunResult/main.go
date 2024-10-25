package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type addRunResultPlugin struct {
	actionplugin.Common
}

func (actPlugin *addRunResultPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-addRunResultPlugin",
		Author:      "Steven GUIHEUX <steven.guiheux@ovhcloud.com>",
		Description: `This action create a run result from an artifact`,
		Version:     sdk.VERSION,
	}, nil
}

func (p *addRunResultPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	resultType := q.GetOptions()["type"]
	path := q.GetOptions()["path"]
	payload := q.GetOptions()["payload"]

	var detail sdk.V2WorkflowRunResultDetail
	if payload != "" {
		if err := sdk.JSONUnmarshal([]byte(payload), &detail); err != nil {
			err := fmt.Errorf("unable to parse payload: %v", err)
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
		}
	}

	ko, err := p.perform(ctx, resultType, path, detail)
	if err != nil {
		err := fmt.Errorf("unable to create run result: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}
	if ko {
		res.Status = sdk.StatusFail
	}

	return stream.Send(res)

}

func (p *addRunResultPlugin) perform(ctx context.Context, resultType, artifactPath string, detail sdk.V2WorkflowRunResultDetail) (bool, error) {
	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		return true, err
	}
	if jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager.Name == "" {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "you must have an artifact manager integration on your job")
	}

	// Get artifact information
	artiConfig := grpcplugins.ArtifactoryConfig{
		URL:   jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigURL),
		Token: jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigToken),
	}

	path := artifactPath
	if resultType == sdk.V2WorkflowRunResultTypeDocker {
		path = strings.Replace(path, ":", "/", -1)
		path += "/manifest.json"
	}

	repository := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigRepositoryPrefix)
	switch resultType {
	case sdk.V2WorkflowRunResultTypeDocker:
		repository += "-docker"
	case sdk.V2WorkflowRunResultTypeDebian:
		repository += "-debian"
	case sdk.V2WorkflowRunResultTypeTest, sdk.V2WorkflowRunResultTypeCoverage, sdk.V2WorkflowRunResultTypeGeneric:
		repository += "-cds"
	case sdk.V2WorkflowRunResultTypeHelm:
		repository += "-helm"
	case sdk.V2WorkflowRunResultTypePython:
		repository += "-pypi"
	case sdk.V2WorkflowRunResultTypeTerraformProvider:
		repository += "-terraformProvider"
	case sdk.V2WorkflowRunResultTypeTerraformModule:
		repository += "-terraformModule"
	case sdk.V2WorkflowRunResultTypeStaticFiles:
		return false, performStaticFiles(ctx, &p.Common, path, detail)
	}

	// get file info
	fileInfo, err := grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, repository, path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			repository = strings.Replace(repository, "-cds", "-generic", -1)
			fileInfo, err = grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, repository, path)
			if err != nil {
				return true, err
			}
		}
	}

	// get file properties
	fileProps, err := grpcplugins.GetArtifactoryFileProperties(ctx, &p.Common, artiConfig, repository, path)
	if err != nil {
		return true, sdk.WrapError(err, "unable to retrieve file properties %s", path)
	}

	//search file
	fileDir, fileName := filepath.Split(fileInfo.Path)
	aqlPath := strings.TrimPrefix(strings.TrimSuffix(fileDir, "/"), "/")
	var aqlSearch string
	if aqlPath == "" {
		aqlSearch = fmt.Sprintf(`items.find({"name" : "%s"}).include("repo","path","name","virtual_repos")`, fileName)
	} else {
		aqlSearch = fmt.Sprintf(`items.find({"name" : "%s", "path" : "%s"}).include("repo","path","name","virtual_repos")`, fileName, aqlPath)
	}

	itemSearch, err := grpcplugins.SearchItem(ctx, &p.Common, artiConfig, aqlSearch)
	if err != nil {
		return true, err
	}
	if len(itemSearch.Results) == 0 {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to find artifact %s in path %s", fileName, fileDir)
	}
	// retrieve localRepository
	virtualRepo := repository
	var localRepo string
	for _, r := range itemSearch.Results {
		if strings.HasPrefix(r.Repo, virtualRepo) {
			localRepo = r.Repo
			break
		}
	}

	if localRepo == "" {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve local repository for artifact %s", path)
	}

	// compute maturity
	maturity := strings.TrimPrefix(localRepo, virtualRepo+"-")

	runResult := sdk.V2WorkflowRunResult{
		IssuedAt:                       fileInfo.Created,
		Status:                         sdk.V2WorkflowRunResultStatusCompleted,
		ArtifactManagerIntegrationName: &jobCtx.Integrations.ArtifactManager.Name,
	}

	grpcplugins.ExtractFileInfoIntoRunResult(&runResult, *fileInfo, fileName, resultType, localRepo, virtualRepo, maturity)

	mustReturnKO := false
	switch resultType {
	case sdk.V2WorkflowRunResultTypeDocker:
		runResult.Type = sdk.V2WorkflowRunResultTypeDocker
		if err := performDocker(ctx, &p.Common, &runResult, jobCtx.Integrations.ArtifactManager, artifactPath, path, *fileInfo); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeDebian:
		runResult.Type = sdk.V2WorkflowRunResultTypeDebian
		if err := performDebian(&runResult, fileInfo, fileProps); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeTest:
		runResult.Type = sdk.V2WorkflowRunResultTypeTest
		nbKo, err := performTests(ctx, &p.Common, *fileInfo, &runResult, jobCtx.Integrations.ArtifactManager, repository, path)
		if err != nil {
			return true, err
		}
		if nbKo > 0 {
			if nbKo == 1 {
				grpcplugins.Errorf(&p.Common, "there is 1 test failed")
				return true, nil
			} else if nbKo > 1 {
				grpcplugins.Errorf(&p.Common, fmt.Sprintf("there are %d tests failed", nbKo))
				return true, nil
			}
		}
	case sdk.V2WorkflowRunResultTypeHelm:
		runResult.Type = sdk.V2WorkflowRunResultTypeHelm
		if err := performHelm(&runResult, fileProps); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypePython:
		runResult.Type = sdk.V2WorkflowRunResultTypePython
		if err := performPython(&runResult, fileName, fileProps); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeCoverage:
		if err := performGeneric(&runResult, fileInfo, sdk.V2WorkflowRunResultTypeCoverage, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeGeneric:
		if err := performGeneric(&runResult, fileInfo, sdk.V2WorkflowRunResultTypeGeneric, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeTerraformProvider:
		runResult.Type = sdk.V2WorkflowRunResultTypeTerraformProvider
		if err := performTerraformProvider(&runResult, fileProps); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeTerraformModule:
		runResult.Type = sdk.V2WorkflowRunResultTypeTerraformModule
		if err := performTerraformModule(&runResult, fileProps); err != nil {
			return true, err
		}

	default:
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unsupported result type %s", resultType)
	}
	if _, err := grpcplugins.CreateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: &runResult}); err != nil {
		return true, err
	}
	grpcplugins.Success(&p.Common, fmt.Sprintf("run result %s created", runResult.Name()))
	return mustReturnKO, err
}

func performStaticFiles(ctx context.Context, c *actionplugin.Common, destinationPath string, detail sdk.V2WorkflowRunResultDetail) error {
	jobCtx, err := grpcplugins.GetJobContext(ctx, c)
	if err != nil {
		return err
	}
	if jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager.Name == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "you must have an artifact manager integration on your job")
	}

	// Get artifact information
	artiConfig := grpcplugins.ArtifactoryConfig{
		URL:   jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigURL),
		Token: jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigToken),
	}

	repository := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-static"

	// get folder info
	folderInfo, err := grpcplugins.GetArtifactoryFolderInfo(ctx, c, artiConfig, repository, destinationPath)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return err
		}
	}

	detail.Type = "V2WorkflowRunResultStaticFilesDetail"
	runResult := sdk.V2WorkflowRunResult{
		IssuedAt:                       time.Now(),
		Status:                         sdk.V2WorkflowRunResultStatusCompleted,
		ArtifactManagerIntegrationName: &jobCtx.Integrations.ArtifactManager.Name,
		Type:                           sdk.V2WorkflowRunResultTypeStaticFiles,
		Detail:                         detail,
	}

	staticFilesDetail, err := sdk.GetConcreteDetail[*sdk.V2WorkflowRunResultStaticFilesDetail](&runResult)
	if err != nil {
		grpcplugins.Errorf(c, "unable to parse detail for staticFiles run result %q. Please check the documentation.", destinationPath)
		return err
	}
	runResult.Detail.Data = staticFilesDetail

	runResult.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
	runResult.ArtifactManagerMetadata.Set("repository", repository) // This is the virtual repository
	runResult.ArtifactManagerMetadata.Set("name", destinationPath)
	runResult.ArtifactManagerMetadata.Set("type", "folder")
	runResult.ArtifactManagerMetadata.Set("path", folderInfo.Path)
	runResult.ArtifactManagerMetadata.Set("createdBy", folderInfo.CreatedBy)
	runResult.ArtifactManagerMetadata.Set("localRepository", repository)
	runResult.ArtifactManagerMetadata.Set("uri", folderInfo.URI)

	var runResultRequest = workerruntime.V2RunResultRequest{RunResult: &runResult}
	runResultResponse, err := grpcplugins.CreateRunResult(ctx, c, &runResultRequest)
	if err != nil {
		grpcplugins.Errorf(c, "unable to create run result: %v", err.Error())
		return err
	}
	grpcplugins.Success(c, fmt.Sprintf("run result %s created", runResultResponse.RunResult.Name()))

	return nil
}

func performDocker(ctx context.Context, c *actionplugin.Common, runResult *sdk.V2WorkflowRunResult, integ sdk.JobIntegrationsContext, dockerImageName string, manifestPath string, fileinfo grpcplugins.ArtifactoryFileInfo) error {
	repository := integ.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-docker"
	imageTag := strings.Split(dockerImageName, ":")
	var tag string
	if len(imageTag) == 1 {
		tag = "latest"
	} else if len(imageTag) == 2 {
		tag = imageTag[1]
	} else {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve tag from image %s", dockerImageName)
	}

	// retrieve image ID
	downloadURI := fmt.Sprintf("%s%s/%s", integ.Get(sdk.ArtifactoryConfigURL), strings.TrimPrefix(repository, "/"), manifestPath)
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return sdk.WrapError(err, "unable to create request to retrieve file docker manifest")
	}

	rtToken := integ.Get(sdk.ArtifactoryConfigToken)
	req.Header.Set("Authorization", "Bearer "+rtToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return sdk.WrapError(err, "unable to get docker manifest file")
	}

	if resp.StatusCode > 200 {
		return sdk.Errorf("unable to download file %s (HTTP %d)", downloadURI, resp.StatusCode)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type dockerManifestConfig struct {
		Digest string `json:"digest"`
	}
	type dockerManifest struct {
		Config dockerManifestConfig `json:"config"`
	}

	var manifest dockerManifest
	if err := json.Unmarshal(bts, &manifest); err != nil {
		return sdk.WrapError(err, "unable to read docker manifest")
	}
	imageID := strings.TrimPrefix(manifest.Config.Digest, "sha256:")[0:12]
	img := grpcplugins.Img{
		Repository: repository,
		Tag:        tag,
		ImageID:    imageID,
		Created:    fileinfo.Created.String(),
		Size:       "",
	}

	url, err := url.Parse(integ.Get(sdk.ArtifactoryConfigURL))
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid artifact_manager url "+integ.Get(sdk.ArtifactoryConfigURL))
	}

	name := fmt.Sprintf("%s.%s/%s", repository, url.Host, dockerImageName)
	runResult.Detail = grpcplugins.ComputeRunResultDockerDetail(name, img)
	return nil
}

func performTests(ctx context.Context, c *actionplugin.Common, fileInfo grpcplugins.ArtifactoryFileInfo, runResult *sdk.V2WorkflowRunResult, jobCtx sdk.JobIntegrationsContext, repository, path string) (int, error) {
	// download file
	downloadURI := fmt.Sprintf("%s%s/%s", jobCtx.Get(sdk.ArtifactoryConfigURL), repository, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return 0, sdk.WrapError(err, "unable to create request to retrieve file "+path)
	}

	rtToken := jobCtx.Get(sdk.ArtifactoryConfigToken)
	req.Header.Set("Authorization", "Bearer "+rtToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, sdk.WrapError(err, "unable to get test file: %s", path)
	}

	if resp.StatusCode > 200 {
		return 0, sdk.Errorf("unable to download file %s (HTTP %d)", downloadURI, resp.StatusCode)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return 0, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to read file size [%s]: %v", fileInfo.Size, err)
	}

	detail, nbKo, err := grpcplugins.ComputeRunResultTestsDetail(c, path, bts, size, fileInfo.Checksums.Md5, fileInfo.Checksums.Sha1, fileInfo.Checksums.Sha256)
	if err != nil {
		return 0, err
	}
	runResult.Detail = *detail
	return nbKo, nil
}

func performDebian(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, props map[string][]string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	_, fileName := filepath.Split(fileInfo.Path)
	runResult.Detail = grpcplugins.ComputeRunResultDebianDetail(fileName, size, fileInfo.Checksums.Md5, fileInfo.Checksums.Sha1, fileInfo.Checksums.Sha256, props["deb.component"], props["deb.distribution"], props["deb.architecture"])
	return nil
}

func performHelm(runResult *sdk.V2WorkflowRunResult, props map[string][]string) error {
	chartNames, ok := props["chart.name"]
	if !ok || len(chartNames) == 0 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "chart doesn't have the chart.name property")
	}

	chartVerions, ok := props["chart.version"]
	if !ok || len(chartVerions) == 0 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "chart doesn't have the chart.version property")
	}
	runResult.Detail = grpcplugins.ComputeRunResultHelmDetail(chartNames[0], "", chartVerions[0])
	return nil
}

func performPython(runResult *sdk.V2WorkflowRunResult, fileName string, props map[string][]string) error {
	pypiVersion, ok := props["pypi.version"]
	if !ok || len(pypiVersion) == 0 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "package doesn't have the pypi.version property")
	}
	runResult.Detail = grpcplugins.ComputeRunResultPythonDetail(fileName, pypiVersion[0], strings.TrimPrefix(filepath.Ext(fileName), "."))

	return nil
}

func performTerraformModule(runResult *sdk.V2WorkflowRunResult, props map[string][]string) error {
	providerProps, ok := props["terraform.provider"]
	if !ok || len(providerProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.provider")
	}
	moduleProps := props["terraform.name"]
	if !ok || len(moduleProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.module")
	}
	nsProps := props["terraform.namespace"]
	if !ok || len(nsProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.namespace")
	}
	typeProps := props["terraform.type"]
	if !ok || len(typeProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.type")
	}
	versionProps := props["terraform.version"]
	if !ok || len(versionProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.version")
	}

	idProps := props["terraform.id"]
	if !ok || len(versionProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.id")
	}

	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultTerraformModuleDetail{
			Provider:  providerProps[0],
			Name:      moduleProps[0],
			Namespace: nsProps[0],
			Type:      typeProps[0],
			Version:   versionProps[0],
			ID:        idProps[0],
		},
	}
	return nil
}

func performTerraformProvider(runResult *sdk.V2WorkflowRunResult, props map[string][]string) error {
	flavorProps, ok := props["terraform.flavor"]
	if !ok || len(flavorProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.flavor")
	}
	nameProps := props["terraform.name"]
	if !ok || len(nameProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.name")
	}
	nsProps := props["terraform.namespace"]
	if !ok || len(nsProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.namespace")
	}
	typeProps := props["terraform.type"]
	if !ok || len(typeProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.type")
	}
	versionProps := props["terraform.version"]
	if !ok || len(versionProps) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property terraform.version")
	}

	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultTerraformProviderDetail{
			Flavor:    flavorProps[0],
			Name:      nameProps[0],
			Namespace: nsProps[0],
			Type:      typeProps[0],
			Version:   versionProps[0],
		},
	}
	return nil
}

func performGeneric(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, resultType sdk.V2WorkflowRunResultType, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	runResult.Type = resultType
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultGenericDetail{
			Name:   fileName,
			Size:   size,
			Mode:   os.FileMode(0755),
			MD5:    fileInfo.Checksums.Md5,
			SHA1:   fileInfo.Checksums.Sha1,
			SHA256: fileInfo.Checksums.Sha256,
		},
	}
	return nil
}

func (actPlugin *addRunResultPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func main() {
	actPlugin := addRunResultPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
