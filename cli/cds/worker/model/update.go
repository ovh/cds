package model

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdWorkerModelUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds worker model update <oldname> <name> <type>",
		Long:  `Update name, type and image value only.`,
		Run:   updateWorkerModel,
	}

	cmd.Flags().StringVar(&imageP, "image", "", "Image value (docker or openstack)")
	cmd.Flags().StringVar(&openstackFlavorP, "flavor", "", "Flavor value (openstack)")
	cmd.Flags().StringVar(&userDataFileP, "userdata", "", "Path to UserData file (openstack)")
	return cmd
}

func updateWorkerModel(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	workerModelName := args[0]
	name := args[1]
	typeS := args[2]
	var value string
	//value := args[3]

	var t string
	switch typeS {
	case string(sdk.Docker):
		t = sdk.Docker
		value = imageP
		if imageP == "" {
			sdk.Exit("Error: Docker image not provided (--image)\n")
		}
		break
	case string(sdk.HostProcess):
		t = sdk.HostProcess
		value = imageP
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
		if userDataFileP == "" {
			sdk.Exit("Error: Openstack UserData file not provided (--userdata)\n")
		}
		file, err := ioutil.ReadFile(userDataFileP)
		if err != nil {
			sdk.Exit("Error: Cannot read Openstack UserData file (%s)\n", err)
		}
		d.UserData = base64.StdEncoding.EncodeToString([]byte(file))
		data, err := json.Marshal(d)
		if err != nil {
			sdk.Exit("Error: Cannot marshal model info (%s)\n", err)
		}
		value = string(data)
		break
	case string(sdk.VSphere):
		t = sdk.VSphere
		d := sdk.OpenstackModelData{
			Image: imageP,
		}
		if d.Image == "" {
			sdk.Exit("Error: VSphere image not provided (--image)\n")
		}
		if userDataFileP == "" {
			sdk.Exit("Error: VSphere UserData file not provided (--userdata)\n")
		}
		file, err := ioutil.ReadFile(userDataFileP)
		if err != nil {
			sdk.Exit("Error: Cannot read VSphere UserData file (%s)\n", err)
		}

		rx := regexp.MustCompile(`(?m)(#.*)$`)
		file = rx.ReplaceAll(file, []byte(""))
		d.UserData = strings.Replace(string(file), "\n", " ; ", -1)

		data, err := jsonWithoutHTMLEncode(d)
		if err != nil {
			sdk.Exit("Error: Cannot marshal model info (%s)\n", err)
		}
		value = string(data)
		break
	default:
		sdk.Exit("Unknown worker type: %s\n", typeS)
	}

	m, err := sdk.GetWorkerModel(workerModelName)
	if err != nil {
		sdk.Exit("Error: cannot retrieve worker model %s (%s)\n", workerModelName, err)
	}
	err = sdk.UpdateWorkerModel(m.ID, name, t, value)
	if err != nil {
		sdk.Exit("Error: cannot update model (%s)\n", err)
	}
}
