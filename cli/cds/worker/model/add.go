package model

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	imageP           string
	openstackFlavorP string
	userDataFileP    string
	groupName        string
)

func cmdWorkerModelAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds worker model add <name> <type> --group <group>",
		Long: `
		Available model type :
		- Docker images ("docker")
		- Openstack image ("openstack")
		- VSphere image ("vsphere")
		`,
		Run: addWorkerModel,
	}

	cmd.Flags().StringVar(&imageP, "image", "", "Image value (docker, openstack, vsphere)")
	cmd.Flags().StringVar(&openstackFlavorP, "flavor", "", "Flavor value (openstack)")
	cmd.Flags().StringVar(&userDataFileP, "userdata", "", "Path to UserData file (vsphere or openstack)")
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
		image = string(data)
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
			sdk.Exit("Error: Cannot read Openstack UserData file (%s)\n", err)
		}

		rx := regexp.MustCompile(`(?m)(#.*)$`)
		file = rx.ReplaceAll(file, []byte(""))
		d.UserData = strings.Replace(string(file), "\n", " ; ", -1)

		data, err := jsonWithoutHTMLEncode(d)
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

func jsonWithoutHTMLEncode(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
