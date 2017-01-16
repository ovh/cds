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
		sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
		return res
	}

	files, err := filepath.Glob(p)
	if err != nil {
		res.Reason = fmt.Sprintf("UnitTest parser: Cannot find requested files, invalid pattern")
		sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
		return res
	}

	var v sdk.Tests
	for _, f := range files {
		var ftests sdk.Tests

		data, err := ioutil.ReadFile(f)
		if err != nil {
			res.Reason = fmt.Sprintf("UnitTest parser: cannot read file %s (%s)", f, err)
			sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
			return res
		}

		err = xml.Unmarshal([]byte(data), &v)
		if err != nil {
			res.Reason = fmt.Sprintf("UnitTest parser: cannot interpret file %s (%s)", f, err)
			sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
			return res
		}

		// Is it nosetests format ?
		if s, ok := parseNoseTests(data); ok {
			ftests.TestSuites = append(ftests.TestSuites, s)
		}

		v.TestSuites = append(v.TestSuites, ftests.TestSuites...)
	}
	// update global stats
	for _, s := range v.TestSuites {
		v.Total += s.Total
		v.TotalOK += (s.Total - s.Failures)
		v.TotalKO += s.Failures
		v.TotalSkipped += s.Skip
	}

	for _, s := range v.TestSuites {
		if s.Failures > 0 {
			res.Reason = fmt.Sprintf("JUnit parser: %s has %d failed tests", s.Name, s.Failures)
			sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
		}
	}

	if v.Total == 0 {
		res.Reason = "JUnit parser: No tests"
		sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
	}

	data, err := json.Marshal(v)
	if err != nil {
		res.Reason = fmt.Sprintf("JUnit parse: failed to send tests details: %s", err)
		sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
		return res
	}

	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%s/test?envName=%s", proj, app, pip, bnS, envName)
	_, code, err := sdk.Request("POST", uri, data)
	if err == nil && code > 300 {
		err = fmt.Errorf("HTTP %d", code)
	}
	if err != nil {
		res.Reason = fmt.Sprintf("JUnit parse: failed to send tests details: %s", err)
		sendLog(pbJob.ID, sdk.JUnitAction, res.Reason, pbJob.PipelineBuildID, stepOrder)
		return res
	}
	res.Status = sdk.StatusSuccess
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
