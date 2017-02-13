package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ovh/cds/sdk"
)

func runParseJunitTestResultAction(a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	var res sdk.Result
	res.Status = sdk.StatusFail

	// Retrieve build info
	var proj, app, pip, bnS, envName string
	for _, p := range pbJob.Parameters {
		switch p.Name {
		case "cds.pipeline":
			pip = p.Value
			break
		case "cds.project":
			proj = p.Value
			break
		case "cds.application":
			app = p.Value
			break
		case "cds.buildNumber":
			bnS = p.Value
			break
		case "cds.environment":
			envName = p.Value
			break
		}
	}

	var p string
	for _, a := range a.Parameters {
		if a.Name == "path" {
			p = a.Value
			break
		}
	}

	if p == "" {
		res.Reason = fmt.Sprintf("UnitTest parser: path not provided")
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	files, err := filepath.Glob(p)
	if err != nil {
		res.Reason = fmt.Sprintf("UnitTest parser: Cannot find requested files, invalid pattern")
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	var v sdk.Tests
	for _, f := range files {
		var ftests sdk.Tests

		data, errRead := ioutil.ReadFile(f)
		if errRead != nil {
			res.Reason = fmt.Sprintf("UnitTest parser: cannot read file %s (%s)", f, errRead)
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
			return res
		}

		err = xml.Unmarshal([]byte(data), &v)
		if err != nil {
			res.Reason = fmt.Sprintf("UnitTest parser: cannot interpret file %s (%s)", f, err)
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
			return res
		}

		// Is it nosetests format ?
		if s, ok := parseNoseTests(data); ok {
			ftests.TestSuites = append(ftests.TestSuites, s)
		}

		v.TestSuites = append(v.TestSuites, ftests.TestSuites...)
	}

	reasons := computeStats(&res, &v)
	for _, r := range reasons {
		sendLog(pbJob.ID, r, pbJob.PipelineBuildID, stepOrder, false)
	}

	data, err := json.Marshal(v)
	if err != nil {
		res.Reason = fmt.Sprintf("JUnit parse: failed to send tests details: %s", err)
		res.Status = sdk.StatusFail
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/test?envName=%s", proj, app, pip, bnS, envName)
	_, code, err := sdk.Request("POST", uri, data)
	if err == nil && code > 300 {
		err = fmt.Errorf("HTTP %d", code)
	}

	if err != nil {
		res.Reason = fmt.Sprintf("JUnit parse: failed to send tests details: %s", err)
		res.Status = sdk.StatusFail
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
	}

	return res
}

// computeStats computes failures / errors on testSuites,
// set result.Status and return a list of log to send to API
func computeStats(res *sdk.Result, v *sdk.Tests) []string {
	// update global stats
	for _, ts := range v.TestSuites {
		nSkipped := 0
		for _, tc := range ts.TestCases {
			nSkipped += tc.Skipped
		}
		if ts.Skipped < nSkipped {
			ts.Skipped = nSkipped
		}
		if ts.Total < len(ts.TestCases)-nSkipped {
			ts.Total = len(ts.TestCases) - nSkipped
		}
		v.Total += ts.Total
		v.TotalOK += (ts.Total - ts.Failures - ts.Errors)
		v.TotalKO += ts.Failures + ts.Errors
		v.TotalSkipped += ts.Skipped
	}

	var nbOK, nbKO int

	reasons := []string{}

	reasons = append(reasons, fmt.Sprintf("JUnit parser: %d testsuite(s)", len(v.TestSuites)))

	for _, ts := range v.TestSuites {
		var nbKOTC, nbFailures, nbErrors int
		reasons = append(reasons, fmt.Sprintf("JUnit parser: testsuite %s has %d testcase(s)", ts.Name, len(ts.TestCases)))
		for _, tc := range ts.TestCases {
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
			}
		}
		nbOK += len(ts.TestCases) - nbKOTC
		nbKO += nbKOTC
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
	}

	if nbKO > v.TotalKO {
		v.TotalKO = nbKO
	}

	if nbOK > v.TotalOK {
		v.TotalOK = nbOK
	}

	if v.TotalKO+v.TotalOK > v.Total {
		v.Total = v.TotalKO + v.TotalOK
	}

	res.Status = sdk.StatusFail
	if v.TotalKO == 0 {
		res.Status = sdk.StatusSuccess
	}
	return reasons
}

func parseNoseTests(data []byte) (sdk.TestSuite, bool) {
	var s sdk.TestSuite
	err := xml.Unmarshal([]byte(data), &s)
	if err != nil {
		return s, false
	}

	if s.Name == "" {
		return s, false
	}

	return s, true
}
