package venom

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"gopkg.in/cheggaaa/pb.v1"
)

func runTestSuite(ts *TestSuite, bars map[string]*pb.ProgressBar, detailsLevel string) {
	l := log.WithField("v.testsuite", ts.Name)
	start := time.Now()

	d, err := dump.ToMap(ts.Vars)
	if err != nil {
		log.Errorf("err:%s", err)
	}
	ts.Templater.Add("", d)

	totalSteps := 0
	for _, tc := range ts.TestCases {
		totalSteps += len(tc.TestSteps)
	}

	runTestCases(ts, bars, detailsLevel, l)

	elapsed := time.Since(start)

	var o string
	if ts.Failures > 0 || ts.Errors > 0 {
		o = fmt.Sprintf("❌ %s", rightPad(ts.Package, " ", 47))
	} else {
		o = fmt.Sprintf("✅ %s", rightPad(ts.Package, " ", 47))
	}
	if detailsLevel == DetailsLow {
		o += fmt.Sprintf("%s", elapsed)
	}
	if detailsLevel != DetailsLow {
		bars[ts.Package].Prefix(o)
		bars[ts.Package].Finish()
	} else {
		PrintFunc("%s\n", o)
	}
}

func runTestCases(ts *TestSuite, bars map[string]*pb.ProgressBar, detailsLevel string, l *log.Entry) {
	for i, tc := range ts.TestCases {
		if tc.Skipped == 0 {
			runTestCase(ts, &tc, bars, l, detailsLevel)
			ts.TestCases[i] = tc
		}

		if len(tc.Failures) > 0 {
			ts.Failures += len(tc.Failures)
		}
		if len(tc.Errors) > 0 {
			ts.Errors += len(tc.Errors)
		}
		if tc.Skipped > 0 {
			ts.Skipped += tc.Skipped
		}
	}
}
