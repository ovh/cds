package venom

import (
	"github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
)

func runTestCase(ts *TestSuite, tc *TestCase, bars map[string]*pb.ProgressBar, l Logger, detailsLevel string) {
	l.Debugf("Init context")
	var errContext error
	tc.Context, errContext = ts.Templater.ApplyOnContext(tc.Context)
	if errContext != nil {
		tc.Errors = append(tc.Errors, Failure{Value: errContext.Error()})
		return
	}
	tcc, errContext := ContextWrap(tc)
	if errContext != nil {
		tc.Errors = append(tc.Errors, Failure{Value: errContext.Error()})
		return
	}
	if err := tcc.Init(); err != nil {
		tc.Errors = append(tc.Errors, Failure{Value: err.Error()})
		return
	}
	defer tcc.Close()

	if _l, ok := l.(*logrus.Entry); ok {
		l = _l.WithField("x.testcase", tc.Name)
	}
	l.Infof("start")

	for _, stepIn := range tc.TestSteps {

		step, erra := ts.Templater.ApplyOnStep(stepIn)
		if erra != nil {
			tc.Errors = append(tc.Errors, Failure{Value: erra.Error()})
			break
		}

		e, err := WrapExecutor(step, tcc)
		if err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: err.Error()})
			break
		}

		RunTestStep(tcc, e, ts, tc, step, ts.Templater, l, detailsLevel)

		if detailsLevel != DetailsLow {
			bars[ts.Package].Increment()
		}
		if len(tc.Failures) > 0 || len(tc.Errors) > 0 {
			break
		}
	}
	l.Infof("end")
}
