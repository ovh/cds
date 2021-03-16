package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminDatabaseCmd = cli.Command{
	Name:  "database",
	Short: "Manage CDS Database",
}

func adminDatabase() *cobra.Command {
	return cli.NewCommand(adminDatabaseCmd, nil, []*cobra.Command{
		cli.NewCommand(adminDatabaseUnlockCmd, adminDatabaseUnlockFunc, nil),
		cli.NewCommand(adminDatabaseDeleteMigrationCmd, adminDatabaseDeleteFunc, nil),
		cli.NewListCommand(adminDatabaseMigrationsList, adminDatabaseMigrationsListFunc, nil),
		cli.NewGetCommand(adminDatabaseSignatureResume, adminDatabaseSignatureResumeFunc, nil),
		cli.NewCommand(adminDatabaseSignatureRoll, adminDatabaseSignatureRollFunc, nil),
		cli.NewCommand(adminDatabaseEncryptionResume, adminDatabaseEncryptionResumeFunc, nil),
		cli.NewCommand(adminDatabaseEncryptionRoll, adminDatabaseEncryptionRollFunc, nil),
	})
}

var adminDatabaseUnlockCmd = cli.Command{
	Name:  "unlock",
	Short: "Unlock a pending migration (Use with caution)",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminDatabaseUnlockFunc(v cli.Values) error {
	return client.AdminDatabaseMigrationUnlock(v.GetString("id"))
}

var adminDatabaseDeleteMigrationCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a database migration from table gorp_migration (use with caution)",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminDatabaseDeleteFunc(v cli.Values) error {
	return client.AdminDatabaseMigrationDelete(v.GetString("id"))
}

var adminDatabaseMigrationsList = cli.Command{
	Name:  "list",
	Short: "List all CDS DB migrations",
}

func adminDatabaseMigrationsListFunc(_ cli.Values) (cli.ListResult, error) {
	migrations, err := client.AdminDatabaseMigrationsList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(migrations), nil
}

const argServiceName = "service"

var adminDatabaseSignatureResume = cli.Command{
	Name:  "list-signed-data",
	Short: "List all signed data in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
}

func adminDatabaseSignatureResumeFunc(args cli.Values) (interface{}, error) {
	return client.AdminDatabaseSignaturesResume(args.GetString(argServiceName))
}

var adminDatabaseSignatureRoll = cli.Command{
	Name:  "roll-signed-data",
	Short: "Roll a signed data in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name: "entity",
	},
}

func adminDatabaseSignatureRollFunc(args cli.Values) error {
	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		return client.AdminDatabaseSignaturesRollAllEntities(args.GetString(argServiceName))
	}

	for _, e := range entities {
		if err := client.AdminDatabaseSignaturesRollEntity(args.GetString(argServiceName), e); err != nil {
			return err
		}
	}

	return nil
}

var adminDatabaseEncryptionResume = cli.Command{
	Name:  "list-encrypted-data",
	Short: "List all encrypted data in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
}

func adminDatabaseEncryptionResumeFunc(args cli.Values) error {
	entities, err := client.AdminDatabaseListEncryptedEntities(args.GetString(argServiceName))
	for _, e := range entities {
		fmt.Println(e)
	}
	return err
}

var adminDatabaseEncryptionRoll = cli.Command{
	Name:  "roll-encrypted-data",
	Short: "Roll a encrypted data in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name: "entity",
	},
}

func adminDatabaseEncryptionRollFunc(args cli.Values) error {
	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		return client.AdminDatabaseRollAllEncryptedEntities(args.GetString(argServiceName))
	}

	for _, e := range entities {
		if err := client.AdminDatabaseRollEncryptedEntity(args.GetString(argServiceName), e); err != nil {
			return err
		}
	}
	return nil
}
