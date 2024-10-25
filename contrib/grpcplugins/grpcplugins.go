package grpcplugins

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/srerickson/checksum"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Logf(c *actionplugin.Common, s string, i ...any) {
	Log(c, fmt.Sprintf(s, i...))
}

func Log(c *actionplugin.Common, s string) {
	if c.StreamServer == nil {
		fmt.Println(s)
	} else {
		if err := c.StreamServer.Send(&actionplugin.StreamResult{Logs: s}); err != nil {
			fmt.Printf("Unable to send logs %s: %v\n", s, err)
		}
	}
}

func Warnf(c *actionplugin.Common, s string, i ...any) {
	Logf(c, WarnColor+"Warning: "+NoColor+s, i...)
}

func Warn(c *actionplugin.Common, s string) {
	Log(c, WarnColor+"Warning: "+NoColor+s)
}

func Errorf(c *actionplugin.Common, s string, i ...any) {
	Logf(c, ErrColor+"Error: "+NoColor+s, i...)
}

func Error(c *actionplugin.Common, s string) {
	Log(c, ErrColor+"Error: "+NoColor+s)
}

func Successf(c *actionplugin.Common, s string, i ...any) {
	Logf(c, SuccessColor+s+NoColor, i...)
}

func Success(c *actionplugin.Common, s string) {
	Log(c, SuccessColor+s+NoColor)
}

const (
	WarnColor    = "\033[1;33m"
	ErrColor     = "\033[1;31m"
	SuccessColor = "\033[1;32m"
	NoColor      = "\033[0m"
)

func GetRunResults(workerHTTPPort int32) ([]sdk.WorkflowRunResult, error) {
	if workerHTTPPort == 0 {
		return nil, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/run-result", workerHTTPPort), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request to get run result: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot get run result /run-result: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body on get run result /run-result: %v", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cannot get run result /run-result: HTTP %d", resp.StatusCode)
	}

	var results []sdk.WorkflowRunResult
	if err := sdk.JSONUnmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %v", err)
	}
	return results, nil
}

func GetV2CacheLink(ctx context.Context, c *actionplugin.Common, cacheKey string) (*sdk.CDNItemLinks, error) {
	path := fmt.Sprintf("/v2/cache/signature/%s/link", cacheKey)
	req, err := c.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable toworker cache signature")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body on get cache signature %s: %v", path, err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cannot get run result %s: HTTP %d", path, resp.StatusCode)
	}

	var result sdk.CDNItemLinks
	if err := sdk.JSONUnmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %v", err)
	}
	return &result, nil
}

func GetV2CacheSignature(ctx context.Context, c *actionplugin.Common, cacheKey string) (*workerruntime.CDNSignature, error) {
	path := "/v2/cache/signature/" + url.PathEscape(cacheKey)
	req, err := c.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable toworker cache signature")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body on get cache signature %s: %v", path, err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cannot get run result %s: HTTP %d", path, resp.StatusCode)
	}

	var result workerruntime.CDNSignature
	if err := sdk.JSONUnmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %v", err)
	}
	return &result, nil
}

func GetV2RunResults(ctx context.Context, c *actionplugin.Common, filter workerruntime.V2FilterRunResult) (*workerruntime.V2GetResultResponse, error) {
	btes, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(ctx, "GET", "/v2/result", bytes.NewReader(btes))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get run results")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body on get run result /v2/result: %v", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cannot get run result /v2/result: HTTP %d", resp.StatusCode)
	}

	var result workerruntime.V2GetResultResponse
	if err := sdk.JSONUnmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %v", err)
	}
	return &result, nil
}

func GetWorkerDirectories(ctx context.Context, c *actionplugin.Common) (*sdk.WorkerDirectories, error) {
	req, err := c.NewRequest(ctx, "GET", "/directories", nil)
	if err != nil {
		return nil, errors.Errorf("unable to create request to get directories: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("unable to read body on get /working-directory: %v", err)
	}

	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("cannot get working directory: HTTP %d", resp.StatusCode)
	}

	var workDir sdk.WorkerDirectories
	if err := sdk.JSONUnmarshal(body, &workDir); err != nil {
		return nil, errors.Errorf("unable to unmarshal response: %v", err)
	}
	return &workDir, nil
}

