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

	ko, err := p.perform(ctx, resultType, path)
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

func (p *addRunResultPlugin) perform(ctx context.Context, resultType, artifactPath string) (bool, error) {

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
		path += "/manifest.json"
	}

	repository := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigRepositoryPrefix)
	switch resultType {
	case sdk.V2WorkflowRunResultTypeDocker:
		repository += "-docker"
	case sdk.V2WorkflowRunResultTypeDebian:
		repository += "-debian"
	case sdk.V2WorkflowRunResultTypeTest, sdk.V2WorkflowRunResultTypeCoverage, sdk.V2WorkflowRunResultTypeGeneric:
		repository += "-generic"
	case sdk.V2WorkflowRunResultTypeHelm:
		repository += "-helm"
	case sdk.V2WorkflowRunResultTypePython:
		repository += "-pypi"
	}

	// get file info
	fileInfo, err := grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, repository, path)
	if err != nil {
		return true, sdk.WrapError(err, "unable to retrieve file %s", path)
	}

	// get file properties
	fileProps, err := grpcplugins.GetArtifactoryFileProperties(ctx, &p.Common, artiConfig, repository, path)
	if err != nil {
		return true, sdk.WrapError(err, "unable to retrieve file properties %s", path)
	}

	//search file
	fileDir, fileName := filepath.Split(fileInfo.Path)
	aqlSearch := fmt.Sprintf(`items.find({"name" : "%s", "path" : "%s"}).include("repo","path","name","virtual_repos")`, fileName, strings.TrimPrefix(strings.TrimSuffix(fileDir, "/"), "/"))

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
		if err := performDocker(ctx, &p.Common, &runResult, jobCtx.Integrations.ArtifactManager, artifactPath, *fileInfo); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeDebian:
		runResult.Type = sdk.V2WorkflowRunResultTypeDebian
		if err := performDebian(&runResult, fileInfo, fileProps.Properties); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeTest:
		runResult.Type = sdk.V2WorkflowRunResultTypeTest
		nbKo, err := performTests(ctx, &p.Common, *fileInfo, &runResult, jobCtx.Integrations.ArtifactManager, path)
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
		if err := performHelm(&runResult, fileProps.Properties); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypePython:
		runResult.Type = sdk.V2WorkflowRunResultTypePython
		if err := performPython(&runResult, fileName, fileProps.Properties); err != nil {
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
	default:
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unsupported result type %s", resultType)
	}
	_, err = grpcplugins.CreateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: &runResult})
	return mustReturnKO, err
}

func performDocker(ctx context.Context, c *actionplugin.Common, runResult *sdk.V2WorkflowRunResult, integ sdk.JobIntegrationsContext, dockerFullImage string, fileinfo grpcplugins.ArtifactoryFileInfo) error {
	dockerURLSplit := strings.SplitN(dockerFullImage, "/", 2)
	if len(dockerURLSplit) != 2 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to parse docker images url: %s", dockerFullImage)
	}

	url, err := url.Parse(integ.Get(sdk.ArtifactoryConfigURL))
	if err != nil {
		return err
	}
	repository := strings.TrimSuffix(dockerURLSplit[0], "."+url.Host)
	imageTag := strings.Split(dockerURLSplit[1], ":")
	var tag string
	if len(imageTag) == 1 {
		tag = "latest"
	} else if len(imageTag) == 2 {
		tag = imageTag[1]
	} else {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve tag from image %s", dockerURLSplit[1])
	}

	// retrieve image ID
	downloadURI := fmt.Sprintf("%s%s/%s/manifest.json", integ.Get(sdk.ArtifactoryConfigURL), strings.TrimPrefix(repository, "/"), dockerURLSplit[1])
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
		return sdk.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
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
	name := dockerFullImage
	runResult.Detail = grpcplugins.ComputeRunResultDockerDetail(name, img)
	return nil
}

func performTests(ctx context.Context, c *actionplugin.Common, fileInfo grpcplugins.ArtifactoryFileInfo, runResult *sdk.V2WorkflowRunResult, jobCtx sdk.JobIntegrationsContext, path string) (int, error) {
	// download file
	downloadURI := fmt.Sprintf("%s%s", jobCtx.Get(sdk.ArtifactoryConfigURL), strings.TrimPrefix(path, "/"))
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
		return 0, sdk.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
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
