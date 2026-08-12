package main

import (
	"context"
	"os"
	"strings"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"

	"github.com/spf13/cobra"
)

var experimentalWorkflowTemplateCmd = cli.Command{
	Name:  "template",
	Short: "CDS Experimental workflow template commands",
}

func experimentalWorkflowTemplate() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowTemplateCmd, nil, []*cobra.Command{
		cli.NewGetCommand(templateGenerateWorkflowCmd, templateGenerateWorkflowFunc, nil, withAllCommandModifiers()...),
	})
}

var templateGenerateWorkflowCmd = cli.Command{
	Name:    "generate-from-file",
	Short:   "Generate workflow from a template file",
	Example: "cdsctl experimental template generate-from-file <path_to_file>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
		{
			Type:      cli.FlagArray,
			Name:      "params",
			ShortHand: "p",
			Usage:     "Specify parameters for template like -p paramKey=paramValue -p param2=value2",
			Default:   "",
		},
		{
			Type:      cli.FlagArray,
			Name:      "vars",
			ShortHand: "v",
			Usage:     "Simulate the existence of variable sets like -v myVarset -v myOtherVarset",
			Default:   "",
		},
	},
}

func templateGenerateWorkflowFunc(v cli.Values) (interface{}, error) {
	path := v.GetString("template-path")
	rawParams := v.GetStringArray("params")

	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var template sdk.V2WorkflowTemplate
	if err := yaml.Unmarshal(bts, &template); err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read template: %v", err)
	}
	params := make(map[string]string)
	for _, p := range rawParams {
		ps := strings.SplitN(p, "=", 2)
		if len(ps) != 2 {
			return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid param %s", p)
		}
		params[ps[0]] = ps[1]
	}

	vars := make([]string, 0)
	for _, name := range v.GetStringArray("vars") {
		if name == "" {
			continue
		}
		vars = append(vars, name)
	}

	req := sdk.V2WorkflowTemplateGenerateRequest{
		Template: template,
		Params:   params,
		Vars:     vars,
	}
	resp, err := client.TemplateGenerateWorkflowFromFile(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
