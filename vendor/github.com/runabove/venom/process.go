package venom

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/yaml.v2"
)

var aliases map[string]string
var bars map[string]*pb.ProgressBar
var mutex = &sync.Mutex{}

// Process runs tests suite and return a Tests result
func Process(path string, alias []string, parallel int, detailsLevel string) (Tests, error) {
	log.Infof("Start processing path %s", path)

	aliases = make(map[string]string)

	for _, a := range alias {
		t := strings.Split(a, ":")
		if len(t) < 2 {
			continue
		}
		aliases[t[0]] = strings.Join(t[1:], "")
	}

	fileInfo, _ := os.Stat(path)
	if fileInfo != nil && fileInfo.IsDir() {
		path = filepath.Dir(path) + "/*.yml"
		log.Debugf("path computed:%s", path)
	}

	filesPath, errg := filepath.Glob(path)
	if errg != nil {
		log.Fatalf("Error reading files on path:%s :%s", path, errg)
	}

	tss := []TestSuite{}

	log.Debugf("Work with parallel %d", parallel)
	var wgPrepare, wg sync.WaitGroup
	wg.Add(len(filesPath))
	wgPrepare.Add(len(filesPath))

	parallels := make(chan TestSuite, parallel)
	chanEnd := make(chan TestSuite, 1)

	tr := Tests{}
	go func() {
		for t := range chanEnd {
			tss = append(tss, t)
			if t.Failures > 0 {
				tr.TotalKO += t.Failures
			} else {
				tr.TotalOK += len(t.TestCases) - t.Failures
			}
			if t.Skipped > 0 {
				tr.TotalSkipped += t.Skipped
			}

			tr.Total = tr.TotalKO + tr.TotalOK + tr.TotalSkipped
			wg.Done()
		}
	}()

	bars = make(map[string]*pb.ProgressBar)
	chanToRun := make(chan TestSuite, len(filesPath)+1)
	totalSteps := 0
	for _, file := range filesPath {
		go func(f string) {

			log.Debugf("read %s", f)
			dat, errr := ioutil.ReadFile(f)
			if errr != nil {
				log.WithError(errr).Errorf("Error while reading file")
				wgPrepare.Done()
				wg.Done()
				return
			}

			ts := TestSuite{}
			ts.Package = f
			log.Debugf("Unmarshal %s", f)
			if err := yaml.Unmarshal(dat, &ts); err != nil {
				log.WithError(err).Errorf("Error while unmarshal file")
				wgPrepare.Done()
				wg.Done()
				return
			}
			ts.Name += " [" + f + "]"

			// compute progress bar
			nSteps := 0
			for _, tc := range ts.TestCases {
				totalSteps += len(tc.TestSteps)
				nSteps += len(tc.TestSteps)
				if tc.Skipped == 1 {
					ts.Skipped++
				}
			}
			ts.Total = len(ts.TestCases)

			b := pb.New(nSteps).Prefix(rightPad("⚙ "+ts.Package, " ", 47))
			b.ShowCounters = false
			if detailsLevel == DetailsLow {
				b.ShowBar = false
				b.ShowFinalTime = false
				b.ShowPercent = false
				b.ShowSpeed = false
				b.ShowTimeLeft = false
			}

			if detailsLevel != DetailsLow {
				mutex.Lock()
				bars[ts.Package] = b
				mutex.Unlock()
			}

			chanToRun <- ts
			wgPrepare.Done()
		}(file)
	}

	wgPrepare.Wait()

	var pbbars []*pb.ProgressBar
	var pool *pb.Pool
	if detailsLevel != DetailsLow {
		for _, b := range bars {
			pbbars = append(pbbars, b)
		}
		var errs error
		pool, errs = pb.StartPool(pbbars...)
		if errs != nil {
			log.Errorf("Error while prepare details bars: %s", errs)
		}
	}

	go func() {
		for ts := range chanToRun {
			go func(ts TestSuite) {
				parallels <- ts
				defer func() { <-parallels }()
				runTestSuite(&ts, detailsLevel)
				chanEnd <- ts
			}(ts)
		}
	}()

	wg.Wait()

	log.Infof("end processing path %s", path)

	if detailsLevel != DetailsLow {
		if err := pool.Stop(); err != nil {
			log.Errorf("Error while closing pool progress bar: %s", err)
		}
	}

	tr.TestSuites = tss
	return tr, nil
}

