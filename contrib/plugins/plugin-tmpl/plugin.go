package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/ovh/cds/sdk/plugin"
)

// Plugin is the tmpl plugin implementation for CDS
type Plugin struct {
	plugin.Common
}

// Name returns the plugin name
func (p Plugin) Name() string {
	return "plugin-tmpl"
}

// Description returns the plugin's description
func (p Plugin) Description() string {
	return `This action helps you generates a file using a template file and text/template golang package.

Check documentation on text/template for more information https://golang.org/pkg/text/template.`
}

// Author returns the author of this plugin
func (p Plugin) Author() string {
	return "Alexandre JIN <alexandre.jin@corp.ovh.com>"
}

// Parameters returns the needed parameters for this plugin
func (p Plugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()

	// mandatory parameters
	params.Add("file", plugin.StringParameter, "Template file to use", "")
	params.Add("output", plugin.StringParameter, "Output path for generated file (default to <file>.out or just trimming .tpl extension)", "")
	params.Add("params", plugin.TextParameter, "Parameters to pass on the template file (key=value newline separated list)", "")

	return params
}

// Run executes the action
func (p Plugin) Run(job plugin.IJob) plugin.Result {
	file := job.Arguments().Get("file")
	output := job.Arguments().Get("output")
	params := job.Arguments().Get("params")

	// if no file was specified
	if file == "" {
		plugin.SendLog(job, "Missing template file")
		return plugin.Fail
	}

	// if output was not specified, either trim .tpl extension if any, or output to .out
	// in order to avoid name collision
	if output == "" {
		if strings.HasSuffix(file, ".tpl") {
			output = strings.TrimSuffix(file, strings.TrimSuffix(file, ".tpl"))
		} else {
			output = file + ".out"
		}
	}

	funcMap := template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"split": strings.Split,
		"join":  strings.Join,
	}

	// get template file content
	content, err := ioutil.ReadFile(file)
	if err != nil {
		plugin.SendLog(job, "Failed to read template file: %s", err)
		return plugin.Fail
	}

	// parse the template file
	tmpl, err := template.New("file").Funcs(funcMap).Parse(string(content))
	if err != nil {
		plugin.SendLog(job, "Failed to parse template file: %s", err)
		return plugin.Fail
	}

	// open the output file
	of, err := os.Create(output)
	if err != nil {
		plugin.SendLog(job, "Failed to create output file: %s", err)
		return plugin.Fail
	}
	defer of.Close()

	// parse template parameters if any
	tmplParams, err := parseTemplateParameters(params)
	if err != nil {
		plugin.SendLog(job, "Failed to parse template parameters: %s", err)
		return plugin.Fail
	}

	// finally, execute the template
	if err := tmpl.Execute(of, tmplParams); err != nil {
		plugin.SendLog(job, "Failed to execute template: %s", err)
		return plugin.Fail
	}

	plugin.SendLog(job, "Generated output file %s", output)
	return plugin.Success
}

// parseTemplateParameters parses a list of key value pairs separated by new lines
func parseTemplateParameters(s string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	for _, l := range strings.Split(s, "\n") {
		components := strings.SplitN(l, "=", 2)
		if len(components) != 2 {
			return nil, fmt.Errorf("invalid key value pair form for %q", l)
		}
		params[components[0]] = components[1]
	}

	return params, nil
}