func CreateRunResult(ctx context.Context, c *actionplugin.Common, result *workerruntime.V2RunResultRequest) (*workerruntime.V2AddResultResponse, error) {
	btes, err := json.Marshal(result)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return nil, errors.WithStack(err)
	}
	req, err := c.NewRequest(ctx, http.MethodPost, "/v2/result", bytes.NewReader(btes))
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return nil, err
	}
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return nil, errors.WithStack(err)
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Wrapf(err, "unable to create run result (status code %d) %v", resp.StatusCode, string(body))
	}

	var response workerruntime.V2AddResultResponse
	if err := sdk.JSONUnmarshal(body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func RunResultsSynchronize(ctx context.Context, c *actionplugin.Common) error {
	req, err := c.NewRequest(ctx, http.MethodPost, "/v2/result/synchronize", nil)
	if err != nil {
		return err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return errors.Wrap(err, "unable to synchronize run results")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.Wrapf(err, "unable to synchronize run result (status code %d)", resp.StatusCode)
	}

	return nil
}

func UpdateRunResult(ctx context.Context, c *actionplugin.Common, result *workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
	btes, err := json.Marshal(result)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req, err := c.NewRequest(ctx, http.MethodPut, "/v2/result", bytes.NewReader(btes))
	if err != nil {
		return nil, err
	}
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("unable to update run result %s (status code %d) %v", result.RunResult.ID, resp.StatusCode, string(body))
	}

	var response workerruntime.V2UpdateResultResponse
	if err := sdk.JSONUnmarshal(body, &response); err != nil {
		return nil, errors.Wrap(err, "unable to parse run result response")
	}
	return &response, nil
}

func CreateOutput(ctx context.Context, c *actionplugin.Common, out workerruntime.OutputRequest) error {
	bts, _ := json.Marshal(out)
	r := bytes.NewReader(bts)
	req, err := c.NewRequest(ctx, "POST", "/v2/output", r)
	if err != nil {
		return sdk.WrapError(err, "unable to prepare request")
	}

	if _, err := c.DoRequest(req); err != nil {
		return sdk.WrapError(err, "unable to post output")
	}
	return nil
}

func GetJobRun(ctx context.Context, c *actionplugin.Common) (*sdk.V2WorkflowRunJob, error) {
	r, err := c.NewRequest(ctx, "GET", "/v2/jobrun", nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to prepare request")
	}

	resp, err := c.DoRequest(r)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get job run")
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to read response")
	}

	defer resp.Body.Close()

	var jobRun sdk.V2WorkflowRunJob
	if err := sdk.JSONUnmarshal(btes, &jobRun); err != nil {
		return nil, sdk.WrapError(err, "unable to read response")
	}
	return &jobRun, nil
}

func GetProjectKey(ctx context.Context, c *actionplugin.Common, keyName string) (*sdk.ProjectKey, error) {
	r, err := c.NewRequest(ctx, "GET", "/v2/key/"+keyName, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to prepare request")
	}

	resp, err := c.DoRequest(r)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get job context")
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to read response")
	}

	defer resp.Body.Close()

	var context sdk.ProjectKey
	if err := sdk.JSONUnmarshal(btes, &context); err != nil {
		return nil, sdk.WrapError(err, "unable to read response")
	}
	return &context, nil
}

func GetJobContext(ctx context.Context, c *actionplugin.Common) (*sdk.WorkflowRunJobsContext, error) {
	r, err := c.NewRequest(ctx, "GET", "/v2/context", nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to prepare request")
	}

	resp, err := c.DoRequest(r)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get job context")
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to read response")
	}

	defer resp.Body.Close()

	var context sdk.WorkflowRunJobsContext
	if err := sdk.JSONUnmarshal(btes, &context); err != nil {
		return nil, sdk.WrapError(err, "unable to read response")
	}
	return &context, nil
}

