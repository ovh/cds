package venom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

const (
	// DetailsLow prints only summary results
	DetailsLow = "low"
	// DetailsMedium prints progress bar and summary
	DetailsMedium = "medium"
	// DetailsHigh prints progress bar and details
	DetailsHigh = "high"
)

var aliases map[string]string
var bars map[string]*pb.ProgressBar
var mutex = &sync.Mutex{}

// Process runs tests suite and return a sdk.Tests result
func Process(path string, alias []string, parallel int, detailsLevel string) (sdk.Tests, error) {
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

	tss := []sdk.TestSuite{}

	log.Debugf("Work with parallel %d", parallel)
	var wgPrepare, wg sync.WaitGroup
	wg.Add(len(filesPath))
	wgPrepare.Add(len(filesPath))

	parallels := make(chan sdk.TestSuite, parallel)
	chanEnd := make(chan sdk.TestSuite, 1)

	tr := sdk.Tests{}
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
	chanToRun := make(chan sdk.TestSuite, len(filesPath)+1)
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

			ts := sdk.TestSuite{}
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
			go func(ts sdk.TestSuite) {
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

func runTestSuite(ts *sdk.TestSuite, detailsLevel string) {
	l := log.WithField("v.testsuite", ts.Name)
	start := time.Now()

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

func runTestCase(ts *sdk.TestSuite, tc *sdk.TestCase, l *log.Entry, detailsLevel string) {
	l = l.WithField("x.testcase", tc.Name)
	l.Infof("start")
	for _, tst := range tc.TestSteps {

		//testTypes["script"].Run(&tst, l, aliases)

		stype := "script"
		t := newTest(stype)
		if t == nil {
			tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("Unknown type '%s'", stype)})
			break
		}

		applyResult(tc, &tst, l, t)
		if detailsLevel != DetailsLow {
			bars[ts.Package].Increment()
		}
		if len(tc.Failures) > 0 {
			break
		}
	}
	l.Infof("end")
}

func applyResult(tc *sdk.TestCase, ts *sdk.TestStep, l *log.Entry, t Test) error {

	buferr := new(bytes.Buffer)
	if err := xml.EscapeText(buferr, []byte(ts.Result.StdErr)); err != nil {
		return err
	}
	bufout := new(bytes.Buffer)
	if err := xml.EscapeText(bufout, []byte(ts.Result.StdErr)); err != nil {
		return err
	}

	tc.Systemerr.Value = buferr.String()
	tc.Systemout.Value = bufout.String()

	if ts.Result.Err != nil {
		tc.Systemerr.Value += ts.Result.Err.Error()
	}

	if len(ts.Assertions) == 0 {
		ts.Assertions = []string{""}
	}

	for _, a := range ts.Assertions {
		assertion := getAssertion(ts, a, l, t)
		t.Check(tc, ts, assertion, l)
	}

	return nil
}

func getAssertion(ts *sdk.TestStep, assertion string, l *log.Entry, t Test) string {
	if assertion != "" {
		return assertion
	}
	return t.GetDefaultAssertion(assertion)
}
