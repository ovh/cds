package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var workerModelCmd = cli.Command{
	Name:  "model",
	Short: "Manage Worker Model",
}

func workerModel() *cobra.Command {
	return cli.NewCommand(workerModelCmd, nil, []*cobra.Command{
		cli.NewListCommand(workerModelListCmd, workerModelListRun, nil),
		cli.NewGetCommand(workerModelShowCmd, workerModelShowRun, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(workerModelDeleteCmd, workerModelDeleteRun, nil),
		cli.NewCommand(workerModelImportCmd, workerModelImportRun, nil),
		cli.NewCommand(workerModelExportCmd, workerModelExportRun, nil, withAllCommandModifiers()...),
	})
}

var workerModelListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS worker models",
	Flags: []cli.Flag{
		{
			Name:      "binary",
			Usage:     "Use this flag to filter worker model list by its binary capabilities",
			ShortHand: "b",
		},
		{
			Name:      "state",
			Usage:     "Use this flag to filter worker model by his state (disabled|error|register|deprecated)",
			ShortHand: "s",
		},
	},
}

func workerModelListRun(v cli.Values) (cli.ListResult, error) {
	var err error
	var workerModels []sdk.Model
	binaryFlag := v.GetString("binary")
	stateFlag := v.GetString("state")

	if binaryFlag != "" {
		workerModels, err = client.WorkerModels(&cdsclient.WorkerModelFilter{
			Binary: binaryFlag,
		})
	} else {
		workerModels, err = client.WorkerModels(&cdsclient.WorkerModelFilter{
			State: stateFlag,
		})
	}

	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workerModels), nil
}

var workerModelImportCmd = cli.Command{
	Name:    "import",
	Example: "cdsctl worker model import my_worker_model_file.yml https://mydomain.com/myworkermodel.yml",
	Long: `
Available model type :
- Docker images ("docker")
- Openstack image ("openstack")
- VSphere image ("vsphere")

For admin:
+ For each type of model you have to indicate the main worker command to run your workflow (example: worker)
+ For Openstack and VSphere model you can indicate a precmd and postcmd that will execute before and after the main worker command
	`,
	Aliases: []string{
		"add",
	},
	VariadicArgs: cli.Arg{
		Name: "path",
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "Use force flag to update your worker model",
			IsValid: func(s string) bool {
				if s != "true" && s != "false" {
					return false
				}
				return true
			},
			Default: "false",
			Type:    cli.FlagBool,
		},
	},
}

func workerModelImportRun(c cli.Values) error {
	force := c.GetBool("force")
	if c.GetString("path") == "" {
		return fmt.Errorf("path for worker model is mandatory")
	}
	files := strings.Split(c.GetString("path"), ",")

	for _, filepath := range files {
		contentFile, format, err := exportentities.OpenPath(filepath)
		if err != nil {
			return err
		}
		formatStr, _ := exportentities.GetFormatStr(format)

		wm, err := client.WorkerModelImport(contentFile, formatStr, force)
		if err != nil {
			_ = contentFile.Close()
			return err
		}
		fmt.Printf("Worker model %s imported with success\n", wm.Name)
		_ = contentFile.Close()
	}

	return nil
}

var workerModelShowCmd = cli.Command{
	Name:    "show",
	Short:   "Show a Worker Model",
	Example: `cdsctl worker model show myGroup/myModel`,
	Args: []cli.Arg{
		{Name: "worker-model-path"},
	},
}

func workerModelShowRun(v cli.Values) (interface{}, error) {
	groupName, modelName, err := cli.ParsePath(v.GetString("worker-model-path"))
	if err != nil {
		return nil, err
	}

	wm, err := client.WorkerModel(groupName, modelName)
	if err != nil {
		return nil, err
	}

	return wm, nil
}

var workerModelDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Delete a CDS worker model",
	Example: `cdsctl worker model delete shared.infra/myModel`,
	Args: []cli.Arg{
		{Name: "worker-model-path"},
	},
}

func workerModelDeleteRun(v cli.Values) error {
	groupName, modelName, err := cli.ParsePath(v.GetString("worker-model-path"))
	if err != nil {
		return err
	}

	if err := client.WorkerModelDelete(groupName, modelName); err != nil {
		if sdk.ErrorIs(err, sdk.ErrNoWorkerModel) && v.GetBool("force") {
			return nil
		}
		return err
	}

	return nil
}

var workerModelExportCmd = cli.Command{
	Name:    "export",
	Short:   "Export a worker model",
	Example: `cdsctl worker model export myGroup/myModel`,
	Args: []cli.Arg{
		{Name: "worker-model-path"},
	},
	Flags: []cli.Flag{
		{
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func workerModelExportRun(c cli.Values) error {
	groupName, modelName, err := cli.ParsePath(c.GetString("worker-model-path"))
	if err != nil {
		return err
	}

	btes, err := client.WorkerModelExport(groupName, modelName, c.GetString("format"))
	if err != nil {
		return err
	}

	fmt.Println(string(btes))
	return nil
}
