package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	workerModelCmd = cli.Command{
		Name:  "model",
		Short: "Manage Worker Model",
	}

	workerModel = cli.NewCommand(workerModelCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workerModelListCmd, workerModelListRun, nil),
			cli.NewGetCommand(workerModelShowCmd, workerModelShowRun, nil, withAllCommandModifiers()...),
			cli.NewDeleteCommand(workerModelDeleteCmd, workerModelDeleteRun, nil),
			cli.NewCommand(workerModelImportCmd, workerModelImportRun, nil),
		})
)

var workerModelListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS worker models",
}

func workerModelListRun(v cli.Values) (cli.ListResult, error) {
	workerModels, err := client.WorkerModels()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workerModels), nil
}

var workerModelImportCmd = cli.Command{
	Name:    "import",
	Example: "cdsctl worker model import my_worker_model_file.yml",
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
		Name: "filepath",
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
			Kind:    reflect.Bool,
		},
	},
}

type workerModelFile struct {
	Name          string            `json:"name" yaml:"name"`
	Group         string            `json:"group" yaml:"group"`
	Communication string            `json:"communication,omitempty" yaml:"communication,omitempty"`
	Provision     int               `json:"provision,omitempty" yaml:"provision,omitempty"`
	Image         string            `json:"image" yaml:"image"`
	Description   string            `json:"description" yaml:"description"`
	Type          string            `json:"type" yaml:"type"`
	Flavor        string            `json:"flavor,omitempty" yaml:"flavor,omitempty"`
	Envs          map[string]string `json:"envs,omitempty" yaml:"envs,omitempty"`
	Shell         string            `json:"shell,omitempty" yaml:"shell,omitempty"`
	PreCmd        string            `json:"pre_cmd,omitempty" yaml:"pre_cmd,omitempty"`
	Cmd           string            `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	PostCmd       string            `json:"post_cmd,omitempty" yaml:"post_cmd,omitempty"`
	Restricted    bool              `json:"restricted" yaml:"restricted"`
}

func workerModelImportRun(c cli.Values) error {
	force := c.GetBool("force")
	if c.GetString("filepath") == "" {
		return fmt.Errorf("filepath for worker model is mandatory")
	}
	files := strings.Split(c.GetString("filepath"), ",")

	for _, filepath := range files {
		reader, format, err := exportentities.OpenFile(filepath)
		if err != nil {
			return fmt.Errorf("Error: Cannot read file %s (%v)", filepath, err)
		}

		buf := new(bytes.Buffer)
		if _, errR := buf.ReadFrom(reader); errR != nil {
			reader.Close()
			return fmt.Errorf("Error: cannot read file content %s : %v", filepath, errR)
		}
		reader.Close()

		var modelInfos workerModelFile
		switch format {
		case exportentities.FormatJSON:
			if err := json.Unmarshal(buf.Bytes(), &modelInfos); err != nil {
				return fmt.Errorf("Error: cannot unmarshal json file %s : %v", filepath, err)
			}
		case exportentities.FormatYAML:
			if err := yaml.Unmarshal(buf.Bytes(), &modelInfos); err != nil {
				return fmt.Errorf("Error: cannot unmarshal yaml file %s : %v", filepath, err)
			}
		default:
			return fmt.Errorf("Invalid file format")
		}

		var t string
		var modelDocker sdk.ModelDocker
		var modelVM sdk.ModelVirtualMachine
		switch modelInfos.Type {
		case sdk.Docker:
			t = sdk.Docker
			if modelInfos.Image == "" {
				sdk.Exit("Error: Docker image not provided\n")
			}
			modelDocker.Shell = modelInfos.Shell
			modelDocker.Image = modelInfos.Image
			modelDocker.Cmd = modelInfos.Cmd
			if modelDocker.Shell == "" {
				sdk.Exit("Error: main shell command not provided\n")
			}
			if modelDocker.Cmd == "" {
				sdk.Exit("Error: main worker command not provided\n")
			}
			break
		case sdk.Openstack:
			t = sdk.Openstack
			d := sdk.ModelVirtualMachine{
				Image:   modelInfos.Image,
				Flavor:  modelInfos.Flavor,
				Cmd:     modelInfos.Cmd,
				PostCmd: modelInfos.PostCmd,
				PreCmd:  modelInfos.PreCmd,
			}
			if d.Image == "" {
				return fmt.Errorf("Error: Openstack image not provided")
			}
			if d.Flavor == "" {
				return fmt.Errorf("Error: Openstack flavor not provided")
			}
			if d.Cmd == "" {
				return fmt.Errorf("Error: Openstack command not provided")
			}
			modelVM = d
			break
		case sdk.VSphere:
			t = sdk.VSphere
			d := sdk.ModelVirtualMachine{
				Image:   modelInfos.Image,
				Flavor:  modelInfos.Flavor,
				Cmd:     modelInfos.Cmd,
				PostCmd: modelInfos.PostCmd,
				PreCmd:  modelInfos.PreCmd,
			}
			if d.Image == "" {
				return fmt.Errorf("Error: VSphere image not provided")
			}

			if d.Cmd == "" {
				return fmt.Errorf("Error: VSphere main worker command empty")
			}

			modelVM = d
			break
		default:
			return fmt.Errorf("Unknown worker type: %s", modelInfos.Type)
		}

		if modelInfos.Name == "" {
			return fmt.Errorf("Error: worker model name is not provided")
		}

		if modelInfos.Group == "" {
			return fmt.Errorf("Error: group is not provided")
		}

		g, err := client.GroupGet(modelInfos.Group)
		if err != nil {
			return fmt.Errorf("Error : Unable to get group %s : %s", modelInfos.Group, err)
		}

		if force {
			if existingWm, err := client.WorkerModel(modelInfos.Name); err != nil {
				if _, errAdd := client.WorkerModelAdd(modelInfos.Name, t, &modelDocker, &modelVM, g.ID); errAdd != nil {
					return fmt.Errorf("Error: cannot add worker model %s (%s)", modelInfos.Name, errAdd)
				}
				fmt.Printf("Worker model %s added with success", modelInfos.Name)
			} else {
				if _, errU := client.WorkerModelUpdate(existingWm.ID, modelInfos.Name, t, &modelDocker, &modelVM, g.ID); errU != nil {
					return fmt.Errorf("Error: cannot update worker model %s (%s)", modelInfos.Name, errU)
				}
				fmt.Printf("Worker model %s updated with success", modelInfos.Name)
			}
		} else {
			if _, errAdd := client.WorkerModelAdd(modelInfos.Name, t, &modelDocker, &modelVM, g.ID); errAdd != nil {
				return fmt.Errorf("Error: cannot add worker model %s (%s)", modelInfos.Name, errAdd)
			}
			fmt.Printf("Worker model %s added with success", modelInfos.Name)
		}
	}

	return nil
}

var workerModelShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a Worker Model",
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func workerModelShowRun(v cli.Values) (interface{}, error) {
	wm, err := client.WorkerModel(v["name"])
	if err != nil {
		return nil, err
	}
	return wm, nil
}

var workerModelDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Delete a CDS worker model",
	Example: `cdsctl worker model delete myModelA myModelB`,
	VariadicArgs: cli.Arg{
		Name: "name",
	},
}

func workerModelDeleteRun(v cli.Values) error {
	if err := client.WorkerModelDelete(v.GetString("name")); err != nil {
		if sdk.ErrorIs(err, sdk.ErrNoWorkerModel) && v.GetBool("force") {
			return nil
		}
		return err
	}
	return nil
}
