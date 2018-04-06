package main

import (
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/urfave/cli"

	"github.com/ovh/cds/sdk"
)

var (
	osRegionName string
)

func init() {
	osRegionName = os.Getenv("OS_REGION_NAME")
}

func main() {
	app := cli.NewApp()
	app.Name = "os-ansible-inventory"
	app.Usage = "Openstack Ansible Inventory based on metadata"
	app.Version = sdk.VERSION
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name: "filter",
		},
		cli.StringFlag{
			Name: "group-by",
		},
		cli.StringFlag{
			Name: "net",
		},
	}
	app.Action = func(c *cli.Context) error {
		filters := c.StringSlice("filter")

		fList := []Filter{}
		for _, f := range filters {
			tuple := strings.Split(f, "=")
			if len(tuple) == 2 {
				fList = append(fList, Filter{
					Key:      tuple[0],
					Value:    tuple[1],
					Operator: "=",
				})
			}

			tuple = strings.Split(f, "~")
			if len(tuple) == 2 {
				fList = append(fList, Filter{
					Key:      tuple[0],
					Value:    tuple[1],
					Operator: "~",
				})
			}
		}

		groupBy := c.String("group-by")
		net := c.String("net")

		return do(fList, groupBy, net)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func do(filters []Filter, groupBy, net string) error {
	auth, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}

	provider, err := openstack.AuthenticatedClient(auth)
	if err != nil {
		return err
	}

	nova, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{Region: osRegionName})
	if err != nil {
		return err
	}

	all, err := servers.List(nova, nil).AllPages()
	if err != nil {
		return err
	}
	serverList, err := servers.ExtractServers(all)
	if err != nil {
		return err
	}

	list := &ServerList{
		l: serverList,
	}

	filter(list, filters)
	if list.err != nil {
		return list.err
	}

	return render(list, groupBy, net)
}

func filter(l *ServerList, filters []Filter) {
	for _, f := range filters {
		if l.err != nil {
			return
		}
		l.Filter(f)
	}
}

func getIPAdress(s *servers.Server, net string, netVersion int) string {
	if netVersion == 0 {
		netVersion = 4
	}

	address := map[string]string{}

	for k, v := range s.Addresses {
		switch v.(type) {
		case []interface{}:
			for _, z := range v.([]interface{}) {
				var addr string
				var version int
				for x, y := range z.(map[string]interface{}) {
					if x == "addr" {
						addr = y.(string)
					}
					if x == "version" {
						version = int(y.(float64))
					}
				}
				//we only support IPV4
				if addr != "" && version == netVersion {
					address[k] = addr
				}
			}
		}
	}

	if net != "" {
		return address[net]
	}

	for net := range address {
		return address[net]
	}

	return ""
}

func getAnsibleMetadata(s *servers.Server) map[string]string {
	data := map[string]string{}
	for k, v := range s.Metadata {
		if strings.HasPrefix(k, "ansible_") {
			data[k] = v
		}
	}
	return data
}

func render(l *ServerList, groupBy string, net string) error {
	i := &Inventory{
		Groups: map[string]Group{},
	}

	if groupBy == "" {
		g := Group{}
		for _, s := range l.l {
			g.Hosts = append(g.Hosts, Host{
				Address: getIPAdress(&s, net, 4),
				Name:    s.Name,
				Extra:   getAnsibleMetadata(&s),
			})
		}
		i.Groups["all"] = g
	}

	t := template.New("t")
	t, err := t.Parse(iniFile)
	if err != nil {
		return err
	}

	err = t.Execute(os.Stdout, i)
	if err != nil {
		return err
	}

	return nil
}

var iniFile = `
{{- range $key, $value := .Groups }}
[{{$key}}]
{{ range $value.Hosts}}{{ .Name }} ansible_connection=ssh ansible_host={{ .Address }} {{- range $ekey, $evalue := .Extra }} {{$ekey}}={{$evalue}} {{ end -}}{{ end }}
{{ end -}}
`
