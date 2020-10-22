package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fsamin/go-repo"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/pkg/browser"
)

var workflowRunManualCmd = cli.Command{
	Name:  "run",
	Short: "Run a CDS workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Flags: []cli.Flag{
		{
			Name:      "data",
			ShortHand: "d",
			Usage:     "Run the workflow with payload data",
			IsValid: func(s string) bool {
				if strings.TrimSpace(s) == "" {
					return true
				}
				data := map[string]interface{}{}
				return json.Unmarshal([]byte(s), &data) == nil
			},
		},
		{
			Name:      "parameter",
			ShortHand: "p",
			Usage:     "Run the workflow with pipeline parameter",
			IsValid: func(s string) bool {
				if s == "" {
					return true
				}
				// Hacking cobra which split param with a double pipe
				splittedParam := strings.Split(s, "||")
				for _, p := range splittedParam {
					if strings.Count(p, "=") < 1 {
						return false
					}
				}
				return true
			},
			Type: cli.FlagSlice,
		},
		{
			Name:  "run-number",
			Usage: "Existing Workflow RUN Number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
		},
		{
			Name:  "node-name",
			Usage: "Node Name to relaunch; Flag run-number is mandatory",
		},
		{
			Name:      "interactive",
			ShortHand: "i",
			Usage:     "Follow the workflow run in an interactive terminal user interface",
			Type:      cli.FlagBool,
		},
		{
			Name:      "open-web-browser",
			ShortHand: "o",
			Usage:     "Open web browser on the workflow run",
			Type:      cli.FlagBool,
		},
		{
			Name:      "sync",
			ShortHand: "s",
			Usage:     "Synchronise your pipelines with your last editions. Must be used with flag run-number",
			Type:      cli.FlagBool,
		},
	},
}

func workflowRunManualRun(v cli.Values) error {
	if v.GetBool("sync") && v.GetString("run-number") == "" {
		return fmt.Errorf("could not use flag --sync without flag --run-number")
	}

	manual := sdk.WorkflowNodeRunManual{}
	if strings.TrimSpace(v.GetString("data")) != "" {
		data := map[string]interface{}{}
		if err := json.Unmarshal([]byte(v.GetString("data")), &data); err != nil {
			return fmt.Errorf("error payload isn't a valid json")
		}
		manual.Payload = data
	} else {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return fmt.Errorf("Unable to get current path: %s", err)
		}
		var gitBranch, currentBranch, remoteURL string
		ctx := context.Background()
		r, err := repo.New(ctx, dir)
		if err == nil { // If the directory is a git repository
			currentBranch, _ = r.CurrentBranch(ctx)
			remoteURL, err = r.FetchURL(ctx)
			if err != nil {
				return sdk.WrapError(err, "cannot fetch the remote url")
			}
		}

		wf, err := client.WorkflowGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName))
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow")
		}

		// Check if we are on the same repository and if we have a git.branch in the default payload
		if wf.WorkflowData.Node.Context != nil && wf.WorkflowData.Node.Context.ApplicationID != 0 {
			app := wf.GetApplication(wf.WorkflowData.Node.Context.ApplicationID)
			if remoteURL != "" && strings.Contains(remoteURL, app.RepositoryFullname) && currentBranch != "" {
				defaultPayload, err := wf.WorkflowData.Node.Context.DefaultPayloadToMap()
				if err == nil && defaultPayload["git.branch"] != "" {
					gitBranch = currentBranch
				}
			}
		}

		if gitBranch != "" {
			m := map[string]string{}
			m["git.branch"] = gitBranch
			manual.Payload = m
		}
	}

	pipParams := v.GetStringSlice("parameter")
	if len(pipParams) > 0 {
		for _, sParam := range pipParams {
			if sParam == "" {
				continue
			}
			splittedParam := strings.SplitN(sParam, "=", 2)
			sdk.AddParameter(&manual.PipelineParameters, splittedParam[0], sdk.StringParameter, splittedParam[1])
		}
	}

	var runNumber, fromNodeID int64

	if v.GetString("run-number") != "" {
		var errp error
		runNumber, errp = strconv.ParseInt(v.GetString("run-number"), 10, 64)
		if errp != nil {
			return fmt.Errorf("run-number invalid: not a integer")
		}
	}

	if v.GetBool("sync") {
		if _, err := client.WorkflowRunResync(v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber); err != nil {
			return fmt.Errorf("Cannot resync your workflow run %d : %v", runNumber, err)
		}
	}

	if v.GetString("node-name") != "" {
		if runNumber <= 0 {
			return fmt.Errorf("You can use flag node-name without flag run-number")
		}
		wr, err := client.WorkflowRunGet(v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
		if err != nil {
			return err
		}
		for _, wnrs := range wr.WorkflowNodeRuns {
			for _, wnr := range wnrs {
				wn := wr.Workflow.WorkflowData.NodeByID(wnr.WorkflowNodeID)
				if wn.Name == v.GetString("node-name") {
					fromNodeID = wnr.WorkflowNodeID
					break
				}
			}
		}
	}

	w, err := client.WorkflowRunFromManual(v.GetString(_ProjectKey), v.GetString(_WorkflowName), manual, runNumber, fromNodeID)
	if err != nil {
		return err
	}

	fmt.Printf("Workflow %s #%d has been launched\n", v.GetString(_WorkflowName), w.Number)

	configUser, err := client.ConfigUser()
	if err != nil {
		return err
	}
	if configUser.URLUI == "" {
		fmt.Println("Unable to retrieve workflow URI")
		return nil
	}

	if !v.GetBool("interactive") {
		url := fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", configUser.URLUI, v.GetString(_ProjectKey), v.GetString(_WorkflowName), w.Number)
		fmt.Println(url)

		if v.GetBool("open-web-browser") {
			return browser.OpenURL(url)
		}

		return nil
	}

	return workflowRunInteractive(v, w, configUser.URLUI)
}
