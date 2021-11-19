package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminIntegrationModelsCmd = cli.Command{
	Name:  "integration-model",
	Short: "Manage CDS Integration models",
}

func adminIntegrationModels() *cobra.Command {
	return cli.NewCommand(adminIntegrationModelsCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminIntegrationModelsListCmd, adminIntegrationModelsListRun, nil),
		cli.NewGetCommand(adminIntegrationModelShowCmd, adminIntegrationModelShowRun, nil),
		cli.NewCommand(adminIntegrationModelExportCmd, adminIntegrationModelExportRun, nil),
		cli.NewCommand(adminIntegrationModelImportCmd, adminIntegrationModelImportRun, nil),
		cli.NewDeleteCommand(adminIntegrationModelDeleteCmd, adminIntegrationModelDeleteRun, nil),
	})
}

// List command
var adminIntegrationModelsListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS Integration models",
}

func adminIntegrationModelsListRun(v cli.Values) (cli.ListResult, error) {
	models, err := client.IntegrationModelList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(models), nil
}

// Show command
var adminIntegrationModelShowCmd = cli.Command{
	Name:  "show",
	Short: "Show details of a CDS Integration model",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminIntegrationModelShowRun(v cli.Values) (interface{}, error) {
	model, err := client.IntegrationModelGet(v.GetString("name"))
	if err != nil {
		return nil, err
	}
	return model, nil
}

// Export command
var adminIntegrationModelExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a CDS Integration model",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminIntegrationModelExportRun(v cli.Values) error {
	model, err := client.IntegrationModelGet(v.GetString("name"))
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(model)
	if err != nil {
		return cli.WrapError(err, "unable to marshal content")
	}

	fmt.Println(string(b))
	return nil
}

// Import command
var adminIntegrationModelImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a CDS Integration model from a yaml file",
	Args: []cli.Arg{
		{
			Name: "file",
		},
	},
}

func adminIntegrationModelImportRun(v cli.Values) error {
	b, err := os.ReadFile(v.GetString("file"))
	if err != nil {
		return cli.WrapError(err, "unable to read file %s", v.GetString("file"))
	}

	m := new(sdk.IntegrationModel)
	if err := yaml.Unmarshal(b, m); err != nil {
		return cli.WrapError(err, "unable to load file")
	}

	//Try to load the model to know if we have to add it or update it
	model, _ := client.IntegrationModelGet(m.Name)
	if model.ID == 0 { // If the model has not been found
		return client.IntegrationModelAdd(m)
	}

	return client.IntegrationModelUpdate(m)
}

var adminIntegrationModelDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS Integration model",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func adminIntegrationModelDeleteRun(v cli.Values) error {
	return client.IntegrationModelDelete(v.GetString("name"))
}
