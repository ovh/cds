package main

import (
	"context"
	"encoding/xml"
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

	var dirFS = os.DirFS(workDirs.WorkingDir)

	if err := p.perform(ctx, dirFS, filePath); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)

}

func (actPlugin *junitPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (actPlugin *junitPlugin) perform(ctx context.Context, dirFS fs.FS, filePath string) error {
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

		runResultRequest, nbFailed, err := createRunResult(&actPlugin.Common, bts, r.Path, r.Result, sizes[r.Path], checksums[r.Path], permissions[r.Path])
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

func createRunResult(p *actionplugin.Common, fileContent []byte, filePath string, fileName string, size int64, checksum grpcplugins.ChecksumResult, perm fs.FileMode) (*workerruntime.V2RunResultRequest, int, error) {
	var ftests sdk.JUnitTestsSuites
	if err := xml.Unmarshal(fileContent, &ftests); err != nil {
		// Check if file contains testsuite only (and no testsuites)
		var s sdk.JUnitTestSuite
		if err := xml.Unmarshal([]byte(fileContent), &s); err != nil {
			grpcplugins.Error(p, fmt.Sprintf("Unable to unmarshal junit file %q: %v.", filePath, err))
			return nil, 0, errors.New("unable to read file " + filePath)
		}

		if s.Name != "" {
			ftests.TestSuites = append(ftests.TestSuites, s)
		}
	}

	reportLogs := computeTestsReasons(ftests)
	for _, l := range reportLogs {
		grpcplugins.Log(p, l)
	}
	ftests = ftests.EnsureData()
	stats := ftests.ComputeStats()

	message := fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sha1: %s, SHA256: %s, Mode: %v", filePath, fileName, size, checksum.Md5, checksum.Sha1, checksum.Sha256, perm)
	grpcplugins.Log(p, message)

	// Create run result at status "pending"
	return &workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypeTest,
			Status:   sdk.V2WorkflowRunResultStatusPending,
			Detail: sdk.V2WorkflowRunResultDetail{
				Data: sdk.V2WorkflowRunResultTestDetail{
					Name:        fileName,
					Size:        size,
					Mode:        perm,
					MD5:         checksum.Md5,
					SHA1:        checksum.Sha1,
					SHA256:      checksum.Sha256,
					TestsSuites: ftests,
					TestStats:   stats,
				},
			},
		},
	}, stats.TotalKO, nil
}

func main() {
	actPlugin := junitPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}

func computeTestsReasons(s sdk.JUnitTestsSuites) []string {
	reasons := []string{fmt.Sprintf("JUnit parser: %d testsuite(s)", len(s.TestSuites))}
	for _, ts := range s.TestSuites {
		reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d testcase(s)", ts.Name, len(ts.TestCases)))
		for _, tc := range ts.TestCases {
			if len(tc.Failures) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d failure(s)", tc.Name, len(tc.Failures)))
			}
			if len(tc.Errors) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d error(s)", tc.Name, len(tc.Errors)))
			}
			if len(tc.Skipped) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d test(s) skipped", tc.Name, len(tc.Skipped)))
			}
		}
		if ts.Failures > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d failure(s)", ts.Name, ts.Failures))
		}
		if ts.Errors > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d error(s)", ts.Name, ts.Errors))
		}
		if ts.Failures+ts.Errors > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) failed", ts.Name, ts.Failures+ts.Errors))
		}
		if ts.Skipped > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) skipped", ts.Name, ts.Skipped))
		}
	}
	return reasons
}