type ArtifactoryConfig struct {
	URL             string
	Token           string
	DistributionURL string
	ReleaseToken    string
}

type ArtifactoryFilePropertiesResponse struct {
	Properties map[string][]string `properties`
}

type ArtifactoryFileInfo struct {
	Repo        string    `json:"repo"`
	Path        string    `json:"path"`
	Created     time.Time `json:"created"`
	CreatedBy   string    `json:"createdBy"`
	DownloadURI string    `json:"downloadUri"`
	MimeType    string    `json:"mimeType"`
	Size        string    `json:"size"`
	Checksums   struct {
		Sha1   string `json:"sha1"`
		Md5    string `json:"md5"`
		Sha256 string `json:"sha256"`
	} `json:"checksums"`
	OriginalChecksums struct {
		Sha1   string `json:"sha1"`
		Md5    string `json:"md5"`
		Sha256 string `json:"sha256"`
	} `json:"originalChecksums"`
	URI string `json:"uri"`
}

type SearchResultResponse struct {
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	Repo         string   `json:"repo"`
	Path         string   `json:"path"`
	Name         string   `json:"name"`
	VirtualRepos []string `json:"virtual_repos"`
}

type RepositoryInfo struct {
	Rclass       string   `json:"rclass"`
	PackageType  string   `json:"packageType"`
	Repositories []string `json:"repositories"`
}

type ArtifactoryFolderInfo struct {
	Repo      string    `json:"repo"`
	Path      string    `json:"path"`
	Created   time.Time `json:"created"`
	CreatedBy string    `json:"createdBy"`
	URI       string    `json:"uri"`
	Children  []struct {
		URI    string `json:"uri"`
		Folder bool   `json:"folder"`
	} `json:"children"`
}

func GetArtifactoryFileProperties(ctx context.Context, c *actionplugin.Common, config ArtifactoryConfig, repo, path string) (map[string][]string, error) {
	if !strings.HasSuffix(config.URL, "/") {
		config.URL = config.URL + "/"
	}
	uri := config.URL + "api/storage/" + filepath.Join(repo, path) + "?properties"
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 200 {
		if resp.StatusCode == 404 {
			return make(map[string][]string), nil
		}
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory file info %s: error %d", uri, resp.StatusCode)
	}

	var res ArtifactoryFilePropertiesResponse
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory file info: %v", err)
	}

	return res.Properties, nil
}

func GetArtifactoryFileInfo(ctx context.Context, c *actionplugin.Common, config ArtifactoryConfig, repo, path string) (*ArtifactoryFileInfo, error) {
	if !strings.HasSuffix(config.URL, "/") {
		config.URL = config.URL + "/"
	}
	uri := config.URL + "api/storage/" + filepath.Join(repo, path)
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 200 {
		if resp.StatusCode != 404 {
			Error(c, string(btes))
		}
		return nil, errors.Errorf("unable to get Artifactory file info %s: error %d", uri, resp.StatusCode)
	}

	var res ArtifactoryFileInfo
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory file info: %v", err)
	}

	return &res, nil
}

func SearchItem(ctx context.Context, c *actionplugin.Common, config ArtifactoryConfig, aql string) (*SearchResultResponse, error) {
	if !strings.HasSuffix(config.URL, "/") {
		config.URL = config.URL + "/"
	}
	reader := bytes.NewReader([]byte(aql))
	uri := config.URL + "api/search/aql"
	req, err := http.NewRequestWithContext(ctx, "POST", uri, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 200 {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to search item on artifactory Artifactory: error %d", resp.StatusCode)
	}

	var res SearchResultResponse
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to read search response: %v", err)
	}

	return &res, nil
}

