package action

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunParseJunitTestResultAction(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {
	var res sdk.Result
	res.Status = sdk.StatusFail

	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return res, err
	}

	p := sdk.ParameterValue(a.Parameters, "path")
	if p == "" {
		return res, errors.New("UnitTest parser: path not provided")
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return res, err
	}

	var abs string
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		abs, _ = x.RealPath(workdir.Name())
		log.Debug(ctx, "RunParseJunitTestResultAction> workdir is %s", abs)
	} else {
		abs = workdir.Name()
	}

	if !sdk.PathIsAbs(p) {
		p = filepath.Join(abs, p)
	}

	log.Debug(ctx, "RunParseJunitTestResultAction> path: %v", p)

	// Global all files matching filePath
	files, errg := afero.Glob(afero.NewOsFs(), p)

	log.Debug(ctx, "RunParseJunitTestResultAction> files: %v", files)

	if errg != nil {
		return res, errors.New("UnitTest parser: Cannot find requested files, invalid pattern")
	}

	var tests sdk.JUnitTestsSuites
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%d", len(files))+" file(s) to analyze")

	for _, f := range files {
		data, errRead := afero.ReadFile(afero.NewOsFs(), f)
		if errRead != nil {
			return res, fmt.Errorf("UnitTest parser: cannot read file %s (%s)", f, errRead)
		}

		var ftests sdk.JUnitTestsSuites
		if err := xml.Unmarshal(data, &ftests); err != nil {
			// Check if file contains testsuite only (and no testsuites)
			if s, ok := ParseTestsuiteAlone(data); ok {
				ftests.TestSuites = append(ftests.TestSuites, s)
			}
		}

		log.Debug(ctx, "found %d testsuites in %q", len(ftests.TestSuites), f)

		if len(ftests.TestSuites) == 0 {
			wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("WARNING: unable to parse %q as valid xUnit report file", filepath.Base(f)))
			continue
		}

		tests.TestSuites = append(tests.TestSuites, ftests.TestSuites...)
	}

	tests = tests.EnsureData()

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("%d", len(tests.TestSuites))+" Total Testsuite(s)")
	reasons := ComputeTestsReasons(tests)
	for _, r := range reasons {
		wk.SendLog(ctx, workerruntime.LevelInfo, r)
	}

	if err := wk.Blur(&tests); err != nil {
		return res, err
	}

	stats := tests.ComputeStats()
	if stats.TotalKO == 0 {
		res.Status = sdk.StatusSuccess
	}

	if err := wk.Client().QueueSendUnitTests(ctx, jobID, tests); err != nil {
		return res, fmt.Errorf("JUnit parse: failed to send tests details: %s", err)
	}

	return res, nil
}

func ParseTestsuiteAlone(data []byte) (sdk.JUnitTestSuite, bool) {
	var s sdk.JUnitTestSuite
	err := xml.Unmarshal([]byte(data), &s)
	if err != nil {
		return s, false
	}

	if s.Name == "" {
		return s, false
	}

	return s, true
}

func ComputeTestsReasons(s sdk.JUnitTestsSuites) []string {
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
