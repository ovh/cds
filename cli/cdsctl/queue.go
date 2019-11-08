package main

import (
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var queueCmd = cli.Command{
	Name:  "queue",
	Short: "CDS Queue",
}

func queue() *cobra.Command {
	return cli.NewListCommand(queueCmd, queueRun, []*cobra.Command{
		cli.NewCommand(queueUICmd, queueUIRun, nil, withAllCommandModifiers()...),
	})
}

var queueUICmd = cli.Command{
	Name:  "interactive",
	Short: "Show the current queue",
}

func queueRun(v cli.Values) (cli.ListResult, error) {
	jobs, err := client.QueueWorkflowNodeJobRun(sdk.StatusWaiting)
	if err != nil {
		return nil, err
	}

	config, err := client.ConfigUser()
	if err != nil {
		return nil, err
	}
	baseURL := config.URLUI

	type job struct {
		Run          string `cli:"run,key"`
		ProjectKey   string `cli:"project_key"`
		WorkflowName string `cli:"workflow_name"`
		NodeName     string `cli:"pipeline_name"`
		Status       string `cli:"status"`
		URL          string `cli:"url"`
		Since        string `cli:"since"`
		BookedBy     string `cli:"booked_by"`
		TriggeredBy  string `cli:"triggered_by"`
	}
	jobList := make([]job, len(jobs))

	for k, jr := range jobs {
		jobList[k] = job{
			Run:          getVarsInPbj("cds.run", jr.Parameters),
			ProjectKey:   getVarsInPbj("cds.project", jr.Parameters),
			WorkflowName: getVarsInPbj("cds.workflow", jr.Parameters),
			NodeName:     getVarsInPbj("cds.node", jr.Parameters),
			Status:       jr.Status,
			URL:          generateQueueJobURL(baseURL, jr.Parameters),
			Since:        fmt.Sprintf(sdk.Round(time.Since(jr.Queued), time.Second).String()),
			BookedBy:     jr.BookedBy.Name,
			TriggeredBy:  getVarsInPbj("cds.triggered_by.username", jr.Parameters),
		}
	}

	return cli.AsListResult(jobList), nil
}

func getVarsInPbj(key string, ps []sdk.Parameter) string {
	for _, p := range ps {
		if p.Name == key {
			return p.Value
		}
	}
	return ""
}

func generateQueueJobURL(baseURL string, parameters []sdk.Parameter) string {
	prj := getVarsInPbj("cds.project", parameters)
	workflow := getVarsInPbj("cds.workflow", parameters)
	runNumber := getVarsInPbj("cds.run.number", parameters)
	return fmt.Sprintf("%s/project/%s/workflow/%s/run/%s", baseURL, prj, workflow, runNumber)
}

func queueUIRun(v cli.Values) error {

	return nil
}