func GetArtifactoryRepositoryInfo(ctx context.Context, c *actionplugin.Common, config ArtifactoryConfig, repo string) (*RepositoryInfo, error) {
	if !strings.HasSuffix(config.URL, "/") {
		config.URL = config.URL + "/"
	}
	uri := config.URL + "api/repositories/" + repo
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 200 {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory folder info %s: error %d", uri, resp.StatusCode)
	}

	var res RepositoryInfo
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory folder info: %v", err)
	}

	return &res, nil
}

func GetArtifactoryFolderInfo(ctx context.Context, c *actionplugin.Common, config ArtifactoryConfig, repo, path string) (*ArtifactoryFolderInfo, error) {
	if !strings.HasSuffix(config.URL, "/") {
		config.URL = config.URL + "/"
	}
	uri := config.URL + "api/storage/" + filepath.Join(repo, path)
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	btes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 200 {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory folder info %s: error %d", uri, resp.StatusCode)
	}

	var res ArtifactoryFolderInfo
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(c, string(btes))
		return nil, errors.Errorf("unable to get Artifactory folder info: %v", err)
	}

	return &res, nil
}

func GetArtifactoryRunResults(ctx context.Context, c *actionplugin.Common, pattern string) (*workerruntime.V2GetResultResponse, error) {
	response, err := GetV2RunResults(ctx, c, workerruntime.V2FilterRunResult{Pattern: pattern})
	if err != nil {
		return nil, err
	}
	var final []sdk.V2WorkflowRunResult
	for i := range response.RunResults {
		if response.RunResults[i].ArtifactManagerIntegrationName != nil {
			final = append(final, response.RunResults[i])
		} else {
			Logf(c, "skipping artifact %s, it has not been uploaded on artifactory.", response.RunResults[i].Name())
		}
	}
	return &workerruntime.V2GetResultResponse{
		RunResults: final,
	}, nil
}

func ExtractFileInfoIntoRunResult(runResult *sdk.V2WorkflowRunResult, fi ArtifactoryFileInfo, name, resultType, localRepository, repository, maturity string) {
	runResult.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
	runResult.ArtifactManagerMetadata.Set("repository", repository) // This is the virtual repository
	runResult.ArtifactManagerMetadata.Set("maturity", maturity)
	runResult.ArtifactManagerMetadata.Set("name", name)
	runResult.ArtifactManagerMetadata.Set("type", resultType)
	runResult.ArtifactManagerMetadata.Set("path", fi.Path)
	runResult.ArtifactManagerMetadata.Set("md5", fi.Checksums.Md5)
	runResult.ArtifactManagerMetadata.Set("sha1", fi.Checksums.Sha1)
	runResult.ArtifactManagerMetadata.Set("sha256", fi.Checksums.Sha256)
	runResult.ArtifactManagerMetadata.Set("mimeType", fi.MimeType)
	runResult.ArtifactManagerMetadata.Set("createdBy", fi.CreatedBy)
	runResult.ArtifactManagerMetadata.Set("localRepository", localRepository)

	// we keep only the virtual repo in hostname
	uri := strings.Replace(fi.URI, repository+"-"+maturity, repository, 1)
	runResult.ArtifactManagerMetadata.Set("uri", uri)
	downloadURI := strings.Replace(fi.DownloadURI, repository+"-"+maturity, repository, 1)
	runResult.ArtifactManagerMetadata.Set("downloadURI", downloadURI)
}

