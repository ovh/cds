package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk/plugin"
	"github.com/ovh/venom"
	"github.com/ovh/venom/context/default"
	"github.com/ovh/venom/context/redis"
	"github.com/ovh/venom/context/webctx"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/ovhapi"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/redis"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/web"
)

// VenomPlugin implements plugin interface
type VenomPlugin struct {
	plugin.Common
}

// Name returns the plugin name
func (s VenomPlugin) Name() string {
	return "plugin-venom"
}

// Description returns the plugin description
func (s VenomPlugin) Description() string {
	return `This plugin helps you to run venom. Venom: https://github.com/ovh/venom.

Add an extra step of type junit on your job to view tests results on CDS UI.`
}

// Author returns the plugin author's name
func (s VenomPlugin) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

// Parameters return parameters description
func (s VenomPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("path", plugin.StringParameter, "Path containers yml venom files. Format: adirectory/, ./*aTest.yml, ./foo/b*/**/z*.yml", ".")
	params.Add("exclude", plugin.TextParameter, "Exclude some files, one file per line", "")
	params.Add("output", plugin.StringParameter, "Directory where output xunit result file", ".")
	params.Add("parallel", plugin.StringParameter, "Launch Test Suites in parallel. Enter here number of routines", "1")
	params.Add("details", plugin.StringParameter, "Output Details Level: low, medium, high", "low")
	params.Add("loglevel", plugin.StringParameter, "Log Level: debug, info, warn or error", "error")
	params.Add("vars", plugin.StringParameter, "Empty: all {{.cds...}} vars will be rewrited. Otherwise, you can limit rewrite to some variables. Example, enter cds.app.yourvar,cds.build.foo,myvar=foo to rewrite {{.cds.app.yourvar}}, {{.cds.build.foo}} and {{.foo}}. Default: Empty", "")
	params.Add("vars-from-file", plugin.StringParameter, "filename.yaml or filename.json. See https://github.com/ovh/venom#run-venom-with-file-var", "")
	return params
}

type venomWriter struct {
	plugin.IJob
}

func (w venomWriter) Write(buf []byte) (int, error) {
	return len(buf), plugin.SendLog(w, "VENOM %s", buf)
}

// Run execute the action
func (s VenomPlugin) Run(a plugin.IJob) plugin.Result {
	// Parse parameters
	path := a.Arguments().Get("path")
	exclude := a.Arguments().Get("exclude")
	output := a.Arguments().Get("output")
	parallelS := a.Arguments().Get("parallel")
	loglevel := a.Arguments().Get("loglevel")
	vars := a.Arguments().Get("vars")
	varsFromFile := a.Arguments().Get("vars-from-file")

	if path == "" {
		path = "."
	}

	parallel, err := strconv.Atoi(parallelS)
	if err != nil {
		plugin.SendLog(a, "VENOM - parallel arg must be an integer\n")
		return plugin.Fail
	}

	v := venom.New()
	v.RegisterExecutor(exec.Name, exec.New())
	v.RegisterExecutor(http.Name, http.New())
	v.RegisterExecutor(imap.Name, imap.New())
	v.RegisterExecutor(ovhapi.Name, ovhapi.New())
	v.RegisterExecutor(readfile.Name, readfile.New())
	v.RegisterExecutor(redis.Name, redis.New())
	v.RegisterExecutor(smtp.Name, smtp.New())
	v.RegisterExecutor(ssh.Name, ssh.New())
	v.RegisterExecutor(web.Name, web.New())
	v.RegisterExecutor(dbfixtures.Name, dbfixtures.New())
	v.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
	v.RegisterTestCaseContext(webctx.Name, webctx.New())
	v.RegisterTestCaseContext(redisctx.Name, redisctx.New())

	v.PrintFunc = func(format string, aa ...interface{}) (n int, err error) {
		plugin.SendLog(a, format, aa)
		return 0, nil
	}

	start := time.Now()
	w := venomWriter{a}
	data := make(map[string]string)
	if vars == "" {
		// no vars -> all .cds... variables can by used in yml
		data = a.Arguments().Data
	} else {
		// if vars is not empty
		// vars could be:
		// cds.foo.bar,cds.foo2.bar2
		// cds.foo.bar,cds.foo2.bar2,anotherVars=foo,anotherVars2=bar
		for _, v := range strings.Split(vars, ",") {
			t := strings.Split(v, "=")
			if len(t) > 1 {
				// if value of current var is setted, we take it
				data[t[0]] = t[1]
				plugin.SendLog(a, "VENOM - var %s has value %s\n", t[0], t[1])
			} else if len(t) == 1 && strings.HasPrefix(v, "cds.") {
				plugin.SendLog(a, "VENOM - try fo find var %s in cds variables\n", v)
				// if var starts with .cds, we try to take value from current CDS variables
				for k := range a.Arguments().Data {
					if k == v {
						plugin.SendLog(a, "VENOM - var %s is found with value %s\n", v, a.Arguments().Data[k])
						data[k] = a.Arguments().Data[k]
						break
					}
				}
			}
		}

		//If we use the var list, it means we do pretty hacky stuffs, so let's ignore all cds vars
		v.IgnoreVariables = append(v.IgnoreVariables, "cds", "workflow", "git")
	}

	if varsFromFile != "" {
		varFileMap := make(map[string]string)
		bytes, err := ioutil.ReadFile(varsFromFile)
		if err != nil {
			plugin.SendLog(a, "VENOM - Error while reading file: %v\n", err)
			return plugin.Fail
		}
		switch filepath.Ext(varsFromFile) {
		case ".json":
			err = json.Unmarshal(bytes, &varFileMap)
		case ".yaml":
			err = yaml.Unmarshal(bytes, &varFileMap)
		default:
			plugin.SendLog(a, "VENOM - unsupported varFile format")
			return plugin.Fail
		}

		if err != nil {
			plugin.SendLog(a, "VENOM - Error on unmarshal file: %v\n", err)
			return plugin.Fail
		}

		for key, value := range varFileMap {
			data[key] = value
		}
	}

	v.AddVariables(data)
	v.LogLevel = loglevel
	v.LogOutput = w
	v.OutputFormat = "xml"
	v.OutputDir = output
	v.Parallel = parallel
	v.OutputDetails = "low"

	filepath := strings.Split(path, ",")
	filepathExcluded := strings.Split(exclude, ",")

	if len(filepath) == 1 {
		filepath = strings.Split(filepath[0], " ")
	}

	if len(filepathExcluded) == 1 {
		filepathExcluded = strings.Split(filepathExcluded[0], " ")
	}

	tests, err := v.Process(filepath, filepathExcluded)
	if err != nil {
		plugin.SendLog(a, "VENOM - Fail on venom: %v\n", err)
		return plugin.Fail
	}

	elapsed := time.Since(start)
	plugin.SendLog(a, "VENOM - Output test results under: %s\n", output)
	if err := v.OutputResult(*tests, elapsed); err != nil {
		plugin.SendLog(a, "VENOM - Error while uploading test results: %v\n", err)
		return plugin.Fail
	}

	return plugin.Success
}

func main() {
	plugin.Main(&VenomPlugin{})
}
