package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/*
This plugin have to be used as an action platform plugin

Tmpl action plugin must be configured as following:
	name: tmpl-action-plugin
	type: action
	author: "François Samin"
	description: "tmpl action Plugin"

$ cdsctl admin plugins import tmpl-action-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add tmpl-action-plugin tmpl-action-plugin-bin.yml <path-to-binary-file>

Arsenal platform must configured as following
	name: tmpl
	default_config:
		host:
			type: string
	action_default_config:
		file:
			type: string
		output:
			type: string
		params:
			type: text
	plugin: tmpl-plugin
*/

type tmplActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *tmplActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:   "tmpl action Plugin",
		Author: "Alexandre JIN  <alexandre.jin@corp.ovh.com>",
		Description: `This action helps you generates a file using a template file and text/template golang package.

	Check documentation on text/template for more information https://golang.org/pkg/text/template.`,
		Version: sdk.VERSION,
	}, nil
}

func (actPlugin *tmplActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	file := q.GetOptions()["file"]
	output := q.GetOptions()["output"]
	params := q.GetOptions()["params"]

	// if no file was specified
	if file == "" {
		return fail("Missing template file")
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
		return fail("Failed to read template file: %v", err)
	}

	// parse the template file
	tmpl, err := template.New("file").Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fail("Failed to parse template file: %v", err)
	}

	// open the output file
	of, err := os.Create(output)
	if err != nil {
		return fail("Failed to create output file: %v", err)
	}
	defer of.Close()

	// parse template parameters if any
	tmplParams, err := parseTemplateParameters(params)
	if err != nil {
		return fail("Failed to parse template parameters: %v", err)
	}

	// finally, execute the template
	if err := tmpl.Execute(of, tmplParams); err != nil {
		return fail("Failed to execute template: %v", err)
	}

	fmt.Printf("Generated output file %s", output)

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func main() {
	actPlugin := tmplActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return

}

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &actionplugin.ActionResult{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
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
