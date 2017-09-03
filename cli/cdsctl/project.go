package main

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	projectCmd = cli.Command{
		Name:  "project",
		Short: "Manage CDS project",
	}

	project = cli.NewCommand(projectCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(projectListCmd, projectListRun, nil),
			cli.NewGetCommand(projectShowCmd, projectShowRun, nil),
			projectKey,
		})
)

var projectListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS projects",
}

func projectListRun(v cli.Values) (cli.ListResult, error) {
	projs, err := client.ProjectList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(projs), nil
}

var projectShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS project",
	Args: []cli.Arg{
		{Name: "key"},
	},
}

func projectShowRun(v cli.Values) (interface{}, error) {
	mods := []cdsclient.RequestModifier{}
	if v["verbose"] == "true" {
		mods = append(mods, func(r *http.Request) {
			q := r.URL.Query()
			q.Set("withApplications", "true")
			q.Set("withPipelines", "true")
			q.Set("withEnvironments", "true")
			r.URL.RawQuery = q.Encode()
		})
	}
	proj, err := client.ProjectGet(v["key"], mods...)
	if err != nil {
		return nil, err
	}
	return *proj, nil
}
