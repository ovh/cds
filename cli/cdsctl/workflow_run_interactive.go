package main

import (
	"context"
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

	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	feature, err := client.FeatureEnabled("cdn-job-logs", map[string]string{
		"project_key": projectKey,
	})
	if err != nil {
		return err
	}

	for {
		var errrg error
		wo, errrg = client.WorkflowRunGet(projectKey, v.GetString(_WorkflowName), w.Number)
		if errrg != nil {
			return errrg
		}

		var newOutput string

		failedOn = ""
		for _, wnrs := range wo.WorkflowNodeRuns {
			for _, wnr := range wnrs {
				for _, stage := range wnr.Stages {
					for _, job := range stage.RunJobs {
						status, _ := statusShort(job.Status)
						var start, end string
						if job.Status != sdk.StatusWaiting {
							start = fmt.Sprintf("start:%s", job.Start)
						}
						if job.Done.After(job.Start) {
							end = fmt.Sprintf(" end:%s", job.Done)
						}

						jobLine := fmt.Sprintf("%s  %s/%s/%s/%s %s %s \n", status, v.GetString(_WorkflowName), wnr.WorkflowNodeName, stage.Name, job.Job.Action.Name, start, end)
						if job.Status == sdk.StatusFail {
							newOutput += fmt.Sprintf(tm.Color(tm.Bold(jobLine), tm.RED))
						} else {
							newOutput += fmt.Sprintf(tm.Bold(jobLine))
						}

						for _, info := range job.SpawnInfos {
							newOutput += fmt.Sprintf("\nInformations: %s - %s", info.APITime, info.UserMessage)
						}
						newOutput += fmt.Sprintf("\n")

						for _, step := range job.Job.StepStatus {
							var link *sdk.CDNLogLink
							if feature.Enabled {
								link, err = client.WorkflowNodeRunJobStepLink(context.Background(), projectKey, workflowName, wnr.ID, job.ID, int64(step.StepOrder))
								if err != nil {
									return err
								}
							}

							var data string
							if link != nil {
								buf, err := client.WorkflowLogDownload(context.Background(), *link)
								if err != nil {
									return err
								}
								data = string(buf)
							} else {
								buildState, err := client.WorkflowNodeRunJobStepLog(context.Background(), projectKey, workflowName, wnr.ID, job.ID, int64(step.StepOrder))
								if err != nil {
									return err
								}
								data = buildState.StepLogs.Val
							}

							vSplitted := strings.Split(data, "\n")
							failedOnStepKnowned := false
							for _, line := range vSplitted {
								line = strings.Trim(line, " ")
								titleStep := fmt.Sprintf("%s / step %d", job.Job.Action.Name, step.StepOrder)
								// RED color on step failed
								if step.Status == sdk.StatusFail {
									if !failedOnStepKnowned {
										// hide "Starting" text on resume
										failedOn = fmt.Sprintf("%s%s / %s / %s / %s %s \n", failedOn, v.GetString(_WorkflowName), wnr.WorkflowNodeName, stage.Name, titleStep, strings.Replace(line, "Starting", "", 1))
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

		if wo.Status == sdk.StatusFail || wo.Status == sdk.StatusSuccess {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if wo != nil {
		iconStatus, _ := statusShort(wo.Status)
		tm.Printf("Workflow: %s - RUN %d.%d - %s %s \n", v.GetString(_WorkflowName), wo.Number, wo.LastSubNumber, wo.Status, iconStatus)
		tm.Printf("Start: %s - End %s\n", wo.Start, wo.LastModified)
		tm.Printf("Duration: %s\n", sdk.Round(wo.LastModified.Sub(wo.Start), time.Second).String())
		if wo.Status == sdk.StatusFail {
			tm.Println(tm.Color(fmt.Sprintf("Failed on: %s", failedOn), tm.RED))
		}

		u := fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", baseURL, v.GetString(_ProjectKey), v.GetString(_WorkflowName), wo.Number)
		tm.Printf("View on web UI: %s\n", u)
	}
	tm.Flush()
	return nil
}

func statusShort(status string) (string, string) {
	switch status {
	case sdk.StatusWaiting:
		return "w", "fg-cyan"
	case sdk.StatusBuilding:
		return "b", "fg-blue"
	case sdk.StatusDisabled:
		return "d", "fg-white"
	case sdk.StatusChecking:
		return "c", "fg-yellow"
	}
	return status, "fg-default"
}
