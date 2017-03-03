package venom

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
)

func runTestCase(ts *TestSuite, tc *TestCase, bars map[string]*pb.ProgressBar, l *log.Entry, detailsLevel string) {
	l.Debugf("Init context")
	tcc, errContext := getContextWrap(tc)
	if errContext != nil {
		tc.Errors = append(tc.Errors, Failure{Value: errContext.Error()})
		return
	}
	if err := tcc.Init(); err != nil {
		tc.Errors = append(tc.Errors, Failure{Value: err.Error()})
		return
	}
	defer tcc.Close()

	l = l.WithField("x.testcase", tc.Name)
	l.Infof("start")

	for _, stepIn := range tc.TestSteps {

		step, erra := ts.Templater.applyOnStep(stepIn)
		if erra != nil {
			tc.Errors = append(tc.Errors, Failure{Value: erra.Error()})
			break
		}

		e, err := getExecutorWrap(step, tcc)
		if err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: err.Error()})
			break
		}

		runTestStep(tcc, e, ts, tc, step, ts.Templater, l, detailsLevel)

		if detailsLevel != DetailsLow {
			bars[ts.Package].Increment()
		}
		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			break
		}
	}
	l.Infof("end")
}
