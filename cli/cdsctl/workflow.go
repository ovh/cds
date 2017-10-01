package main

import (
	"fmt"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	workflowCmd = cli.Command{
		Name:  "workflow",
		Short: "Manage CDS workflow",
	}

	workflow = cli.NewCommand(workflowCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workflowListCmd, workflowListRun, nil),
			cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil),
			cli.NewCommand(workflowRunManualCmd, workflowRunManualRun, nil),
			workflowArtifact,
		})
)

var workflowListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS workflows",
	Args: []cli.Arg{
		{Name: "project-key"},
	},
}

func workflowListRun(v cli.Values) (cli.ListResult, error) {
	w, err := client.WorkflowList(v["project-key"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(w), nil
}

var workflowShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
}

func workflowShowRun(v cli.Values) (interface{}, error) {
	w, err := client.WorkflowGet(v["project-key"], v["workflow-name"])
	if err != nil {
		return nil, err
	}
	return *w, nil
}

var workflowRunManualCmd = cli.Command{
	Name:  "run",
	Short: "Run a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "payload"},
	},
}

func workflowRunManualRun(v cli.Values) error {
	manual := sdk.WorkflowNodeRunManual{}
	if v["payload"] != "" {
		manual.Payload = v["payload"]
	}
	w, err := client.WorkflowRunFromManual(v["project-key"], v["workflow-name"], manual)
	if err != nil {
		return err
	}

	var wo *sdk.WorkflowRun
	var failedOn string

	for {
		tm.Clear() // Clear current screen
		tm.MoveCursor(1, 1)
		var errrg error
		wo, errrg = client.WorkflowRunGet(v["project-key"], v["workflow-name"], w.Number)
		if errrg != nil {
			return errrg
		}

		failedOn = ""
		for _, wnrs := range wo.WorkflowNodeRuns {
			for _, wnr := range wnrs {
				wn := w.Workflow.GetNode(wnr.WorkflowNodeID)
				for _, stage := range wnr.Stages {
					for _, job := range stage.RunJobs {
						status, _ := statusShort(job.Status)
						var start, end string
						if job.Status != sdk.StatusWaiting.String() {
							start = fmt.Sprintf("start:%s", job.Start)
						}
						if job.Done.After(job.Start) {
							end = fmt.Sprintf(" end:%s", job.Done)
						}

						jobLine := fmt.Sprintf("%s  %s/%s/%s/%s %s %s \n", status, v["workflow-name"], wn.Name, stage.Name, job.Job.Action.Name, start, end)
						if job.Status == sdk.StatusFail.String() {
							tm.Printf(tm.Color(tm.Bold(jobLine), tm.RED))
						} else {
							tm.Printf(tm.Bold(jobLine))
						}

						for _, info := range job.SpawnInfos {
							tm.Printf("\nInformations: %s - %s", info.APITime, info.UserMessage)
						}
						tm.Printf("\n")
						tm.Flush()

						for _, step := range job.Job.StepStatus {
							buildState, errb := client.WorkflowNodeRunJobStep(v["project-key"], v["workflow-name"], wo.Number, wnr.ID, job.ID, step.StepOrder)
							if errb != nil {
								return errb
							}

							vSplitted := strings.Split(buildState.StepLogs.Val, "\n")
							failedOnStepKnowned := false
							for _, line := range vSplitted {
								line = strings.Trim(line, " ")
								titleStep := fmt.Sprintf("%s / step %d", job.Job.Action.Name, step.StepOrder)
								// RED color on step failed
								if step.Status == sdk.StatusFail.String() {
									if !failedOnStepKnowned {
										// hide "Starting" text on resume
										failedOn = fmt.Sprintf("%s%s / %s / %s / %s %s \n", failedOn, v["workflow-name"], wn.Name, stage.Name, titleStep, strings.Replace(line, "Starting", "", 1))
									}
									failedOnStepKnowned = true
									titleStep = fmt.Sprintf(tm.Color(titleStep, tm.RED))
								}

								if line != "" {
									tm.Printf("%s\t\t %s\n", titleStep, line)
								}
							}
						}
						if job.Done.After(job.Start) {
							tm.Printf("\n")
						}

						tm.Flush()
					}
				}
			}
		}

		if wo.Status == sdk.StatusFail.String() || wo.Status == sdk.StatusSuccess.String() {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if wo != nil {
		iconStatus, _ := statusShort(wo.Status)
		fmt.Printf("Workflow: %s - RUN %d - %s %s \n", v["workflow-name"], wo.Number, wo.Status, iconStatus)
		fmt.Printf("Start: %s - End %s\n", wo.Start, wo.LastModified)
		fmt.Printf("Duration: %s\n", sdk.Round(wo.LastModified.Sub(wo.Start), time.Second).String())
		if wo.Status == sdk.StatusFail.String() {
			fmt.Printf("Failed on: %s", failedOn)
		}

		var baseURL string
		configUser, err := client.ConfigUser()
		if err != nil {
			return err
		}

		if b, ok := configUser[sdk.ConfigURLUIKey]; ok {
			baseURL = b
		}

		u := fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", baseURL, v["project-key"], v["workflow-name"], wo.Number)
		fmt.Printf("View on web UI: %s\n", u)
	}

	return nil
}
