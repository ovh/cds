package run

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var (
	path           string
	alias          []string
	format         string
	parallel       int
	logLevel       string
	outputDir      string
	detailsLevel   string
	resumeFailures bool
	resume         bool
)

const (
	detailsLow    = "low"
	detailsMedium = "medium"
	detailsHigh   = "high"
)

func init() {
	Cmd.Flags().StringSliceVarP(&alias, "alias", "", []string{""}, "--alias cds:'cds -f config.json' --alias cds2:'cds -f config.json'")
	Cmd.Flags().StringVarP(&format, "format", "", "xml", "--formt:yaml, json, xml")
	Cmd.Flags().IntVarP(&parallel, "parallel", "", 1, "--parallel=2 : launches 2 Test Suites in parallel")
	Cmd.PersistentFlags().StringVarP(&logLevel, "log", "", "warn", "Log Level : debug, info or warn")
	Cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "", "", "Output Directory: create tests results file inside this directory")
	Cmd.PersistentFlags().StringVarP(&detailsLevel, "details", "", "medium", "Output Details Level : low, medium, high")
	Cmd.PersistentFlags().BoolVarP(&resume, "resume", "", true, "Output Resume: one line with Total, TotalOK, TotalKO, TotalSkipped, TotalTestSuite")
	Cmd.PersistentFlags().BoolVarP(&resumeFailures, "resumeFailures", "", true, "Output Resume Failures")
}

// Cmd run
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Tests",
	PreRun: func(cmd *cobra.Command, args []string) {

		if len(args) > 1 {
			log.Fatalf("Invalid path: venom run <path>")
		}
		if len(args) == 1 {
			path = args[0]
		} else {
			path = "."
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		if parallel < 0 {
			parallel = 1
		}

		switch logLevel {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "error":
			log.SetLevel(log.WarnLevel)
		default:
			log.SetLevel(log.WarnLevel)
		}

		switch detailsLevel {
		case detailsLow, "medium", "high":
			log.Infof("Detail Level: %s", detailsLevel)
		default:
			log.Fatalf("Invalid details. Must be low, medium or high")
		}

		start := time.Now()
		tests, err := process()
		if err != nil {
			log.Fatal(err)
		}

		elapsed := time.Since(start)

		outputResult(tests, elapsed)
	},
}

func outputResult(tests sdk.Tests, elapsed time.Duration) {
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.Marshal(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output json (%s)", err)
		}
	case "yml", "yaml":
		data, err = yaml.Marshal(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output yaml (%s)", err)
		}
	default:
		for _, tss := range tests.TestSuites {
			data, err = xml.Marshal(tss)
			if err != nil {
				log.Fatalf("Error: cannot format output xml (%s)", err)
			}
		}
	}

	if detailsLevel == "high" {
		fmt.Printf(string(data))
	}

	if resume {
		outputResume(tests, elapsed)
	}

	if outputDir != "" {
		if format == "xml" {
			for i, ts := range tests.TestSuites {

				tuple := strings.Split(ts.Name, " ") // remove " [...] from name"
				lname := ts.Name
				if len(tuple) >= 1 {
					lname = tuple[0]
				}
				dataxml := append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), data...)
				filename := fmt.Sprintf("%s/test_results_%d_%s.xml", outputDir, i, strings.Replace(lname, " ", "", -1))

				if err := ioutil.WriteFile(filename, dataxml, 0644); err != nil {
					fmt.Printf("Error while creating file %s, err:%s", filename, err)
					os.Exit(1)
				}
			}
			return
		}

		filename := outputDir + "/" + "test_results" + "." + format
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			fmt.Printf("Error while creating file %s, err:%s", filename, err)
			os.Exit(1)
		}
	}

}

func outputResume(tests sdk.Tests, elapsed time.Duration) {

	if resumeFailures {
		for _, t := range tests.TestSuites {
			if t.Failures > 0 || t.Errors > 0 {
				fmt.Printf("FAILED %s\n", t.Name)
				fmt.Printf("--------------\n")

				for _, tc := range t.TestCases {
					for _, f := range tc.Failures {
						fmt.Printf("%s\n", f.Value)
					}
					for _, f := range tc.Errors {
						fmt.Printf("%s\n", f.Value)
					}
				}
				fmt.Printf("-=-=-=-=-=-=-=-=-\n")
			}
		}
	}

	totalTestCases := 0
	totalTestSteps := 0
	for _, t := range tests.TestSuites {
		if t.Failures > 0 || t.Errors > 0 {
			fmt.Printf("FAILED %s\n", t.Name)
		}
		totalTestCases += len(t.TestCases)
		for _, tc := range t.TestCases {
			totalTestSteps += len(tc.TestSteps)
		}
	}

	fmt.Printf("Total:%d TotalOK:%d TotalKO:%d TotalSkipped:%d TotalTestSuite:%d TotalTestCase:%d TotalTestStep:%d Duration:%s\n",
		tests.Total,
		tests.TotalOK,
		tests.TotalKO,
		tests.TotalSkipped,
		len(tests.TestSuites),
		totalTestCases,
		totalTestSteps,
		elapsed,
	)

}
