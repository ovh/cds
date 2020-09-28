package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/slug"
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
		},
	},
}

func workflowLogSearchNumber(v cli.Values) (int64, error) {
	num, err := v.GetInt64("run-number")
	if err != nil {
		return 0, err
	}
	if num > 0 {
		return num, nil
	}

	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	fmt.Printf("Searching latest run for workflow %s/%s...\n", projectKey, workflowName)
	runs, err := client.WorkflowRunSearch(projectKey, 0, 0, cdsclient.Filter{
		Name:  "workflow",
		Value: workflowName,
	})
	if err != nil {
		return 0, err
	}
	if len(runs) < 1 {
		return 0, sdk.WithStack(fmt.Errorf("no run found for workflow %s/%s", projectKey, workflowName))
	}
	return runs[0].Number, nil
}

func workflowLogListRun(v cli.Values) error {
	runNumber, err := workflowLogSearchNumber(v)
	if err != nil {
		return err
	}

	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	wr, err := client.WorkflowRunGet(projectKey, workflowName, runNumber)
	if err != nil {
		return err
	}

	fmt.Printf("List logs files on workflow %s/%s run %d\n", projectKey, workflowName, runNumber)
	logs := workflowLogProcess(wr)
	for _, log := range logs {
		fmt.Println(log.getFilename())
	}
	return nil
}

type workflowLogDetailType string

const (
	workflowLogDetailTypeStep    workflowLogDetailType = "step"
	workflowLogDetailTypeService workflowLogDetailType = "service"
)

type workflowLogDetail struct {
	detailType        workflowLogDetailType
	workflowName      string
	pipelineName      string
	stageName         string
	jobName           string
	runID             int64
	jobID             int64
	number            int64
	subNumber         int64
	countUsageJobName int64
	status            string

	// for step log
	stepOrder int64

	// for service log
	serviceName string
}

func (w workflowLogDetail) getFilename() string {
	jobName := strings.Replace(w.jobName, " ", "", -1)
	if w.countUsageJobName > 0 {
		jobName = fmt.Sprintf("%s.%d", jobName, w.countUsageJobName)
	}

	var suffix string
	if w.detailType == workflowLogDetailTypeService {
		suffix = fmt.Sprintf("service.%s", w.serviceName)
	} else {
		suffix = fmt.Sprintf("step.%d", w.stepOrder)
	}

	return fmt.Sprintf("%s-%d.%d-pipeline.%s-stage.%s-job.%s-status.%s-%s.log",
		w.workflowName,
		w.number,
		w.subNumber,
		w.pipelineName,
		strings.Replace(w.stageName, " ", "", -1),
		jobName,
		w.status,
		suffix,
	)
}

func workflowLogProcess(wr *sdk.WorkflowRun) []workflowLogDetail {
	var logs []workflowLogDetail
	for _, nodeRuns := range wr.WorkflowNodeRuns {
		for _, nodeRun := range nodeRuns {
			for _, stage := range nodeRun.Stages {
				jobNames := map[string]int64{}
				for _, runJob := range stage.RunJobs {
					jobName := slug.Convert(runJob.Job.Job.Action.Name)
					countUsageJobName, ok := jobNames[jobName]
					if !ok {
						jobNames[jobName] = 1
					} else {
						jobNames[jobName]++
					}

					commonLogDetail := workflowLogDetail{
						workflowName:      wr.Workflow.Name,
						pipelineName:      nodeRun.WorkflowNodeName,
						stageName:         stage.Name,
						jobName:           runJob.Job.Job.Action.Name,
						jobID:             runJob.ID,
						runID:             nodeRun.ID,
						number:            wr.Number,
						subNumber:         nodeRun.SubNumber,
						countUsageJobName: countUsageJobName,
						status:            runJob.Status,
					}

					for _, req := range runJob.Job.Action.Requirements {
						if req.Type == sdk.ServiceRequirement {
							detail := commonLogDetail
							detail.detailType = workflowLogDetailTypeService
							detail.serviceName = req.Name
							logs = append(logs, detail)
						}
					}

					for _, step := range runJob.Job.StepStatus {
						detail := commonLogDetail
						detail.detailType = workflowLogDetailTypeStep
						detail.stepOrder = int64(step.StepOrder)
						logs = append(logs, detail)
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
		},
	},
	Flags: []cli.Flag{
		{
			Name:  "pattern",
			Usage: "Filter on log filename",
		},
	},
}

func workflowLogDownloadRun(v cli.Values) error {
	runNumber, err := workflowLogSearchNumber(v)
	if err != nil {
		return err
	}

	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	fmt.Printf("Downloading logs files from workflow %s/%s run %d\n", projectKey, workflowName, runNumber)

	wr, err := client.WorkflowRunGet(projectKey, workflowName, runNumber)
	if err != nil {
		return err
	}
	logs := workflowLogProcess(wr)

	var reg *regexp.Regexp
	if v.GetString("pattern") != "" {
		reg, err = regexp.Compile(v.GetString("pattern"))
		if err != nil {
			return sdk.NewErrorFrom(err, "invalid pattern %s", v.GetString("pattern"))
		}
	}

	feature, err := client.FeatureEnabled("cdn-job-logs", map[string]string{
		"project_key": projectKey,
	})
	if err != nil {
		return err
	}

	var ok bool
	for _, log := range logs {
		if reg != nil && !reg.MatchString(log.getFilename()) {
			continue
		}

		// If cdn logs is enabled for current project, first check if logs can be downloaded from it
		var link *sdk.CDNLogLink
		if feature.Enabled {
			if log.detailType == workflowLogDetailTypeService {
				link, err = client.WorkflowNodeRunJobServiceLink(projectKey, workflowName, log.runID, log.jobID, log.serviceName)
			} else {
				link, err = client.WorkflowNodeRunJobStepLink(projectKey, workflowName, log.runID, log.jobID, log.stepOrder)
			}
			if err != nil {
				return err
			}
		}

		var data []byte
		if link != nil && link.Exists {
			data, _, _, err = client.Request(context.Background(), http.MethodGet, link.CDNURL+link.DownloadPath, nil)
			if err != nil {
				return err
			}
		} else {
			if log.detailType == workflowLogDetailTypeService {
				serviceLog, err := client.WorkflowNodeRunJobServiceLog(projectKey, workflowName, log.runID, log.jobID, log.serviceName)
				if err != nil {
					return err
				}
				data = []byte(serviceLog.Val)
			} else {
				buildState, err := client.WorkflowNodeRunJobStepLog(projectKey, workflowName, log.runID, log.jobID, log.stepOrder)
				if err != nil {
					return err
				}
				data = []byte(buildState.StepLogs.Val)
			}
		}

		if err := ioutil.WriteFile(log.getFilename(), data, 0644); err != nil {
			return err
		}
		fmt.Printf("file %s created\n", log.getFilename())

		ok = true
	}

	if !ok {
		return sdk.WithStack(fmt.Errorf("no log downloaded"))
	}
	return nil
}
