package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var schemaCmd = cli.Command{
	Name:  "schema",
	Short: "Manage JSON schemas and examples",
}

func schema() *cobra.Command {
	return cli.NewCommand(schemaCmd, nil, []*cobra.Command{
		cli.NewCommand(schemaWorkerModelCmd, schemaWorkerModelRun, nil),
		cli.NewCommand(schemaActionCmd, schemaActionRun, nil),
		cli.NewCommand(schemaWorkflowCmd, schemaWorkflowRun, nil),
	})
}

var schemaWorkerModelCmd = cli.Command{
	Name:  "worker-model",
	Short: "Generate worker model YAML example with descriptions",
	Flags: []cli.Flag{
		{
			Name:    "output",
			Type:    cli.FlagString,
			Default: "",
			Usage:   "Output file (default: stdout)",
		},
	},
}

func schemaWorkerModelRun(v cli.Values) error {
	schema := sdk.GetWorkerModelJsonSchema()
	gen := sdk.NewYAMLGenerator()

	out := os.Stdout
	if outputFile := v.GetString("output"); outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return cli.NewError("failed to create output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	if err := gen.Generate(out, schema); err != nil {
		return cli.NewError("failed to generate YAML: %v", err)
	}

	return nil
}

var schemaActionCmd = cli.Command{
	Name:  "action",
	Short: "Generate action YAML example with descriptions",
	Flags: []cli.Flag{
		{
			Name:    "output",
			Type:    cli.FlagString,
			Default: "",
			Usage:   "Output file (default: stdout)",
		},
	},
}

func schemaActionRun(v cli.Values) error {
	// TODO: récupérer la liste des actions publiques depuis l'API
	publicActions := []string{"checkout", "script", "coverage", "artifact-upload", "artifact-download"}

	schema := sdk.GetActionJsonSchema(publicActions)
	gen := sdk.NewYAMLGenerator()

	out := os.Stdout
	if outputFile := v.GetString("output"); outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return cli.NewError("failed to create output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	if err := gen.Generate(out, schema); err != nil {
		return cli.NewError("failed to generate YAML: %v", err)
	}

	return nil
}

var schemaWorkflowCmd = cli.Command{
	Name:  "workflow",
	Short: "Generate workflow YAML example with descriptions",
	Flags: []cli.Flag{
		{
			Name:    "output",
			Type:    cli.FlagString,
			Default: "",
			Usage:   "Output file (default: stdout)",
		},
	},
}

func schemaWorkflowRun(v cli.Values) error {
	// TODO: récupérer la liste des actions publiques, régions, worker models si besoin
	publicActions := []string{"checkout", "script", "coverage", "artifact-upload", "artifact-download"}
	regionNames := []string{"par1", "gra1"}
	workerModels := []string{"my-docker-model", "my-vsphere-model"}

	schema := sdk.GetWorkflowJsonSchema(publicActions, regionNames, workerModels)
	gen := sdk.NewYAMLGenerator()

	out := os.Stdout
	if outputFile := v.GetString("output"); outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return cli.NewError("failed to create output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	if err := gen.Generate(out, schema); err != nil {
		return cli.NewError("failed to generate YAML: %v", err)
	}

	return nil
}

func init() {
	// plus d'affectation Run, wiring fait via cli.NewCommand dans schema()
}
