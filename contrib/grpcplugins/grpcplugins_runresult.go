package grpcplugins

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/pkg/errors"
)

func ComputeRunResultTests(c *actionplugin.Common, filePath string, fileContent []byte, size int64, md5, sha1, sha256 string) (*sdk.V2WorkflowRunResultDetail, int, error) {
	var ftests sdk.JUnitTestsSuites
	if err := xml.Unmarshal(fileContent, &ftests); err != nil {
		// Check if file contains testsuite only (and no testsuites)
		var s sdk.JUnitTestSuite
		if err := xml.Unmarshal([]byte(fileContent), &s); err != nil {
			Error(c, fmt.Sprintf("Unable to unmarshal junit file %q: %v.", filePath, err))
			return nil, 0, errors.New("unable to read file " + filePath)
		}

		if s.Name != "" {
			ftests.TestSuites = append(ftests.TestSuites, s)
		}
	}

	reportLogs := computeTestsReasons(ftests)
	for _, l := range reportLogs {
		Log(c, l)
	}
	ftests = ftests.EnsureData()
	stats := ftests.ComputeStats()

	_, fileName := filepath.Split(filePath)
	perm := os.FileMode(0755)

	message := fmt.Sprintf("\nStarting upload of file %q as %q \n  Size: %d, MD5: %s, sha1: %s, SHA256: %s, Mode: %v", filePath, fileName, size, md5, sha1, sha256, perm)
	Log(c, message)

	// Create run result at status "pending"
	return &sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultTestDetail{
			Name:        fileName,
			Size:        size,
			Mode:        perm,
			MD5:         md5,
			SHA1:        sha1,
			SHA256:      sha256,
			TestsSuites: ftests,
			TestStats:   stats,
		}}, stats.TotalKO, nil

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

func ComputeRunResultHelmDetail(chartName, appVersion, chartVersion string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultHelmDetail{
			Name:         chartName,
			AppVersion:   appVersion,
			ChartVersion: chartVersion,
		},
	}
}

func ComputeRunResultPythonDetail(packageName string, version string, extension string) sdk.V2WorkflowRunResultDetail {
	return sdk.V2WorkflowRunResultDetail{
		Data: sdk.V2WorkflowRunResultPythonDetail{
			Name:      packageName,
			Version:   version,
			Extension: extension,
		},
	}
}
