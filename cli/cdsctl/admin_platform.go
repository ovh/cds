package main

import (
	"fmt"
	"io/ioutil"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	adminPlatformModelsCmd = cli.Command{
		Name:  "platform-model",
		Short: "Manage CDS Platform models",
	}

	adminPlatformModels = cli.NewCommand(adminPlatformModelsCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminPlatformModelsListCmd, adminPlatformModelsListRun, nil),
			cli.NewGetCommand(adminPlatformModelShowCmd, adminPlatformModelShowRun, nil),
			cli.NewCommand(adminPlatformModelExportCmd, adminPlatformModelExportRun, nil),
			cli.NewCommand(adminPlatformModelImportCmd, adminPlatformModelImportRun, nil),
		})
)

// List command
var adminPlatformModelsListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS Platform models",
}

func adminPlatformModelsListRun(v cli.Values) (cli.ListResult, error) {
	models, err := client.PlatformModelList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(models), nil
}

// Show command
var adminPlatformModelShowCmd = cli.Command{
	Name:  "show",
	Short: "Show details of a CDS Platform model",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminPlatformModelShowRun(v cli.Values) (interface{}, error) {
	model, err := client.PlatformModelGet(v.GetString("name"))
	if err != nil {
		return nil, err
	}
	return model, nil
}

// Export command
var adminPlatformModelExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a CDS Platform model",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminPlatformModelExportRun(v cli.Values) error {
	model, err := client.PlatformModelGet(v.GetString("name"))
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(model)
	if err != nil {
		return fmt.Errorf("unable to marshal: %v", err)
	}

	fmt.Println(string(b))
	return nil
}

// Import command
var adminPlatformModelImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a CDS Platform model from a yaml file",
	Args: []cli.Arg{
		{
			Name: "file",
		},
	},
}

func adminPlatformModelImportRun(v cli.Values) error {
	b, err := ioutil.ReadFile(v.GetString("file"))
	if err != nil {
		return fmt.Errorf("unable to read file %s: %v", v.GetString("file"), err)
	}

	m := new(sdk.PlatformModel)
	if err := yaml.Unmarshal(b, m); err != nil {
		return fmt.Errorf("unable to load file: %v", err)
	}

	model, _ := client.PlatformModelGet(m.Name)
	if model.ID == 0 {
		if err := client.PlatformModelAdd(m); err != nil {
			return err
		}
		return nil
	}

	return client.PlatformModelUpdate(m)
}
