package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/fsamin/go-dump"
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
			cli.NewListCommand(workflowHistoryCmd, workflowHistoryRun, nil),
			cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil),
			cli.NewDeleteCommand(workflowDeleteCmd, workflowDeleteRun, nil),
			cli.NewCommand(workflowRunManualCmd, workflowRunManualRun, nil),
			cli.NewCommand(workflowExportCmd, workflowExportRun, nil),
			cli.NewCommand(workflowImportCmd, workflowImportRun, nil),
			cli.NewCommand(workflowPullCmd, workflowPullRun, nil),
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

var workflowHistoryCmd = cli.Command{
	Name:  "history",
	Short: "History of a CDS workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	OptionalArgs: []cli.Arg{
		{
			Name: "offset",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 1,
		},
		{
			Name: "limit",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 2,
		},
	},
}

func workflowHistoryRun(v cli.Values) (cli.ListResult, error) {
	var offset int64
	if v.GetString("offset") != "" {
		var errn error
		offset, errn = v.GetInt64("offset")
		if errn != nil {
			return nil, errn
		}
	}

	var limit int64
	if v.GetString("limit") != "" {
		var errl error
		limit, errl = v.GetInt64("limit")
		if errl != nil {
			return nil, errl
		}
	}

	w, err := client.WorkflowRunList(v["project-key"], v["workflow-name"], offset, limit)
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

func workflowShowRun(v cli.Values) (interface{}, error) {
	var runNumber int64
	if v.GetString("run-number") != "" {
		var errl error
		runNumber, errl = v.GetInt64("run-number")
		if errl != nil {
			return nil, errl
		}
	}

	if runNumber == 0 {
		w, err := client.WorkflowGet(v["project-key"], v["workflow-name"])
		if err != nil {
			return nil, err
		}
		return *w, nil
	}

	w, err := client.WorkflowRunGet(v["project-key"], v["workflow-name"], runNumber)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, tag := range w.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", tag.Tag, tag.Value))
	}

	type wtags struct {
		sdk.WorkflowRun
		Payload string `cli:"payload"`
		Tags    string `cli:"tags"`
	}

	var payload []string
	if v, ok := w.WorkflowNodeRuns[w.Workflow.RootID]; ok {
		if len(v) > 0 {
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			pl, errm1 := e.ToStringMap(v[0].Payload)
			if errm1 != nil {
				return nil, errm1
			}
			for k, kv := range pl {
				payload = append(payload, fmt.Sprintf("%s:%s", k, kv))
			}
			payload = append(payload)
		}
	}

	wt := &wtags{*w, strings.Join(payload, " "), strings.Join(tags, " ")}
	return *wt, nil
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

var workflowExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.String,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func workflowExportRun(c cli.Values) error {
	btes, err := client.WorkflowExport(c.GetString("project-key"), c.GetString("workflow-name"), c.GetBool("with-permissions"), c.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(btes))
	return nil
}

var workflowPullCmd = cli.Command{
	Name:  "pull",
	Short: "Pull a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "workflow-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "output-dir",
			ShortHand: "d",
			Usage:     "Output directory",
			Default:   ".cds",
		},
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Force, may override files",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "quiet",
			Usage:   "If true, do not output filename created",
			Default: "false",
		},
	},
}

func workflowPullRun(c cli.Values) error {
	dir := strings.TrimSpace(c.GetString("output-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return fmt.Errorf("Unable to create directory %s: %v", c.GetString("output-dir"), err)
	}

	tr, err := client.WorkflowPull(c.GetString("project-key"), c.GetString("workflow-name"), c.GetBool("with-permissions"))
	if err != nil {
		return err
	}

	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fname := filepath.Join(dir, hdr.Name)
		if _, err = os.Stat(fname); err == nil || os.IsExist(err) {
			if !c.GetBool("force") {
				if !cli.AskForConfirmation(fmt.Sprintf("This will override %s. Do you want to continue?", fname)) {
					os.Exit(0)
				}
			}
		}

		if verbose {
			fmt.Println("Creating file", fname)
		}
		fi, err := os.Create(fname)
		if err != nil {
			return err
		}
		if _, err := io.Copy(fi, tr); err != nil {
			return err
		}
		if err := fi.Close(); err != nil {
			return err
		}
		if !c.GetBool("quiet") {
			fmt.Println(fname)
		}
	}
	return nil
}

var workflowImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a workflow",
	Args: []cli.Arg{
		{Name: "project-key"},
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Override workflow if exists",
			Default: "false",
		},
	},
}

func workflowImportRun(c cli.Values) error {
	path := c.GetString("filename")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var format = "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	}

	msgs, err := client.WorkflowImport(c.GetString("project-key"), f, format, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}
