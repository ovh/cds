package main

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
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
			cli.NewDeleteCommand(workflowDeleteCmd, workflowDeleteRun, nil),
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

var workflowDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
}

func workflowDeleteRun(v cli.Values) error {
	err := client.WorkflowDelete(v["project-key"], v["workflow-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	return err
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
	Flags: []cli.Flag{
		{
			Name:  "run-number",
			Usage: "Existing Workflow RUN Number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Kind: reflect.String,
		},
		{
			Name:  "node-name",
			Usage: "Node Name to relaunch; Flag run-number is mandatory",
			Kind:  reflect.String,
		},
	},
}

func workflowRunManualRun(v cli.Values) error {
	manual := sdk.WorkflowNodeRunManual{}
	if v["payload"] != "" {
		manual.Payload = v["payload"]
	}

	var runNumber, fromNodeID int64

	if v.GetString("run-number") != "" {
		var errp error
		runNumber, errp = strconv.ParseInt(v.GetString("run-number"), 10, 64)
		if errp != nil {
			return fmt.Errorf("run-number invalid: not a integer")
		}
	}

	if v.GetString("node-name") != "" {
		if runNumber <= 0 {
			return fmt.Errorf("You can use flag node-name without flag run-number")
		}
		wr, err := client.WorkflowRunGet(v["project-key"], v["workflow-name"], runNumber)
		if err != nil {
			return err
		}
		for _, wnrs := range wr.WorkflowNodeRuns {
			for _, wnr := range wnrs {
				wn := wr.Workflow.GetNode(wnr.WorkflowNodeID)
				if wn.Name == v.GetString("node-name") {
					fromNodeID = wnr.WorkflowNodeID
					break
				}
			}
		}
	}

	w, err := client.WorkflowRunFromManual(v["project-key"], v["workflow-name"], manual, runNumber, fromNodeID)
	if err != nil {
		return err
	}

	var wo *sdk.WorkflowRun
	var failedOn, output string

	for {
		var errrg error
		wo, errrg = client.WorkflowRunGet(v["project-key"], v["workflow-name"], w.Number)
		if errrg != nil {
			return errrg
		}

		var newOutput string

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
							newOutput += fmt.Sprintf(tm.Color(tm.Bold(jobLine), tm.RED))
						} else {
							newOutput += fmt.Sprintf(tm.Bold(jobLine))
						}

						for _, info := range job.SpawnInfos {
							newOutput += fmt.Sprintf("\nInformations: %s - %s", info.APITime, info.UserMessage)
						}
						newOutput += fmt.Sprintf("\n")

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
									newOutput += fmt.Sprintf("%s\t\t %s\n", titleStep, line)
								}
							}
						}
						if job.Done.After(job.Start) {
							newOutput += fmt.Sprintf("\n")
						}

						if newOutput != output {
							tm.Clear() // Clear current screen
							tm.MoveCursor(1, 1)
							output = newOutput
							tm.Printf(output)
							tm.Flush()
						}
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
		tm.Printf("Workflow: %s - RUN %d.%d - %s %s \n", v["workflow-name"], wo.Number, wo.LastSubNumber, wo.Status, iconStatus)
		tm.Printf("Start: %s - End %s\n", wo.Start, wo.LastModified)
		tm.Printf("Duration: %s\n", sdk.Round(wo.LastModified.Sub(wo.Start), time.Second).String())
		if wo.Status == sdk.StatusFail.String() {
			tm.Println(tm.Color(fmt.Sprintf("Failed on: %s", failedOn), tm.RED))
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
		tm.Printf("View on web UI: %s\n", u)
	}
	tm.Flush()
	return nil
}
