package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/ovh/cds/sdk/plugin"
)

// Plugin name for CDS
const PluginName = "plugin-group-tmpl"

// Plugin identifier for logs
var PluginIdentifier = strings.ToUpper(PluginName)

// Plugin is the tmpl plugin implementation for CDS
type Plugin struct {
	plugin.Common
}

// Name returns the plugin name
func (p Plugin) Name() string {
	return PluginName
}

// Description returns the plugin's description
func (p Plugin) Description() string {
	return `This actions helps you generate a marathon group application file.
It takes a config template file as a single application, and creates the group with the variables specified for each application in the applications files.
Check documentation on text/template for more information https://golang.org/pkg/text/template.`
}

// Author returns the author of this plugin
func (p Plugin) Author() string {
	return "Yannick BRIFFA <yannick.briffa@corp.ovh.com>"
}

// Parameters returns the needed parameters for this plugin
func (p Plugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()

	// mandatory parameters
	params.Add("config", plugin.StringParameter, "Template file to use", "")
	params.Add("output", plugin.StringParameter, "Output path for generated file (default to <file>.out or just trimming .tpl extension)", "")
	params.Add("applications", plugin.StringParameter, "Applications file variables", "")

	return params
}

// Run executes the action
func (p Plugin) Run(action plugin.IJob) plugin.Result {
	config := action.Arguments().Get("config")
	output := action.Arguments().Get("output")
	applications := action.Arguments().Get("applications")

	// if no file was specified
	if config == "" {
		plugin.SendLog(action, "Missing config template")
		return plugin.Fail
	}

	// if no applications were specified
	if applications == "" {
		plugin.SendLog(action, "Missing applications variables file")
		return plugin.Fail
	}

	// if output was not specified, either trim .tpl extension if any, or output to .out
	// in order to avoid name collision
	if output == "" {
		if strings.HasSuffix(config, ".tpl") {
			output = strings.TrimSuffix(config, ".tpl")
		} else {
			output = config + ".out"
		}
	}

	// get template config content
	configContent, err := ioutil.ReadFile(config)
	if err != nil {
		plugin.SendLog(action, "Failed to read config template file: %s", err)
		return plugin.Fail
	}

	// parse the template file
	configTemplate, err := template.New("file").Funcs(funcMap).Parse(string(configContent))
	if err != nil {
		plugin.SendLog(action, "Failed to parse config template: %s", err)
		return plugin.Fail
	}

	// open the output file
	of, err := os.Create(output)
	if err != nil {
		plugin.SendLog(action, "Failed to create output file: %s", err)
		return plugin.Fail
	}
	defer of.Close()

	// fetching the apps variables
	apps, err := NewApplications(applications)
	if err != nil {
		plugin.SendLog(action, "Failed to read applications variables file: %s", err)
		return plugin.Fail
	}

	// executing the template for each application in the applicationsFiles
	appsConfigs, err := getConfigByApplication(apps, configTemplate)
	if err != nil {
		plugin.SendLog(action, "Failed to read applications variables file: %s", err)
		return plugin.Fail
	}

	// finally, execute the template
	tmplParams := &outputBodyVars{
		Configs:         appsConfigs,
		SubApplications: apps.Names(),
	}
	buf := new(bytes.Buffer)
	if err := outputBodyTemplate.Execute(buf, tmplParams); err != nil {
		plugin.SendLog(action, "Failed to execute main template: %s", err)
		return plugin.Fail
	}
	indent := new(bytes.Buffer)
	if err := json.Indent(indent, buf.Bytes(), "", "    "); err != nil {
		plugin.SendLog(action, "Failed to indent generated content: %s", err)
		return plugin.Fail
	}
	if _, err := indent.WriteTo(of); err != nil {
		plugin.SendLog(action, "Failed to write generated file: %s", err)
		return plugin.Fail
	}

	plugin.SendLog(action, "Generated output file %s", output)
	return plugin.Success
}

func getConfigByApplication(apps *Applications, tmpl *template.Template) (map[string]string, error) {
	appsConfigs := map[string]string{}

	for _, app := range apps.Names() {
		// getting the variables for the specific application
		vars, err := apps.Variables(app)
		if err != nil {
			return nil, fmt.Errorf("%s : %s ", app, err)
		}

		// executing the template and getting the result as a string
		appConfig, err := executeTemplate(tmpl, vars)
		if err != nil {
			return nil, fmt.Errorf("%s : %s ", app, err)
		}
		appsConfigs[app] = appConfig
	}
	return appsConfigs, nil
}
