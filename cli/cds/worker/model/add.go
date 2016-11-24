package model

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	imageP                 string
	openstackFlavorP       string
	openstackUserDataFileP string
)

func cmdWorkerModelAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds worker model add <name> <type>",
		Long: `
		Available model type :
		- Docker images ("docker")
		- Openstack image ("openstack")
		`,
		Run: addWorkerModel,
	}

	cmd.Flags().StringVar(&imageP, "image", "", "Image value (docker or openstack)")
	cmd.Flags().StringVar(&openstackFlavorP, "flavor", "", "Flavor value (openstack)")
	cmd.Flags().StringVar(&openstackUserDataFileP, "userdata", "", "Path to UserData file (openstack)")

	return cmd
}

func addWorkerModel(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]
	modelType := args[1]

	var t sdk.WorkerType
	var image string
	switch modelType {
	case string(sdk.Docker):
		t = sdk.Docker
		image = imageP
		if image == "" {
			sdk.Exit("Error: Docker image not provided (--image)\n")
		}
		break
	case string(sdk.Openstack):
		t = sdk.Openstack
		d := sdk.OpenstackModelData{
			Image:  imageP,
			Flavor: openstackFlavorP,
		}
		if d.Image == "" {
			sdk.Exit("Error: Openstack image not provided (--image)\n")
		}
		if d.Flavor == "" {
			sdk.Exit("Error: Openstack flavor not provided (--flavor)\n")
		}
		if openstackUserDataFileP == "" {
			sdk.Exit("Error: Openstack UserData file not provided (--userdata)\n")
		}
		file, err := ioutil.ReadFile(openstackUserDataFileP)
		if err != nil {
			sdk.Exit("Error: Cannot read Openstack UserData file (%s)\n", err)
		}
		d.UserData = base64.StdEncoding.EncodeToString([]byte(file))
		data, err := json.Marshal(d)
		if err != nil {
			sdk.Exit("Error: Cannot marshal model info (%s)\n", err)
		}
		image = string(data)
		break
	default:
		sdk.Exit("Unknown worker type: %s\n", modelType)
	}

	_, err := sdk.AddWorkerModel(name, t, image)
	if err != nil {
		sdk.Exit("Error: cannot add worker model (%s)\n", err)
	}
}

func cmdWorkerModelCapabilityAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds worker model capability add <workerModelName> <name> <type> <value>",
		Long: `
		Available capability types:
		- Binary installed ("binary")
		- Network access ("network")
		`,
		Run: addWorkerModelCapability,
	}

	return cmd
}

func addWorkerModelCapability(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	workerModelName := args[0]
	name := args[1]
	typeS := args[2]
	value := args[3]

	var t sdk.RequirementType
	switch typeS {
	case string(sdk.BinaryRequirement):
		t = sdk.BinaryRequirement
		break
	case string(sdk.NetworkAccessRequirement):
		t = sdk.NetworkAccessRequirement
		break
	}

	m, err := sdk.GetWorkerModel(workerModelName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve worker model %s (%s)\n", workerModelName, err)
	}

	err = sdk.AddCapabilityToWorkerModel(m.ID, name, t, value)
	if err != nil {
		sdk.Exit("Error: cannot add capability to model (%s)\n", err)
	}
}
