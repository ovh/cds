package main

import (
	"fmt"

	"github.com/ovh/cds/cli"
)

var (
	encryptCmd = cli.Command{
		Name:  "encrypt",
		Short: "Encrypt variable into your CDS project",
		Long:  "To be able to write secret in the CDS yaml files, you have encrypt data with the project GPG key",
		Example: `cdsctl encrypt MYPROJECT my-data my-super-secret-value
mydata: 01234567890987654321`,
		Args: []cli.Arg{
			{Name: "project-key"},
			{Name: "variable-name"},
		},
		OptionalArgs: []cli.Arg{
			{Name: "secret-value"},
		},
	}

	encrypt = cli.NewCommand(encryptCmd, encryptRun, nil)
)

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