func UploadRunResult(ctx context.Context, actplugin *actionplugin.Common, jobContext sdk.WorkflowRunJobsContext, runresultReq *workerruntime.V2RunResultRequest, fileName string, f fs.File, size int64, fileChecksum ChecksumResult) (*workerruntime.V2UpdateResultResponse, error) {
	response, err := CreateRunResult(ctx, actplugin, runresultReq)
	if err != nil {
		Error(actplugin, err.Error())
		return nil, err
	}

	// Upload the file to an artifactory or CDN
	var d time.Duration
	var runResultRequest workerruntime.V2RunResultRequest
	switch {
	case response.CDNAddress != "":
		reader, ok := f.(io.ReadSeeker)
		var item *sdk.CDNItem
		var err error
		if ok {
			item, d, err = CDNItemUpload(ctx, actplugin, response.CDNAddress, response.CDNSignature, reader)
			if err != nil {
				Error(actplugin, "An error occurred during file upload upload: "+err.Error())
				return nil, err
			}
		} else {
			// unable to cast the file
			return nil, fmt.Errorf("unable to cast reader")
		}

		// Update the run result status
		runResultRequest = workerruntime.V2RunResultRequest{RunResult: response.RunResult}
		i := sdk.CDNItemLink{CDNHttpURL: response.CDNAddress, Item: *item}
		runResultRequest.RunResult.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{
			"uri":              i.CDNHttpURL,
			"cdn_http_url":     i.CDNHttpURL,
			"cdn_id":           i.Item.ID,
			"cdn_type":         string(i.Item.Type),
			"cdn_api_ref_hash": i.Item.APIRefHash,
			"downloadURI":      fmt.Sprintf("%s/item/%s/%s/download", i.CDNHttpURL, string(i.Item.Type), i.Item.APIRefHash),
		}
		Logf(actplugin, "  CDN API Ref Hash: %s", i.Item.APIRefHash)
		Logf(actplugin, "  CDN HTTP URL: %s", i.CDNHttpURL)

	case response.RunResult.ArtifactManagerIntegrationName != nil:
		// Get integration from the local cache, or from the worker
		if jobContext.Integrations == nil || jobContext.Integrations.ArtifactManager.Name == "" {
			err := errors.New("unable to find artifactory integration")
			Errorf(actplugin, err.Error())
			return nil, err
		}

		integ := jobContext.Integrations.ArtifactManager

		repository := integ.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-cds"
		maturity := integ.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
		path := filepath.Join(
			strings.ToLower(jobContext.Git.Server),
			strings.ToLower(jobContext.Git.Repository),
			jobContext.CDS.ProjectKey,
			jobContext.CDS.Workflow,
			jobContext.CDS.Version)

		response.RunResult.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
		response.RunResult.ArtifactManagerMetadata.Set("repository", repository) // This is the virtual repository
		response.RunResult.ArtifactManagerMetadata.Set("type", "generic")
		response.RunResult.ArtifactManagerMetadata.Set("maturity", maturity)
		response.RunResult.ArtifactManagerMetadata.Set("name", fileName)
		response.RunResult.ArtifactManagerMetadata.Set("path", path)
		response.RunResult.ArtifactManagerMetadata.Set("md5", fileChecksum.Md5)
		response.RunResult.ArtifactManagerMetadata.Set("sha1", fileChecksum.Sha1)
		response.RunResult.ArtifactManagerMetadata.Set("sha256", fileChecksum.Sha256)

		reader, ok := f.(io.ReadSeeker)
		if !ok {
			// unable to cast the file
			return nil, fmt.Errorf("unable to cast reader")
		}

		Logf(actplugin, "  Artifactory URL: %s", integ.Get(sdk.ArtifactoryConfigURL))
		Logf(actplugin, "  Artifactory repository: %s", repository)

		var res *ArtifactoryUploadResult
		res, d, err = ArtifactoryItemUploadRunResult(ctx, actplugin, response.RunResult, integ, reader)
		if err != nil {
			Error(actplugin, err.Error())
			return nil, err
		}

		response.RunResult.ArtifactManagerMetadata.Set("uri", res.URI)
		response.RunResult.ArtifactManagerMetadata.Set("mimeType", res.MimeType)
		response.RunResult.ArtifactManagerMetadata.Set("downloadURI", res.DownloadURI)
		response.RunResult.ArtifactManagerMetadata.Set("createdBy", res.CreatedBy)
		response.RunResult.ArtifactManagerMetadata.Set("localRepository", res.Repo) // This contains the localrepository
		response.RunResult.ArtifactManagerMetadata.Set("path", res.Path)
		response.RunResult.ArtifactManagerMetadata.Set("name", filepath.Base(res.Path))

		runResultRequest = workerruntime.V2RunResultRequest{RunResult: response.RunResult}

		Logf(actplugin, "  Artifactory download URI: %s", res.DownloadURI)

	default:
		err := errors.Errorf("unsupported run result %s", response.RunResult.ID)
		Error(actplugin, err.Error())
		return nil, err
	}

	// Update run result
	runResultRequest.RunResult.Status = sdk.V2WorkflowRunResultStatusCompleted
	updateResponse, err := UpdateRunResult(ctx, actplugin, &runResultRequest)
	if err != nil {
		Error(actplugin, err.Error())
		return nil, err
	}

	Successf(actplugin, "  %d bytes uploaded in %.3fs", size, d.Seconds())

	if _, err := updateResponse.RunResult.GetDetail(); err != nil {
		Error(actplugin, err.Error())
		return nil, err
	}
	Logf(actplugin, "  Result %s (%s) created", updateResponse.RunResult.Name(), updateResponse.RunResult.ID)
	return updateResponse, nil
}

