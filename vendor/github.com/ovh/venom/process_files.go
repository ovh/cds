package venom

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/yaml.v2"
)

func getFilesPath(path []string, exclude []string) ([]string, error) {
	var filesPath []string
	var fpathsExcluded []string

	if len(exclude) > 0 {
		for _, p := range exclude {
			pe, erre := filepath.Glob(p)
			if erre != nil {
				log.Fatalf("Error reading files on path:%s :%s", path, erre)
			}
			fpathsExcluded = append(fpathsExcluded, pe...)
		}
	}

	for _, p := range path {
		p = strings.TrimSpace(p)

		fileInfo, _ := os.Stat(p)
		if fileInfo != nil && fileInfo.IsDir() {
			p = p + string(os.PathSeparator) + "*.yml"
		}

		fpaths, errg := filepath.Glob(p)
		if errg != nil {
			log.Errorf("Error reading files on path:%s :%s", path, errg)
			return nil, errg
		}
		for _, fp := range fpaths {
			toExclude := false
			for _, te := range fpathsExcluded {
				if te == fp {
					toExclude = true
					break
				}
			}
			if !toExclude && strings.HasSuffix(fp, ".yml") {
				filesPath = append(filesPath, fp)
			}
		}
	}

	sort.Strings(filesPath)
	return filesPath, nil
}

func (v *Venom) readFiles(filesPath []string) error {
	v.outputProgressBar = make(map[string]*pb.ProgressBar)

	for _, f := range filesPath {
		log.Info("Reading ", f)
		dat, errr := ioutil.ReadFile(f)
		if errr != nil {
			return fmt.Errorf("Error while reading file %s err:%s", f, errr)
		}

		ts := TestSuite{}
		ts.Templater = newTemplater(v.variables)
		ts.Package = f

		// Apply templater unitl there is no more modifications
		// it permits to include testcase from env
		out := ts.Templater.apply(dat)
		for i := 0; i < 10; i++ {
			tmp := ts.Templater.apply(out)
			if string(tmp) == string(out) {
				break
			}
			out = tmp
		}

		if err := yaml.Unmarshal(out, &ts); err != nil {
			return fmt.Errorf("Error while unmarshal file %s err:%v", f, err)
		}
		ts.Name += " [" + f + "]"

		nSteps := 0
		for _, tc := range ts.TestCases {
			nSteps += len(tc.TestSteps)
			if len(tc.Skipped) >= 1 {
				ts.Skipped += len(tc.Skipped)
			}
		}
		ts.Total = len(ts.TestCases)

		b := pb.New(nSteps).Prefix(rightPad("⚙ "+ts.Package, " ", 47))
		b.ShowCounters = false
		b.Output = v.LogOutput
		if v.OutputDetails == DetailsLow {
			b.ShowBar = false
			b.ShowFinalTime = false
			b.ShowPercent = false
			b.ShowSpeed = false
			b.ShowTimeLeft = false
		}

		if v.OutputDetails != DetailsLow {
			v.outputProgressBar[ts.Package] = b
		}
		v.testsuites = append(v.testsuites, ts)
	}
	return nil
}
