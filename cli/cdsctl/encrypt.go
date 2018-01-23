package main

import (
	"fmt"

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
        value: registry.ovh.net/engine/http2kafka
    my-data:
        type: password
        value: 01234567890987654321
    EOF


Then, import then environment:

    cds environment import MYPROJECT your-environment.yml

Or push your workflow

	cds workflow push MYPROJECT *.yml
`,
	Example: `cdsctl encrypt MYPROJECT my-data my-super-secret-value
my-data: 01234567890987654321`,
	Ctx: []cli.Arg{
		{Name: "project-key"},
	},
	Args: []cli.Arg{
		{Name: "variable-name"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "secret-value"},
	},
}

var encrypt = cli.NewCommand(encryptCmd, encryptRun, nil, withAllCommandModifiers()...)

func encryptRun(v cli.Values) error {
	secretValue := v.GetString("secret-value")
	if secretValue == "" {
		secretValue = cli.ReadLine()
	}

	variable, err := client.VariableEncrypt(v.GetString("project-key"), v.GetString("variable-name"), secretValue)
	if err != nil {
		return err
	}

	fmt.Printf("%s: %s\n", variable.Name, variable.Value)
	return nil
}
