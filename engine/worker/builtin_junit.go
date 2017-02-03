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
	// update global stats
	for _, ts := range v.TestSuites {
		v.Total += ts.Total
		v.TotalOK += (ts.Total - ts.Failures - ts.Errors)
		v.TotalKO += ts.Failures + ts.Errors
		v.TotalSkipped += ts.Skipped
	}

	var nbOK, nbKO int

	for _, ts := range v.TestSuites {
		var nbKOTC, nbFailures, nbErrors int
		for _, tc := range ts.TestCases {
			if len(tc.Failures) > 0 {
				res.Reason = fmt.Sprintf("JUnit parser: testcase %s has %d failure(s)", tc.Name, len(tc.Failures))
				sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
				nbFailures += len(tc.Failures)
			}
			if len(tc.Errors) > 0 {
				res.Reason = fmt.Sprintf("JUnit parser: %s testcase has %d error(s)", tc.Name, len(tc.Errors))
				sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
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
			res.Reason = fmt.Sprintf("JUnit parser: testsuite %s has %d failure(s)", ts.Name, nbFailures)
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		}
		if nbErrors > 0 {
			res.Reason = fmt.Sprintf("JUnit parser: testsuite %s has %d error(s)", ts.Name, nbErrors)
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		}
		if nbKOTC > 0 {
			res.Reason = fmt.Sprintf("JUnit parser: testsuite %s has %d test(s) failed", ts.Name, nbKOTC)
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		}
	}

	if nbKO > v.TotalKO {
		v.TotalKO = nbKO
	}

	if nbOK > v.TotalOK {
		v.TotalOK = nbOK
	}

	if nbOK+nbKO > v.Total {
		v.Total = nbOK + nbKO
	}

	res.Status = sdk.StatusFail
	if v.TotalKO == 0 {
		res.Status = sdk.StatusSuccess
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
