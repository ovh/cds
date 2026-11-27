package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminKeysCmd = cli.Command{
	Name:  "keys",
	Short: "Manage the email of project PGP keys (admin only)",
}

func adminKeys() *cobra.Command {
	return cli.NewCommand(adminKeysCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminKeysOutdatedCmd, adminKeysOutdatedRun, nil),
		cli.NewCommand(adminKeysUpdateEmailCmd, adminKeysUpdateEmailRun, nil),
	})
}

var adminKeysOutdatedCmd = cli.Command{
	Name:    "outdated",
	Short:   "List the project PGP keys whose email differs from the API template",
	Example: "cdsctl admin keys outdated",
}

// adminKeyOutdatedItem joins the emails in one column: the list display skips slice fields.
type adminKeyOutdatedItem struct {
	ProjectKey    string `json:"project_key" cli:"project"`
	KeyName       string `json:"key_name" cli:"key"`
	Default       bool   `json:"default" cli:"default"`
	Disabled      bool   `json:"disabled" cli:"disabled"`
	Emails        string `json:"emails" cli:"emails"`
	ExpectedEmail string `json:"expected_email" cli:"expected_email"`
	LongKeyID     string `json:"long_key_id" cli:"long_key_id"`
}

func adminKeysOutdatedRun(_ cli.Values) (cli.ListResult, error) {
	statuses, err := client.AdminProjectKeysOutdatedEmail(context.Background())
	if err != nil {
		return nil, err
	}
	items := make([]adminKeyOutdatedItem, 0, len(statuses))
	for _, s := range statuses {
		items = append(items, adminKeyOutdatedItem{
			ProjectKey:    s.ProjectKey,
			KeyName:       s.KeyName,
			Default:       s.Default,
			Disabled:      s.Disabled,
			Emails:        strings.Join(s.Emails, ","),
			ExpectedEmail: s.ExpectedEmail,
			LongKeyID:     s.LongKeyID,
		})
	}
	return cli.AsListResult(items), nil
}

var adminKeysUpdateEmailCmd = cli.Command{
	Name:  "update-email",
	Short: "Set the API template email on the default PGP key of every project, or on one given PGP key",
	Long: `Without argument, rewrites the default PGP key (proj-pgp-<project>) of every project whose email differs
from the API template. With a project key and a key name, rewrites that PGP key only, default or not.
The key IDs do not change.`,
	Example: "cdsctl admin keys update-email\ncdsctl admin keys update-email MY-PROJECT proj-my-key",
	OptionalArgs: []cli.Arg{
		{Name: "project-key"},
		{Name: "key-name"},
	},
}

func adminKeysUpdateEmailRun(v cli.Values) error {
	projectKey, keyName := v.GetString("project-key"), v.GetString("key-name")
	var report *sdk.ProjectKeyEmailUpdateReport
	var err error
	switch {
	case projectKey == "" && keyName == "":
		report, err = client.AdminProjectKeysUpdateEmail(context.Background())
	case projectKey != "" && keyName != "":
		report, err = client.AdminProjectKeyUpdateEmail(context.Background(), projectKey, keyName)
	default:
		return cli.NewError("give both the project key and the key name, or none to update every default key")
	}
	if err != nil {
		return err
	}

	for _, s := range report.Updated {
		fmt.Printf("Updated %s/%s [%s]: %s -> %s\n", s.ProjectKey, s.KeyName, s.LongKeyID, strings.Join(s.Emails, ","), s.ExpectedEmail)
	}
	for _, f := range report.Failed {
		fmt.Printf("Failed %s/%s: %s\n", f.ProjectKey, f.KeyName, f.Error)
	}
	if len(report.MissingDefaultKey) > 0 {
		fmt.Printf("Projects without default PGP key (%d): %s\n", len(report.MissingDefaultKey), strings.Join(report.MissingDefaultKey, ", "))
	}
	fmt.Printf("Updated: %d, skipped: %d, failed: %d\n", len(report.Updated), report.Skipped, len(report.Failed))
	if len(report.Failed) > 0 {
		return cli.NewError("%d key(s) could not be updated", len(report.Failed))
	}
	return nil
}
