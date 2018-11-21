package main

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var workflowLogCmd = cli.Command{
	Name:    "logs",
	Aliases: []string{"log"},
	Short:   "Manage CDS Workflow Run Logs",
	Long: `Download logs from a workflow run.

	# list all logs files on latest run
	$ cdsctl workflow logs list KEY WF

	# list all logs files on run number 1
	$ cdsctl workflow logs list KEY WF 1

	# download all logs files on latest run
	$ cdsctl workflow logs download KEY WF

	# download only one file, for run number 1
	$ cdsctl workflow logs download KEY WF 1 --pattern="MyJob"
	# this will download file WF-1.0-pipeline.myPipeline-stage.MyStage-job.MyJob-status.Success-step.0.log

`,
}

func workflowLog() *cobra.Command {
	return cli.NewCommand(workflowLogCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowLogListCmd, workflowLogListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowLogDownloadCmd, workflowLogDownloadRun, nil, withAllCommandModifiers()...),
	})
}

var workflowLogListCmd = cli.Command{
	Name:  "list",
	Short: "List logs from a workflow run",
	Long: `List logs from a workflow run. There on log file for each step.

	# list all logs files from projet KEY, with workflow named WD on latest run
	$ cdsctl workflow logs list KEY WF

	# list all logs files from projet KEY, with workflow named WD on run 1
	$ cdsctl workflow logs list KEY WF 1

`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	OptionalArgs: []cli.Arg{
		{
			Name: "run-number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 1,
		},
	},
}

func workflowLogSearchNumber(v cli.Values) (int64, error) {
	runNumber, errRunNumber := v.GetInt64("run-number")
	if errRunNumber != nil {
		return 0, errRunNumber
	}
	if runNumber == 0 {
		filters := []cdsclient.Filter{
			{
				Name:  "workflow",
				Value: v.GetString(_WorkflowName),
			},
		}

		fmt.Printf("Searching latest run on workflow %s...\n", v.GetString(_WorkflowName))
		runs, err := client.WorkflowRunSearch(v.GetString(_ProjectKey), 0, 0, filters...)
		if err != nil {
			return 0, err
		}
		if len(runs) < 1 {
			return 0, fmt.Errorf("workflow run not found")
		}
		runNumber = runs[0].Number
	}
	return runNumber, nil
}

func workflowLogListRun(v cli.Values) error {
	runNumber, err := workflowLogSearchNumber(v)
	if err != nil {
		return err
	}

	wr, err := client.WorkflowRunGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return err
	}
	fmt.Printf("List logs files on workflow %s run %d\n", v.GetString(_WorkflowName), runNumber)
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

	# download all logs files on latest run
	$ cdsctl workflow logs download KEY WF

	# download all logs files on run number 1
	$ cdsctl workflow logs download KEY WF 1

	# download only one file:
	$ cdsctl workflow logs download KEY WF 1 --pattern="MyStage"
	# this will download WF-1.0-pipeline.myPipeline-stage.MyStage-job.MyJob-status.Success-step.0.log for example

`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	OptionalArgs: []cli.Arg{
		{
			Name: "run-number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 1,
		},
	},
	Flags: []cli.Flag{
		{
			Name:  "pattern",
			Usage: "Filter on log filename",
			Kind:  reflect.String,
		},
	},
}

func workflowLogDownloadRun(v cli.Values) error {
	runNumber, err := workflowLogSearchNumber(v)
	if err != nil {
		return err
	}

	fmt.Printf("Downloding logs files from workflow %s run %d\n", v.GetString(_WorkflowName), runNumber)

	wr, err := client.WorkflowRunGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return err
	}
	logs := workflowLogProcess(wr)

	var reg *regexp.Regexp
	if v.GetString("pattern") != "" {
		var errp error
		reg, errp = regexp.Compile(v.GetString("pattern"))
		if errp != nil {
			return fmt.Errorf("Invalid pattern %s: %v", v.GetString("pattern"), errp)
		}
	}

	var ok bool
	for _, log := range logs {
		if v.GetString("pattern") != "" && !reg.MatchString(log.getFilename()) {
			continue
		}

		buildState, err := client.WorkflowNodeRunJobStep(v.GetString(_ProjectKey),
			v.GetString(_WorkflowName),
			runNumber,
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
