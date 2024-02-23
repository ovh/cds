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

	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/pkg/errors"
	"github.com/srerickson/checksum"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Logf(s string, i ...any) {
	fmt.Println(fmt.Sprintf(s, i...))
}

func Println(a ...interface{}) {
	fmt.Println(a...)
}

func Log(s string) {
	fmt.Println(s)
}

func Warnf(s string, i ...any) {
	Logf(WarnColor+"Warning: "+NoColor+s, i...)
}

func Warn(s string) {
	Log(WarnColor + "Warning: " + NoColor + s)
}

func Errorf(s string, i ...any) {
	Logf(ErrColor+"Error: "+NoColor+s, i...)
}

func Error(s string) {
	Log(ErrColor + "Error: " + NoColor + s)
}

func Successf(s string, i ...any) {
	Logf(SuccessColor+s+NoColor, i...)
}

func Success(s string) {
	Log(SuccessColor + s + NoColor)
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
		return nil, errors.WithStack(err)
	}
	Logf("Result create: %s", string(btes))
	req, err := c.NewRequest(ctx, http.MethodPost, "/v2/result", bytes.NewReader(btes))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
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

func GetIntegrationByName(ctx context.Context, c *actionplugin.Common, name string) (*sdk.ProjectIntegration, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, fmt.Sprintf("/v2/integrations/%s", url.QueryEscape(name)), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get integration")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("unable to get integration (status code %d) %v", resp.StatusCode, string(body))
	}

	var response sdk.ProjectIntegration
	if err := sdk.JSONUnmarshal(body, &response); err != nil {
		return nil, errors.Wrap(err, "unable to parse response")
	}
	return &response, nil

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
		Error(string(btes))
		return nil, errors.Errorf("unable to get Artifactory file info %s: error %d", uri, resp.StatusCode)
	}

	var res ArtifactoryFileInfo
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(string(btes))
		return nil, errors.Errorf("unable to get Artifactory file info: %v", err)
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
		Error(string(btes))
		return nil, errors.Errorf("unable to get Artifactory folder info %s: error %d", uri, resp.StatusCode)
	}

	var res ArtifactoryFolderInfo
	if err := json.Unmarshal(btes, &res); err != nil {
		Error(string(btes))
		return nil, errors.Errorf("unable to get Artifactory folder info: %v", err)
	}

	return &res, nil
}

func GetArtifactoryRunResults(ctx context.Context, c *actionplugin.Common, pattern string) (*workerruntime.V2GetResultResponse, error) {
	response, err := GetV2RunResults(ctx, c, workerruntime.V2FilterRunResult{Pattern: pattern, WithClearIntegration: true})
	if err != nil {
		return nil, err
	}
	var final []sdk.V2WorkflowRunResult
	for i := range response.RunResults {
		if response.RunResults[i].ArtifactManagerIntegrationName != nil {
			final = append(final, response.RunResults[i])
		}
	}
	return &workerruntime.V2GetResultResponse{
		RunResults: final,
	}, nil
}

