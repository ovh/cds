package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
 */

type runActionUploadArtifactPlugin struct {
	actionplugin.Common
	integrationCache *grpcplugins.IntegrationCache
}

func main() {
	actPlugin := runActionUploadArtifactPlugin{
		integrationCache: grpcplugins.NewIntegrationCache(),
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

func (p *runActionUploadArtifactPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}
	path := q.GetOptions()["path"]
	ifNoFilesFound := q.GetOptions()["if-no-files-found"]

	runResultType := sdk.V2WorkflowRunResultType(sdk.V2WorkflowRunResultTypeGeneric)
	if q.GetOptions()["type"] == sdk.V2WorkflowRunResultTypeCoverage {
		runResultType = sdk.V2WorkflowRunResultType(sdk.V2WorkflowRunResultTypeCoverage)
	}

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	var dirFS = os.DirFS(workDirs.WorkingDir)

	if err := p.perform(ctx, dirFS, path, ifNoFilesFound, runResultType); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
	}

	return stream.Send(res)
}

func (actPlugin *runActionUploadArtifactPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (actPlugin *runActionUploadArtifactPlugin) perform(ctx context.Context, dirFS fs.FS, path, ifNoFilesFound string, runResultType sdk.V2WorkflowRunResultType) error {
	results, sizes, permissions, openFiles, checksums, err := grpcplugins.RetrieveFilesToUpload(ctx, &actPlugin.Common, dirFS, path, ifNoFilesFound)
	if err != nil {
		return err
	}

	for _, r := range results {
		message := fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sha1: %s, SHA256: %s, Mode: %v", r.Path, r.Result, sizes[r.Path], checksums[r.Path].Md5, checksums[r.Path].Sha1, checksums[r.Path].Sha256, permissions[r.Path])
		grpcplugins.Log(&actPlugin.Common, message)

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
						MD5:    checksums[r.Path].Md5,
						SHA1:   checksums[r.Path].Sha1,
						SHA256: checksums[r.Path].Sha256,
					},
				},
			},
		}

		if _, err := grpcplugins.UploadRunResult(ctx, &actPlugin.Common, actPlugin.integrationCache, &runResultRequest, r.Result, openFiles[r.Path], sizes[r.Path], checksums[r.Path]); err != nil {
			_ = openFiles[r.Path].Close()
			return err
		}
		_ = openFiles[r.Path].Close()
	}

	return nil
}
