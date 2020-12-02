package main

import (
	"fmt"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var workflowImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a workflow",
	Long: `
In case you want to import just your workflow. Instead of use a local file you can also use an URL to your yaml file.

If you want to update also dependencies likes pipelines, applications or environments at same time you have to use workflow push instead workflow import.

Without --force, CDS won't update an existing workflow.
With --force, CDS will allow you to update an existing workflow. If this workflow is managed 'as-code', CDS will
override it. This workflow will be detached from the repository, until it is re-imported again following a commit on the repo.
	`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Type:    cli.FlagBool,
			Name:    "force",
			Usage:   "Override workflow if exists",
			Default: "false",
		},
	},
}

func workflowImportRun(c cli.Values) error {
	path := c.GetString("path")
	contentFile, format, err := exportentities.OpenPath(path)
	if err != nil {
		return err
	}
	defer contentFile.Close() //nolint

	mods := []cdsclient.RequestModifier{
		cdsclient.ContentType(format.ContentType()),
	}
	if c.GetBool("force") {
		mods = append(mods, cdsclient.Force())
	}

	msgs, err := client.WorkflowImport(c.GetString(_ProjectKey), contentFile, mods...)
	for _, s := range msgs {
		fmt.Println(s)
	}
	return err
}
