package main

import (
	"context"
	"fmt"
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

	if err := p.perform(ctx, resultType, path); err != nil {
		err := fmt.Errorf("unable to create run result: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)

}

func (p *addRunResultPlugin) perform(ctx context.Context, resultType, path string) error {

	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
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

	pathSplit := strings.SplitN(path, "/", 2)
	if len(pathSplit) != 2 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid path. Must be <reposutory_name>/path/to/file")
	}

	// get file info
	fileInfo, err := grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, pathSplit[0], pathSplit[1])
	if err != nil {
		return sdk.WrapError(err, "unable to retrieve file %s", path)
	}

	// get file properties
	fileProps, err := grpcplugins.GetArtifactoryFileProperties(ctx, &p.Common, artiConfig, pathSplit[0], pathSplit[1])
	if err != nil {
		return sdk.WrapError(err, "unable to retrieve file properties %s", path)
	}

	// get repository info
	repositoryInfo, err := grpcplugins.GetArtifactoryRepositoryInfo(ctx, &p.Common, artiConfig, pathSplit[0])
	if err != nil {
		return sdk.WrapError(err, "unable to retrieve repository %s", pathSplit[0])
	}

	//search file
	fileDir, fileName := filepath.Split(fileInfo.Path)
	aqlSearch := fmt.Sprintf(`items.find({"name" : "%s", "path" : "%s"}).include("repo","path","name","virtual_repos")`, fileName, strings.TrimPrefix(strings.TrimSuffix(fileDir, "/"), "/"))

	itemSearch, err := grpcplugins.SearchItem(ctx, &p.Common, artiConfig, aqlSearch)
	if err != nil {
		return err
	}
	if len(itemSearch.Results) == 0 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to find artifact %s in path %s", fileName, fileDir)
	}
	var localRepo, virtualRepo string
	if repositoryInfo.Rclass == "virtual" {
		// retrieve localRepository
		virtualRepo = pathSplit[0]
		for _, r := range itemSearch.Results {
			if strings.HasPrefix(r.Repo, virtualRepo) {
				localRepo = r.Repo
				break
			}
		}
	} else {
		// retrieve virtual
		localRepo = pathSplit[0]
		for _, r := range itemSearch.Results {
			if r.Repo != localRepo {
				continue
			}
			for _, v := range r.VirtualRepos {
				if strings.HasPrefix(localRepo, v) {
					virtualRepo = v
					break
				}
			}
			break
		}
	}
	if localRepo == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve local repository for artifact %s", path)
	}
	if virtualRepo == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve virtual repository for artifact %s", path)
	}

	// compute maturity
	maturity := strings.TrimPrefix(localRepo, virtualRepo+"-")

	runResult := sdk.V2WorkflowRunResult{
		IssuedAt:                       fileInfo.Created,
		Status:                         sdk.V2WorkflowRunResultStatusCompleted,
		ArtifactManagerIntegrationName: &jobCtx.Integrations.ArtifactManager.Name,
	}

	grpcplugins.ExtractFileInfoIntoRunResult(&runResult, *fileInfo, fileName, resultType, localRepo, virtualRepo, maturity)

	switch resultType {
	case sdk.V2WorkflowRunResultTypeDebian:
	case sdk.V2WorkflowRunResultTypeArsenalDeployment:
	case sdk.V2WorkflowRunResultTypeDocker:
	case sdk.V2WorkflowRunResultTypeRelease:
	case sdk.V2WorkflowRunResultTypeTest:
	case sdk.V2WorkflowRunResultTypeHelm:
		if err := performHelm(&runResult, fileProps.Properties); err != nil {
			return err
		}
	case sdk.V2WorkflowRunResultTypePython:
		if err := performPython(&runResult, fileName, fileProps.Properties); err != nil {
			return err
		}
	case sdk.V2WorkflowRunResultTypeGeneric:
		if err := performGeneric(&runResult, *fileInfo, fileName); err != nil {
			return err
		}
	case sdk.V2WorkflowRunResultTypeCoverage:
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unsupported result type %s", resultType)
	}
	_, err = grpcplugins.CreateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: &runResult})
	return err
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

func performGeneric(runResult *sdk.V2WorkflowRunResult, fileInfo grpcplugins.ArtifactoryFileInfo, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	runResult.Type = sdk.V2WorkflowRunResultTypeGeneric
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
