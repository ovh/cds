package application

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var (
	cmdApplicationPipelineScheduler = &cobra.Command{
		Use:   "scheduler",
		Short: "cds application pipeline scheduler",
	}

	cmdApplicationPipelineSchedulerList = &cobra.Command{
		Use:   "list",
		Short: "cds application pipeline scheduler list <projectKey> <applicationName> <pipelineName>",
		Run:   applicationPipelineSchedulerList,
	}

	cmdApplicationPipelineSchedulerAdd = &cobra.Command{
		Use:   "add",
		Short: "cds application pipeline scheduler add <projectKey> <applicationName> <pipelineName> <cron expression> [-e environment] [-p <param>=<value>]",
		Run:   applicationPipelineSchedulerAdd,
	}

	cmdApplicationPipelineSchedulerAddEnv string

	cmdApplicationPipelineSchedulerUpdate = &cobra.Command{
		Use:   "update",
		Short: "cds application pipeline scheduler update <projectKey> <applicationName> <pipelineName> <ID> [-c <cron expression>] [-e environment] [-p <pipelineParam>=<value>] [--disable true|false]",
		Run:   applicationPipelineSchedulerUpdate,
	}

	cmdApplicationPipelineSchedulerUpdateCronExpr, cmdApplicationPipelineSchedulerUpdateDisable string

	cmdApplicationPipelineSchedulerDelete = &cobra.Command{
		Use:   "delete",
		Short: "cds application pipeline scheduler delete <projectKey> <applicationName> <pipelineName> <ID>",
		Run:   applicationPipelineSchedulerDelete,
	}
)

func applicationPipelineSchedulerList(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]

	ps, err := sdk.GetPipelineScheduler(projectKey, appName, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot list pipeline schedulers: (%s)\n", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Frequency", "Parameters", "Environment", "Enabled", "Last Execution", "Next Execution"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for i := range ps {
		args, err := yaml.Marshal(ps[i].Args)
		if err != nil {
			sdk.Exit("Error: cannot format output (%s)\n", err)
		}

		var last = "never"
		var next = "unknown"

		if ps[i].LastExecution != nil {
			loc, err := time.LoadLocation(ps[i].Timezone)
			if err != nil {
				last = fmt.Sprintf("%v", ps[i].LastExecution.ExecutionDate)
			} else {
				t := ps[i].LastExecution.ExecutionDate.In(loc)
				last = fmt.Sprintf("%v", t)
			}
		}

		if ps[i].NextExecution != nil {
			loc, err := time.LoadLocation(ps[i].Timezone)
			if err != nil {
				next = fmt.Sprintf("%v", ps[i].NextExecution.ExecutionPlannedDate)
			} else {
				t := ps[i].NextExecution.ExecutionPlannedDate.In(loc)
				next = fmt.Sprintf("%v", t)
			}
		}

		table.Append([]string{
			fmt.Sprintf("%d", ps[i].ID),
			ps[i].Crontab, string(args),
			ps[i].EnvironmentName,
			fmt.Sprintf("%v", !ps[i].Disabled),
			last,
			next,
		})
	}
	table.Render()
}

func applicationPipelineSchedulerAdd(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]
	cronExpr := args[3]

	var params []sdk.Parameter
	// Parameters
	for i := range cmdApplicationAddPipelineParams {
		p, err := sdk.NewStringParameter(cmdApplicationAddPipelineParams[i])
		if err != nil {
			sdk.Exit("Error: cannot parse parameter '%s' (%s)\n", cmdApplicationAddPipelineParams[i])
		}
		params = append(params, p)
	}

	if _, err := sdk.AddPipelineScheduler(projectKey, appName, pipelineName, cronExpr, cmdApplicationPipelineSchedulerAddEnv, params); err != nil {
		sdk.Exit("Error: cannot add pipeline scheduler : %s\n", err)
	}

	fmt.Println("OK")

}

func applicationPipelineSchedulerUpdate(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}

	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]
	ids := args[3]

	id, err := strconv.ParseInt(ids, 10, 64)
	if err != nil {
		sdk.Exit("Error: invalid parameter ID : %s", err)
	}

	var params []sdk.Parameter
	// Parameters
	for i := range cmdApplicationAddPipelineParams {
		p, err := sdk.NewStringParameter(cmdApplicationAddPipelineParams[i])
		if err != nil {
			sdk.Exit("Error: cannot parse parameter '%s' (%s)\n", cmdApplicationAddPipelineParams[i])
		}
		params = append(params, p)
	}

	ps, err := sdk.GetPipelineScheduler(projectKey, appName, pipelineName)
	if err != nil {
		sdk.Exit("Error: Unable to list pipeline schedulers: %s", err)
	}

	var s *sdk.PipelineScheduler
	for i := range ps {
		if ps[i].ID == id {
			s = &ps[i]
			break
		}
	}

	if s == nil {
		sdk.Exit("Error: Unable to find pipeline scheduler with id %d", id)
	}

	if cmdApplicationPipelineSchedulerUpdateCronExpr != "" {
		s.Crontab = cmdApplicationPipelineSchedulerUpdateCronExpr
	}

	if cmdApplicationPipelineSchedulerUpdateDisable == "true" {
		s.Disabled = true
	} else if cmdApplicationPipelineSchedulerUpdateDisable == "false" {
		s.Disabled = false
	}

	if len(params) > 0 {
		s.Args = params
	}

	if cmdApplicationPipelineSchedulerAddEnv != "" {
		s.EnvironmentName = cmdApplicationPipelineSchedulerAddEnv
	}

	if _, err := sdk.UpdatePipelineScheduler(projectKey, appName, pipelineName, s); err != nil {
		sdk.Exit("Error: Unable to update pipeline scheduler with id %d", id)
	}

	fmt.Println("OK")
}

func applicationPipelineSchedulerDelete(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}

	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]
	ids := args[3]

	id, err := strconv.ParseInt(ids, 10, 64)
	if err != nil {
		sdk.Exit("Error: invalid parameter ID : %s", err)
	}

	ps, err := sdk.GetPipelineScheduler(projectKey, appName, pipelineName)
	if err != nil {
		sdk.Exit("Error: Unable to list pipeline schedulers: %s", err)
	}

	var s *sdk.PipelineScheduler
	for i := range ps {
		if ps[i].ID == id {
			s = &ps[i]
			break
		}
	}

	if s == nil {
		sdk.Exit("Error: Unable to find pipeline scheduler with id %d", id)
	}

	if err := sdk.DeletePipelineScheduler(projectKey, appName, pipelineName, s); err != nil {
		sdk.Exit("Error: Unable to delete pipeline scheduler with id %d", id)
	}

	fmt.Println("OK")
}
