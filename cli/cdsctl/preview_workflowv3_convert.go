package main

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/workflowv3"
)

var workflowV3ConvertCmd = cli.Command{
	Name:  "workflowv3-convert",
	Short: "Convert existing workflow to Workflow V3 files.",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Flags: []cli.Flag{
		{
			Name:  "full",
			Type:  cli.FlagBool,
			Usage: "Set the flag to export pipeline, application and environment content.",
		},
		{
			Name:    "format",
			Type:    cli.FlagString,
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func workflowV3ConvertRun(v cli.Values) error {
	isFullExport := v.GetBool("full")

	w, err := client.WorkflowGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), cdsclient.WithDeepPipelines())
	if err != nil {
		return err
	}

	res := workflowv3.Convert(*w, isFullExport)

	format := v.GetString("format")
	var buf []byte
	switch format {
	case "yaml":
		buf, err = yaml.Marshal(res)
	case "json":
		buf, err = json.Marshal(res)
	default:
		return fmt.Errorf("invalid given export format %q", format)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(buf))

	return nil
}
