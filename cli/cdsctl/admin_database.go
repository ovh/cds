package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
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
		cli.NewGetCommand(adminDatabaseEncryptionResume, adminDatabaseEncryptionResumeFunc, nil),
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

var adminDatabaseSignatureResume = cli.Command{
	Name:  "list-signed-data",
	Short: "List all signed data in database",
}

func adminDatabaseSignatureResumeFunc(_ cli.Values) (interface{}, error) {
	return client.AdminDatabaseSignaturesResume()
}

var adminDatabaseSignatureRoll = cli.Command{
	Name:  "roll-signed-data",
	Short: "Roll a signed data in database",
	VariadicArgs: cli.Arg{
		Name: "entity",
	},
}

func adminDatabaseSignatureRollFunc(args cli.Values) error {

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		return client.AdminDatabaseSignaturesRollAllEntities()
	}

	for _, e := range entities {
		if err := client.AdminDatabaseSignaturesRollEntity(e); err != nil {
			return err
		}
	}

	return nil

}

var adminDatabaseEncryptionResume = cli.Command{
	Name:  "list-encrypted-data",
	Short: "List all encrypted data in database",
}

func adminDatabaseEncryptionResumeFunc(_ cli.Values) (interface{}, error) {
	return client.AdminDatabaseListEncryptedEntities()
}

var adminDatabaseEncryptionRoll = cli.Command{
	Name:  "roll-encrypteddata",
	Short: "Roll a encrypted data in database",
	VariadicArgs: cli.Arg{
		Name: "entity",
	},
}

func adminDatabaseEncryptionRollFunc(args cli.Values) error {
	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		return client.AdminDatabaseRollAllEncryptedEntities()
	}

	for _, e := range entities {
		if err := client.AdminDatabaseRollEncryptedEntity(e); err != nil {
			return err
		}
	}

	return nil

}
