package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type runActionDownloadArtifactlugin struct {
	actionplugin.Common
}

func main() {
	actPlugin := runActionDownloadArtifactlugin{}
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

	if err := perform(ctx, &actPlugin.Common, name, path); err != nil {
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	return res, nil
}

func perform(ctx context.Context, c *actionplugin.Common, name, path string) error {
	if name == "" {
		grpcplugins.Log("No artifact name specified, downloading all artifacts")
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, c)
	if err != nil {
		return err
	}

	response, err := grpcplugins.GetV2RunResults(ctx, c, workerruntime.V2FilterRunResult{Pattern: name, Type: sdk.V2WorkflowRunResultTypeGeneric})
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

		if r.ArtifactManagerIntegration == nil && r.ArtifactManagerMetadata != nil { // download from CDN
			cdnApirefhash, has := (*r.ArtifactManagerMetadata)["cdn_api_ref_hash"]
			if !has {
				grpcplugins.Errorf("unable to download artifact %q (caused by: missing cdn_api_ref_hash property")
				hasError = true
				continue
			}

			cdnType, has := (*r.ArtifactManagerMetadata)["cdn_type"]
			if !has {
				grpcplugins.Errorf("unable to download artifact %q (caused by: missing cdn_type property")
				hasError = true
				continue
			}

			cdnAddr, has := (*r.ArtifactManagerMetadata)["cdn_http_url"]
			if !has {
				grpcplugins.Errorf("unable to download artifact %q (caused by: missing cdn_http_url property")
				hasError = true
				continue
			}

			req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/item/%s/%s/download", cdnAddr, cdnType, cdnApirefhash), nil)
			if err != nil {
				grpcplugins.Errorf(err.Error())
				hasError = true
				continue
			}

			req.Header.Set("X-CDS-WORKER-SIGNATURE", response.CDNSignature)

			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				grpcplugins.Errorf(err.Error())
				hasError = true
				continue
			}

			if resp.StatusCode > 200 {
				grpcplugins.Errorf("unable to download file (HTTP %d)", resp.StatusCode)
				hasError = true
				continue
			}

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
					grpcplugins.Errorf("unable to create directory %q :%v", destinationDir, err.Error())
					hasError = true
					continue
				}

				fi, err := os.OpenFile(destinationFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, x.Mode)
				if err != nil {
					grpcplugins.Errorf("unable to create file %q: %v", destinationFile, err.Error())
					hasError = true
					continue
				}

				n, err := io.Copy(fi, resp.Body)
				if err != nil {
					grpcplugins.Errorf("unable to write file %q: %v", destinationFile, err.Error())
					hasError = true
					continue
				}
				_ = resp.Body.Close()
				grpcplugins.Logf("Artifact %q was downloaded to %s (%d bytes downloaded in %.3f seconds).", x.Name, destinationFile, n, time.Since(t0).Seconds())
				nbSuccess++
			}
		}
	}

	if hasError {
		return errors.New("artifacts download failed")
	}

	grpcplugins.Logf("There were %d artifacts downloaded", nbSuccess)

	return nil
}