func rightPad(s string, padStr string, pLen int) string {
	o := s + strings.Repeat(padStr, pLen)
	return o[0:pLen]
}

func runTestSuite(ts *TestSuite, detailsLevel string) {
	l := log.WithField("v.testsuite", ts.Name)
	start := time.Now()

	d, err := dump.ToMap(ts.Vars)
	if err != nil {
		log.Errorf("err:%s", err)
	}
	ts.Templater = newTemplater(d)

	totalSteps := 0
	for _, tc := range ts.TestCases {
		totalSteps += len(tc.TestSteps)
	}

	for i, tc := range ts.TestCases {
		if tc.Skipped == 0 {
			runTestCase(ts, &tc, l, detailsLevel)
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
		fmt.Println(o)
	}
}

func runTestCase(ts *TestSuite, tc *TestCase, l *log.Entry, detailsLevel string) {
	l = l.WithField("x.testcase", tc.Name)
	l.Infof("start")
	for _, stepIn := range tc.TestSteps {

		step, erra := ts.Templater.Apply(stepIn)
		if erra != nil {
			tc.Errors = append(tc.Errors, Failure{Value: erra.Error()})
			break
		}

		e, err := getExecutorWrap(step)
		if err != nil {
			tc.Errors = append(tc.Errors, Failure{Value: err.Error()})
			break
		}

		runTestStep(e, ts, tc, step, l, detailsLevel, ts.Templater)

		if detailsLevel != DetailsLow {
			bars[ts.Package].Increment()
		}
		if len(tc.Failures) > 0 {
			break
		}
	}
	l.Infof("end")
}

func runTestStep(e *executorWrap, ts *TestSuite, tc *TestCase, step TestStep, l *log.Entry, detailsLevel string, templater *Templater) {

	var isOK bool
	var errors []Failure
	var failures []Failure

	var retry int
	for retry = 0; retry <= e.retry && !isOK; retry++ {
		if retry > 1 && !isOK {
			log.Debugf("Sleep %d, it's %d attempt", e.delay, retry)
			time.Sleep(time.Duration(e.delay) * time.Second)
		}

		result, err := e.executor.Run(l, aliases, step)
		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: err.Error()})
			continue
		}

		ts.Templater.Add(tc.Name, result)

		log.Debugf("result:%+v", ts.Templater)

		if h, ok := e.executor.(executorWithDefaultAssertions); ok {
			isOK, errors, failures = applyChecks(result, step, h.GetDefaultAssertions(), l)
		} else {
			isOK, errors, failures = applyChecks(result, step, nil, l)
		}
		if isOK {
			break
		}
	}
	tc.Errors = append(tc.Errors, errors...)
	tc.Failures = append(tc.Failures, failures...)
	if retry > 0 && (len(failures) > 0 || len(errors) > 0) {
		tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempt(s)", retry)})
	}
}

func runTestStepExecutor(e *executorWrap, ts *TestSuite, step TestStep, l *log.Entry, templater *Templater) (ExecutorResult, error) {
	if e.timeout == 0 {
		return e.executor.Run(l, aliases, step)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.timeout)*time.Second)
	defer cancel()

	ch := make(chan ExecutorResult)
	cherr := make(chan error)
	go func(e *executorWrap, step TestStep, l *log.Entry) {
		result, err := e.executor.Run(l, aliases, step)
		cherr <- err
		ch <- result
	}(e, step, l)

	select {
	case err := <-cherr:
		return nil, err
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("Timeout after %d second(s)", e.timeout)
	}

}