type ArtifactoryUploadResult struct {
	Repo        string    `json:"repo"`
	Path        string    `json:"path"`
	Created     time.Time `json:"created"`
	CreatedBy   string    `json:"createdBy"`
	DownloadURI string    `json:"downloadUri"`
	MimeType    string    `json:"mimeType"`
	Size        string    `json:"size"`
	Checksums   struct {
		Sha1   string `json:"sha1"`
		Md5    string `json:"md5"`
		Sha256 string `json:"sha256"`
	} `json:"checksums"`
	OriginalChecksums struct {
		Sha1   string `json:"sha1"`
		Md5    string `json:"md5"`
		Sha256 string `json:"sha256"`
	} `json:"originalChecksums"`
	URI string `json:"uri"`
}

func ArtifactoryItemUploadRunResult(ctx context.Context, c *actionplugin.Common, runResult *sdk.V2WorkflowRunResult, integ sdk.JobIntegrationsContext, reader io.ReadSeeker) (*ArtifactoryUploadResult, time.Duration, error) {
	rtURL := integ.Get(sdk.ArtifactoryConfigURL)
	repo := runResult.ArtifactManagerMetadata.Get("repository")
	path := runResult.ArtifactManagerMetadata.Get("path")
	filename := runResult.ArtifactManagerMetadata.Get("name")

	headers := make(map[string]string)
	headers["X-Checksum-Sha1"] = runResult.ArtifactManagerMetadata.Get("sha1")
	headers["X-Checksum-Sha256"] = runResult.ArtifactManagerMetadata.Get("sha256")
	headers["X-Checksum-MD5"] = runResult.ArtifactManagerMetadata.Get("md5")

	uploadURL := rtURL + filepath.Join(repo, path, filename)
	return ArtifactoryItemUpload(ctx, c, integ, reader, headers, uploadURL)
}

func ArtifactoryItemUpload(ctx context.Context, c *actionplugin.Common, integ sdk.JobIntegrationsContext, reader io.ReadSeeker, headers map[string]string, uploadURL string) (*ArtifactoryUploadResult, time.Duration, error) {
	t0 := time.Now()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	rtToken := integ.Get(sdk.ArtifactoryConfigToken)

	for i := 0; i < 3; i++ {
		reader.Seek(0, io.SeekStart)
		req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, reader)
		if err != nil {
			return nil, time.Since(t0), err
		}

		req.Header.Set("Authorization", "Bearer "+rtToken)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, time.Since(t0), err
		}
		if resp.StatusCode >= 200 && resp.StatusCode <= 204 {
			btes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, time.Since(t0), err
			}
			var result ArtifactoryUploadResult
			if err := json.Unmarshal(btes, &result); err != nil {
				return nil, time.Since(t0), err
			}
			return &result, time.Since(t0), nil
		} else {
			bts, err := io.ReadAll(resp.Body)
			if err != nil {
				Error(c, err.Error())
			}
			defer resp.Body.Close()
			Error(c, string(bts))
			Error(c, fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		Log(c, "retrying file upload...")
	}

	return nil, time.Since(t0), errors.New("unable to upload artifact")
}

