package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var experimentalWorkerModelCmd = cli.Command{
	Name:    "worker-model",
	Aliases: []string{"wm"},
	Short:   "CDS Experimental worker model commands",
}

func experimentalWorkerModel() *cobra.Command {
	return cli.NewCommand(experimentalWorkerModelCmd, nil, []*cobra.Command{
		cli.NewListCommand(wmListCmd, workerModelListFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(wmMigrateCmd, workerModelMigrateFunc, nil, withAllCommandModifiers()...),
	})
}

var wmMigrateCmd = cli.Command{
	Name:    "migrate",
	Example: "cdsctl worker-model migrate <group_name> <model_name>",
	Short:   "Display the v2 worker model yaml file from an existing v1 worker model ",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "group"},
		{Name: "model"},
	},
}

func workerModelMigrateFunc(v cli.Values) error {
	groupName := v.GetString("group")
	model := v.GetString("model")

	workerModel, err := client.WorkerModelGet(groupName, model)
	if err != nil {
		return err
	}

	modelV2 := sdk.V2WorkerModel{
		Name:        workerModel.Name,
		Description: workerModel.Description,
		Type:        workerModel.Type,
		OSArch:      "linux/amd64",
	}
	if workerModel.RegisteredOS != nil && workerModel.RegisteredArch != nil {
		modelV2.OSArch = *workerModel.RegisteredOS + "/" + *workerModel.RegisteredArch
	}
	switch modelV2.Type {
	case sdk.WorkerModelTypeOpenstack:
		openstackSpec := sdk.V2WorkerModelOpenstackSpec{
			Image: workerModel.ModelVirtualMachine.Image,
		}
		bts, _ := json.Marshal(openstackSpec)
		modelV2.Spec = bts
	case sdk.WorkerModelTypeDocker:
		dockerSpec := sdk.V2WorkerModelDockerSpec{
			Image: workerModel.ModelDocker.Image,
			Envs:  make(map[string]string),
		}
		for k, v := range workerModel.ModelDocker.Envs {
			if strings.HasPrefix(k, "CDS_") {
				continue
			}
			dockerSpec.Envs[k] = v
		}
		bts, _ := json.Marshal(dockerSpec)
		modelV2.Spec = bts
	case sdk.WorkerModelTypeVSphere:
		vsphereSpec := sdk.V2WorkerModelVSphereSpec{
			Image: workerModel.ModelVirtualMachine.Image,
		}
		bts, _ := json.Marshal(vsphereSpec)
		modelV2.Spec = bts
	}

	bts, _ := yaml.Marshal(modelV2)
	fmt.Printf("%s\n", string(bts))

	return nil
}

var wmListCmd = cli.Command{
	Name:    "list",
	Example: "cdsctl worker-model list",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository"},
	},
	Flags: []cli.Flag{
		{Name: "branch", Usage: "Filter on a specific branch"},
	},
}

func workerModelListFunc(v cli.Values) (cli.ListResult, error) {
	vcsName := v.GetString("vcs-name")
	repositoryName := v.GetString("repository")

	branch := v.GetString("branch")
	var filter *cdsclient.WorkerModelV2Filter
	if branch != "" {
		filter = &cdsclient.WorkerModelV2Filter{
			Branch: branch,
		}
	}

	wms, err := client.WorkerModelv2List(context.Background(), v.GetString(_ProjectKey), vcsName, repositoryName, filter)
	if err != nil {
		return nil, err
	}

	return cli.AsListResult(wms), err
}
