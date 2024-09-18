package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type junitPlugin struct {
	actionplugin.Common
}

func (actPlugin *junitPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "junit",
		Author:      "Steven GUIHEUX <steven.guiheux@ovhcloud.com>",
		Description: `This action upload and parse a junit report`,
		Version:     sdk.VERSION,
	}, nil
}

func (p *junitPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	filePath := q.GetOptions()["path"]

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	if err := p.perform(ctx, workDirs.WorkingDir, filePath); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)

}

func (actPlugin *junitPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (actPlugin *junitPlugin) perform(ctx context.Context, dirFS, filePath string) error {
	results, sizes, permissions, openFiles, checksums, err := grpcplugins.RetrieveFilesToUpload(ctx, &actPlugin.Common, dirFS, filePath, "ERROR")
	if err != nil {
		return err
	}

	jobCtx, err := grpcplugins.GetJobContext(ctx, &actPlugin.Common)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to get job context: %v", err))
	}

	testFailed := 0
	for _, r := range results {
		bts, err := os.ReadFile(r.Path)
		if err != nil {
			_ = openFiles[r.Path].Close()
			return errors.New(fmt.Sprintf("Unable to read file %q: %v.", r.Path, err))
		}

		runResultRequest, nbFailed, err := createRunResult(&actPlugin.Common, bts, r.Path, sizes[r.Path], checksums[r.Path], permissions[r.Path])
		if err != nil {
			_ = openFiles[r.Path].Close()
			return err
		}
		testFailed += nbFailed

		if _, err := grpcplugins.UploadRunResult(ctx, &actPlugin.Common, *jobCtx, runResultRequest, r.Result, openFiles[r.Path], sizes[r.Path], checksums[r.Path]); err != nil {
			_ = openFiles[r.Path].Close()
			return err
		}
		_ = openFiles[r.Path].Close()
	}

	if testFailed == 1 {
		return fmt.Errorf("there is 1 test failed")
	} else if testFailed > 1 {
		return fmt.Errorf("there are %d tests failed", testFailed)
	}
	return nil
}

func createRunResult(p *actionplugin.Common, fileContent []byte, filePath string, size int64, checksum grpcplugins.ChecksumResult, perm fs.FileMode) (*workerruntime.V2RunResultRequest, int, error) {
	runResult := workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypeTest,
			Status:   sdk.V2WorkflowRunResultStatusPending,
		},
	}

	detail, nbKo, err := grpcplugins.ComputeRunResultTestsDetail(p, filePath, fileContent, size, checksum.Md5, checksum.Sha1, checksum.Sha256)
	if err != nil {
		return nil, 0, err
	}
	runResult.RunResult.Detail = *detail

	// Create run result at status "pending"
	return &runResult, nbKo, nil
}

func main() {
	actPlugin := junitPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
