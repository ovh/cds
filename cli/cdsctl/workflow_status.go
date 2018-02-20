package main

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	dump "github.com/fsamin/go-dump"
	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowStatusCmd = cli.Command{
	Name:  "status",
	Short: "Check the status of the run",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	OptionalArgs: []cli.Arg{
		{Name: "run-number"},
	},
	Flags: []cli.Flag{
		{
			Name:  "track",
			Kind:  reflect.Bool,
			Usage: "Wait the workflow to be over",
		},
	},
}

func workflowStatusRun(v cli.Values) (interface{}, error) {
	var track = v.GetBool("track")
	if !track {
		return workflowStatusRunWithoutTrack(v)
	}
	return workflowStatusRunWithTrack(v)
}

func workflowStatusRunWithTrack(v cli.Values) (interface{}, error) {
	var runNumber int64
	var currentDisplay = new(cli.Display)

	//Try to get the latest commit
	r, err := repo.New(".")
	if err != nil {
		return nil, fmt.Errorf("unable to get latest commit: %v", err)
	}
	latestCommit, err := r.LatestCommit()
	if err != nil {
		return nil, fmt.Errorf("unable to get latest commit: %v", err)
	}

	currentDisplay.Printf("Looking for %s...", magenta(latestCommit.Hash))
	currentDisplay.Do(context.Background())

	for runNumber == 0 {
		runNumber, _ = workflowNodeForCurrentRepo(v[_ProjectKey], v.GetString(_WorkflowName))
	}

	run, err := client.WorkflowRunGet(v[_ProjectKey], v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return nil, err
	}

	for {
		run, err = client.WorkflowRunGet(v[_ProjectKey], v.GetString(_WorkflowName), runNumber)
		if err != nil {
			return nil, err
		}

		workflowRunFormatDisplay(run, latestCommit, currentDisplay)
		time.Sleep(500 * time.Millisecond)
		if sdk.StatusIsTerminated(run.Status) {
			break
		}
	}
	fmt.Println()
	return nil, nil
}

var (
	red          = color.New(color.FgRed).SprintfFunc()
	blue         = color.New(color.FgBlue).SprintfFunc()
	magenta      = color.New(color.FgMagenta).SprintfFunc()
	green        = color.New(color.FgGreen).SprintfFunc()
	cyan         = color.New(color.FgCyan).SprintfFunc()
	buildingChar = blue("↻")
	okChar       = green("✓")
	koChar       = red("✗")
	arrow        = cyan("➤")
)

//w-cds #10.0 ✓root -> ✓build -> ✓test -> ✗deploy
func workflowRunFormatDisplay(run *sdk.WorkflowRun, commit repo.Commit, currentDisplay *cli.Display) {
	var output = "%s [%s | %s] " + cyan("#%d.%d", run.Number, run.LastSubNumber)
	nodeIDs := []int64{}
	for id := range run.WorkflowNodeRuns {
		nodeIDs = append(nodeIDs, id)
	}

	sort.Slice(nodeIDs, func(i, j int) bool {
		return nodeIDs[i] < nodeIDs[j]
	})

	for _, nodeID := range nodeIDs {
		nodeRuns := run.WorkflowNodeRuns[nodeID]
		if len(nodeRuns) == 0 {
			continue
		}
		sort.Slice(nodeRuns, func(i, j int) bool {
			return nodeRuns[i].SubNumber > nodeRuns[j].SubNumber
		})

		if output[len(output):] != " " {
			output += fmt.Sprintf(" %s ", arrow)
		}

		nodeRun := nodeRuns[0]
		switch nodeRun.Status {
		case sdk.StatusSuccess.String():
			output += green("%s %s", okChar, green(nodeRun.WorkflowNodeName))
		case sdk.StatusFail.String():
			output += red("%s %s", koChar, red(nodeRun.WorkflowNodeName))
		default:
			output += blue("%s %s", buildingChar, blue(nodeRun.WorkflowNodeName))
		}
	}

	currentDisplay.Printf(output, run.Workflow.Name, magenta(commit.Hash), magenta(commit.Author))
}

func workflowStatusRunWithoutTrack(v cli.Values) (interface{}, error) {
	var runNumber int64
	var errRunNumber error
	// If no run number, get the latest
	runNumberStr := v.GetString("run-number")
	if runNumberStr != "" {
		runNumber, errRunNumber = strconv.ParseInt(runNumberStr, 10, 64)
	} else {
		runNumber, errRunNumber = workflowNodeForCurrentRepo(v[_ProjectKey], v.GetString(_WorkflowName))
	}
	if errRunNumber != nil {
		return nil, errRunNumber
	}

	run, err := client.WorkflowRunGet(v[_ProjectKey], v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, tag := range run.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", tag.Tag, tag.Value))
	}

	type wtags struct {
		sdk.WorkflowRun
		Payload string `cli:"payload"`
		Tags    string `cli:"tags"`
	}

	var payload []string
	if v, ok := run.WorkflowNodeRuns[run.Workflow.RootID]; ok {
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

	wt := &wtags{*run, strings.Join(payload, " "), strings.Join(tags, " ")}
	return wt, nil
}
