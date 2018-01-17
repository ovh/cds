package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	groupTokenCmd = cli.Command{
		Name:  "token",
		Short: "Manage CDS group token",
	}

	groupToken = cli.NewCommand(groupTokenCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(groupTokenListCmd, groupTokenListRun, nil),
			cli.NewGetCommand(groupTokenCreateCmd, groupTokenCreateRun, nil),
			cli.NewDeleteCommand(groupTokenDeleteCmd, groupTokenDeleteRun, nil),
		})
)

var groupTokenCreateCmd = cli.Command{
	Name:  "generate",
	Short: "Generate a new token",
	Long: `
		Useful to generate a new token when you use the cli or the api in scripts or for your worker, hatchery, uservices.
		The expiration must be [daily|persistent|session]
		Daily expirate after one day
		Persistent doesn't expirate until you revoke them
		Pay attention you must be an administrator of the group to launch this command
	`,
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "expiration"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "description"},
	},
}

func groupTokenCreateRun(v cli.Values) (interface{}, error) {
	token, err := client.GroupGenerateToken(v["groupname"], v["expiration"], v.GetString("description"))
	if err != nil {
		return nil, err
	}
	return *token, nil
}

var groupTokenDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a token linked to a group",
	Long: `
		Useful to delete a token from a group and so revoke it to unauthorize future connection
		Pay attention you must be an administrator of the group to launch this command
	`,
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "tokenId"},
	},
}

func groupTokenDeleteRun(v cli.Values) error {
	tokenID, err := v.GetInt64("tokenId")
	if err != nil {
		return fmt.Errorf("Token id is bad formatted")
	}

	if err := client.GroupDeleteToken(v["groupname"], tokenID); err != nil {
		return err
	}
	return nil
}

var groupTokenListCmd = cli.Command{
	Name:  "list",
	Short: "List tokens from group",
	Long: `
		You can list tokens linked to a groups to know the id of a token to delete it or know the creator of this token.
		Pay attention you must be an administrator of the group to launch this command
	`,
	Args: []cli.Arg{
		{Name: "groupname"},
	},
}

func groupTokenListRun(v cli.Values) (cli.ListResult, error) {
	tokens, err := client.GroupListToken(v["groupname"])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(tokens), nil
}
