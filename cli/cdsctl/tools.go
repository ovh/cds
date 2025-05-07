package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
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

type yamlSchemaPathV2 struct {
	Workflow         string
	WorkerModel      string
	Action           string
	Job              string
	WorkflowTemplate string
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
	v1, err := client.UserGetSchema(context.Background())
	if err != nil {
		return err
	}

	v2Workflow, err := client.UserGetSchemaV2(context.Background(), sdk.EntityTypeWorkflow)
	if err != nil {
		return err
	}

	v2WorkerModel, err := client.UserGetSchemaV2(context.Background(), sdk.EntityTypeWorkerModel)
	if err != nil {
		return err
	}

	v2Action, err := client.UserGetSchemaV2(context.Background(), sdk.EntityTypeAction)
	if err != nil {
		return err
	}

	v2Job, err := client.UserGetSchemaV2(context.Background(), sdk.EntityTypeJob)
	if err != nil {
		return err
	}

	v2WorkflowTemplate, err := client.UserGetSchemaV2(context.Background(), sdk.EntityTypeWorkflowTemplate)
	if err != nil {
		return err
	}

	var installer yamlSchemaInstaller

	switch v.GetString("ide-name") {
	case "vscode":
		installer = &yamlSchemaVSCodeInstaller{}
	default:
		return cli.NewError("Invalid given IDE name")
	}

	home, err := os.UserHomeDir()
	targetFolder := home + "/" + targetFolderName
	if err != nil {
		return cli.WrapError(err, "Cannot get user home directory info")
	}
	if err := os.RemoveAll(targetFolder); err != nil {
		return cli.WrapError(err, "Cannot remove folder %s", targetFolder)
	}
	if err := os.MkdirAll(targetFolder, 0775); err != nil {
		return cli.WrapError(err, "Cannot create folder %s", targetFolder)
	}

	paths := yamlSchemaPath{
		Workflow:    path.Join(targetFolder, "workflow.schema.json"),
		Pipeline:    path.Join(targetFolder, "pipeline.schema.json"),
		Application: path.Join(targetFolder, "application.schema.json"),
		Environment: path.Join(targetFolder, "environment.schema.json"),
	}

	pathsV2 := yamlSchemaPathV2{
		Workflow:         path.Join(targetFolder, "workflow.v2.schema.json"),
		WorkerModel:      path.Join(targetFolder, "worker-model.v2.schema.json"),
		Action:           path.Join(targetFolder, "action.v2.schema.json"),
		Job:              path.Join(targetFolder, "job.v2.schema.json"),
		WorkflowTemplate: path.Join(targetFolder, "workflow-template.v2.schema.json"),
	}

	if err := os.WriteFile(paths.Workflow, []byte(v1.Workflow), 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", paths.Workflow)
	}
	fmt.Printf("File %s successfully written.\n", paths.Workflow)

	if err := os.WriteFile(paths.Pipeline, []byte(v1.Pipeline), 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", paths.Pipeline)
	}
	fmt.Printf("File %s successfully written.\n", paths.Pipeline)

	if err := os.WriteFile(paths.Application, []byte(v1.Application), 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", paths.Application)
	}
	fmt.Printf("File %s successfully written.\n", paths.Application)

	if err := os.WriteFile(paths.Environment, []byte(v1.Environment), 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", paths.Environment)
	}
	fmt.Printf("File %s successfully written.\n", paths.Environment)

	if err := os.WriteFile(pathsV2.Workflow, v2Workflow, 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", pathsV2.Workflow)
	}
	fmt.Printf("File %s successfully written.\n", pathsV2.Workflow)

	if err := os.WriteFile(pathsV2.WorkerModel, v2WorkerModel, 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", pathsV2.WorkerModel)
	}
	fmt.Printf("File %s successfully written.\n", pathsV2.WorkerModel)

	if err := os.WriteFile(pathsV2.Action, v2Action, 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", pathsV2.Action)
	}
	fmt.Printf("File %s successfully written.\n", pathsV2.Action)

	if err := os.WriteFile(pathsV2.Job, v2Job, 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", pathsV2.Job)
	}
	fmt.Printf("File %s successfully written.\n", pathsV2.Job)

	if err := os.WriteFile(pathsV2.WorkflowTemplate, v2WorkflowTemplate, 0644); err != nil {
		return cli.WrapError(err, "Cannot write file at %s", pathsV2.WorkflowTemplate)
	}
	fmt.Printf("File %s successfully written.\n", pathsV2.WorkflowTemplate)

	paths.Workflow = "file://" + paths.Workflow
	paths.Pipeline = "file://" + paths.Pipeline
	paths.Application = "file://" + paths.Application
	paths.Environment = "file://" + paths.Environment

	return installer.Install(paths)
}
