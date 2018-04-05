package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	projectCmd = cli.Command{
		Name:  "project",
		Short: "Manage CDS project",
	}
)

func project() *cobra.Command {
	cmds := []*cobra.Command{
		cli.NewListCommand(projectListCmd, projectListRun, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectShowCmd, projectShowRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectCreateCmd, projectCreateRun, nil),
		cli.NewDeleteCommand(projectDeleteCmd, projectDeleteRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectFavoriteCmd, projectFavoriteRun, nil, withAllCommandModifiers()...),
		projectKey,
		projectGroup,
		projectVariable,
	}
	if cli.ShellMode {
		cmds = append(cmds, application, workflow, environment)
	}
	return cli.NewCommand(projectCmd, nil, cmds)
}

var projectListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS projects",
}

func projectListRun(v cli.Values) (cli.ListResult, error) {
	projs, err := client.ProjectList(false, false)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(projs), nil
}

var projectShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
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
	proj, err := client.ProjectGet(v[_ProjectKey], mods...)
	if err != nil {
		return nil, err
	}
	return *proj, nil
}

var projectCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: "project-name"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "group-name"},
	},
	Aliases: []string{"add"},
}

func projectCreateRun(v cli.Values) error {
	proj := &sdk.Project{Name: v["project-name"], Key: v[_ProjectKey]}
	return client.ProjectCreate(proj, v["group-name"])
}

var projectDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectDeleteRun(v cli.Values) error {
	projKey := v[_ProjectKey]
	if v.GetBool("force") {
		// Delete all workflow
		ws, errW := client.WorkflowList(projKey)
		if errW != nil && !sdk.ErrorIs(errW, sdk.ErrNoProject) {
			return errW
		}
		for _, w := range ws {
			if err := client.WorkflowDelete(projKey, w.Name); err != nil && !sdk.ErrorIs(err, sdk.ErrNoProject) {
				return err
			}
		}

		// Delete all apps
		apps, errA := client.ApplicationList(projKey)
		if errA != nil && !sdk.ErrorIs(errA, sdk.ErrNoProject) {
			return errA
		}
		for _, app := range apps {
			if err := client.ApplicationDelete(projKey, app.Name); err != nil && !sdk.ErrorIs(err, sdk.ErrNoProject) {
				return err
			}
		}

		// Delete all pipelines
		pips, errP := client.PipelineList(projKey)
		if errP != nil && !sdk.ErrorIs(errP, sdk.ErrNoProject) {
			return errP
		}
		for _, pip := range pips {
			if err := client.PipelineDelete(projKey, pip.Name); err != nil && !sdk.ErrorIs(err, sdk.ErrNoProject) {
				return err
			}
		}

		// Delete all environments
		envs, errE := client.EnvironmentList(projKey)
		if errE != nil && !sdk.ErrorIs(errE, sdk.ErrNoProject) {
			return errE
		}
		for _, env := range envs {
			if err := client.EnvironmentDelete(projKey, env.Name); err != nil && !sdk.ErrorIs(err, sdk.ErrNoProject) {
				return err
			}
		}
	}

	if err := client.ProjectDelete(projKey); err != nil {
		if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNoProject) {
			fmt.Println(err.Error())
			os.Exit(0)
		}
		return err
	}
	return nil
}
