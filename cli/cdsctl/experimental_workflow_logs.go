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

func experimentalWorkflowRunJobs() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowRunJobsCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowRunJobLogsDownloadCmd, workflowRunJobLogsDownloadFunc, nil, withAllCommandModifiers()...),
	})
}

var workflowRunJobLogsDownloadCmd = cli.Command{
	Name:    "download",
	Aliases: []string{"dl"},
	Short:   "Get the workflow run job status",
	Example: "cdsctl experimental workflow logs download <proj_key> <vcs_identifier> <repo_identifier> <workflow_name> <run_number>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "vcs_identifier"},
		{Name: "repo_identifier"},
		{Name: "workflow_name"},
		{Name: "run-number"},
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
	vcsId := v.GetString("vcs_identifier")
	repoId := v.GetString("repo_identifier")
	wkfName := v.GetString("workflow_name")
	runNumber, err := v.GetInt64("run-number")
	if err != nil {
		return err
	}

	var reg *regexp.Regexp
	if v.GetString("pattern") != "" {
		reg, err = regexp.Compile(v.GetString("pattern"))
		if err != nil {
			return cli.NewError("invalid pattern %q", v.GetString("pattern"))
		}
	}

	runJobs, err := client.WorkflowV2RunJobs(context.Background(), projKey, vcsId, repoId, wkfName, runNumber)
	if err != nil {
		return err
	}

	for _, rj := range runJobs {
		if reg != nil && !reg.MatchString(rj.JobID) {
			continue
		}
		links, err := client.WorkflowV2RunJobLogLinks(context.Background(), projKey, vcsId, repoId, wkfName, runNumber, rj.JobID)
		if err != nil {
			return err
		}

		for _, link := range links.Data {
			fileName := getFileName(rj, rj.Job.Steps[link.StepOrder].ID, link.StepOrder)
			data, err := client.WorkflowLogDownload(context.Background(), sdk.CDNLogLink{APIRef: link.APIRef, ItemType: links.ItemType})
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

func getFileName(rj sdk.V2WorkflowRunJob, stepID string, stepOrder int64) string {
	return fmt.Sprintf("%s-%d-%d-%s-%s", rj.WorkflowName, rj.RunNumber, rj.RunAttempt, rj.JobID, sdk.GetJobStepName(stepID, int(stepOrder)))
}
