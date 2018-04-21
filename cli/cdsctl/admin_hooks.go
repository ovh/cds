package main

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminHooksCmd = cli.Command{
		Name:  "hooks",
		Short: "Manage CDS Hooks tasks",
	}

	adminHooks = cli.NewCommand(adminHooksCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminHooksTaskListCmd, adminHooksTaskListRun, nil),
		})
)

var adminHooksTaskListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS Hooks Tasks",
}

func adminHooksTaskListRun(v cli.Values) (cli.ListResult, error) {
	btes, err := client.ServiceCallGET("hooks", "/task")
	if err != nil {
		return nil, err
	}
	ts := []sdk.Task{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}

	type TaskDisplay struct {
		UUID         string `cli:"UUID,key"`
		Type         string `cli:"Type"`
		Stopped      bool   `cli:"Stopped"`
		Project      string `cli:"Project"`
		Workflow     string `cli:"Workflow"`
		VCSServer    string `cli:"VCSServer"`
		RepoFullname string `cli:"RepoFullname"`
		Cron         string `cli:"Cron"`
	}

	tss := []TaskDisplay{}
	for _, p := range ts {
		tss = append(tss, TaskDisplay{
			UUID:         p.UUID,
			Type:         p.Type,
			Stopped:      p.Stopped,
			Project:      p.Config["project"].Value,
			Workflow:     p.Config["workflow"].Value,
			VCSServer:    p.Config["vcsServer"].Value,
			RepoFullname: p.Config["repoFullName"].Value,
			Cron:         p.Config["cron"].Value,
		})
	}

	return cli.AsListResult(tss), nil
}
