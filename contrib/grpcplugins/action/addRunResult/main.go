package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
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

	resultType := sdk.V2WorkflowRunResultType(q.GetOptions()["type"])
	path := q.GetOptions()["path"]
	payload := q.GetOptions()["payload"]
	ifNoFilesFound := q.GetOptions()["if-no-files-found"]

	var detail sdk.V2WorkflowRunResultDetail
	if payload != "" {
		if err := sdk.JSONUnmarshal([]byte(payload), &detail); err != nil {
			err := fmt.Errorf("unable to parse payload: %v", err)
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
		}
	}

	ko, err := p.perform(ctx, resultType, path, ifNoFilesFound, detail)
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

func (p *addRunResultPlugin) perform(ctx context.Context, resultType sdk.V2WorkflowRunResultType, artifactPath, ifNoFilesFound string, detail sdk.V2WorkflowRunResultDetail) (bool, error) {
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

	isGlob := containsGlob(artifactPath)
	if isGlob && !globSupportedType(resultType) {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "wildcard path %q is not supported for result type %s", artifactPath, resultType)
	}

	repository := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigRepositoryPrefix)
	switch resultType {
	case sdk.V2WorkflowRunResultTypeDocker:
		// Docker needs a very specific behavior due to the layout. All the common steps bellow are skipped and handled by performDocker.
		// A glob is the exception: the <image>/<tag> folders are enumerated like oci, then each match goes back through performDocker.
		if !isGlob {
			return p.performDocker(ctx, artifactPath)
		}
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
	case sdk.V2WorkflowRunResultTypeNpm:
		repository += "-npm"
	case sdk.V2WorkflowRunResultTypeStaticFiles:
		return false, performStaticFiles(ctx, &p.Common, artifactPath, detail)
	case sdk.V2WorkflowRunResultTypeMaven:
		repository += "-maven"
	case sdk.V2WorkflowRunResultTypeGradle:
		repository += "-gradle"
	case sdk.V2WorkflowRunResultTypeSbt:
		repository += "-sbt"
	case sdk.V2WorkflowRunResultTypeNuget:
		repository += "-nuget"
	case sdk.V2WorkflowRunResultTypePuppet:
		repository += "-puppet"
	case sdk.V2WorkflowRunResultTypeConan:
		repository += "-conan"
	case sdk.V2WorkflowRunResultTypeOCI:
		repository += "-oci"
	}

	if !isGlob {
		return p.performOne(ctx, resultType, artiConfig, jobCtx.Integrations.ArtifactManager, repository, artifactPath)
	}

	enum, err := p.enumerateGlobMatches(ctx, artiConfig, jobCtx.Integrations.ArtifactManager, repository, artifactPath, resultType)
	if err != nil {
		return true, err
	}
	if len(enum.candidates) == 0 {
		msg := fmt.Sprintf("no artifact found matching %q in repository %s", artifactPath, repository)
		switch strings.ToUpper(ifNoFilesFound) {
		case "ERROR":
			return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "%s", msg)
		case "IGNORE":
			grpcplugins.Log(&p.Common, msg)
		default:
			grpcplugins.Warn(&p.Common, msg)
		}
		return false, nil
	}

	// a conan/oci run result must list every file of the package; the enumeration only carried
	// the manifest marker, so those types go through performOne whose AQL fetches the full
	// listing. File-based types (1 artifact = 1 file) reuse the enumeration item directly.
	needsPackageFilesListing := resultType == sdk.V2WorkflowRunResultTypeConan || resultType == sdk.V2WorkflowRunResultTypeOCI

	var nbKO int
	for _, candidate := range enum.candidates {
		var ko bool
		var err error
		switch {
		case resultType == sdk.V2WorkflowRunResultTypeDocker:
			// for docker, candidate.path is an image:tag reference
			ko, err = p.performDockerFromItem(ctx, jobCtx.Integrations.ArtifactManager, artiConfig, repository, candidate, enum.archByPath)
		case needsPackageFilesListing:
			ko, err = p.performOne(ctx, resultType, artiConfig, jobCtx.Integrations.ArtifactManager, repository, candidate.path)
		default:
			ko, err = p.performOneFromItem(ctx, resultType, jobCtx.Integrations.ArtifactManager, artiConfig, repository, candidate)
		}
		if err != nil {
			grpcplugins.Errorf(&p.Common, "unable to create run result for %q: %v", candidate.path, err)
			nbKO++
			continue
		}
		if ko {
			nbKO++
		}
	}
	grpcplugins.Logf(&p.Common, "%d run result(s) created, %d failed (pattern %q)", len(enum.candidates)-nbKO, nbKO, artifactPath)
	return nbKO > 0, nil
}

