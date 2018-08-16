package venom

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsamin/go-dump"
	log "github.com/sirupsen/logrus"
)

func (v *Venom) runTestSuite(ts *TestSuite) {
	if v.EnableProfiling {
		var filename, filenameCPU, filenameMem string
		if v.OutputDir != "" {
			filename = v.OutputDir + "/"
		}
		filenameCPU = filename + "pprof_cpu_profile_" + ts.Filename + ".prof"
		filenameMem = filename + "pprof_mem_profile_" + ts.Filename + ".prof"
		fCPU, errCPU := os.Create(filenameCPU)
		fMem, errMem := os.Create(filenameMem)
		if errCPU != nil || errMem != nil {
			log.Errorf("error while create profile file CPU:%v MEM:%v", errCPU, errMem)
		} else {
			pprof.StartCPUProfile(fCPU)
			p := pprof.Lookup("heap")
			defer p.WriteTo(fMem, 1)
			defer pprof.StopCPUProfile()
		}
	}

	l := log.WithField("v.testsuite", ts.Name)
	start := time.Now()

	d, err := dump.ToStringMap(ts.Vars)
	if err != nil {
		log.Errorf("err:%s", err)
	}
	ts.Templater.Add("", d)
	ts.Templater.Add("", map[string]string{"venom.testsuite": ts.ShortName})
	ts.Templater.Add("", map[string]string{"venom.testsuite.filename": ts.Filename})

	// we apply templater on current vars only
	for index := 0; index < 10; index++ {
		var toApply bool
		for k, v := range ts.Templater.Values {
			if strings.Contains(v, "{{") {
				toApply = true
				_, s := ts.Templater.apply([]byte(v))
				ts.Templater.Values[k] = string(s)
			}
		}
		if !toApply {
			break
		}
	}

	totalSteps := 0
	for _, tc := range ts.TestCases {
		totalSteps += len(tc.TestSteps)
	}

	v.runTestCases(ts, l)

	elapsed := time.Since(start)

	var o string
	if ts.Failures > 0 || ts.Errors > 0 {
		red := color.New(color.FgRed).SprintFunc()
		o = fmt.Sprintf("%s %s", red("FAILURE"), rightPad(ts.Package, " ", 47))
	} else {
		green := color.New(color.FgGreen).SprintFunc()
		o = fmt.Sprintf("%s %s", green("SUCCESS"), rightPad(ts.Package, " ", 47))
	}
	o += fmt.Sprintf("%s", elapsed)
	v.PrintFunc("%s\n", o)
}

func (v *Venom) runTestCases(ts *TestSuite, l Logger) {
	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		if len(tc.Skipped) == 0 {
			v.runTestCase(ts, tc, l)
		}

		if len(tc.Failures) > 0 {
			ts.Failures += len(tc.Failures)
		}
		if len(tc.Errors) > 0 {
			ts.Errors += len(tc.Errors)
		}
		if len(tc.Skipped) > 0 {
			ts.Skipped += len(tc.Skipped)
		}

		if v.StopOnFailure && (len(tc.Failures) > 0 || len(tc.Errors) > 0) {
			// break TestSuite
			return
		}
	}
}

//Parse the suite to find unreplaced and extracted variables
func (v *Venom) parseTestSuite(ts *TestSuite) ([]string, []string, error) {
	d, err := dump.ToStringMap(ts.Vars)
	if err != nil {
		log.Errorf("err:%s", err)
	}
	ts.Templater.Add("", d)

	return v.parseTestCases(ts)
}

//Parse the testscases to find unreplaced and extracted variables
func (v *Venom) parseTestCases(ts *TestSuite) ([]string, []string, error) {
	vars := []string{}
	extractsVars := []string{}
	for i := range ts.TestCases {
		tc := &ts.TestCases[i]
		if len(tc.Skipped) == 0 {
			tvars, tExtractedVars, err := v.parseTestCase(ts, tc)
			if err != nil {
				return nil, nil, err
			}
			for _, k := range tvars {
				var found bool
				for i := 0; i < len(vars); i++ {
					if vars[i] == k {
						found = true
						break
					}
				}
				if !found {
					vars = append(vars, k)
				}
			}
			for _, k := range tExtractedVars {
				var found bool
				for i := 0; i < len(extractsVars); i++ {
					if extractsVars[i] == k {
						found = true
						break
					}
				}
				if !found {
					extractsVars = append(extractsVars, k)
				}
			}
		}
	}

	return vars, extractsVars, nil
}
