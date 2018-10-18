package main

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
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
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.Slice,
			Name:      "params",
			ShortHand: "p",
			Usage:     "Specify params for template",
			Default:   "",
		},
		{
			Kind:      reflect.Bool,
			Name:      "ignore-prompt",
			ShortHand: "i",
			Usage:     "Set to not ask interactively for params",
		},
	},
}

func templateExecuteRun(v cli.Values) error {
	projectKey := v.GetString(_ProjectKey)

	// try to get the template fon cds
	templateIDString := v.GetString("template-id")
	templateID, err := strconv.ParseInt(templateIDString, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid given template id")
	}
	wt, err := client.TemplateGet(templateID)
	if err != nil {
		return err
	}

	// init params from cli flags
	paramPairs := v.GetStringSlice("params")
	params := make(map[string]string, len(paramPairs))
	for _, p := range paramPairs {
		ps := strings.Split(p, "=")
		if len(ps) < 2 {
			return fmt.Errorf("Invalid given param %s", ps[0])
		}
		params[ps[0]] = strings.Join(ps[1:], "=")
	}

	// for parameters not given with flags, ask interactively if not disabled
	if !v.GetBool("ignore-prompt") {
		for _, p := range wt.Parameters {
			if _, ok := params[p.Key]; !ok {
				fmt.Printf("Value for param %s (type: %s, required: %t): ", p.Key, p.Type, p.Required)
				v, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				params[p.Key] = strings.TrimSuffix(v, "\n")
			}
		}
	}

	req := sdk.WorkflowTemplateRequest{
		Name:       v["name"],
		Parameters: params,
	}

	if err := wt.CheckParams(req); err != nil {
		return err
	}

	res, err := client.TemplateExecute(projectKey, templateID, req)
	if err != nil {
		return err
	}

	for _, r := range res {
		fmt.Println(r)
	}

	return nil
}