func PromoteArtifactoryRunResult(ctx context.Context, c *actionplugin.Common, r sdk.V2WorkflowRunResult, promotionType sdk.WorkflowRunResultPromotionType, maturity string, props *utils.Properties) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	integration, err := GetIntegrationByName(ctx, c, *r.ArtifactManagerIntegrationName)
	if err != nil {
		return err
	}

	rtConfig := ArtifactoryConfig{
		URL:   integration.Config[sdk.ArtifactoryConfigURL].Value,
		Token: integration.Config[sdk.ArtifactoryConfigToken].Value,
	}

	artifactClient, err := artifact_manager.NewClient("artifactory", rtConfig.URL, rtConfig.Token)
	if err != nil {
		return errors.Errorf("Failed to create artifactory client: %v", err)
	}

	if r.DataSync == nil {
		r.DataSync = &sdk.WorkflowRunResultSync{}
	}

	latestPromotion := r.DataSync.LatestPromotionOrRelease()
	currentMaturity := integration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
	if latestPromotion != nil {
		currentMaturity = latestPromotion.ToMaturity
	}

	if maturity == "" {
		maturity = integration.Config[sdk.ArtifactoryConfigPromotionHighMaturity].Value
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
		Path:     strings.TrimPrefix(filepath.Dir(r.ArtifactManagerMetadata.Get("path")), "/"), // strip the first "/" and remove "/manifest.json"
	}

	switch r.Type {
	case "docker":
		if err := art.PromoteDockerImage(ctx, artifactClient, data, newPromotion.FromMaturity, newPromotion.ToMaturity, props, false); err != nil {
			return errors.Errorf("unable to promote docker image: %s to %s: %v", data.Name, newPromotion.ToMaturity, err)
		}
	default:
		if err := art.PromoteFile(artifactClient, data, newPromotion.FromMaturity, newPromotion.ToMaturity, props, false); err != nil {
			return errors.Errorf("unable to promote file: %s: %v", data.Name, err)
		}
	}

	switch promotionType {
	case sdk.WorkflowRunResultPromotionTypePromote:
		r.Status = sdk.V2WorkflowRunResultStatusPromoted
		r.DataSync.Promotions = append(r.DataSync.Promotions, newPromotion)
	case sdk.WorkflowRunResultPromotionTypeRelease:
		r.Status = sdk.V2WorkflowRunResultStatusReleased
		r.DataSync.Releases = append(r.DataSync.Releases, newPromotion)
	}

	// Update metadata
	r.ArtifactManagerMetadata.Set("localRepository", r.ArtifactManagerMetadata.Get("repository")+"-"+newPromotion.ToMaturity)
	r.ArtifactManagerMetadata.Set("maturity", newPromotion.ToMaturity)

	if _, err := UpdateRunResult(ctx, c, &workerruntime.V2RunResultRequest{RunResult: &r}); err != nil {
		return err
	}

	return nil
}

func UploadRunResult(ctx context.Context, actplugin *actionplugin.Common, integrationCache *IntegrationCache, runresultReq *workerruntime.V2RunResultRequest, fileName string, f fs.File, size int64, fileChecksum ChecksumResult) (*workerruntime.V2UpdateResultResponse, error) {
	response, err := CreateRunResult(ctx, actplugin, runresultReq)
	if err != nil {
		Error(err.Error())
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
				Error("An error occured during file upload upload: " + err.Error())
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
		}
		Logf("  CDN API Ref Hash: %s", i.Item.APIRefHash)
		Logf("  CDN HTTP URL: %s", i.CDNHttpURL)

	case response.RunResult.ArtifactManagerIntegrationName != nil:
		// Get integration from the local cache, or from the worker
		integrationCache.lockCacheIntegrations.Lock()
		integ, has := integrationCache.cacheIntegrations[*response.RunResult.ArtifactManagerIntegrationName]
		if !has {
			integFromWorker, err := GetIntegrationByName(ctx, actplugin, *response.RunResult.ArtifactManagerIntegrationName)
			if err != nil {
				Errorf(err.Error())
				return nil, err
			}
			integrationCache.cacheIntegrations[*response.RunResult.ArtifactManagerIntegrationName] = *integFromWorker
			integ = *integFromWorker
		}
		integrationCache.lockCacheIntegrations.Unlock()
		jobRun, err := GetJobRun(ctx, actplugin)
		if err != nil {
			Error(err.Error())
			return nil, err
		}

		jobContext, err := GetJobContext(ctx, actplugin)
		if err != nil {
			Error(err.Error())
			return nil, err
		}

		repository := integ.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value + "-cds"
		maturity := integ.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
		path := filepath.Join(jobRun.ProjectKey, jobRun.WorkflowName, jobContext.Git.SemverCurrent)

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

		Logf("  Artifactory URL: %s", integ.Config[sdk.ArtifactoryConfigURL].Value)
		Logf("  Artifactory repository: %s", repository)

		var res *ArtifactoryUploadResult
		res, d, err = ArtifactoryItemUpload(ctx, actplugin, response.RunResult, integ, reader)
		if err != nil {
			Error(err.Error())
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

		Logf("  Artifactory download URI: %s", res.DownloadURI)

	default:
		err := errors.Errorf("unsupported run result %s", response.RunResult.ID)
		Error(err.Error())
		return nil, err
	}

	// Update run result
	runResultRequest.RunResult.Status = sdk.V2WorkflowRunResultStatusCompleted
	updateResponse, err := UpdateRunResult(ctx, actplugin, &runResultRequest)
	if err != nil {
		Error(err.Error())
		return nil, err
	}

	Logf("  %d bytes uploaded in %.3fs", size, d.Seconds())

	if _, err := updateResponse.RunResult.GetDetail(); err != nil {
		Error(err.Error())
		return nil, err
	}
	Logf("  Result %s (%s) created", updateResponse.RunResult.Name(), updateResponse.RunResult.ID)
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

func ArtifactoryItemUpload(ctx context.Context, c *actionplugin.Common, runResult *sdk.V2WorkflowRunResult, integ sdk.ProjectIntegration, reader io.ReadSeeker) (*ArtifactoryUploadResult, time.Duration, error) {
	t0 := time.Now()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	rtURL := integ.Config[sdk.ArtifactoryConfigURL].Value
	rtToken := integ.Config[sdk.ArtifactoryConfigToken].Value

	repo := runResult.ArtifactManagerMetadata.Get("repository")
	path := runResult.ArtifactManagerMetadata.Get("path")
	filename := runResult.ArtifactManagerMetadata.Get("name")

	for i := 0; i < 3; i++ {
		reader.Seek(0, io.SeekStart)
		req, err := http.NewRequestWithContext(ctx, "PUT", rtURL+filepath.Join(repo, path, filename), reader)
		if err != nil {
			return nil, time.Since(t0), err
		}

		req.Header.Set("Authorization", "Bearer "+rtToken)
		req.Header.Set("X-Checksum-Sha1", runResult.ArtifactManagerMetadata.Get("sha1"))
		req.Header.Set("X-Checksum-Sha256", runResult.ArtifactManagerMetadata.Get("sha256"))
		req.Header.Set("X-Checksum-MD5", runResult.ArtifactManagerMetadata.Get("md5"))

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
				Error(err.Error())
			}
			defer resp.Body.Close()
			Error(string(bts))
			Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		Log("retrying file upload...")
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
				Error(err.Error())
			}
			if err := sdk.DecodeError(bts); err != nil {
				Error(err.Error())
			}
			Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		Log("retrying file upload...")
	}

	return nil, time.Since(t0), errors.New("unable to upload artifact")
}

