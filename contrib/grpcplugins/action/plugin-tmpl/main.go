package main

import (
	"context"
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
$ make build tmpl
$ make publish tmpl
*/

type tmplActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *tmplActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:   "plugin-tmpl",
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
		return actionplugin.Fail("Missing template file")
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
		return actionplugin.Fail("Failed to read template file: %v", err)
	}

	// parse the template file
	tmpl, err := template.New("file").Funcs(funcMap).Parse(string(content))
	if err != nil {
		return actionplugin.Fail("Failed to parse template file: %v", err)
	}

	// open the output file
	of, err := os.Create(output)
	if err != nil {
		return actionplugin.Fail("Failed to create output file: %v", err)
	}
	defer of.Close()

	// parse template parameters if any
	tmplParams, err := parseTemplateParameters(params)
	if err != nil {
		return actionplugin.Fail("Failed to parse template parameters: %v", err)
	}

	// finally, execute the template
	if err := tmpl.Execute(of, tmplParams); err != nil {
		return actionplugin.Fail("Failed to execute template: %v", err)
	}

	fmt.Printf("Generated output file %s", output)

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := tmplActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
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
