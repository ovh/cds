package venom

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/yaml.v2"
)

func getFilesPath(path []string, exclude []string) []string {
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
		fileInfo, _ := os.Stat(p)
		if fileInfo != nil && fileInfo.IsDir() {
			p = filepath.Dir(p) + "/*.yml"
			log.Debugf("path computed:%s", path)
		}
		fpaths, errg := filepath.Glob(p)
		if errg != nil {
			log.Fatalf("Error reading files on path:%s :%s", path, errg)
		}
		for _, fp := range fpaths {
			toExclude := false
			for _, te := range fpathsExcluded {
				if te == fp {
					toExclude = true
					break
				}
			}
			if !toExclude && (strings.HasSuffix(fp, ".yml") || strings.HasSuffix(fp, ".yaml")) {
				filesPath = append(filesPath, fp)
			}
		}
	}

	log.Debugf("files to run: %v", filesPath)

	sort.Strings(filesPath)
	return filesPath
}

func readFiles(variables map[string]string, detailsLevel string, filesPath []string, chanToRun chan<- TestSuite, writer io.Writer) (map[string]*pb.ProgressBar, error) {
	bars := make(map[string]*pb.ProgressBar)

	for _, f := range filesPath {
		dat, errr := ioutil.ReadFile(f)
		if errr != nil {
			return nil, fmt.Errorf("Error while reading file %s err:%s", f, errr)
		}

		ts := TestSuite{}
		ts.Templater = newTemplater(variables)
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
			return nil, fmt.Errorf("Error while unmarshal file %s err:%s data:%s variables:%s", f, err, out, variables)
		}
		ts.Name += " [" + f + "]"

		nSteps := 0
		for _, tc := range ts.TestCases {
			nSteps += len(tc.TestSteps)
			if tc.Skipped == 1 {
				ts.Skipped++
			}
		}
		ts.Total = len(ts.TestCases)

		b := pb.New(nSteps).Prefix(rightPad("⚙ "+ts.Package, " ", 47))
		b.ShowCounters = false
		b.Output = writer
		if detailsLevel == DetailsLow {
			b.ShowBar = false
			b.ShowFinalTime = false
			b.ShowPercent = false
			b.ShowSpeed = false
			b.ShowTimeLeft = false
		}

		if detailsLevel != DetailsLow {
			bars[ts.Package] = b
		}

		chanToRun <- ts
	}
	return bars, nil
}
