package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/pkg/browser"
)

var workflowRunManualCmd = cli.Command{
	Name:  "run",
	Short: "Run a CDS workflow",
	Ctx: []cli.Arg{
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
		{
			Name:      "interactive",
			ShortHand: "i",
			Usage:     "Follow the workflow run in an interactive terminal user interface",
			Kind:      reflect.Bool,
		},
		{
			Name:      "open-web-browser",
			ShortHand: "o",
			Usage:     "Open web browser on the workflow run",
			Kind:      reflect.Bool,
		},
	},
}

func workflowRunManualRun(v cli.Values) error {
	manual := sdk.WorkflowNodeRunManual{}
	if v["payload"] != "" {
		if err := json.Unmarshal([]byte(v["payload"]), &manual.Payload); err != nil {
			return fmt.Errorf("Error payload isn't a valid json")
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

	fmt.Printf("Workflow %s #%d has been launched\n", v["workflow-name"], w.Number)

	var baseURL string
	configUser, err := client.ConfigUser()
	if err != nil {
		return err
	}

	if b, ok := configUser[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	if baseURL == "" {
		fmt.Println("Unable to retrieve workflow URI")
		return nil
	}

	if !v.GetBool("interactive") {
		url := fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", baseURL, v["project-key"], v["workflow-name"], w.Number)
		fmt.Println(url)

		if v.GetBool("open-web-browser") {
			return browser.OpenURL(url)
		}

		return nil
	}

	return workflowRunInteractive(v, w, baseURL)
}
