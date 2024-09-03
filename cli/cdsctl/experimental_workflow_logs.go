package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var experimentalWorkflowRunJobsCmd = cli.Command{
	Name:    "logs",
	Short:   "CDS Experimental workflow run jobs logs commands",
	Aliases: []string{"log"},
}

func experimentalWorkflowRunLogs() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowRunJobsCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowRunJobLogsDownloadCmd, workflowRunJobLogsDownloadFunc, nil, withAllCommandModifiers()...),
	})
}

var workflowRunJobLogsDownloadCmd = cli.Command{
	Name:    "download",
	Aliases: []string{"dl"},
	Short:   "Get the workflow run job status",
	Example: "cdsctl experimental workflow logs download <proj_key> <workflow_run_id>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "workflow_run_id"},
	},
	Flags: []cli.Flag{
		{
			Name:  "pattern",
			Usage: "Filter on job name",
		},
	},
}

func workflowRunJobLogsDownloadFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	workflowRunID := v.GetString("workflow_run_id")

	var reg *regexp.Regexp
	var err error
	if v.GetString("pattern") != "" {
		reg, err = regexp.Compile(v.GetString("pattern"))
		if err != nil {
			return cli.NewError("invalid pattern %q", v.GetString("pattern"))
		}
	}

	runJobs, err := client.WorkflowV2RunJobs(context.Background(), projKey, workflowRunID)
	if err != nil {
		return err
	}

	for _, rj := range runJobs {
		if reg != nil && !reg.MatchString(rj.JobID) {
			continue
		}
		links, err := client.WorkflowV2RunJobLogLinks(context.Background(), projKey, workflowRunID, rj.ID)
		if err != nil {
			return err
		}

		steps := make([]string, 0)

		// Foreach step create a ref if a step status exists
		for i, s := range rj.Job.Steps {
			stepName := sdk.GetJobStepName(s.ID, i)
			if _, ok := rj.StepsStatus[stepName]; ok {
				steps = append(steps, stepName)
			}
		}

		// Foreach step create a ref if a post step status exists
		for i := len(rj.Job.Steps) - 1; i >= 0; i-- {
			stepName := sdk.GetJobStepName(rj.Job.Steps[i].ID, i)
			if _, ok := rj.StepsStatus["Post-"+stepName]; ok {
				steps = append(steps, "Post-"+stepName)
			}
		}

		for idx, link := range links.Data {
			fileName := getFileName(rj, steps[idx])
			data, err := client.WorkflowLogDownload(context.Background(), sdk.CDNLogLink{APIRef: link.APIRef, ItemType: link.ItemType})
			if err != nil {
				if strings.Contains(err.Error(), "resource not found") {
					continue
				}
				fmt.Printf("unable to download log: %s\n", fileName)
				return err
			}
			if err := os.WriteFile(fileName, data, 0644); err != nil {
				return err
			}
			fmt.Printf("file %s created\n", fileName)
		}
	}
	return nil
}

func getFileName(rj sdk.V2WorkflowRunJob, name string) string {
	return fmt.Sprintf("%s-%d-%d-%s-%s", rj.WorkflowName, rj.RunNumber, rj.RunAttempt, rj.JobID, name)
}