func CDNItemUpload(ctx context.Context, c *actionplugin.Common, cdnAddr string, signature string, reader io.ReadSeeker) (*sdk.CDNItem, time.Duration, error) {
	t0 := time.Now()

	for i := 0; i < 3; i++ {
		reader.Seek(0, io.SeekStart)

		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/item/upload", cdnAddr), reader)
		if err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to prepare HTTP request: %v", err)
		}
		req.Header.Set("X-CDS-WORKER-SIGNATURE", signature)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, time.Since(t0), err
		}

		if resp.StatusCode >= 200 && resp.StatusCode <= 204 {
			btes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, time.Since(t0), err
			}
			var item sdk.CDNItem
			if err := sdk.JSONUnmarshal(btes, &item); err != nil {
				return nil, time.Since(t0), err
			}
			return &item, time.Since(t0), nil
		} else {
			bts, err := io.ReadAll(resp.Body)
			if err != nil {
				Error(c, err.Error())
			}
			if err := sdk.DecodeError(bts); err != nil {
				Error(c, err.Error())
			}
			Error(c, fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		Log(c, "retrying file upload...")
	}

	return nil, time.Since(t0), errors.New("unable to upload artifact")
}

type ChecksumResult struct {
	Md5    string
	Sha1   string
	Sha256 string
}

func checksums(ctx context.Context, c *actionplugin.Common, dir fs.FS, path ...string) (map[string]ChecksumResult, error) {
	pipe, err := checksum.NewPipe(dir, checksum.WithCtx(ctx), checksum.WithMD5(), checksum.WithSHA1(), checksum.WithSHA256())
	if err != nil {
		return nil, err
	}

	go func() {
		for _, p := range path {
			if err := pipe.Add(p); err != nil {
				Error(c, p)
			}
		}
		pipe.Close()
	}()

	var result = map[string]ChecksumResult{}

	for out := range pipe.Out() {
		md5, err := out.Sum(checksum.MD5)
		if err != nil {
			Error(c, err.Error())
			continue
		}
		sha1, err := out.Sum(checksum.SHA1)
		if err != nil {
			Error(c, err.Error())
			continue
		}
		sha256, err := out.Sum(checksum.SHA256)
		if err != nil {
			Error(c, err.Error())
			continue
		}
		result[out.Path()] = ChecksumResult{
			Md5:    hex.EncodeToString(md5),
			Sha1:   hex.EncodeToString(sha1),
			Sha256: hex.EncodeToString(sha256),
		}
	}

	return result, nil
}

func RetrieveFilesToUpload(ctx context.Context, c *actionplugin.Common, cwd, filePath string, ifNoFilesFound string) (*glob.FileResults, map[string]int64, map[string]os.FileMode, map[string]fs.File, map[string]ChecksumResult, error) {
	results, err := glob.Glob(cwd, filePath)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	dirFS := results.DirFS

	var message string
	switch len(results.Results) {
	case 0:
		message = fmt.Sprintf("No files were found with the provided path: %q. No artifacts will be uploaded.", filePath)
	case 1:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d file uploaded.", filePath, len(results.Results))
	default:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d files uploaded.", filePath, len(results.Results))
	}

	if len(results.Results) == 0 {
		switch strings.ToUpper(ifNoFilesFound) {
		case "ERROR":
			Error(c, message)
			return nil, nil, nil, nil, nil, errors.New("no files were found")
		case "WARN":
			Warn(c, message)
		default:
			Log(c, message)
		}
	} else {
		Log(c, message)
	}

	var files []string
	var sizes = map[string]int64{}
	var permissions = map[string]os.FileMode{}
	var openFiles = map[string]fs.File{}
	for _, r := range results.Results {
		files = append(files, r.Path)
		f, err := dirFS.Open(r.Path)
		if err != nil {
			Errorf(c, "unable to open file %q: %v", r.Path, err)
			continue
		}
		stat, err := f.Stat()
		if err != nil {
			Errorf(c, "unable to stat file %q: %v", r.Path, err)
			f.Close()
			continue
		}
		sizes[r.Path] = stat.Size()
		permissions[r.Path] = stat.Mode()
		openFiles[r.Path] = f
	}

	checksums, err := checksums(ctx, c, dirFS, files...)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return results, sizes, permissions, openFiles, checksums, nil
}

