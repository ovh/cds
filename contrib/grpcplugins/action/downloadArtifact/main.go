package main

import (
	"context"
	"errors"
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

func (actPlugin *runActionDownloadArtifactlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (p *runActionDownloadArtifactlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	name := q.GetOptions()["name"]
	path := q.GetOptions()["path"]
	_ = path

	if err := p.perform(ctx, name, path); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
	}

	return stream.Send(res)
}

func (actPlugin *runActionDownloadArtifactlugin) perform(ctx context.Context, name, path string) error {
	if name == "" {
		grpcplugins.Log(&actPlugin.Common, "No artifact name specified, downloading all artifacts")
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &actPlugin.Common)
	if err != nil {
		return err
	}

	response, err := grpcplugins.GetV2RunResults(ctx, &actPlugin.Common, workerruntime.V2FilterRunResult{Pattern: name, Type: []sdk.V2WorkflowRunResultType{sdk.V2WorkflowRunResultTypeCoverage, sdk.V2WorkflowRunResultTypeGeneric}})
	if err != nil {
		return err
	}

	filteredRunResults := make([]sdk.V2WorkflowRunResult, 0)
	for _, r := range response.RunResults {
		if r.Type == sdk.V2WorkflowRunResultTypeGeneric || r.Type == sdk.V2WorkflowRunResultTypeCoverage {
			filteredRunResults = append(filteredRunResults, r)
		}
	}

	if len(filteredRunResults) == 0 {
		grpcplugins.Log(&actPlugin.Common, "Unable to find any artifacts for the associated workflow")
	}

	var nbSuccess int
	var hasError bool

	jobCtx, err := grpcplugins.GetJobContext(ctx, &actPlugin.Common)
	if err != nil {
		grpcplugins.Errorf(&actPlugin.Common, err.Error())
		return errors.New("unable to retrieve job context")
	}

	grpcplugins.Logf(&actPlugin.Common, "Total number of files that will be downloaded: %d", len(response.RunResults))

	for _, r := range filteredRunResults {
		t0 := time.Now()

		if r.Detail.Type != "V2WorkflowRunResultGenericDetail" {
			return sdk.Errorf("unsupported run result")
		}
		x, _ := r.GetDetailAsV2WorkflowRunResultGenericDetail()
		switch {
		case r.ArtifactManagerIntegrationName == nil: // download from CDN

			cdnApirefhash, has := (*r.ArtifactManagerMetadata)["cdn_api_ref_hash"]
			if !has {
				return sdk.Errorf("unable to download artifact %q (caused by: missing cdn_api_ref_hash property", r.Name())
			}

			cdnType, has := (*r.ArtifactManagerMetadata)["cdn_type"]
			if !has {
				return sdk.Errorf("unable to download artifact %q (caused by: missing cdn_type property", r.Name())
			}

			cdnAddr, has := (*r.ArtifactManagerMetadata)["cdn_http_url"]
			if !has {
				return sdk.Errorf("unable to download artifact %q (caused by: missing cdn_http_url property", r.Name())
			}

			destinationFile, n, err := grpcplugins.DownloadFromCDN(ctx, &actPlugin.Common, r, response.CDNSignature, *workDirs, cdnApirefhash, cdnType, cdnAddr, path, x.Name, x.Mode)
			if err != nil {
				grpcplugins.Errorf(&actPlugin.Common, err.Error())
				hasError = true
				continue
			}
			grpcplugins.Logf(&actPlugin.Common, "Artifact %q was downloaded to %s (%d bytes downloaded in %.3f seconds).", x.Name, destinationFile, n, time.Since(t0).Seconds())
		case r.ArtifactManagerIntegrationName != nil: // download from artifactory

			// Get integration from the local cache, or from the worker
			if jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager.Name == "" {
				grpcplugins.Errorf(&actPlugin.Common, "unable to retrieve artifactory integration")
				return errors.New("artifactory integration not found")
			}
			integ := jobCtx.Integrations.ArtifactManager
			destinationFile, n, err := grpcplugins.DownloadFromArtifactory(ctx, &actPlugin.Common, integ, *workDirs, path, x.Name, x.Mode, r.ArtifactManagerMetadata.Get("downloadURI"))
			if err != nil {
				grpcplugins.Errorf(&actPlugin.Common, err.Error())
				hasError = true
				continue
			}
			grpcplugins.Successf(&actPlugin.Common, "Artifact %q was downloaded to %s (%d bytes downloaded in %.3f seconds).", x.Name, destinationFile, n, time.Since(t0).Seconds())
		}
		nbSuccess++
	}

	if hasError {
		return errors.New("artifacts download failed")
	}

	grpcplugins.Logf(&actPlugin.Common, "There were %d artifacts downloaded", nbSuccess)

	return nil
}
