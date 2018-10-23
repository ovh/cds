package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	workflowLogCmd = cli.Command{
		Name:    "logs",
		Aliases: []string{"log"},
		Short:   "Manage CDS Workflow Run Logs",
		Long: `Download logs from a workflow run.

	# list all logs files
	$ cdsctl workflow logs list KEY WF 1
	
	# list all logs files
	$ cdsctl workflow logs download KEY WF 1
	
	# download only one file:
	$ cdsctl workflow logs download KEY WF 1 WF-1.0-pipeline.myPipeline-stage.MyStage-job.MyJob-status.Success-step.0.log

`,
	}

	workflowLog = cli.NewCommand(workflowLogCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(workflowLogListCmd, workflowLogListRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowLogDownloadCmd, workflowLogDownloadRun, nil, withAllCommandModifiers()...),
		})
)

var workflowLogListCmd = cli.Command{
	Name:  "list",
	Short: "List logs from a workflow run",
	Long: `List logs from a workflow run. There on log file for each step.

	# list all logs files from projet KEY, with workflow named WD on run 1
	$ cdsctl workflow logs list KEY WF 1

`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "number"},
	},
}

func workflowLogListRun(v cli.Values) error {
	number, err := strconv.ParseInt(v["number"], 10, 64)
	if err != nil {
		return fmt.Errorf("number parameter have to be an integer")
	}

	wr, err := client.WorkflowRunGet(v[_ProjectKey], v[_WorkflowName], number)
	if err != nil {
		return err
	}
	logs := workflowLogProcess(wr)
	for _, log := range logs {
		fmt.Println(log.getFilename())
	}
	return nil
}

type workflowLogDetail struct {
	workflowName string
	pipelineName string
	stageName    string
	status       string
	jobName      string
	runID        int64
	jobID        int64
	stepOrder    int
	number       int64
	subNumber    int64
}

func (w workflowLogDetail) getFilename() string {
	return fmt.Sprintf("%s-%d.%d-pipeline.%s-stage.%s-job.%s-status.%s-step.%d.log",
		w.workflowName,
		w.number,
		w.subNumber,
		w.pipelineName,
		strings.Replace(w.stageName, " ", "", -1),
		strings.Replace(w.jobName, " ", "", -1),
		w.status,
		w.stepOrder,
	)
}

func workflowLogProcess(wr *sdk.WorkflowRun) []workflowLogDetail {
	logs := []workflowLogDetail{}
	for _, noderuns := range wr.WorkflowNodeRuns {
		for _, node := range noderuns {
			for _, stage := range node.Stages {
				for _, job := range stage.RunJobs {

					for _, step := range job.Job.StepStatus {
						logs = append(logs,
							workflowLogDetail{
								workflowName: wr.Workflow.Name,
								pipelineName: node.WorkflowNodeName,
								stageName:    stage.Name,
								jobName:      job.Job.Job.Action.Name,
								jobID:        job.ID,
								status:       job.Status,
								stepOrder:    step.StepOrder,
								runID:        node.ID,
								number:       wr.Number,
								subNumber:    wr.LastSubNumber,
							})
					}
				}
			}
		}
	}
	return logs
}

var workflowLogDownloadCmd = cli.Command{
	Name:  "download",
	Short: "Download logs from a workflow run.",
	Long: `Download logs from a workflow run. You can download all logs files or just one log if you want.

	# list all logs files
	$ cdsctl workflow logs download KEY WF 1

	# download only one file:
	$ cdsctl workflow logs download KEY WF 1 WF-1.0-pipeline.myPipeline-stage.MyStage-job.MyJob-status.Success-step.0.log

`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "number"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "filename"},
	},
}

func workflowLogDownloadRun(v cli.Values) error {
	number, err := strconv.ParseInt(v["number"], 10, 64)
	if err != nil {
		return fmt.Errorf("number parameter have to be an integer")
	}

	wr, err := client.WorkflowRunGet(v[_ProjectKey], v[_WorkflowName], number)
	if err != nil {
		return err
	}
	logs := workflowLogProcess(wr)

	var ok bool
	for _, log := range logs {
		if v["filename"] != "" && v["filename"] != log.getFilename() {
			continue
		}

		buildState, err := client.WorkflowNodeRunJobStep(v[_ProjectKey],
			v[_WorkflowName],
			number,
			log.runID,
			log.jobID,
			log.stepOrder,
		)
		if err != nil {
			return err
		}

		d1 := []byte(buildState.StepLogs.Val)
		if err := ioutil.WriteFile(log.getFilename(), d1, 0644); err != nil {
			return err
		}
		fmt.Printf("file %s created\n", log.getFilename())
		ok = true
	}

	if !ok {
		return fmt.Errorf("No log downloaded")
	}
	return nil
}
