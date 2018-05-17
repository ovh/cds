package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/venom"
)

func runParseJunitTestResultAction(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		var res sdk.Result
		res.Status = sdk.StatusFail.String()

		pip := sdk.ParameterValue(*params, "cds.pipeline")
		proj := sdk.ParameterValue(*params, "cds.project")
		app := sdk.ParameterValue(*params, "cds.application")
		envName := sdk.ParameterValue(*params, "cds.environment")
		bnS := sdk.ParameterValue(*params, "cds.buildNumber")

		p := sdk.ParameterValue(a.Parameters, "path")
		if p == "" {
			res.Reason = fmt.Sprintf("UnitTest parser: path not provided")
			sendLog(res.Reason)
			return res
		}

		files, errg := filepath.Glob(p)
		if errg != nil {
			res.Reason = fmt.Sprintf("UnitTest parser: Cannot find requested files, invalid pattern")
			sendLog(res.Reason)
			return res
		}

		var tests venom.Tests
		sendLog(fmt.Sprintf("%d", len(files)) + " file(s) to analyze")

		for _, f := range files {
			var ftests venom.Tests

			data, errRead := ioutil.ReadFile(f)
			if errRead != nil {
				res.Reason = fmt.Sprintf("UnitTest parser: cannot read file %s (%s)", f, errRead)
				sendLog(res.Reason)
				return res
			}

			var vf venom.Tests
			if err := xml.Unmarshal(data, &vf); err != nil {
				// Check if file contains testsuite only (and no testsuites)
				if s, ok := parseTestsuiteAlone(data); ok {
					ftests.TestSuites = append(ftests.TestSuites, s)
				}
				tests.TestSuites = append(tests.TestSuites, ftests.TestSuites...)
			} else {
				tests.TestSuites = append(tests.TestSuites, vf.TestSuites...)
			}
		}

		sendLog(fmt.Sprintf("%d", len(tests.TestSuites)) + " Total Testsuite(s)")
		reasons := computeStats(&res, &tests)
		for _, r := range reasons {
			sendLog(r)
		}

		data, err := json.Marshal(tests)
		if err != nil {
			res.Reason = fmt.Sprintf("JUnit parse: failed to send tests details: %s", err)
			res.Status = sdk.StatusFail.String()
			sendLog(res.Reason)
			return res
		}

		var uri string
		if w.currentJob.wJob != nil {
			uri = fmt.Sprintf("/queue/workflows/%d/test", w.currentJob.wJob.ID)
		} else {
			uri = fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/test?envName=%s", proj, app, pip, bnS, url.QueryEscape(envName))
		}

		_, code, err := sdk.Request("POST", uri, data)
		if err == nil && code > 300 {
			err = fmt.Errorf("HTTP %d", code)
		}

		if err != nil {
			res.Reason = fmt.Sprintf("JUnit parse: failed to send tests details: %s", err)
			res.Status = sdk.StatusFail.String()
			sendLog(res.Reason)
			return res
		}

		return res
	}
}

// computeStats computes failures / errors on testSuites,
// set result.Status and return a list of log to send to API
func computeStats(res *sdk.Result, v *venom.Tests) []string {
	// update global stats
	for _, ts := range v.TestSuites {
		nSkipped := 0
		for _, tc := range ts.TestCases {
			nSkipped += len(tc.Skipped)
		}
		if ts.Skipped < nSkipped {
			ts.Skipped = nSkipped
		}
		if ts.Total < len(ts.TestCases)-nSkipped {
			ts.Total = len(ts.TestCases) - nSkipped
		}
		v.Total += ts.Total
		v.TotalOK += ts.Total - ts.Failures - ts.Errors
		v.TotalKO += ts.Failures + ts.Errors
		v.TotalSkipped += ts.Skipped
	}

	var nbOK, nbKO, nbSkipped int

	reasons := []string{}
	reasons = append(reasons, fmt.Sprintf("JUnit parser: %d testsuite(s)", len(v.TestSuites)))

	for i, ts := range v.TestSuites {
		var nbKOTC, nbFailures, nbErrors, nbSkippedTC int
		if ts.Name == "" {
			ts.Name = fmt.Sprintf("TestSuite.%d", i)
		}
		reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d testcase(s)", ts.Name, len(ts.TestCases)))
		for k, tc := range ts.TestCases {
			if tc.Name == "" {
				tc.Name = fmt.Sprintf("TestCase.%d", k)
			}
			if len(tc.Failures) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d failure(s)", tc.Name, len(tc.Failures)))
				nbFailures += len(tc.Failures)
			}
			if len(tc.Errors) > 0 {
				reasons = append(reasons, fmt.Sprintf("JUnit parser: testcase %s has %d error(s)", tc.Name, len(tc.Errors)))
				nbErrors += len(tc.Errors)
			}
			if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
				nbKOTC++
			} else if len(tc.Skipped) > 0 {
				nbSkippedTC += len(tc.Skipped)
			}
			v.TestSuites[i].TestCases[k] = tc
		}
		nbOK += len(ts.TestCases) - nbKOTC
		nbKO += nbKOTC
		nbSkipped += nbSkippedTC
		if ts.Failures > nbFailures {
			nbFailures = ts.Failures
		}
		if ts.Errors > nbErrors {
			nbErrors = ts.Errors
		}

		if nbFailures > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d failure(s)", ts.Name, nbFailures))
		}
		if nbErrors > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d error(s)", ts.Name, nbErrors))
		}
		if nbKOTC > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) failed", ts.Name, nbKOTC))
		}
		if nbSkippedTC > 0 {
			reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) skipped", ts.Name, nbSkippedTC))
		}
		v.TestSuites[i] = ts
	}

	if nbKO > v.TotalKO {
		v.TotalKO = nbKO
	}

	if nbOK != v.TotalOK {
		v.TotalOK = nbOK
	}

	if nbSkipped != v.TotalSkipped {
		v.TotalSkipped = nbSkipped
	}

	if v.TotalKO+v.TotalOK != v.Total {
		v.Total = v.TotalKO + v.TotalOK + v.TotalSkipped
	}

	res.Status = sdk.StatusFail.String()
	if v.TotalKO == 0 {
		res.Status = sdk.StatusSuccess.String()
	}
	return reasons
}

func parseTestsuiteAlone(data []byte) (venom.TestSuite, bool) {
	var s venom.TestSuite
	err := xml.Unmarshal([]byte(data), &s)
	if err != nil {
		return s, false
	}

	if s.Name == "" {
		return s, false
	}

	return s, true
}