type ChecksumResult struct {
	Md5    string
	Sha1   string
	Sha256 string
}

func checksums(ctx context.Context, dir fs.FS, path ...string) (map[string]ChecksumResult, error) {
	pipe, err := checksum.NewPipe(dir, checksum.WithCtx(ctx), checksum.WithMD5(), checksum.WithSHA1(), checksum.WithSHA256())
	if err != nil {
		return nil, err
	}

	go func() {
		for _, p := range path {
			if err := pipe.Add(p); err != nil {
				Error(p)
			}
		}
		pipe.Close()
	}()

	var result = map[string]ChecksumResult{}

	for out := range pipe.Out() {
		md5, err := out.Sum(checksum.MD5)
		if err != nil {
			Error(err.Error())
			continue
		}
		sha1, err := out.Sum(checksum.SHA1)
		if err != nil {
			Error(err.Error())
			continue
		}
		sha256, err := out.Sum(checksum.SHA256)
		if err != nil {
			Error(err.Error())
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

func RetrieveFilesToUpload(ctx context.Context, dirFS fs.FS, filePath string, ifNoFilesFound string) (glob.Results, map[string]int64, map[string]os.FileMode, map[string]fs.File, map[string]ChecksumResult, error) {
	results, err := glob.Glob(dirFS, ".", filePath)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var message string
	switch len(results) {
	case 0:
		message = fmt.Sprintf("No files were found with the provided path: %q. No artifacts will be uploaded.", filePath)
	case 1:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d file uploaded.", filePath, len(results))
	default:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d files uploaded.", filePath, len(results))
	}

	if len(results) == 0 {
		switch strings.ToUpper(ifNoFilesFound) {
		case "ERROR":
			Error(message)
			return nil, nil, nil, nil, nil, errors.New("no files were found")
		case "WARN":
			Warn(message)
		default:
			Log(message)
		}
	} else {
		Log(message)
	}

	var files []string
	var sizes = map[string]int64{}
	var permissions = map[string]os.FileMode{}
	var openFiles = map[string]fs.File{}
	for _, r := range results {
		files = append(files, r.Path)
		f, err := dirFS.Open(r.Path)
		if err != nil {
			Errorf("unable to open file %q: %v", r.Path, err)
			continue
		}
		stat, err := f.Stat()
		if err != nil {
			Errorf("unable to stat file %q: %v", r.Path, err)
			f.Close()
			continue
		}
		defer f.Close()
		sizes[r.Path] = stat.Size()
		permissions[r.Path] = stat.Mode()
		openFiles[r.Path] = f
	}

	checksums, err := checksums(ctx, dirFS, files...)
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
