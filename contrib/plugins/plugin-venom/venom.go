package main

import (
	"strconv"
	"time"

	"github.com/ovh/cds/sdk/plugin"
	"github.com/runabove/venom"
	"github.com/runabove/venom/context/default"
	"github.com/runabove/venom/context/webctx"
	"github.com/runabove/venom/executors/exec"
	"github.com/runabove/venom/executors/http"
	"github.com/runabove/venom/executors/imap"
	"github.com/runabove/venom/executors/readfile"
	"github.com/runabove/venom/executors/smtp"
	"github.com/runabove/venom/executors/ssh"
	"github.com/runabove/venom/executors/web"
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
	return "This plugin helps you to run venom. Venom: https://github.com/runabove/venom. Add an extra step of type junit on your job to view tests results on CDS UI."
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
	params.Add("parallel", plugin.StringParameter, "Launch Test Suites in parallel. Enter here number of routines", "2")
	params.Add("output", plugin.StringParameter, "Directory where output xunit result file", ".")
	params.Add("details", plugin.StringParameter, "Output Details Level: low, medium, high", "low")
	params.Add("loglevel", plugin.StringParameter, "Log Level: debug, info, warn or error", "error")
	return params
}

// Run execute the action
func (s VenomPlugin) Run(a plugin.IJob) plugin.Result {
	// Parse parameters
	path := a.Arguments().Get("path")
	exclude := a.Arguments().Get("exclude")
	parallel := a.Arguments().Get("parallel")
	output := a.Arguments().Get("output")
	details := a.Arguments().Get("details")
	loglevel := a.Arguments().Get("loglevel")

	if path == "" {
		path = "."
	}
	p, err := strconv.Atoi(parallel)
	if err != nil {
		plugin.SendLog(a, "PLUGIN", "parallel arg must be an integer.")
		return plugin.Fail
	}

	venom.RegisterExecutor(exec.Name, exec.New())
	venom.RegisterExecutor(http.Name, http.New())
	venom.RegisterExecutor(imap.Name, imap.New())
	venom.RegisterExecutor(readfile.Name, readfile.New())
	venom.RegisterExecutor(smtp.Name, smtp.New())
	venom.RegisterExecutor(ssh.Name, ssh.New())
	venom.RegisterExecutor(web.Name, web.New())

	venom.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
	venom.RegisterTestCaseContext(webctx.Name, webctx.New())

	venom.PrintFunc = func(format string, aa ...interface{}) (n int, err error) {
		plugin.SendLog(a, format, aa)
		return 0, nil
	}

	start := time.Now()
	tests, err := venom.Process([]string{path}, a.Arguments().Data, []string{exclude}, p, loglevel, details)
	if err != nil {
		plugin.SendLog(a, "PLUGIN", "Fail on venom: %s", err)
		return plugin.Fail
	}

	elapsed := time.Since(start)
	venom.OutputResult("xml", false, true, output, *tests, elapsed, "low")

	return plugin.Success
}

func main() {
	p := VenomPlugin{}
	plugin.Serve(&p)
}
