package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type runActionDownloadArtifactlugin struct {
	actionplugin.Common
	cacheIntegrations     map[string]sdk.ProjectIntegration
	lockCacheIntegrations *sync.Mutex
}

func main() {
	actPlugin := runActionDownloadArtifactlugin{
		cacheIntegrations:     make(map[string]sdk.ProjectIntegration),
		lockCacheIntegrations: new(sync.Mutex),
	}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

func (actPlugin *runActionDownloadArtifactlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "downloadArtifact",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "Download a build artifact that was previously uploaded in the workflow by the upload-artifact action.",
		Version:     sdk.VERSION,
	}, nil
}

// Run implements actionplugin.ActionPluginServer.
func (actPlugin *runActionDownloadArtifactlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}

	name := q.GetOptions()["name"]
	path := q.GetOptions()["path"]
	_ = path

	if err := actPlugin.perform(ctx, name, path); err != nil {
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	return res, nil
}

func (actPlugin *runActionDownloadArtifactlugin) perform(ctx context.Context, name, path string) error {
	if name == "" {
		grpcplugins.Log("No artifact name specified, downloading all artifacts")
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &actPlugin.Common)
	if err != nil {
		return err
	}

	response, err := grpcplugins.GetV2RunResults(ctx, &actPlugin.Common, workerruntime.V2FilterRunResult{Pattern: name, Type: sdk.V2WorkflowRunResultTypeGeneric, WithClearIntegration: true})
	if err != nil {
		return err
	}

	if len(response.RunResults) == 0 {
		grpcplugins.Log("Unable to find any artifacts for the associated workflow")
	}

	var nbSuccess int
	var hasError bool

	grpcplugins.Logf("Total number of files that will be downloaded: %d", len(response.RunResults))

	for _, r := range response.RunResults {
		t0 := time.Now()

		switch {
		case r.ArtifactManagerIntegrationName == nil: // download from CDN
			x, destinationFile, n, err := downloadFromCDN(ctx, &actPlugin.Common, r, response.CDNSignature, *workDirs, path)
			if err != nil {
				grpcplugins.Errorf(err.Error())
				hasError = true
				continue
			}
			grpcplugins.Logf("Artifact %q was downloaded to %s (%d bytes downloaded in %.3f seconds).", x.Name, destinationFile, n, time.Since(t0).Seconds())
		case r.ArtifactManagerIntegrationName != nil: // download from artifactory

			// Get integration from the local cache, or from the worker
			actPlugin.lockCacheIntegrations.Lock()
			integ, has := actPlugin.cacheIntegrations[*r.ArtifactManagerIntegrationName]
			if !has {
				integFromWorker, err := grpcplugins.GetIntegrationByName(ctx, &actPlugin.Common, *r.ArtifactManagerIntegrationName)
				if err != nil {
					grpcplugins.Errorf(err.Error())
					hasError = true
					actPlugin.lockCacheIntegrations.Unlock()
					continue
				}
				actPlugin.cacheIntegrations[*r.ArtifactManagerIntegrationName] = *integFromWorker
				integ = *integFromWorker
			}
			actPlugin.lockCacheIntegrations.Unlock()

			x, destinationFile, n, err := downloadFromArtifactory(ctx, &actPlugin.Common, integ, *workDirs, r, path)
			if err != nil {
				grpcplugins.Errorf(err.Error())
				hasError = true
				continue
			}
			grpcplugins.Logf("Artifact %q was downloaded to %s (%d bytes downloaded in %.3f seconds).", x.Name, destinationFile, n, time.Since(t0).Seconds())
		}
		nbSuccess++
	}

	if hasError {
		return errors.New("artifacts download failed")
	}

	grpcplugins.Logf("There were %d artifacts downloaded", nbSuccess)

	return nil
}

func downloadFromArtifactory(ctx context.Context, c *actionplugin.Common, integration sdk.ProjectIntegration, workDirs sdk.WorkerDirectories, r sdk.V2WorkflowRunResult, path string) (*sdk.V2WorkflowRunResultGenericDetail, string, int64, error) {
	downloadURI := r.ArtifactManagerMetadata.Get("downloadURI")
	if downloadURI == "" {
		return nil, "", 0, sdk.Errorf("no downloadURI specified")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return nil, "", 0, err
	}

	rtToken := integration.Config[sdk.ArtifactoryConfigToken].Value
	req.Header.Set("Authorization", "Bearer "+rtToken)

	grpcplugins.Logf("Downloading file from %s...", downloadURI)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", 0, err
	}

	if resp.StatusCode > 200 {
		return nil, "", 0, sdk.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
	}

	return bodyToFile(resp, r, workDirs, path)
}

func bodyToFile(resp *http.Response, r sdk.V2WorkflowRunResult, workDirs sdk.WorkerDirectories, path string) (*sdk.V2WorkflowRunResultGenericDetail, string, int64, error) {
	switch r.Detail.Type {
	case "V2WorkflowRunResultGenericDetail":
		x, _ := r.GetDetailAsV2WorkflowRunResultGenericDetail()
		var destinationDir string
		if path != "" && filepath.IsAbs(path) {
			destinationDir = path
		} else if path != "" {
			destinationDir = filepath.Join(workDirs.WorkingDir, path)
		} else {
			destinationDir = workDirs.WorkingDir
		}
		destinationFile := filepath.Join(destinationDir, x.Name)
		destinationDir = filepath.Dir(destinationFile)
		if err := os.MkdirAll(destinationDir, os.FileMode(0750)); err != nil {
			return nil, "", 0, sdk.Errorf("unable to create directory %q :%v", destinationDir, err.Error())
		}

		fi, err := os.OpenFile(destinationFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, x.Mode)
		if err != nil {
			return nil, "", 0, sdk.Errorf("unable to create file %q: %v", destinationFile, err.Error())
		}

		n, err := io.Copy(fi, resp.Body)
		if err != nil {
			return nil, "", 0, sdk.Errorf("unable to write file %q: %v", destinationFile, err.Error())
		}
		_ = resp.Body.Close()
		return x, destinationFile, n, nil
	}

	return nil, "", 0, sdk.Errorf("unsupported run result")
}

func downloadFromCDN(ctx context.Context, c *actionplugin.Common, r sdk.V2WorkflowRunResult, CDNSignature string, workDirs sdk.WorkerDirectories, path string) (*sdk.V2WorkflowRunResultGenericDetail, string, int64, error) {
	cdnApirefhash, has := (*r.ArtifactManagerMetadata)["cdn_api_ref_hash"]
	if !has {
		return nil, "", 0, sdk.Errorf("unable to download artifact %q (caused by: missing cdn_api_ref_hash property", r.Name())
	}

	cdnType, has := (*r.ArtifactManagerMetadata)["cdn_type"]
	if !has {
		return nil, "", 0, sdk.Errorf("unable to download artifact %q (caused by: missing cdn_type property", r.Name())
	}

	cdnAddr, has := (*r.ArtifactManagerMetadata)["cdn_http_url"]
	if !has {
		return nil, "", 0, sdk.Errorf("unable to download artifact %q (caused by: missing cdn_http_url property", r.Name())
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, cdnType, cdnApirefhash), nil)
	if err != nil {
		return nil, "", 0, err
	}

	req.Header.Set("X-CDS-WORKER-SIGNATURE", CDNSignature)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", 0, err
	}

	if resp.StatusCode > 200 {
		return nil, "", 0, sdk.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
	}

	return bodyToFile(resp, r, workDirs, path)
}
