package run

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var (
	path      string
	alias     []string
	format    string
	parallel  int
	logLevel  string
	outputDir string
)

func init() {
	Cmd.Flags().StringVarP(&path, "path", "", "", "Path containing TestSuites")
	Cmd.Flags().StringSliceVarP(&alias, "alias", "", []string{""}, "--alias cds:'cds -f config.json' --alias cds2:'cds -f config.json'")
	Cmd.Flags().StringVarP(&format, "format", "", "xml", "--formt:yaml, json, xml")
	Cmd.Flags().IntVarP(&parallel, "parallel", "", 1, "--parallel=2")
	Cmd.PersistentFlags().StringVarP(&logLevel, "log", "", "warn", "Log Level : debug, info or warn")
	Cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "", "", "Output Directory: create tests results file inside this directory")
}

// Cmd run
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Tests",
	PreRun: func(cmd *cobra.Command, args []string) {
		if path == "" {
			log.Fatalf("Invalid --path")
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

		tests, err := process()
		if err != nil {
			log.Fatal(err)
		}

		outputResult(tests)
	},
}

func outputResult(tests sdk.Tests) {
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.Marshal(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output (%s)", err)
		}
	case "yml", "yaml":
		data, err = yaml.Marshal(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output (%s)", err)
		}
	default:
		for _, tss := range tests.TestSuites {
			data, err = xml.Marshal(tss)
			if err != nil {
				log.Fatalf("Error: cannot format output (%s)", err)
			}
		}
	}

	if outputDir == "" {
		fmt.Printf(string(data))
		return
	}

	if format == "xml" {
		for i, ts := range tests.TestSuites {
			dataxml := append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), data...)
			filename := fmt.Sprintf("%s/test_results_%d_%s.xml", outputDir, i, strings.Replace(ts.Name, " ", "", -1))
			writeFile(filename, dataxml)
		}
		return
	}

	filename := outputDir + "/" + "test_results" + "." + format
	writeFile(filename, data)
}

func writeFile(filename string, data []byte) {

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error while creating file %s, err:%s", filename, err)
		os.Exit(1)
	}

	if _, err := f.Write(data); err != nil {
		fmt.Printf("Error while writing content of file %s, err:%s", filename, err)
		os.Exit(1)
	}
}
