package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/fsamin/go-repo"

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
			Type:  cli.FlagBool,
			Usage: "Wait the workflow to be over",
		},
	},
}

func workflowStatusRun(v cli.Values) (interface{}, error) {
	if !v.GetBool("track") {
		return workflowStatusRunWithoutTrack(v)
	}
	return workflowStatusRunWithTrack(v)
}

func workflowStatusRunWithTrack(v cli.Values) (interface{}, error) {
	var runNumber int64
	var currentDisplay = new(cli.Display)

	// try to get the latest commit
	ctx := context.Background()
	r, err := repo.New(ctx, ".")
	if err != nil {
		return nil, fmt.Errorf("unable to get latest commit: %v", err)
	}
	latestCommit, err := r.LatestCommit(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get latest commit: %v", err)
	}

	currentDisplay.Printf("Looking for %s...\n", cli.Magenta(latestCommit.LongHash))
	currentDisplay.Do(context.Background())

	for runNumber == 0 {
		runNumber, _ = workflowNodeForCurrentRepo(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
		time.Sleep(500 * time.Millisecond)
	}

	if runNumber == 0 {
		runs, err := client.WorkflowRunList(v.GetString(_ProjectKey), v.GetString(_WorkflowName), 0, 1)
		if err != nil {
			return nil, err
		}
		if len(runs) != 1 {
			return nil, fmt.Errorf("workflow run not found")
		}
		runNumber = runs[0].Number
	}

	for {
		run, err := client.WorkflowRunGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
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

// w-cds #10.0 ✓root -> ✓build -> ✓test -> ✗deploy
func workflowRunFormatDisplay(run *sdk.WorkflowRun, commit repo.Commit, currentDisplay *cli.Display) {
	var output = "%s [%s | %s] " + cli.Cyan("#%d.%d", run.Number, run.LastSubNumber)
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
			output += fmt.Sprintf(" %s ", cli.Arrow)
		}

		nodeRun := nodeRuns[0]
		switch nodeRun.Status {
		case sdk.StatusSuccess:
			output += cli.Green("%s %s", cli.OKChar, cli.Green(nodeRun.WorkflowNodeName))
		case sdk.StatusFail:
			output += cli.Red("%s %s", cli.KOChar, cli.Red(nodeRun.WorkflowNodeName))
		default:
			output += cli.Blue("%s %s", cli.BuildingChar, cli.Blue(nodeRun.WorkflowNodeName))
		}
	}

	currentDisplay.Printf(output, run.Workflow.Name, cli.Magenta(commit.Hash), cli.Magenta(commit.Author))
}

func workflowStatusRunWithoutTrack(v cli.Values) (interface{}, error) {
	var runNumber int64
	var errRunNumber error
	// if no run number, get the latest
	runNumberStr := v.GetString("run-number")
	if runNumberStr != "" {
		runNumber, errRunNumber = strconv.ParseInt(runNumberStr, 10, 64)
	} else {
		runNumber, errRunNumber = workflowNodeForCurrentRepo(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
	}
	if errRunNumber != nil {
		return nil, errRunNumber
	}
	if runNumber == 0 {
		runs, err := client.WorkflowRunList(v.GetString(_ProjectKey), v.GetString(_WorkflowName), 0, 1)
		if err != nil {
			return nil, err
		}
		if len(runs) != 1 {
			return 0, fmt.Errorf("workflow run not found")
		}
		runNumber = runs[0].Number
	}

	run, err := client.WorkflowRunGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
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
	var payloadString interface{}
	if len(run.WorkflowNodeRuns) > 0 {
		if v, ok := run.WorkflowNodeRuns[run.Workflow.WorkflowData.Node.ID]; ok {
			if len(v) > 0 {
				payloadString = v[0].Payload
			}
		}
		e := dump.NewDefaultEncoder()
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false
		pl, errm1 := e.ToStringMap(payloadString)
		if errm1 != nil {
			return nil, errm1
		}
		for k, kv := range pl {
			payload = append(payload, fmt.Sprintf("%s:%s", k, kv))
		}
		payload = append(payload)
	}
	wt := &wtags{*run, strings.Join(payload, " "), strings.Join(tags, " ")}
	return *wt, nil
}
