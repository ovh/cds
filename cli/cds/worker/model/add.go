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
	groupName              string
)

func cmdWorkerModelAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds worker model add <name> <type> --group <group>",
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
	cmd.Flags().StringVar(&groupName, "group", "", "Group name")

	return cmd
}

func addWorkerModel(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]
	modelType := args[1]

	var t string
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

	if groupName == "" {
		sdk.Exit("Wrong usage : Missing group\n")
	}

	g, err := sdk.GetGroup(groupName)
	if err != nil {
		sdk.Exit("Error : Unable to get group %s : %s\n", groupName, err)
	}

	if _, err := sdk.AddWorkerModel(name, t, image, g.ID); err != nil {
		sdk.Exit("Error: cannot add worker model (%s)\n", err)
	}
}
