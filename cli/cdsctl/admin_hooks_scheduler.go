package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

var adminHooksSchedulerCmd = cli.Command{
	Name:    "scheduler",
	Aliases: []string{"s", "schedule", "scheudlers"},
	Short:   "Manage repositories where there were events",
}

func adminHooksSchedulers() *cobra.Command {
	return cli.NewCommand(adminHooksSchedulerCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminHooksSchedulersListAllCmd, adminHooksSchedulersListAllRun, nil),
		cli.NewGetCommand(adminHooksGetSchedulerCmd, adminHooksGetSchedulerRun, nil),
		cli.NewDeleteCommand(adminHookSchedulerDeleteCmd, adminHookSchedulerDeleteRun, nil),
		adminHooksRepositoryEvents(),
	})
}

var adminHookSchedulerDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm", "remove", "del"},
	Short:   "Delete a specific scheduler by his identifier",
	Flags: []cli.Flag{
		{Name: "hookID"},
		{Name: "vcs"},
		{Name: "repository"},
		{Name: "workflow"},
	},
}

func adminHookSchedulerDeleteRun(v cli.Values) error {
	hookID := v.GetString("hookID")

	vcs := v.GetString("vcs")
	repository := v.GetString("repository")
	workflow := v.GetString("workflow")

	if hookID == "" && (vcs == "" || repository == "" || workflow == "") {
		return fmt.Errorf("you must must provide the flag `hookID` or the three flags `vcs` `repository` `workflow`")
	}
	if hookID != "" && (vcs != "" || repository != "" || workflow != "") {
		return fmt.Errorf("you must must provide the flag `hookID` or the three flags `vcs` `repository` `workflow`")
	}

	if hookID != "" {
		path := fmt.Sprintf("/admin/scheduler/execution/" + hookID)
		if err := client.ServiceCallDELETE("hooks", path); err != nil {
			return err
		}
	} else {
		path := fmt.Sprintf("/v2/workflow/scheduler/%s/%s/%s", vcs, repository, workflow)
		if err := client.ServiceCallDELETE("hooks", path); err != nil {
			return err
		}
	}

	return nil
}

var adminHooksGetSchedulerCmd = cli.Command{
	Name:    "get",
	Aliases: []string{"show"},
	Short:   "Get a scheduler by his identifier",
	Flags: []cli.Flag{
		{Name: "hookID"},
		{Name: "vcs"},
		{Name: "repository"},
		{Name: "workflow"},
	},
}

func adminHooksGetSchedulerRun(v cli.Values) (interface{}, error) {
	hookID := v.GetString("hookID")

	path := fmt.Sprintf("/admin/scheduler/execution/" + hookID)
	bts, err := client.ServiceCallGET("hooks", path)
	if err != nil {
		return nil, err
	}

	var exec sdk.SchedulerExecution
	if err := json.Unmarshal(bts, &exec); err != nil {
		return nil, err
	}

	resp := struct {
		HookID        string    `cli:"id"`
		Vcs           string    `cli:"vcs"`
		Repository    string    `cli:"repository"`
		Workflow      string    `cli:"workflow"`
		Cron          string    `cli:"cron"`
		Timezone      string    `cli:"timezone"`
		Ref           string    `cli:"ref"`
		Commit        string    `cli:"commit"`
		NextExecution time.Time `cli:"new_execution"`
	}{
		Vcs:           exec.SchedulerDef.VCSName,
		Repository:    exec.SchedulerDef.RepositoryName,
		Workflow:      exec.SchedulerDef.WorkflowName,
		Cron:          exec.SchedulerDef.Data.Cron,
		Timezone:      exec.SchedulerDef.Data.CronTimeZone,
		Ref:           exec.SchedulerDef.Ref,
		Commit:        exec.SchedulerDef.Commit,
		NextExecution: time.Unix(0, exec.NextExecutionTime),
	}

	return resp, nil
}

var adminHooksSchedulersListAllCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List schedulers. You can use flag \"--vcs myvcs --repository my/repo --workflow myworkflow\" to list all schedulers on a workflow",
	Flags: []cli.Flag{
		{Name: "vcs"},
		{Name: "repository"},
		{Name: "workflow"},
	},
}

func adminHooksSchedulersListAllRun(v cli.Values) (cli.ListResult, error) {
	vcs := v.GetString("vcs")
	repository := v.GetString("repository")
	workflow := v.GetString("workflow")

	if vcs != "" && repository != "" && workflow != "" {
		var scheds []sdk.V2WorkflowHook
		path := fmt.Sprintf("/admin/scheduler/%s/%s/%s", url.PathEscape(vcs), url.PathEscape(repository), workflow)
		bts, err := client.ServiceCallGET("hooks", path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(bts, &scheds); err != nil {
			return nil, err
		}

		type Scheduler struct {
			HookID     string `cli:"id"`
			Vcs        string `cli:"vcs"`
			Repository string `cli:"repository"`
			Workflow   string `cli:"workflow"`
			Cron       string `cli:"cron"`
			Timezone   string `cli:"timezone"`
			Ref        string `cli:"ref"`
			Commit     string `cli:"commit"`
		}
		hookSchedulers := make([]Scheduler, 0)
		for _, s := range scheds {
			hookSchedulers = append(hookSchedulers, Scheduler{
				Vcs:        s.VCSName,
				Repository: s.RepositoryName,
				Workflow:   s.WorkflowName,
				Cron:       s.Data.Cron,
				Timezone:   s.Data.CronTimeZone,
				Ref:        s.Ref,
				Commit:     s.Commit,
			})
		}
		return cli.AsListResult(hookSchedulers), nil
	} else {
		var scheds []sdk.V2WorkflowHookShort
		bts, err := client.ServiceCallGET("hooks", "/admin/scheduler")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(bts, &scheds); err != nil {
			return nil, err
		}

		type Scheduler struct {
			HookID     string `cli:"id"`
			Vcs        string `cli:"vcs"`
			Repository string `cli:"repository"`
			Workflow   string `cli:"workflow"`
		}
		hookSchedulers := make([]Scheduler, 0)
		for _, s := range scheds {
			hookSchedulers = append(hookSchedulers, Scheduler{
				Vcs:        s.VCSName,
				Repository: s.RepositoryName,
				Workflow:   s.WorkflowName,
			})
		}
		return cli.AsListResult(hookSchedulers), nil
	}
}
