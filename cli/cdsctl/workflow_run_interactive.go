package main

import (
	"fmt"
	"strings"
	"time"

	tm "github.com/buger/goterm"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

func workflowRunInteractive(v cli.Values, w *sdk.WorkflowRun, baseURL string) error {
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

		u := fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", baseURL, v["project-key"], v["workflow-name"], wo.Number)
		tm.Printf("View on web UI: %s\n", u)
	}
	tm.Flush()
	return nil
}
