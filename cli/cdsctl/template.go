package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var (
	templateCmd = cli.Command{
		Name:  "template",
		Short: "Manage CDS workflow template",
	}

	template = cli.NewCommand(templateCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(templateExecuteCmd, templateExecuteRun, nil, withAllCommandModifiers()...),
		})
)

var templateExecuteCmd = cli.Command{
	Name:  "execute",
	Short: "Execute CDS workflow template",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "template-id"},
		{Name: "name"},
		{Name: "param-names"},
		{Name: "param-values"},
	},
}

func templateExecuteRun(v cli.Values) error {
	projectKey := v[_ProjectKey]

	templateIDString := v["template-id"]
	templateID, err := strconv.ParseInt(templateIDString, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid given template id")
	}

	paramNames := strings.Split(v["param-names"], ",")
	paramValues := strings.Split(v["param-values"], ",")
	if len(paramNames) != len(paramValues) {
		return fmt.Errorf("Invalid given params, length of params names should be the same as values length")
	}

	params := make(map[string]string, len(paramNames))
	for i := 0; i < len(paramNames); i++ {
		params[paramNames[i]] = paramValues[i]
	}

	res, err := client.TemplateExecute(projectKey, templateID, sdk.WorkflowTemplateRequest{
		Name:       v["name"],
		Parameters: params,
	})
	if err != nil {
		return err
	}

	for _, r := range res {
		fmt.Println(r)
	}

	return nil
}
