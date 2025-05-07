package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminHooksOutgoingEventCmd = cli.Command{
	Name:    "outgoing",
	Aliases: []string{"o", "out"},
	Short:   "Manage outgoing events",
}

func adminHooksOutgoingEvents() *cobra.Command {
	return cli.NewCommand(adminHooksOutgoingEventCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminHooksOutgoingEventListCmd, adminHooksOutgoingEventListRun, nil),
		cli.NewCommand(adminHooksOutgoingEventGetCmd, adminHooksOutgoingEventGetRun, nil),
	})
}

var adminHooksOutgoingEventGetCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get outgoing event",
	Args: []cli.Arg{
		{Name: "project"},
		{Name: "vcs-server"},
		{Name: "repository"},
		{Name: "workflow"},
		{Name: "event-id"},
	},
}

func adminHooksOutgoingEventGetRun(v cli.Values) error {
	path := fmt.Sprintf("/admin/outgoing/%s/%s/%s/%s/%s",
		v.GetString("project"),
		url.PathEscape(v.GetString("vcs-server")),
		url.PathEscape(v.GetString("repository")),
		v.GetString("workflow"),
		v.GetString("event-id"))
	btes, err := client.ServiceCallGET("hooks", path)
	if err != nil {
		return err
	}

	var event sdk.HookWorkflowRunOutgoingEvent
	if err := sdk.JSONUnmarshal(btes, &event); err != nil {
		return err
	}

	type CliTrigger struct {
		Project             string `json:"project" cli:"project"`
		VCS                 string `json:"vcs" cli:"vcs"`
		Repository          string `json:"repository" cli:"repository"`
		Workflow            string `json:"workflow" cli:"workflow"`
		RepositoryEventUUID string `json:"repository_event_id" cli:"repository_event_id"`
		Status              string `json:"status" cli:"status"`
		Error               string `json:"error,omitempty" cli:"error"`
	}
	type CliResult struct {
		ID         string       `json:"id" cli:"id"`
		Created    time.Time    `json:"created" cli:"created"`
		Project    string       `json:"project" cli:"project"`
		VCS        string       `json:"vcs" cli:"vcs"`
		Repository string       `json:"repository" cli:"repository"`
		Workflow   string       `json:"workflow" cli:"workflow"`
		RunID      string       `json:"run_id" cli:"run_id"`
		Ref        string       `json:"ref" cli:"ref"`
		Status     string       `json:"status" cli:"status"`
		LastError  string       `json:"error,omitempty" cli:"error"`
		Triggers   []CliTrigger `json:"triggers" cli:"triggers"`
	}

	result := CliResult{
		ID:         event.UUID,
		Created:    time.Unix(0, event.Created),
		Project:    event.Event.WorkflowProject,
		VCS:        event.Event.WorkflowVCSServer,
		Repository: event.Event.WorkflowRepository,
		Workflow:   event.Event.WorkflowName,
		RunID:      event.Event.WorkflowRunID,
		Ref:        event.Event.WorkflowRef,
		Status:     event.Status,
		LastError:  event.LastError,
		Triggers:   make([]CliTrigger, 0, len(event.HooksToTriggers)),
	}

	for _, e := range event.HooksToTriggers {
		result.Triggers = append(result.Triggers, CliTrigger{
			Project:             e.ProjectKey,
			VCS:                 e.VCSName,
			Repository:          e.RepositoryName,
			Workflow:            e.WorkflowName,
			RepositoryEventUUID: e.HookRepositoryEventID,
			Status:              e.Status,
			Error:               e.Error,
		})
	}

	bts, _ := json.Marshal(result)
	fmt.Printf("%s\n", string(bts))
	return nil
}

var adminHooksOutgoingEventListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"l", "ls"},
	Short:   "List outgoing events for the given workflows",
	Args: []cli.Arg{
		{Name: "project"},
		{Name: "vcs-server"},
		{Name: "repository"},
		{Name: "workflow"},
	},
}

func adminHooksOutgoingEventListRun(v cli.Values) (cli.ListResult, error) {
	path := fmt.Sprintf("/admin/outgoing/%s/%s/%s/%s",
		v.GetString("project"),
		url.PathEscape(v.GetString("vcs-server")),
		url.PathEscape(v.GetString("repository")),
		v.GetString("workflow"))
	btes, err := client.ServiceCallGET("hooks", path)
	if err != nil {
		return nil, err
	}
	var events []sdk.HookWorkflowRunOutgoingEvent
	if err := sdk.JSONUnmarshal(btes, &events); err != nil {
		return nil, err
	}
	type CliResult struct {
		ID        string    `cli:"id"`
		Created   time.Time `cli:"created"`
		Ref       string    `cli:"ref"`
		Status    string    `cli:"status"`
		LastError string    `cli:"error"`
	}
	results := make([]CliResult, 0, len(events))
	for _, e := range events {
		results = append(results, CliResult{
			ID:        e.UUID,
			Created:   time.Unix(0, e.Created),
			Ref:       e.Event.WorkflowRef,
			Status:    e.Status,
			LastError: e.LastError,
		})
	}
	return cli.AsListResult(results), nil
}
