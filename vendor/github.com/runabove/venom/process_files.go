package venom

import (
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

func readFiles(variables map[string]string, detailsLevel string, filesPath []string, chanToRun chan<- TestSuite) map[string]*pb.ProgressBar {
	bars := make(map[string]*pb.ProgressBar)

	for _, f := range filesPath {
		dat, errr := ioutil.ReadFile(f)
		if errr != nil {
			log.WithError(errr).Errorf("Error while reading file %s", f)
			continue
		}

		ts := TestSuite{}
		ts.Templater = newTemplater(variables)
		ts.Package = f

		out := ts.Templater.apply(dat)

		if err := yaml.Unmarshal(out, &ts); err != nil {
			log.WithError(err).Errorf("Error while unmarshal file %s", f)
			continue
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
	return bars
}
