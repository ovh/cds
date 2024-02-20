package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/srerickson/checksum"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
 */

type runActionUploadArtifactPlugin struct {
	actionplugin.Common
	cacheIntegrations     map[string]sdk.ProjectIntegration
	lockCacheIntegrations *sync.Mutex
}

func main() {
	actPlugin := runActionUploadArtifactPlugin{
		cacheIntegrations:     make(map[string]sdk.ProjectIntegration),
		lockCacheIntegrations: new(sync.Mutex),
	}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

func (actPlugin *runActionUploadArtifactPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "uploadArtifact",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "This uploads artifacts from your workflow allowing you to share data between jobs and store data once a workflow is complete.",
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *runActionUploadArtifactPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}
	path := q.GetOptions()["path"]
	ifNoFilesFound := q.GetOptions()["if-no-files-found"]

	runResultType := sdk.V2WorkflowRunResultType(sdk.V2WorkflowRunResultTypeGeneric)
	if q.GetOptions()["type"] == sdk.V2WorkflowRunResultTypeCoverage {
		runResultType = sdk.V2WorkflowRunResultType(sdk.V2WorkflowRunResultTypeCoverage)
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &actPlugin.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	var dirFS = os.DirFS(workDirs.WorkingDir)

	if err := actPlugin.perform(ctx, dirFS, path, ifNoFilesFound, runResultType); err != nil {
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	return res, nil
}

func (actPlugin *runActionUploadArtifactPlugin) perform(ctx context.Context, dirFS fs.FS, path, ifNoFilesFound string, runResultType sdk.V2WorkflowRunResultType) error {
	results, err := glob.Glob(dirFS, ".", path)
	if err != nil {
		return err
	}

	var message string
	switch len(results) {
	case 0:
		message = fmt.Sprintf("No files were found with the provided path: %q. No artifacts will be uploaded.", path)
	case 1:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d file uploaded.", path, len(results))
	default:
		message = fmt.Sprintf("With the provided pattern %q, there will be %d files uploaded.", path, len(results))
	}

	if len(results) == 0 {
		switch strings.ToUpper(ifNoFilesFound) {
		case "ERROR":
			grpcplugins.Error(message)
			return errors.New("no files were found")
		case "WARN":
			grpcplugins.Warn(message)
		default:
			grpcplugins.Log(message)
		}
	} else {
		grpcplugins.Log(message)
	}

	var files []string
	var sizes = map[string]int64{}
	var permissions = map[string]os.FileMode{}
	var openFiles = map[string]fs.File{}
	for _, r := range results {
		files = append(files, r.Path)
		f, err := dirFS.Open(r.Path)
		if err != nil {
			grpcplugins.Errorf("unable to open file %q: %v", r.Path, err)
			continue
		}
		stat, err := f.Stat()
		if err != nil {
			grpcplugins.Errorf("unable to stat file %q: %v", r.Path, err)
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
		return err
	}

	for _, r := range results {
		message = fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sh1: %s, SHA256: %s, Mode: %v", r.Path, r.Result, sizes[r.Path], checksums[r.Path].md5, checksums[r.Path].sha1, checksums[r.Path].sha256, permissions[r.Path])
		grpcplugins.Log(message)

		// Create run result at status "pending"
		var runResultRequest = workerruntime.V2RunResultRequest{
			RunResult: &sdk.V2WorkflowRunResult{
				IssuedAt: time.Now(),
				Type:     runResultType,
				Status:   sdk.V2WorkflowRunResultStatusPending,
				Detail: sdk.V2WorkflowRunResultDetail{
					Data: sdk.V2WorkflowRunResultGenericDetail{
						Name:   r.Result,
						Size:   sizes[r.Path],
						Mode:   permissions[r.Path],
						MD5:    checksums[r.Path].md5,
						SHA1:   checksums[r.Path].sha1,
						SHA256: checksums[r.Path].sha256,
					},
				},
			},
		}

		response, err := grpcplugins.CreateRunResult(ctx, &actPlugin.Common, &runResultRequest)
		if err != nil {
			grpcplugins.Error(err.Error())
			return err
		}

		// Upload the file to an artifactory or CDN
		var d time.Duration

		switch {
		case response.CDNAddress != "":
			reader, ok := openFiles[r.Path].(io.ReadSeeker)
			var item *sdk.CDNItem
			if ok {
				item, d, err = actPlugin.CDNItemUpload(ctx, response.CDNAddress, response.CDNSignature, reader)
				if err != nil {
					grpcplugins.Error("An error occured during file upload upload: " + err.Error())
					continue
				}

			} else {
				// unable to cast the file
				return fmt.Errorf("unable to cast reader")
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

			grpcplugins.Logf("  CDN API Ref Hash: %s", i.Item.APIRefHash)
			grpcplugins.Logf("  CDN HTTP URL: %s", i.CDNHttpURL)

		case response.RunResult.ArtifactManagerIntegrationName != nil:
			// Get integration from the local cache, or from the worker
			actPlugin.lockCacheIntegrations.Lock()
			integ, has := actPlugin.cacheIntegrations[*response.RunResult.ArtifactManagerIntegrationName]
			if !has {
				integFromWorker, err := grpcplugins.GetIntegrationByName(ctx, &actPlugin.Common, *response.RunResult.ArtifactManagerIntegrationName)
				if err != nil {
					grpcplugins.Errorf(err.Error())
					return err
				}
				actPlugin.cacheIntegrations[*response.RunResult.ArtifactManagerIntegrationName] = *integFromWorker
				integ = *integFromWorker
			}
			actPlugin.lockCacheIntegrations.Unlock()

			jobRun, err := grpcplugins.GetJobRun(ctx, &actPlugin.Common)
			if err != nil {
				grpcplugins.Error(err.Error())
				return err
			}

			jobContext, err := grpcplugins.GetJobContext(ctx, &actPlugin.Common)
			if err != nil {
				grpcplugins.Error(err.Error())
				return err
			}

			repository := integ.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value + "-cds"
			maturity := integ.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
			path := filepath.Join(jobRun.ProjectKey, jobRun.WorkflowName, jobContext.Git.SemverCurrent)

			response.RunResult.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
			response.RunResult.ArtifactManagerMetadata.Set("repository", repository) // This is the virtual repository
			response.RunResult.ArtifactManagerMetadata.Set("type", "generic")
			response.RunResult.ArtifactManagerMetadata.Set("maturity", maturity)
			response.RunResult.ArtifactManagerMetadata.Set("name", r.Result)
			response.RunResult.ArtifactManagerMetadata.Set("path", path)
			response.RunResult.ArtifactManagerMetadata.Set("md5", checksums[r.Path].md5)
			response.RunResult.ArtifactManagerMetadata.Set("sha1", checksums[r.Path].sha1)
			response.RunResult.ArtifactManagerMetadata.Set("sha256", checksums[r.Path].sha256)

			reader, ok := openFiles[r.Path].(io.ReadSeeker)
			if !ok {
				// unable to cast the file
				return fmt.Errorf("unable to cast reader")
			}

			grpcplugins.Logf("  Artifactory URL: %s", integ.Config[sdk.ArtifactoryConfigURL].Value)
			grpcplugins.Logf("  Artifactory repository: %s", repository)

			var res *ArtifactoryUploadResult
			res, d, err = actPlugin.ArtifactoryItemUpload(ctx, response.RunResult, integ, reader)
			if err != nil {
				grpcplugins.Error(err.Error())
				return err
			}

			response.RunResult.ArtifactManagerMetadata.Set("uri", res.URI)
			response.RunResult.ArtifactManagerMetadata.Set("mimeType", res.MimeType)
			response.RunResult.ArtifactManagerMetadata.Set("downloadURI", res.DownloadURI)
			response.RunResult.ArtifactManagerMetadata.Set("createdBy", res.CreatedBy)
			response.RunResult.ArtifactManagerMetadata.Set("localRepository", res.Repo) // This contains the localrepository
			response.RunResult.ArtifactManagerMetadata.Set("path", res.Path)
			response.RunResult.ArtifactManagerMetadata.Set("name", filepath.Base(res.Path))

			runResultRequest = workerruntime.V2RunResultRequest{RunResult: response.RunResult}

			grpcplugins.Logf("  Artifactory download URI: %s", res.DownloadURI)

		default:
			err := errors.Errorf("unsupported run result %s", response.RunResult.ID)
			grpcplugins.Error(err.Error())
			return err
		}
		// Update run result
		runResultRequest.RunResult.Status = sdk.V2WorkflowRunResultStatusCompleted
		updateResponse, err := grpcplugins.UpdateRunResult(ctx, &actPlugin.Common, &runResultRequest)
		if err != nil {
			grpcplugins.Error(err.Error())
			return err
		}

		grpcplugins.Logf("  %d bytes uploaded in %.3fs", sizes[r.Path], d.Seconds())

		if _, err := updateResponse.RunResult.GetDetail(); err != nil {
			grpcplugins.Error(err.Error())
			return err
		}

		grpcplugins.Logf("  Result %s (%s) created", updateResponse.RunResult.Name(), updateResponse.RunResult.ID)
	}

	return nil
}

type checksumResult struct {
	md5    string
	sha1   string
	sha256 string
}

func checksums(ctx context.Context, dir fs.FS, path ...string) (map[string]checksumResult, error) {
	pipe, err := checksum.NewPipe(dir, checksum.WithCtx(ctx), checksum.WithMD5(), checksum.WithSHA1(), checksum.WithSHA256())
	if err != nil {
		return nil, err
	}

	go func() {
		for _, p := range path {
			if err := pipe.Add(p); err != nil {
				grpcplugins.Error(p)
			}
		}
		pipe.Close()
	}()

	var result = map[string]checksumResult{}

	for out := range pipe.Out() {
		md5, err := out.Sum(checksum.MD5)
		if err != nil {
			grpcplugins.Error(err.Error())
			continue
		}
		sha1, err := out.Sum(checksum.SHA1)
		if err != nil {
			grpcplugins.Error(err.Error())
			continue
		}
		sha256, err := out.Sum(checksum.SHA256)
		if err != nil {
			grpcplugins.Error(err.Error())
			continue
		}
		result[out.Path()] = checksumResult{
			md5:    hex.EncodeToString(md5),
			sha1:   hex.EncodeToString(sha1),
			sha256: hex.EncodeToString(sha256),
		}
	}

	return result, nil
}

func (actPlugin *runActionUploadArtifactPlugin) CDNItemUpload(ctx context.Context, cdnAddr string, signature string, reader io.ReadSeeker) (*sdk.CDNItem, time.Duration, error) {
	t0 := time.Now()

	for i := 0; i < 3; i++ {
		reader.Seek(0, io.SeekStart)

		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/item/upload", cdnAddr), reader)
		if err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to prepare HTTP request: %v", err)
		}
		req.Header.Set("X-CDS-WORKER-SIGNATURE", signature)

		resp, err := actPlugin.HTTPClient.Do(req)
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
				grpcplugins.Error(err.Error())
			}
			if err := sdk.DecodeError(bts); err != nil {
				grpcplugins.Error(err.Error())
			}
			grpcplugins.Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		grpcplugins.Log("retrying file upload...")
	}

	return nil, time.Since(t0), errors.New("unable to upload artifact")
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

func (actPlugin *runActionUploadArtifactPlugin) ArtifactoryItemUpload(ctx context.Context, runResult *sdk.V2WorkflowRunResult, integ sdk.ProjectIntegration, reader io.ReadSeeker) (*ArtifactoryUploadResult, time.Duration, error) {
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

		resp, err := actPlugin.HTTPClient.Do(req)
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
				grpcplugins.Error(err.Error())
			}
			defer resp.Body.Close()
			grpcplugins.Error(string(bts))
			grpcplugins.Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}

		grpcplugins.Log("retrying file upload...")
	}

	return nil, time.Since(t0), errors.New("unable to upload artifact")
}
