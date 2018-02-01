package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	workerModelCmd = cli.Command{
		Name:  "model",
		Short: "Manage Worker Model",
	}

	workerModel = cli.NewCommand(workerModelCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workerModelListCmd, workerModelListRun, nil),
			cli.NewCommand(workerModelAddCmd, workerModelAddRun, nil),
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

var workerModelAddCmd = cli.Command{
	Name:  "add",
	Short: "cdsctl worker model add [name] [docker|openstack|vsphere] --group [group]",
	Long: `
Available model type :
- Docker images ("docker")
- Openstack image ("openstack")
- VSphere image ("vsphere")
	`,
	Args: []cli.Arg{
		{Name: "name"},
		{Name: "type"},
		{Name: "group"},
	},
	Flags: []cli.Flag{
		{
			Name:  "image",
			Kind:  reflect.String,
			Usage: "Image value",
		},
		{
			Name:  "flavor",
			Kind:  reflect.String,
			Usage: "Flavor value (only for openstack)",
		},
		{
			Name:  "userdata",
			Kind:  reflect.String,
			Usage: "Path to UserData file (only for vsphere or openstack)",
		},
	},
}

func workerModelAddRun(c cli.Values) error {
	name := c.GetString("name")
	modelType := c.GetString("type")
	groupName := c.GetString("group")
	userdata := c.GetString("userdata")

	var t string
	var image string
	switch modelType {
	case string(sdk.Docker):
		t = sdk.Docker
		image = c.GetString("image")
		if image == "" {
			sdk.Exit("Error: Docker image not provided (--image)\n")
		}
		break
	case string(sdk.Openstack):
		t = sdk.Openstack
		d := sdk.OpenstackModelData{
			Image:  c.GetString("image"),
			Flavor: c.GetString("flavor"),
		}
		if d.Image == "" {
			return fmt.Errorf("Error: Openstack image not provided (--image)")
		}
		if d.Flavor == "" {
			return fmt.Errorf("Error: Openstack flavor not provided (--flavor)")
		}
		if userdata == "" {
			return fmt.Errorf("Error: Openstack UserData file not provided (--userdata)")
		}
		file, err := ioutil.ReadFile(userdata)
		if err != nil {
			return fmt.Errorf("Error: Cannot read Openstack UserData file (%s)", err)
		}
		d.UserData = base64.StdEncoding.EncodeToString([]byte(file))
		data, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("Error: Cannot marshal model info (%s)", err)
		}
		image = string(data)
		break
	case string(sdk.VSphere):
		t = sdk.VSphere
		d := sdk.OpenstackModelData{
			Image: c.GetString("image"),
		}
		if d.Image == "" {
			return fmt.Errorf("Error: VSphere image not provided (--image)")
		}

		if userdata == "" {
			return fmt.Errorf("Error: VSphere UserData file not provided (--userdata)")
		}
		file, err := ioutil.ReadFile(userdata)
		if err != nil {
			return fmt.Errorf("Error: Cannot read Openstack UserData file (%s)", err)
		}

		rx := regexp.MustCompile(`(?m)(#.*)$`)
		file = rx.ReplaceAll(file, []byte(""))
		d.UserData = strings.Replace(string(file), "\n", " ; ", -1)

		data, err := sdk.JSONWithoutHTMLEncode(d)
		if err != nil {
			return fmt.Errorf("Error: Cannot marshal model info (%s)", err)
		}
		image = string(data)
		break
	default:
		return fmt.Errorf("Unknown worker type: %s", modelType)
	}

	g, err := client.GroupGet(groupName)
	if err != nil {
		return fmt.Errorf("Error : Unable to get group %s : %s", groupName, err)
	}

	if _, err := client.WorkerModelAdd(name, t, image, g.ID); err != nil {
		return fmt.Errorf("Error: cannot add worker model (%s)", err)
	}

	fmt.Printf("Worker model %s added with success", name)
	return nil
}
