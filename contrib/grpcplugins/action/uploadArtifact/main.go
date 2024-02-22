package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

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
		res.Details = err.Error()
		return res, err
	}

	var dirFS = os.DirFS(workDirs.WorkingDir)

	if err := actPlugin.perform(ctx, dirFS, path, ifNoFilesFound, runResultType); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
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

	checksums, err := grpcplugins.Checksums(ctx, dirFS, files...)
	if err != nil {
		return err
	}

	for _, r := range results {
		message = fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sh1: %s, SHA256: %s, Mode: %v", r.Path, r.Result, sizes[r.Path], checksums[r.Path].Md5, checksums[r.Path].Sha1, checksums[r.Path].Sha256, permissions[r.Path])
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
						MD5:    checksums[r.Path].Md5,
						SHA1:   checksums[r.Path].Sha1,
						SHA256: checksums[r.Path].Sha256,
					},
				},
			},
		}

		if _, err := grpcplugins.UploadRunResult(ctx, &actPlugin.Common, actPlugin.integrationCache, &runResultRequest, r.Result, openFiles[r.Path], sizes[r.Path], checksums[r.Path]); err != nil {
			return err
		}
	}

	return nil
}
