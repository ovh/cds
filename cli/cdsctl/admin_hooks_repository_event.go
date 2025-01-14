package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminHooksRepositoryEventCmd = cli.Command{
	Name:    "event",
	Aliases: []string{"e", "events"},
	Short:   "Manage repositories events",
}

func adminHooksRepositoryEvents() *cobra.Command {
	return cli.NewCommand(adminHooksRepositoryEventCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminHooksRepoEventListCmd, adminHooksRepoEventListRun, nil),
		cli.NewGetCommand(adminHooksRepoEventGetCmd, adminHooksRepoEventGetRun, nil),
		cli.NewCommand(adminHookRepoEventRestartCmd, adminHookRepoEventRestartRun, nil),
	})
}

var adminHookRepoEventRestartCmd = cli.Command{
	Name:    "restart",
	Aliases: []string{"reboot"},
	Short:   "Restart an event",
	Args: []cli.Arg{
		{Name: "vcs-server"},
		{Name: "repository"},
		{Name: "event-id"},
	},
}

func adminHookRepoEventRestartRun(v cli.Values) error {
	path := fmt.Sprintf("/admin/repository/event/%s/%s/%s/restart", url.PathEscape(v.GetString("vcs-server")), url.PathEscape(v.GetString("repository")), v.GetString("event-id"))
	if _, err := client.ServiceCallPOST("hooks", path, nil); err != nil {
		return err
	}
	return nil
}

var adminHooksRepoEventGetCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get event",
	Args: []cli.Arg{
		{Name: "vcs-server"},
		{Name: "repository"},
		{Name: "event-id"},
	},
}

func adminHooksRepoEventGetRun(v cli.Values) (interface{}, error) {
	path := fmt.Sprintf("/v2/repository/event/%s/%s/%s", url.PathEscape(v.GetString("vcs-server")), url.PathEscape(v.GetString("repository")), v.GetString("event-id"))
	btes, err := client.ServiceCallGET("hooks", path)
	if err != nil {
		return nil, err
	}

	var event sdk.HookRepositoryEvent
	if err := sdk.JSONUnmarshal(btes, &event); err != nil {
		return nil, err
	}

	type HookEventCLI struct {
		ID                  string                            `cli:"id"`
		Created             time.Time                         `cli:"created"`
		LastUpdate          time.Time                         `cli:"last_update"`
		EventName           sdk.WorkflowHookEventName         `cli:"event_name"`
		VCSServerName       string                            `cli:"vcs_server_name"`
		RepositoryName      string                            `cli:"repository_name"`
		Ref                 string                            `cli:"ref"`
		Commit              string                            `cli:"commit"`
		Path                []string                          `cli:"path"`
		Event               string                            `cli:"event"`
		Status              string                            `cli:"status"`
		Error               string                            `cli:"last_error"`
		NbErrors            int64                             `cli:"nb_errors"`
		Analyses            []sdk.HookRepositoryEventAnalysis `cli:"analyses"`
		UserID              string                            `cli:"user_id"`
		Username            string                            `cli:"username"`
		SignKey             string                            `cli:"sign_key"`
		SigningKeyOperation string                            `cli:"signing_key_operation"`
	}

	cli := HookEventCLI{
		ID:                  event.UUID,
		Created:             time.Unix(0, event.Created),
		LastUpdate:          time.UnixMilli(event.LastUpdate),
		EventName:           event.EventName,
		VCSServerName:       event.VCSServerName,
		RepositoryName:      event.RepositoryName,
		Ref:                 event.ExtractData.Ref,
		Commit:              event.ExtractData.Commit,
		Path:                event.ExtractData.Paths,
		Event:               string(event.Body),
		Status:              event.Status,
		Error:               event.LastError,
		NbErrors:            event.NbErrors,
		UserID:              event.Initiator.UserID,
		Username:            event.Initiator.Username(),
		Analyses:            event.Analyses,
		SignKey:             event.SignKey,
		SigningKeyOperation: event.SigningKeyOperation,
	}

	return cli, nil
}

var adminHooksRepoEventListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"l", "ls"},
	Short:   "List repository events",
	Args: []cli.Arg{
		{Name: "vcs-server"},
		{Name: "repository"},
	},
}

func adminHooksRepoEventListRun(v cli.Values) (cli.ListResult, error) {
	path := fmt.Sprintf("/v2/repository/event/%s/%s", url.PathEscape(v.GetString("vcs-server")), url.PathEscape(v.GetString("repository")))
	btes, err := client.ServiceCallGET("hooks", path)
	if err != nil {
		return nil, err
	}
	var events []sdk.HookRepositoryEvent
	if err := sdk.JSONUnmarshal(btes, &events); err != nil {
		return nil, err
	}
	type Result struct {
		UUID      string                    `cli:"uuid"`
		Created   time.Time                 `cli:"created"`
		EventName sdk.WorkflowHookEventName `cli:"event_name"`
		Status    string                    `cli:"status"`
		Ref       string                    `cli:"ref"`
		Commit    string                    `cli:"commit"`
	}
	rs := make([]Result, 0, len(events))
	for _, e := range events {
		rs = append(rs, Result{
			UUID:      e.UUID,
			Status:    e.Status,
			EventName: e.EventName,
			Created:   time.Unix(0, e.Created),
			Ref:       e.ExtractData.Ref,
			Commit:    e.ExtractData.Commit,
		})
	}
	return cli.AsListResult(rs), nil
}
