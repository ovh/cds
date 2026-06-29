package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminUserConsumerCmd = cli.Command{
	Name:    "consumer",
	Aliases: []string{"consumers"},
	Short:   "Manage auth consumers of a CDS user (admin only)",
}

func adminUserConsumer() *cobra.Command {
	return cli.NewCommand(adminUserConsumerCmd, nil, []*cobra.Command{
		cli.NewCommand(adminUserConsumerNewCmd, adminUserConsumerNewRun, nil),
	})
}

var adminUserConsumerNewCmd = cli.Command{
	Name:  "new",
	Short: "Create a new builtin auth consumer for a given user",
	Long:  "As an admin, create a builtin auth consumer (token) on behalf of another user, typically a service/technical account that can't sign in interactively.",
	Args: []cli.Arg{
		{
			Name: "username",
		},
	},
	Flags: []cli.Flag{
		{
			Name:  "name",
			Usage: "What is the name of this consumer",
		},
		{
			Name:  "description",
			Usage: "What is the purpose of this consumer",
		},
		{
			Name:  "groups",
			Type:  cli.FlagSlice,
			Usage: "Define the list of groups for the consumer",
		},
		{
			Name:  "scopes",
			Type:  cli.FlagSlice,
			Usage: "Define the list of scopes for the consumer",
		},
		{
			Name:  "duration",
			Usage: "Validity period of the token generated for the consumer (in days)",
		},
	},
}

func adminUserConsumerNewRun(v cli.Values) error {
	username := v.GetString("username")

	name := v.GetString("name")
	if name == "" && !v.GetBool("no-interactive") {
		name = cli.AskValue("Name")
	}

	description := v.GetString("description")
	if description == "" && !v.GetBool("no-interactive") {
		description = cli.AskValue("Description")
	}

	allGroups, err := client.GroupList()
	if err != nil {
		return err
	}
	var groupIDs []int64
	for _, g := range v.GetStringSlice("groups") {
		var found bool
		for j := range allGroups {
			if g == allGroups[j].Name {
				groupIDs = append(groupIDs, allGroups[j].ID)
				found = true
				break
			}
		}
		if !found {
			return errors.Errorf("invalid given group name: %q", g)
		}
	}
	if len(groupIDs) == 0 && !v.GetBool("no-interactive") {
		opts := make([]string, len(allGroups))
		for i := range allGroups {
			opts[i] = allGroups[i].Name
		}
		choices := cli.AskSelect("Select groups availables for the new consumer", opts...)
		for _, choice := range choices {
			groupIDs = append(groupIDs, allGroups[choice].ID)
		}
	}

	var scopes []sdk.AuthConsumerScope
	for _, s := range v.GetStringSlice("scopes") {
		scope := sdk.AuthConsumerScope(s)
		if !scope.IsValid() {
			return errors.Errorf("invalid given scope value: %q", scope)
		}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 && !v.GetBool("no-interactive") {
		opts := make([]string, len(sdk.AuthConsumerScopes))
		for i := range sdk.AuthConsumerScopes {
			opts[i] = string(sdk.AuthConsumerScopes[i])
		}
		choices := cli.AskSelect("Select scopes availables for the new consumer", opts...)
		for _, choice := range choices {
			scopes = append(scopes, sdk.AuthConsumerScopes[choice])
		}
	}

	var duration time.Duration
	if v.GetString("duration") != "" {
		iDuration, err := v.GetInt64("duration")
		if err != nil {
			return errors.Errorf("invalid given duration: %q", v.GetString("duration"))
		}
		duration = time.Duration(iDuration) * (24 * time.Hour)
	}

	consumer := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            name,
			Description:     description,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), duration),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     groupIDs,
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(scopes...),
		},
	}

	res, err := client.AuthConsumerCreateForUserAsAdmin(username, consumer)
	if err != nil {
		return err
	}

	fmt.Printf("Builtin consumer successfully created for user %q, use the following token to sign in:\n", username)
	fmt.Println(res.Token)

	return nil
}