type IntegrationCache struct {
	cacheIntegrations     map[string]sdk.ProjectIntegration
	lockCacheIntegrations *sync.Mutex
}

func NewIntegrationCache() *IntegrationCache {
	return &IntegrationCache{
		cacheIntegrations:     make(map[string]sdk.ProjectIntegration),
		lockCacheIntegrations: new(sync.Mutex),
	}
}

func DownloadFromArtifactory(ctx context.Context, c *actionplugin.Common, integration sdk.JobIntegrationsContext, workDirs sdk.WorkerDirectories, path string, name string, mode fs.FileMode, downloadURI string) (string, int64, error) {
	if downloadURI == "" {
		return "", 0, sdk.Errorf("no downloadURI specified")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return "", 0, err
	}

	rtToken := integration.Get(sdk.ArtifactoryConfigToken)
	req.Header.Set("Authorization", "Bearer "+rtToken)

	Logf(c, "Downloading file from %s...", downloadURI)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode > 200 {
		return "", 0, sdk.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
	}

	return BodyToFile(resp, workDirs, path, name, mode)
}

func DownloadFromCDN(ctx context.Context, c *actionplugin.Common, CDNSignature string, workDirs sdk.WorkerDirectories, apirefHash, cdnType, cdnAdresse, path string, name string, mode fs.FileMode) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/item/%s/%s/download", cdnAdresse, cdnType, apirefHash), nil)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("X-CDS-WORKER-SIGNATURE", CDNSignature)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode > 200 {
		return "", 0, sdk.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
	}

	return BodyToFile(resp, workDirs, path, name, mode)
}

func BodyToFile(resp *http.Response, workDirs sdk.WorkerDirectories, path string, name string, mode fs.FileMode) (string, int64, error) {

	var destinationDir string
	if path != "" && filepath.IsAbs(path) {
		destinationDir = path
	} else if path != "" {
		destinationDir = filepath.Join(workDirs.WorkingDir, path)
	} else {
		destinationDir = workDirs.WorkingDir
	}
	destinationFile := filepath.Join(destinationDir, name)
	destinationDir = filepath.Dir(destinationFile)
	if err := os.MkdirAll(destinationDir, os.FileMode(0750)); err != nil {
		return "", 0, sdk.Errorf("unable to create directory %q :%v", destinationDir, err.Error())
	}

	fi, err := os.OpenFile(destinationFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
	if err != nil {
		return "", 0, sdk.Errorf("unable to create file %q: %v", destinationFile, err.Error())
	}

	n, err := io.Copy(fi, resp.Body)
	if err != nil {
		return "", 0, sdk.Errorf("unable to write file %q: %v", destinationFile, err.Error())
	}
	_ = resp.Body.Close()
	return destinationFile, n, nil

}

func BuildCacheURL(integ sdk.JobIntegrationsContext, projKey string, cacheKey string) string {
	return fmt.Sprintf("%s%s/.cache/%s/%s/cache.tar.gz", integ.Get(sdk.ArtifactoryConfigURL), integ.Get(sdk.ArtifactoryConfigCdsRepository), projKey, cacheKey)
}
