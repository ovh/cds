package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build group-tmpl
$ make publish group-tmpl
*/

type groupTmplActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *groupTmplActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:   "plugin-group-tmpl",
		Author: "Yannick BRIFFA <yannick.briffa@corp.ovh.com>",
		Description: `This actions helps you generate a marathon group application file.
It takes a config template file as a single application, and creates the group with the variables specified for each application in the applications files.
Check documentation on text/template for more information https://golang.org/pkg/text/template.`,
		Version: sdk.VERSION,
	}, nil
}

func (actPlugin *groupTmplActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	config := q.GetOptions()["config"]
	output := q.GetOptions()["output"]
	applications := q.GetOptions()["applications"]

	// if no file was specified
	if config == "" {
		return actionplugin.Fail("Missing config template")
	}

	// if no applications were specified
	if applications == "" {
		return actionplugin.Fail("Missing applications variables file")
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
		return actionplugin.Fail("Failed to read config template file: %s", err)
	}

	// parse the template file
	configTemplate, err := template.New("file").Funcs(funcMap).Parse(string(configContent))
	if err != nil {
		return actionplugin.Fail("Failed to parse config template: %v", err)
	}

	// open the output file
	of, err := os.Create(output)
	if err != nil {
		return actionplugin.Fail("Failed to create output file: %v", err)
	}
	defer of.Close()

	// fetching the apps variables
	apps, err := NewApplications(applications)
	if err != nil {
		return actionplugin.Fail("Failed to read applications variables file: %v", err)
	}

	// executing the template for each application in the applicationsFiles
	appsConfigs, err := getConfigByApplication(apps, configTemplate)
	if err != nil {
		return actionplugin.Fail("Failed to read applications variables file: %v", err)
	}

	// finally, execute the template
	tmplParams := &outputBodyVars{
		Configs:         appsConfigs,
		SubApplications: apps.Names(),
	}
	buf := new(bytes.Buffer)
	if err := outputBodyTemplate.Execute(buf, tmplParams); err != nil {
		return actionplugin.Fail("Failed to execute main template: %s", err)
	}
	indent := new(bytes.Buffer)
	if err := json.Indent(indent, buf.Bytes(), "", "    "); err != nil {
		return actionplugin.Fail("Failed to indent generated content: %v", err)
	}
	if _, err := indent.WriteTo(of); err != nil {
		return actionplugin.Fail("Failed to write generated file: %v", err)
	}

	fmt.Printf("Generated output file %s", output)

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := groupTmplActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
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
