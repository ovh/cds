package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

const consumerIDArg = "consumer-id"

func consumer() *cobra.Command {
	cmd := cli.Command{
		Name:    "consumer",
		Aliases: []string{"consumers"},
		Short:   "Manage CDS auth consumers",
	}

	return cli.NewCommand(cmd, nil,
		cli.SubCommands{
			cli.NewListCommand(authConsumerListCmd, authConsumerListRun, nil),
			cli.NewCommand(authConsumerNewCmd, authConsumerNewRun, nil),
			cli.NewCommand(authConsumerDeleteCmd, authConsumerDeleteRun, nil),
			cli.NewCommand(authConsumerRegenCmd, authConsumerRegenRun, nil),
		},
	)
}

var authConsumerListCmd = cli.Command{
	Name:  "list",
	Short: "List your auth consumers for given user",
	OptionalArgs: []cli.Arg{
		{
			Name: "username",
		},
	},
	Flags: []cli.Flag{
		{
			Name:      "group",
			Type:      cli.FlagSlice,
			ShortHand: "g",
			Usage:     "filter by group",
		},
	},
}

func authConsumerListRun(v cli.Values) (cli.ListResult, error) {
	username := v.GetString("username")
	if username == "" {
		username = "me"
	}

	consumers, err := client.AuthConsumerListByUser(username)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(consumers), nil
}

var authConsumerNewCmd = cli.Command{
	Name:  "new",
	Short: "Create a new auth consumer for current user",
	OptionalArgs: []cli.Arg{
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
	},
}

func authConsumerNewRun(v cli.Values) error {
	username := v.GetString("username")
	if username == "" {
		username = "me"
	}

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
			return errors.Errorf("invalid given group name: '%s'", g)
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
			return errors.Errorf("invalid given scope value: '%s'", scope)
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

	res, err := client.AuthConsumerCreateForUser(username, sdk.AuthConsumer{
		Name:         name,
		Description:  description,
		GroupIDs:     groupIDs,
		ScopeDetails: sdk.NewAuthConsumerScopeDetails(scopes...),
	})
	if err != nil {
		return err
	}

	fmt.Println("Builtin consumer successfully created, use the following token to sign in:")
	fmt.Println(res.Token)

	return nil
}

var authConsumerDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete an auth consumer",
	OptionalArgs: []cli.Arg{
		{
			Name: "username",
		},
	},
	Args: []cli.Arg{
		{
			Name: consumerIDArg,
		},
	},
}

func authConsumerDeleteRun(v cli.Values) error {
	username := v.GetString("username")
	if username == "" {
		username = "me"
	}

	consumerID := v.GetString(consumerIDArg)
	if err := client.AuthConsumerDelete(username, consumerID); err != nil {
		return err
	}
	fmt.Printf("Consumer '%s' successfully deleted.\n", consumerID)

	return nil
}

var authConsumerRegenCmd = cli.Command{
	Name:    "regen",
	Aliases: []string{"regenerate"},
	Short:   "Regenerate an existing auth consumer",
	OptionalArgs: []cli.Arg{
		{
			Name: "username",
		},
	},
	Args: []cli.Arg{
		{
			Name: consumerIDArg,
		},
	},
}

func authConsumerRegenRun(v cli.Values) error {
	username := v.GetString("username")
	if username == "" {
		username = "me"
	}

	consumerID := v.GetString(consumerIDArg)
	consumer, err := client.AuthConsumerRegen(username, consumerID)
	if err != nil {
		return err
	}
	fmt.Printf("Consumer '%s' successfully regenerated.\n", consumerID)
	fmt.Printf("Token: %s\n", consumer.Token)

	return nil
}
