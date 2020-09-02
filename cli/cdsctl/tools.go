package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

const (
	targetFolderName = ".cds-schema"
	pluginVSCodeName = "redhat.vscode-yaml"
)

var toolsCmd = cli.Command{
	Name:    "tools",
	Aliases: []string{"tool"},
	Short:   "Some tooling for CDS",
}

func tools() *cobra.Command {
	return cli.NewCommand(toolsCmd, nil, []*cobra.Command{
		cli.NewCommand(toolsYamlSchema, toolsYamlSchemaRun, nil, withAllCommandModifiers()...),
	})
}

var toolsYamlSchema = cli.Command{
	Name:    "yaml-schema",
	Short:   "Generate and install CDS yaml schema for given IDE",
	Example: "cdsctl tools yaml-schema vscode",
	Args: []cli.Arg{
		{Name: "ide-name"},
	},
}

type yamlSchemaPath struct {
	Workflow    string
	Pipeline    string
	Application string
	Environment string
}

type yamlSchemaInstaller interface {
	Install(schemas yamlSchemaPath) error
}

type yamlSchemaVSCodeInstaller struct{}

func (y yamlSchemaVSCodeInstaller) Install(schemas yamlSchemaPath) error {
	fmt.Println("Installing yaml-syntax for VSCode.")

	fmt.Println("You will need to execute the following command:")
	fmt.Println(cli.Cyan("code --install-extension %s", pluginVSCodeName))

	// manually constructs a json to preserve rules order
	paths := []string{schemas.Workflow, schemas.Application, schemas.Environment, schemas.Pipeline}
	globPatterns := []string{"*.cds*.yml", "*.cds*.app.yml", "*.cds*.env.yml", "*.cds*.pip.yml"}
	var schs []string
	for i := range paths {
		schs = append(schs, fmt.Sprintf("\n\t\t\"%s\": \"%s\"", paths[i], globPatterns[i]))
	}
	res := fmt.Sprintf("{\n\t\"yaml.schemas\": {%s\n\t}\n}", strings.Join(schs, ","))

	fmt.Println("You need to add the following part in your VSCode settings.json file:")
	fmt.Println(cli.Cyan(res))

	return nil
}

func toolsYamlSchemaRun(v cli.Values) error {
	res, err := client.UserGetSchema()
	if err != nil {
		return err
	}

	var installer yamlSchemaInstaller

	switch v.GetString("ide-name") {
	case "vscode":
		installer = &yamlSchemaVSCodeInstaller{}
	default:
		return fmt.Errorf("Invalid given IDE name")
	}

	home, err := os.UserHomeDir()
	targetFolder := home + "/" + targetFolderName
	if err != nil {
		return fmt.Errorf("Cannot get user home directory info: %s", err)
	}
	if err := os.RemoveAll(targetFolder); err != nil {
		return fmt.Errorf("Cannot remove folder %s: %s", targetFolder, err)
	}
	if err := os.MkdirAll(targetFolder, 0775); err != nil {
		return fmt.Errorf("Cannot create folder %s: %s", targetFolder, err)
	}

	paths := yamlSchemaPath{
		Workflow:    fmt.Sprintf("%s/workflow.schema.json", targetFolder),
		Pipeline:    fmt.Sprintf("%s/pipeline.schema.json", targetFolder),
		Application: fmt.Sprintf("%s/application.schema.json", targetFolder),
		Environment: fmt.Sprintf("%s/environment.schema.json", targetFolder),
	}

	if err := ioutil.WriteFile(paths.Workflow, []byte(res.Workflow), 0775); err != nil {
		return fmt.Errorf("Cannot write file at %s: %s", paths.Workflow, err)
	}
	fmt.Printf("File %s successfully written.\n", paths.Workflow)

	if err := ioutil.WriteFile(paths.Pipeline, []byte(res.Pipeline), 0775); err != nil {
		return fmt.Errorf("Cannot write file at %s: %s", paths.Pipeline, err)
	}
	fmt.Printf("File %s successfully written.\n", paths.Pipeline)

	if err := ioutil.WriteFile(paths.Application, []byte(res.Application), 0775); err != nil {
		return fmt.Errorf("Cannot write file at %s: %s", paths.Application, err)
	}
	fmt.Printf("File %s successfully written.\n", paths.Application)

	if err := ioutil.WriteFile(paths.Environment, []byte(res.Environment), 0775); err != nil {
		return fmt.Errorf("Cannot write file at %s: %s", paths.Environment, err)
	}
	fmt.Printf("File %s successfully written.\n", paths.Environment)

	paths.Workflow = "file://" + paths.Workflow
	paths.Pipeline = "file://" + paths.Pipeline
	paths.Application = "file://" + paths.Application
	paths.Environment = "file://" + paths.Environment

	return installer.Install(paths)
}
