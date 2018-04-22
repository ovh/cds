package main

import (
	"encoding/json"
	"fmt"
	"time"

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
			cli.NewListCommand(adminHooksTaskExecutionListCmd, adminHooksTaskExecutionListRun, nil),
			cli.NewCommand(adminHooksTaskExecutionStartCmd, adminHooksTaskExecutionStartRun, nil),
			cli.NewCommand(adminHooksTaskExecutionStopCmd, adminHooksTaskExecutionStopRun, nil),
			cli.NewCommand(adminHooksTaskExecutionDeleteAllCmd, adminHooksTaskExecutionDeleteAllRun, nil),
			cli.NewCommand(adminHooksTaskExecutionStartAllCmd, adminHooksTaskExecutionStartAllRun, nil),
			cli.NewCommand(adminHooksTaskExecutionStopAllCmd, adminHooksTaskExecutionStopAllRun, nil),
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
		UUID              string `cli:"UUID,key"`
		Type              string `cli:"Type"`
		Stopped           bool   `cli:"Stopped"`
		Project           string `cli:"Project"`
		Workflow          string `cli:"Workflow"`
		VCSServer         string `cli:"VCSServer"`
		RepoFullname      string `cli:"RepoFullname"`
		Cron              string `cli:"Cron"`
		NbExecutionsTotal int    `cli:"Execs_Total"`
		NbExecutionsTodo  int    `cli:"Execs_Todo"`
	}

	tss := []TaskDisplay{}
	for _, p := range ts {
		tss = append(tss, TaskDisplay{
			UUID:              p.UUID,
			Type:              p.Type,
			Stopped:           p.Stopped,
			Project:           p.Config["project"].Value,
			Workflow:          p.Config["workflow"].Value,
			VCSServer:         p.Config["vcsServer"].Value,
			RepoFullname:      p.Config["repoFullName"].Value,
			Cron:              p.Config["cron"].Value,
			NbExecutionsTotal: p.NbExecutionsTotal,
			NbExecutionsTodo:  p.NbExecutionsTodo,
		})
	}

	return cli.AsListResult(tss), nil
}

var adminHooksTaskExecutionListCmd = cli.Command{
	Name:    "executions",
	Short:   "List CDS Executions for one task",
	Example: "cdsctl admin hooks executions 5178ce1f-2f76-45c5-a203-58c10c3e2c73",
	Args: []cli.Arg{
		{Name: "uuid"},
	},
}

func adminHooksTaskExecutionListRun(v cli.Values) (cli.ListResult, error) {
	uuid := v.GetString("uuid")
	if uuid == "" {
		return nil, fmt.Errorf("please use uuid argument")
	}
	btes, err := client.ServiceCallGET("hooks", fmt.Sprintf("/task/%s/execution", uuid))
	if err != nil {
		return nil, err
	}
	type TaskExecutionDisplay struct {
		sdk.TaskExecution
		ProcessingH string `cli:"Processing H"`
		TimestampH  string `cli:"Timestamp H"`
	}
	ts := sdk.Task{}
	if err := json.Unmarshal(btes, &ts); err != nil {
		return nil, err
	}
	te := []TaskExecutionDisplay{}
	for _, v := range ts.Executions {
		var processingH, timestampH string
		if v.ProcessingTimestamp != 0 {
			processingH = time.Unix(0, v.ProcessingTimestamp).Format(time.RFC3339)
		}
		if v.Timestamp != 0 {
			timestampH = time.Unix(0, v.Timestamp).Format(time.RFC3339)
		}
		te = append(te, TaskExecutionDisplay{
			TaskExecution: v,
			ProcessingH:   processingH,
			TimestampH:    timestampH,
		})
	}

	return cli.AsListResult(te), nil
}

var adminHooksTaskExecutionDeleteAllCmd = cli.Command{
	Name:    "purge",
	Short:   "Delete all executions for a task",
	Example: "cdsctl admin hooks purge 5178ce1f-2f76-45c5-a203-58c10c3e2c73",
	Args: []cli.Arg{
		{Name: "uuid"},
	},
}

func adminHooksTaskExecutionDeleteAllRun(v cli.Values) error {
	uuid := v.GetString("uuid")
	if uuid == "" {
		return fmt.Errorf("please use uuid argument")
	}
	return client.ServiceCallDELETE("hooks", fmt.Sprintf("/task/%s/execution", uuid))
}

var adminHooksTaskExecutionStartCmd = cli.Command{
	Name:    "start",
	Short:   "Start a task",
	Example: "cdsctl admin hooks start 5178ce1f-2f76-45c5-a203-58c10c3e2c73",
	Args: []cli.Arg{
		{Name: "uuid"},
	},
}

func adminHooksTaskExecutionStartRun(v cli.Values) error {
	uuid := v.GetString("uuid")
	if uuid == "" {
		return fmt.Errorf("please use uuid argument")
	}
	_, err := client.ServiceCallGET("hooks", fmt.Sprintf("/task/%s/start", uuid))
	return err
}

var adminHooksTaskExecutionStopCmd = cli.Command{
	Name:    "stop",
	Short:   "Stop a task",
	Example: "cdsctl admin hooks stop 5178ce1f-2f76-45c5-a203-58c10c3e2c73",
	Args: []cli.Arg{
		{Name: "uuid"},
	},
}

func adminHooksTaskExecutionStopRun(v cli.Values) error {
	uuid := v.GetString("uuid")
	if uuid == "" {
		return fmt.Errorf("please use uuid argument")
	}
	_, err := client.ServiceCallGET("hooks", fmt.Sprintf("/task/%s/stop", uuid))
	return err
}

var adminHooksTaskExecutionStopAllCmd = cli.Command{
	Name:    "stopall",
	Short:   "Stop all tasks",
	Example: "cdsctl admin hooks stopall",
}

func adminHooksTaskExecutionStopAllRun(v cli.Values) error {
	_, err := client.ServiceCallGET("hooks", "/task/bulk/stop")
	return err
}

var adminHooksTaskExecutionStartAllCmd = cli.Command{
	Name:    "startall",
	Short:   "Start all tasks",
	Example: "cdsctl admin hooks startall",
}

func adminHooksTaskExecutionStartAllRun(v cli.Values) error {
	_, err := client.ServiceCallGET("hooks", "/task/bulk/start")
	return err
}
