package main

import (
	"fmt"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"net/url"
	"time"
)

var adminHooksRepositoryEventCmd = cli.Command{
	Name:    "event",
	Aliases: []string{"e", "events"},
	Short:   "Manage repositories events",
}

func adminHooksRepositoryEvents() *cobra.Command {
	return cli.NewCommand(adminHooksRepositoryEventCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminHooksRepoEventListCmd, adminHooksRepoEventListRun, nil),
		cli.NewCommand(adminHooksRepoEventGetCmd, adminHooksRepoEventGetRun, nil),
		cli.NewCommand(adminHookRepoEventRestartCmd, adminHookRepoEventRestartRun, nil),
	})
}

var adminHookRepoEventRestartCmd = cli.Command{
	Name:    "restart",
	Aliases: []string{"reboot"},
	Short:   "Get event",
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

func adminHooksRepoEventGetRun(v cli.Values) error {
	path := fmt.Sprintf("/v2/repository/event/%s/%s/%s", url.PathEscape(v.GetString("vcs-server")), url.PathEscape(v.GetString("repository")), v.GetString("event-id"))
	btes, err := client.ServiceCallGET("hooks", path)
	if err != nil {
		return err
	}

	var event sdk.HookRepositoryEvent
	if err := sdk.JSONUnmarshal(btes, &event); err != nil {
		return err
	}

	fmt.Printf("ID: %s\n", event.UUID)
	fmt.Printf("Created: %s\n", time.Unix(0, event.Created))
	fmt.Printf("Last Update%s\n", time.Unix(0, event.LastUpdate))
	fmt.Printf("EventName: %s\n", event.EventName)
	fmt.Printf("Event: %s\n", string(event.Body))
	fmt.Printf("Status: %s\n", event.Status)
	for _, a := range event.Analyses {
		fmt.Printf("Analyze %s: %s\n", a.AnalyzeID, a.Status)
	}
	fmt.Printf("User: %s\n", event.UserID)

	return nil
}

var adminHooksRepoEventListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"l", "ls"},
	Short:   "List events",
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
		UUID      string    `cli:"uuid"`
		Created   time.Time `cli:"created"`
		EventName string    `cli:"event_name"`
		Status    string    `cli:"status"`
	}
	rs := make([]Result, 0, len(events))
	for _, e := range events {
		rs = append(rs, Result{
			UUID:      e.UUID,
			Status:    e.Status,
			EventName: e.EventName,
			Created:   time.Unix(0, e.Created),
		})
	}
	return cli.AsListResult(rs), nil
}