// performOne creates a single run result for a concrete artifact path: a file for most types,
// a package folder for oci and conan.
func (p *addRunResultPlugin) performOne(ctx context.Context, resultType sdk.V2WorkflowRunResultType, artiConfig grpcplugins.ArtifactoryConfig, integ sdk.JobIntegrationsContext, repository, path string) (bool, error) {
	// get file info
	fileInfo, err := grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, repository, path)
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return true, err
		}
		repository = strings.Replace(repository, "-cds", "-generic", -1)
		fileInfo, err = grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, repository, path)
		if err != nil {
			return true, err
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
	if resultType == sdk.V2WorkflowRunResultTypeConan {
		aqlPath := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/") + "/*"
		aqlSearch = fmt.Sprintf(`items.find({"path" : {"$match": "%s"}, "repo": {"$eq":"%s"}}).include("repo","path","name","virtual_repos", "actual_md5", "actual_sha1", "sha256", "size", "property")`, aqlPath, repository)
	} else if resultType == sdk.V2WorkflowRunResultTypeOCI {
		// An OCI package is a folder (<name>/<version>) holding the manifest and blobs directly in it.
		// We match the version folder itself and any nested path. The AQL "repo" field is the local
		// repository, so we match the virtual repo's maturities (e.g. "<repo>-snapshot") with a prefix.
		ociPath := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/")
		aqlSearch = fmt.Sprintf(`items.find({"$or":[{"path":{"$eq":"%s"}},{"path":{"$match":"%s/*"}}], "repo":{"$match":"%s-*"}}).include("repo","path","name","virtual_repos", "actual_md5", "actual_sha1", "sha256", "size", "property")`, ociPath, ociPath, repository)
	} else if aqlPath == "" {
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

	return p.createRunResult(ctx, resultType, integ, virtualRepo, localRepo, maturity, path, fileInfo, fileProps, itemSearch)
}

// createRunResult builds the typed detail from the artifact data and submits the run result.
// itemSearch carries the package files listing, only read for conan and oci.
func (p *addRunResultPlugin) createRunResult(ctx context.Context, resultType sdk.V2WorkflowRunResultType, integ sdk.JobIntegrationsContext, virtualRepo, localRepo, maturity, path string, fileInfo *grpcplugins.ArtifactoryFileInfo, fileProps map[string][]string, itemSearch *grpcplugins.SearchResultResponse) (bool, error) {
	_, fileName := filepath.Split(fileInfo.Path)

	runResult := sdk.V2WorkflowRunResult{
		IssuedAt:                       fileInfo.Created,
		Status:                         sdk.V2WorkflowRunResultStatusCompleted,
		ArtifactManagerIntegrationName: &integ.Name,
	}

	grpcplugins.ExtractFileInfoIntoRunResult(&runResult, *fileInfo, fileName, resultType, localRepo, virtualRepo, maturity)

	switch resultType {
	case sdk.V2WorkflowRunResultTypeDebian:
		runResult.Type = sdk.V2WorkflowRunResultTypeDebian
		if err := performDebian(&runResult, fileInfo, fileProps); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeTest:
		runResult.Type = sdk.V2WorkflowRunResultTypeTest
		nbKo, err := performTests(ctx, &p.Common, *fileInfo, &runResult, integ, virtualRepo, path)
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
	case sdk.V2WorkflowRunResultTypeNpm:
		runResult.Type = sdk.V2WorkflowRunResultTypeNpm
		if err := performNpm(&runResult, fileInfo, fileProps, fileName); err != nil {
			return true, err

		}
	case sdk.V2WorkflowRunResultTypeMaven:
		if err := performMaven(&runResult, fileInfo, resultType, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeGradle:
		if err := performGradle(&runResult, fileInfo, resultType, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeSbt:
		if err := performSbt(&runResult, fileInfo, resultType, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeNuget:
		if err := performNuget(&runResult, fileInfo, fileProps, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypePuppet:
		if err := performPuppet(&runResult, fileInfo, fileProps, fileName); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeConan:
		if err := performConan(&runResult, itemSearch, path); err != nil {
			return true, err
		}
	case sdk.V2WorkflowRunResultTypeOCI:
		if err := performOCI(&runResult, itemSearch, path); err != nil {
			return true, err
		}
	default:
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unsupported result type %s", resultType)
	}
	if _, err := grpcplugins.CreateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: &runResult}); err != nil {
		return true, err
	}
	grpcplugins.Success(&p.Common, fmt.Sprintf("run result %s created", runResult.Name()))
	return false, nil
}

// performOneFromItem creates a run result for a file enumerated by the glob search, reusing
// the data carried by the AQL item instead of re-fetching it from artifactory.
func (p *addRunResultPlugin) performOneFromItem(ctx context.Context, resultType sdk.V2WorkflowRunResultType, integ sdk.JobIntegrationsContext, artiConfig grpcplugins.ArtifactoryConfig, repository string, candidate globCandidate) (bool, error) {
	item := candidate.item
	virtualRepo := virtualRepoFor(item.Repo, repository)
	if virtualRepo == "" {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to match local repository %s with virtual repository %s", item.Repo, repository)
	}
	maturity := strings.TrimPrefix(item.Repo, virtualRepo+"-")
	fileInfo := fileInfoFromItem(artiConfig, virtualRepo, item)
	return p.createRunResult(ctx, resultType, integ, virtualRepo, item.Repo, maturity, candidate.path, fileInfo, propertiesMap(item.Properties), nil)
}

// performDockerFromItem creates a docker run result for an image enumerated by the glob
// search. performDocker is the same work, with the manifest located from artifactory.
func (p *addRunResultPlugin) performDockerFromItem(ctx context.Context, integ sdk.JobIntegrationsContext, artiConfig grpcplugins.ArtifactoryConfig, repository string, candidate globCandidate, archByPath map[string]grpcplugins.SearchResult) (bool, error) {
	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		return true, err
	}

	item := candidate.item
	virtualRepo := virtualRepoFor(item.Repo, repository)
	if virtualRepo == "" {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to match local repository %s with virtual repository %s", item.Repo, repository)
	}
	maturity := strings.TrimPrefix(item.Repo, virtualRepo+"-")

	rtURL, err := url.Parse(artiConfig.URL)
	if err != nil {
		return true, err
	}
	image, tag, _ := strings.Cut(candidate.path, ":")
	destinationImageName := virtualRepo + "." + rtURL.Host + "/" + candidate.path
	dockerImg := grpcplugins.Img{Repository: image, Tag: tag}

	files := grpcplugins.DockerManifestFiles{
		FolderPath:   "/" + strings.Trim(item.Path, "/"),
		FileInfo:     *fileInfoFromItem(artiConfig, virtualRepo, item),
		MultiArch:    item.Name == "list.manifest.json",
		ArchFileInfo: p.archFileInfoFrom(ctx, artiConfig, virtualRepo, item.Repo, image, archByPath),
	}

	runResult := sdk.V2WorkflowRunResult{
		WorkflowRunID:                  jobCtx.CDS.RunID,
		IssuedAt:                       time.Now(),
		Status:                         sdk.V2WorkflowRunResultStatusCompleted,
		ArtifactManagerIntegrationName: &integ.Name,
		Type:                           sdk.V2WorkflowRunResultTypeDocker,
	}

	if err := grpcplugins.FinalizeRunResultDockerDetailFromFiles(ctx, &p.Common, artiConfig, &runResult, destinationImageName, &dockerImg, item.Repo, virtualRepo, maturity, files); err != nil {
		return true, err
	}

	runResultResponse, err := grpcplugins.CreateRunResult(ctx, &p.Common, &workerruntime.V2RunResultRequest{RunResult: &runResult})
	if err != nil {
		grpcplugins.Errorf(&p.Common, "unable to create result: %v", err.Error())
		return true, err
	}
	grpcplugins.Success(&p.Common, fmt.Sprintf("run result %s created", runResultResponse.RunResult.Name()))
	return false, nil
}

// archFileInfoFrom serves the per-architecture manifests out of the glob enumeration. The
// fallback is insurance: staticPrefix never goes deeper than the image folder, so the
// digest folders are always in the scope of the search.
func (p *addRunResultPlugin) archFileInfoFrom(ctx context.Context, artiConfig grpcplugins.ArtifactoryConfig, virtualRepo, localRepo, image string, archByPath map[string]grpcplugins.SearchResult) func(string) (*grpcplugins.ArtifactoryFileInfo, error) {
	return func(digest string) (*grpcplugins.ArtifactoryFileInfo, error) {
		folder := image + "/" + digest
		if item, ok := archByPath[folder]; ok {
			return fileInfoFromItem(artiConfig, virtualRepo, item), nil
		}
		return grpcplugins.GetArtifactoryFileInfo(ctx, &p.Common, artiConfig, localRepo, folder+"/manifest.json")
	}
}

// performDocker is very specific to docker artifactory layout. It doesn't share anything with other perform functions
func (p *addRunResultPlugin) performDocker(ctx context.Context, dockerImageName string) (bool, error) {
	jobCtx, err := grpcplugins.GetJobContext(ctx, &p.Common)
	if err != nil {
		return true, err
	}

	if jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager.Name == "" {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "you must have an artifact manager integration on your job")
	}

	rtURLRaw := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigURL)
	if !strings.HasSuffix(rtURLRaw, "/") {
		rtURLRaw = rtURLRaw + "/"
	}
	rtURL, err := url.Parse(rtURLRaw)
	if err != nil {
		return true, err
	}

	repository := jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-docker" + "." + rtURL.Host
	splittedDockerImageName := strings.Split(dockerImageName, ":")
	destinationImageName := repository + "/" + splittedDockerImageName[0]
	var tag string
	if len(splittedDockerImageName) == 1 {
		tag = "latest"
	} else if len(splittedDockerImageName) == 2 {
		tag = splittedDockerImageName[1]
	} else {
		return true, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to retrieve tag from image %s", dockerImageName)
	}
	destinationImageName += ":" + tag

	dockerImg := grpcplugins.Img{
		Repository: splittedDockerImageName[0],
		Tag:        tag,
	}

	runResult := sdk.V2WorkflowRunResult{
		WorkflowRunID:                  jobCtx.CDS.RunID,
		IssuedAt:                       time.Now(),
		Status:                         sdk.V2WorkflowRunResultStatusCompleted,
		ArtifactManagerIntegrationName: &jobCtx.Integrations.ArtifactManager.Name,
		Detail:                         grpcplugins.ComputeRunResultDockerDetail(destinationImageName, dockerImg),
		Type:                           sdk.V2WorkflowRunResultTypeDocker,
	}

	rtConfig := grpcplugins.ArtifactoryConfig{
		URL:   rtURL.String(),
		Token: jobCtx.Integrations.ArtifactManager.Get(sdk.ArtifactoryConfigToken),
	}

	if err := grpcplugins.FinalizeRunResultDockerDetail(ctx, &p.Common, rtConfig, &runResult, destinationImageName, &dockerImg); err != nil {
		return true, err
	}

	var runResultRequest = workerruntime.V2RunResultRequest{RunResult: &runResult}
	runResultResponse, err := grpcplugins.CreateRunResult(ctx, &p.Common, &runResultRequest)
	if err != nil {
		grpcplugins.Errorf(&p.Common, "unable to create result: %v", err.Error())
		return true, err
	}
	grpcplugins.Success(&p.Common, fmt.Sprintf("run result %s created", runResultResponse.RunResult.Name()))

	return false, nil
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
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid artifact_manager url %s", integ.Get(sdk.ArtifactoryConfigURL))
	}

	name := fmt.Sprintf("%s.%s/%s", repository, url.Host, dockerImageName)
	runResult.Detail = grpcplugins.ComputeRunResultDockerDetail(name, img)

	runResult.ArtifactManagerMetadata.Set("dir", "") // <- repository where sits the manifest.json
	runResult.ArtifactManagerMetadata.Set("name", manifestPath)

	return nil
}

func performTests(ctx context.Context, c *actionplugin.Common, fileInfo grpcplugins.ArtifactoryFileInfo, runResult *sdk.V2WorkflowRunResult, jobCtx sdk.JobIntegrationsContext, repository, path string) (int, error) {
	// download file
	downloadURI := fmt.Sprintf("%s%s/%s", jobCtx.Get(sdk.ArtifactoryConfigURL), repository, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return 0, sdk.WrapError(err, "unable to create request to retrieve file %s", path)
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

func performNpm(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, props map[string][]string, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	npmName, ok := props["npm.name"]
	if !ok || len(npmName) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property npm.name")
	}
	npmVersion, ok := props["npm.version"]
	if !ok || len(npmName) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property npm.version")
	}

	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultNpmDetail{
			FileName: fileName,
			Package:  npmName[0],
			Version:  npmVersion[0],
			Size:     size,
			Mode:     os.FileMode(0755),
			MD5:      fileInfo.Checksums.Md5,
			SHA1:     fileInfo.Checksums.Sha1,
			SHA256:   fileInfo.Checksums.Sha256,
		},
	}
	return nil
}

func performConan(runResult *sdk.V2WorkflowRunResult, aqlSearch *grpcplugins.SearchResultResponse, packagePath string) error {
	var packageName, packageVersion string
	files := make([]sdk.V2WorkflowRunResultConanDetailFile, 0)

	for _, item := range aqlSearch.Results {
		file := sdk.V2WorkflowRunResultConanDetailFile{
			FileName: item.Name,
			Path:     item.Path,
			Size:     item.Size,
			MD5:      item.ActualMd5,
			SHA1:     item.ActualSha1,
			SHA256:   item.Sha256,
		}
		files = append(files, file)

		for _, prop := range item.Properties {
			if prop.Key == "conan.package.name" {
				packageName = prop.Value
			}
			if prop.Key == "conan.package.version" {
				packageVersion = prop.Value
			}
		}
	}
	pathSplit := strings.Split(packagePath, "/")

	runResult.Type = sdk.V2WorkflowRunResultTypeConan
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultConanDetail{
			Name:            packageName,
			Version:         packageVersion,
			PackageRevision: pathSplit[len(pathSplit)-1],
			Files:           files,
		},
	}

	return nil
}

// performOCI handles generic OCI packages. Like Conan, an OCI package is a folder (<name>/<version>)
// containing several files (manifest + blobs). The package name and version are derived from the path:
// the version is the last path segment, the name is everything before it.
func performOCI(runResult *sdk.V2WorkflowRunResult, aqlSearch *grpcplugins.SearchResultResponse, packagePath string) error {
	files := make([]sdk.V2WorkflowRunResultOCIDetailFile, 0)
	for _, item := range aqlSearch.Results {
		files = append(files, sdk.V2WorkflowRunResultOCIDetailFile{
			FileName: item.Name,
			Path:     item.Path,
			Size:     item.Size,
			MD5:      item.ActualMd5,
			SHA1:     item.ActualSha1,
			SHA256:   item.Sha256,
		})
	}

	pathSplit := strings.Split(strings.Trim(packagePath, "/"), "/")
	version := pathSplit[len(pathSplit)-1]
	name := strings.Join(pathSplit[:len(pathSplit)-1], "/")

	runResult.Type = sdk.V2WorkflowRunResultTypeOCI
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultOCIDetail{
			Name:    name,
			Version: version,
			Files:   files,
		},
	}

	return nil
}

func performPuppet(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, props map[string][]string, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	puppetName, ok := props["puppet.name"]
	if !ok || len(puppetName) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property puppet.name")
	}
	puppetVersion, ok := props["puppet.version"]
	if !ok || len(puppetVersion) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property puppet.version")
	}
	runResult.Type = sdk.V2WorkflowRunResultTypePuppet
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultPuppetDetail{
			Name:    puppetName[0],
			Size:    size,
			Mode:    os.FileMode(0755),
			MD5:     fileInfo.Checksums.Md5,
			SHA1:    fileInfo.Checksums.Sha1,
			SHA256:  fileInfo.Checksums.Sha256,
			Version: puppetVersion[0],
		},
	}
	return nil
}

func performNuget(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, props map[string][]string, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	nugetID, ok := props["nuget.id"]
	if !ok || len(nugetID) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property nuget.id")
	}
	nugetVersion, ok := props["nuget.version"]
	if !ok || len(nugetVersion) != 1 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid property nuget.version")
	}
	runResult.Type = sdk.V2WorkflowRunResultTypeNuget
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultNugetDetail{
			Name:    fileName,
			Size:    size,
			Mode:    os.FileMode(0755),
			MD5:     fileInfo.Checksums.Md5,
			SHA1:    fileInfo.Checksums.Sha1,
			SHA256:  fileInfo.Checksums.Sha256,
			ID:      nugetID[0],
			Version: nugetVersion[0],
		},
	}
	return nil
}

func performSbt(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, resultType sdk.V2WorkflowRunResultType, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	runResult.Type = resultType
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultSbtDetail{
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

func performGradle(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, resultType sdk.V2WorkflowRunResultType, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	runResult.Type = resultType
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultGradleDetail{
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

func performMaven(runResult *sdk.V2WorkflowRunResult, fileInfo *grpcplugins.ArtifactoryFileInfo, resultType sdk.V2WorkflowRunResultType, fileName string) error {
	size, err := strconv.ParseInt(fileInfo.Size, 10, 64)
	if err != nil {
		return err
	}
	runResult.Type = resultType
	runResult.Detail = sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultMavenDetail{
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

// aqlSearchLimit is the ceiling a glob enumeration may return. It is deliberately not sent
// to artifactory as a limit(): alongside the property include, limit() applies to the
// database rows (one per property), not to the items, and silently returns a fraction of
// the matching artifacts while reporting that fraction as the total.
const aqlSearchLimit = 10000

func containsGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// globSupportedType returns false for the types whose path is not a globbable artifact path
// (staticFiles takes a destination folder).
func globSupportedType(resultType sdk.V2WorkflowRunResultType) bool {
	switch resultType {
	case sdk.V2WorkflowRunResultTypeStaticFiles:
		return false
	default:
		return true
	}
}

// staticPrefix returns the folder prefix common to all positive patterns of the expression,
// each truncated before its first wildcard. It only narrows the AQL search: a shorter prefix
// widens the search, the exact selection is done by the glob matcher afterwards.
func staticPrefix(expression string) string {
	var common []string
	first := true
	for _, pattern := range strings.FieldsFunc(expression, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ','
	}) {
		if strings.HasPrefix(pattern, "!") { // exclusion patterns don't widen the search
			continue
		}
		if i := strings.IndexAny(pattern, "*?["); i >= 0 {
			pattern = pattern[:i]
		}
		var segments []string
		if i := strings.LastIndex(pattern, "/"); i > 0 {
			segments = strings.Split(strings.Trim(pattern[:i], "/"), "/")
		}
		if first {
			common, first = segments, false
		} else {
			common = commonSegments(common, segments)
		}
		if len(common) == 0 {
			return ""
		}
	}
	return strings.Join(common, "/")
}

func commonSegments(a, b []string) []string {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return a[:n]
}

// repoCriteria returns the AQL criteria matching the local repositories (maturities) behind
// the virtual repository. For -cds the -generic family is also included, mirroring the 404
// fallback of the single path flow.
func repoCriteria(repository string) string {
	if prefix, ok := strings.CutSuffix(repository, "-cds"); ok {
		return fmt.Sprintf(`{"$or":[{"repo":{"$match":"%s-*"}},{"repo":{"$match":"%s-generic-*"}}]}`, repository, prefix)
	}
	return fmt.Sprintf(`{"repo":{"$match":"%s-*"}}`, repository)
}

// dockerRepoCriteria scopes the docker enumeration to the low maturity repository, the only
// one FinalizeRunResultDockerDetail reads.
func dockerRepoCriteria(repository, maturity string) string {
	return fmt.Sprintf(`{"repo":{"$eq":"%s-%s"}}`, repository, maturity)
}

// dockerPatternToPath rewrites the image:tag patterns of an expression into the folder
// layout they match in artifactory (<image>/<tag>). Only staticPrefix reads it, the glob
// keeps matching image references.
func dockerPatternToPath(expression string) string {
	patterns := strings.FieldsFunc(expression, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ','
	})
	for i, p := range patterns {
		if j := strings.LastIndex(p, ":"); j >= 0 {
			patterns[i] = p[:j] + "/" + p[j+1:]
		}
	}
	return strings.Join(patterns, " ")
}

// globCandidate is an artifact matched by the glob pattern: the path to register and the
// enumeration item it came from, carrying repo, checksums, size, dates and properties.
type globCandidate struct {
	path string
	item grpcplugins.SearchResult
}

// globEnumeration is the outcome of the AQL sweep: the matched candidates, and for docker
// the per-architecture manifests of the manifest lists, indexed by <image>/<digest>.
type globEnumeration struct {
	candidates []globCandidate
	archByPath map[string]grpcplugins.SearchResult
}

// dockerArchPath returns the <image>/<digest> folder of a per-architecture manifest, or ""
// for a tag folder. A digest folder is not a taggable image, it is the companion of a
// manifest list.
func dockerArchPath(r grpcplugins.SearchResult) string {
	folder := strings.Trim(r.Path, "/")
	i := strings.LastIndex(folder, "/")
	if i <= 0 || !strings.Contains(folder[i+1:], ":") {
		return ""
	}
	return folder
}

// virtualRepoFor returns the virtual repository a local repository (maturity) belongs to:
// the searched one, or its -generic sibling for -cds (both families are enumerated,
// mirroring the 404 fallback of the single path flow).
func virtualRepoFor(localRepo, repository string) string {
	if strings.HasPrefix(localRepo, repository+"-") {
		return repository
	}
	if prefix, ok := strings.CutSuffix(repository, "-cds"); ok {
		if generic := prefix + "-generic"; strings.HasPrefix(localRepo, generic+"-") {
			return generic
		}
	}
	return ""
}

// fileInfoFromItem rebuilds the storage-API view of a file from its AQL item: path in the
// "/dir/file" form expected downstream, uri/downloadURI on the virtual repository (the form
// stored in run results). mimeType is left empty, it has no consumer.
func fileInfoFromItem(config grpcplugins.ArtifactoryConfig, virtualRepo string, item grpcplugins.SearchResult) *grpcplugins.ArtifactoryFileInfo {
	url := config.URL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	fullPath := "/" + strings.Trim(pathpkg.Join(item.Path, item.Name), "/")
	fi := &grpcplugins.ArtifactoryFileInfo{
		Repo:        virtualRepo,
		Path:        fullPath,
		Created:     item.Created,
		CreatedBy:   item.CreatedBy,
		Size:        strconv.FormatInt(item.Size, 10),
		URI:         url + "api/storage/" + virtualRepo + fullPath,
		DownloadURI: url + virtualRepo + fullPath,
	}
	fi.Checksums.Md5 = item.ActualMd5
	fi.Checksums.Sha1 = item.ActualSha1
	fi.Checksums.Sha256 = item.Sha256
	return fi
}

// propertiesMap converts AQL item properties to the properties-API shape, preserving
// multi-valued keys.
func propertiesMap(props []grpcplugins.SearchResultProperty) map[string][]string {
	m := make(map[string][]string, len(props))
	for _, p := range props {
		m[p.Key] = append(m[p.Key], p.Value)
	}
	return m
}

// deriveCandidate returns the path to match against the glob pattern: the file path for
// file-based types, the package folder for oci and conan. It returns "" when the item must
// be ignored.
func deriveCandidate(r grpcplugins.SearchResult, resultType sdk.V2WorkflowRunResultType) string {
	var candidate string
	switch resultType {
	case sdk.V2WorkflowRunResultTypeOCI:
		candidate = strings.Trim(r.Path, "/")
	case sdk.V2WorkflowRunResultTypeDocker:
		// Same layout as oci (<image>/<tag>/manifest.json), rebuilt into the image:tag form
		// a docker path takes.
		folder := strings.Trim(r.Path, "/")
		i := strings.LastIndex(folder, "/")
		if i <= 0 {
			return ""
		}
		tag := folder[i+1:]
		if strings.Contains(tag, ":") {
			return "" // digest folder <image>/sha256:<digest>: not a taggable image
		}
		candidate = folder[:i] + ":" + tag
	case sdk.V2WorkflowRunResultTypeConan:
		folder, ok := strings.CutSuffix(strings.Trim(r.Path, "/"), "/export")
		if !ok {
			return ""
		}
		candidate = folder
	default:
		candidate = strings.Trim(pathpkg.Join(r.Path, r.Name), "/")
	}
	if candidate == "." {
		return ""
	}
	return candidate
}

// globSearchAQL builds the enumeration query. It is kept apart from enumerateGlobMatches so
// the query can be asserted in a unit test: adding a limit() here silently trims the result
// set (see aqlSearchLimit).
func globSearchAQL(criteria []string) string {
	return fmt.Sprintf(`items.find({"$and":[%s]}).include("repo","path","name","actual_md5","actual_sha1","sha256","size","created","created_by","property")`, strings.Join(criteria, ","))
}

// enumerateGlobMatches lists the artifacts matching the glob pattern in the local
// repositories behind the virtual repository. One AQL search scoped to the static prefix of
// the pattern enumerates the candidates with the data needed to build the run results
// (checksums, size, dates, properties), then the glob matcher selects them in Go:
//   - file-based types: every file is a candidate;
//   - oci: a package is the folder directly holding a manifest.json or list.manifest.json
//     (digest folders <image>/sha256:<digest> are standalone packages and are kept);
//   - docker: the same folders as oci, rebuilt into <image>:<tag> references. Digest
//     folders are indexed apart, they feed the manifests of a manifest list. The search is
//     scoped to the low maturity repository, the only one performDocker reads;
//   - conan: a package revision is the parent of the export folder holding conanmanifest.txt.
//
// The search carries no limit (see aqlSearchLimit) and is not paginated: when the results
// are trimmed by a server-side limit, the search fails rather than registering an
// incomplete set.
func (p *addRunResultPlugin) enumerateGlobMatches(ctx context.Context, artiConfig grpcplugins.ArtifactoryConfig, integ sdk.JobIntegrationsContext, repository, pattern string, resultType sdk.V2WorkflowRunResultType) (*globEnumeration, error) {
	criteria := []string{repoCriteria(repository)}
	prefixPattern := pattern
	if resultType == sdk.V2WorkflowRunResultTypeDocker {
		criteria = []string{dockerRepoCriteria(repository, integ.Get(sdk.ArtifactoryConfigPromotionLowMaturity))}
		prefixPattern = dockerPatternToPath(pattern)
	}
	if base := staticPrefix(prefixPattern); base != "" {
		criteria = append(criteria, fmt.Sprintf(`{"$or":[{"path":{"$eq":"%s"}},{"path":{"$match":"%s/*"}}]}`, base, base))
	}
	switch resultType {
	case sdk.V2WorkflowRunResultTypeOCI, sdk.V2WorkflowRunResultTypeDocker:
		criteria = append(criteria, `{"$or":[{"name":{"$eq":"manifest.json"}},{"name":{"$eq":"list.manifest.json"}}]}`)
	case sdk.V2WorkflowRunResultTypeConan:
		criteria = append(criteria, `{"name":{"$eq":"conanmanifest.txt"}}`, `{"path":{"$match":"*/export"}}`)
	default:
		criteria = append(criteria, `{"type":"file"}`)
	}
	res, err := grpcplugins.SearchItem(ctx, &p.Common, artiConfig, globSearchAQL(criteria))
	if err != nil {
		return nil, err
	}
	if res.Range.Notification != "" {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "glob search truncated by artifactory (%s): use a more specific pattern", res.Range.Notification)
	}
	if len(res.Results) >= aqlSearchLimit {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "glob search returned %d items, above the %d supported: use a more specific pattern", len(res.Results), aqlSearchLimit)
	}

	g := glob.New(pattern)
	seen := make(map[string]struct{}, len(res.Results))
	enum := globEnumeration{archByPath: map[string]grpcplugins.SearchResult{}}
	for _, r := range res.Results {
		if resultType == sdk.V2WorkflowRunResultTypeDocker {
			if arch := dockerArchPath(r); arch != "" {
				enum.archByPath[arch] = r
				continue
			}
		}
		candidate := deriveCandidate(r, resultType)
		if candidate == "" {
			continue
		}
		// the same path can exist in several local repositories (maturities)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		m, err := g.MatchString(candidate)
		if err != nil {
			return nil, err
		}
		if m != nil {
			enum.candidates = append(enum.candidates, globCandidate{path: candidate, item: r})
		}
	}
	sort.Slice(enum.candidates, func(i, j int) bool { return enum.candidates[i].path < enum.candidates[j].path })
	return &enum, nil
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
