package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	tokenCmd = cli.Command{
		Name:  "token",
		Short: "Manage CDS group token",
	}

	token = cli.NewCommand(tokenCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(tokenListCmd, tokenListRun, nil),
			cli.NewGetCommand(tokenCreateCmd, tokenCreateRun, nil),
			cli.NewGetCommand(tokenFindCmd, tokenFindRun, nil),
			cli.NewDeleteCommand(tokenDeleteCmd, tokenDeleteRun, nil),
		})
)

var tokenCreateCmd = cli.Command{
	Name:  "generate",
	Short: "Generate a new token",
	Long: `
Generate a new token when you use the cli or the api in scripts or for your worker, hatchery, uservices.

The expiration must be [daily|persistent|session].

Daily expirate after one day.

Persistent doesn't expirate until you revoke them.

Pay attention you must be an administrator of the group to launch this command.
	`,
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "expiration"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "description"},
	},
}

func tokenCreateRun(v cli.Values) (interface{}, error) {
	token, err := client.GroupGenerateToken(v["groupname"], v["expiration"], v.GetString("description"))
	if err != nil {
		return nil, err
	}
	return *token, nil
}

var tokenFindCmd = cli.Command{
	Name:  "find",
	Short: "Find an existing token",
	Long: `
Find an existing token with his value to have his description, creation date and the name of the creator.
	`,
	Example: `cdsctl token find "myTokenValue"`,
	Aliases: []string{"check", "describe"},
	Args: []cli.Arg{
		{Name: "token"},
	},
}

func tokenFindRun(v cli.Values) (interface{}, error) {
	token, err := client.FindToken(v["token"])
	if err != nil {
		return nil, err
	}
	return token, nil
}

var tokenDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a token linked to a group",
	Long: `
Delete a token from a group and so revoke it to unauthorize future connection.

Pay attention you must be an administrator of the group to launch this command.
	`,
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "tokenId"},
	},
}

func tokenDeleteRun(v cli.Values) error {
	tokenID, err := v.GetInt64("tokenId")
	if err != nil {
		return fmt.Errorf("Token id is bad formatted")
	}

	if err := client.GroupDeleteToken(v["groupname"], tokenID); err != nil {
		return err
	}
	return nil
}

var tokenListCmd = cli.Command{
	Name:  "list",
	Short: "List tokens from group",
	Long: `
You can list tokens linked to a groups to know the id of a token to delete it or know the creator of this token.

Pay attention, if you mention a group, you must be an administrator of the group to launch this command
	`,
	OptionalArgs: []cli.Arg{
		{Name: "groupname"},
	},
}

func tokenListRun(v cli.Values) (cli.ListResult, error) {
	if v.GetString("groupname") != "" {
		tokens, err := client.GroupListToken(v.GetString("groupname"))
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(tokens), nil
	}

	tokens, err := client.ListAllTokens()
	if err != nil {
		return nil, err
	}

	return cli.AsListResult(tokens), err
}
