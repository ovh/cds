package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var encryptCmd = cli.Command{
	Name:  "encrypt",
	Short: "Encrypt variable into your CDS project",
	Long: `To be able to write secret in the CDS yaml files, you have to encrypt data with the project GPG key.

Create a secret variable:


    $ cdsctl encrypt MYPROJECT my-data my-super-secret-value
    my-data: 01234567890987654321

The command returns the value: 01234567890987654321. You can use this value in a configuration file.

Example of use case: Import an environment with a secret.

Create an environment file to import :

    $ cat << EOF > your-environment.yml
    name: your-environment
    values:
    a-readable-variable:
        type: string
        value: value
    my-data:
        type: password
        value: 01234567890987654321
    EOF


Then, import then environment:

    cdsctl environment import MYPROJECT your-environment.yml

Or push your workflow

	cdsctl workflow push MYPROJECT *.yml
`,
	Example: `cdsctl encrypt MYPROJECT my-data my-super-secret-value
my-data: 01234567890987654321`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "secret-value"},
	},
}

func encrypt() *cobra.Command {
	return cli.NewCommand(encryptCmd, encryptRun, cli.SubCommands{
		encryptList(), encryptDelete(),
	}, withAllCommandModifiers()...)
}

func encryptRun(v cli.Values) error {
	secretValue := v.GetString("secret-value")
	if secretValue == "" {
		secretValue = cli.ReadLine()
	}

	variable, err := client.VariableEncrypt(v.GetString(_ProjectKey), v.GetString("variable-name"), secretValue)
	if err != nil {
		return err
	}

	fmt.Printf("%s: %s\n", variable.Name, variable.Value)
	return nil
}

var encryptListCmd = cli.Command{
	Name:  "list",
	Short: "List all the encrypted variable of your CDS project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func encryptList() *cobra.Command {
	return cli.NewListCommand(encryptListCmd, encryptListRun, nil, withAllCommandModifiers()...)
}

func encryptListRun(v cli.Values) (cli.ListResult, error) {
	secrets, err := client.VariableListEncrypt(v.GetString(_ProjectKey))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(secrets), nil
}

var encryptDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete the given encrypted variable of your CDS project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func encryptDelete() *cobra.Command {
	return cli.NewDeleteCommand(encryptDeleteCmd, encryptDeleteRun, nil, withAllCommandModifiers()...)
}

func encryptDeleteRun(v cli.Values) error {
	return client.VariableEncryptDelete(v.GetString(_ProjectKey), v.GetString("name"))
}
