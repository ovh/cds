package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

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
		cli.NewCommand(adminDatabaseSignatureRollSigner, adminDatabaseSignatureRollSignerFunc, nil),
		cli.NewCommand(adminDatabaseSignatureRoll, adminDatabaseSignatureRollFunc, nil),
		cli.NewCommand(adminDatabaseSignatureInfo, adminDatabaseSignatureInfoFunc, nil),
		cli.NewCommand(adminDatabaseEncryptionResume, adminDatabaseEncryptionResumeFunc, nil),
		cli.NewCommand(adminDatabaseEncryptionRoll, adminDatabaseEncryptionRollFunc, nil),
		cli.NewCommand(adminDatabaseEncryptionInfo, adminDatabaseEncryptionInfoFunc, nil),
	})
}

var adminDatabaseUnlockCmd = cli.Command{
	Name:  "unlock",
	Short: "Unlock a pending migration (Use with caution)",
	Example: `
$ cdsctl admin database unlock api id-to-unlock
$ cdsctl admin database unlock cdn id-to-unlock
	`,
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
		{Name: "id"},
	},
}

func adminDatabaseUnlockFunc(args cli.Values) error {
	return client.AdminDatabaseMigrationUnlock(args.GetString(argServiceName), args.GetString("id"))
}

var adminDatabaseDeleteMigrationCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a database migration from table gorp_migration (use with caution)",
	Example: `
$ cdsctl admin database delete api id-migration-to-delete
$ cdsctl admin database delete cdn id-migration-to-delete
	`,
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
		{Name: "id"},
	},
}

func adminDatabaseDeleteFunc(args cli.Values) error {
	return client.AdminDatabaseMigrationDelete(args.GetString(argServiceName), args.GetString("id"))
}

var adminDatabaseMigrationsList = cli.Command{
	Name:  "list",
	Short: "List all CDS DB migrations",
	Example: `
$ cdsctl admin database list api
$ cdsctl admin database list cdn
	`,
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
}

func adminDatabaseMigrationsListFunc(args cli.Values) (cli.ListResult, error) {
	migrations, err := client.AdminDatabaseMigrationsList(args.GetString(argServiceName))
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

var adminDatabaseSignatureRollSigner = cli.Command{
	Name:  "roll-signed-data-signer",
	Short: "Roll signed data in database that are not using the latest signer",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name:       "entity",
		AllowEmpty: true,
	},
}

func adminDatabaseSignatureRollSignerFunc(args cli.Values) error {
	service := args.GetString(argServiceName)

	resume, err := client.AdminDatabaseSignaturesResume(service)
	if err != nil {
		return err
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		for e := range resume {
			entities = append(entities, e)
		}
	}

	sort.Strings(entities)
	for _, e := range entities {
		for _, signer := range resume[e] {
			if signer.Latest {
				continue
			}
			pks, err := client.AdminDatabaseSignaturesTuplesBySigner(service, e, signer.Signer)
			if err != nil {
				return err
			}
			if err := client.AdminDatabaseRollSignedEntity(service, e, pks); err != nil {
				return err
			}
		}
	}

	return nil
}

var adminDatabaseSignatureRoll = cli.Command{
	Name:  "roll-signed-data",
	Short: "Roll signed data in database that are using specific signature key by timestamp",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
		{
			Name: "timestamp",
			IsValid: func(s string) bool {
				_, err := strconv.Atoi(s)
				return err == nil
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name:       "entity",
		AllowEmpty: true,
	},
	Flags: []cli.Flag{
		{
			Name:    "report-dir",
			Usage:   "Path to load report for entities",
			Default: "./report",
		},
	},
}

func adminDatabaseSignatureRollFunc(args cli.Values) error {
	service := args.GetString(argServiceName)
	timestamp, _ := args.GetInt64("timestamp")

	dir := strings.TrimSpace(args.GetString("report-dir"))
	if dir == "" {
		dir = "."
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		resume, err := client.AdminDatabaseSignaturesResume(service)
		if err != nil {
			return err
		}
		for e := range resume {
			entities = append(entities, e)
		}
	}

	sort.Strings(entities)
	for _, e := range entities {
		reportPath := path.Join(dir, service+"."+e+".signature.json")
		buf, err := os.ReadFile(reportPath)
		if err != nil {
			return err
		}
		var report map[int64][]string
		if err := json.Unmarshal(buf, &report); err != nil {
			return err
		}
		if _, ok := report[timestamp]; !ok {
			continue
		}
		if err := client.AdminDatabaseRollSignedEntity(service, e, report[timestamp]); err != nil {
			return err
		}
	}

	return nil
}

var adminDatabaseSignatureInfo = cli.Command{
	Name:  "info-signed-data",
	Short: "Get info for signed data in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name:       "entity",
		AllowEmpty: true,
	},
	Flags: []cli.Flag{
		{
			Name:    "report-dir",
			Usage:   "Path to save report for entities",
			Default: "./report",
		},
	},
}

func adminDatabaseSignatureInfoFunc(args cli.Values) error {
	service := args.GetString(argServiceName)

	dir := strings.TrimSpace(args.GetString("report-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return cli.WrapError(err, "Unable to create directory %s", args.GetString("output-dir"))
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		resume, err := client.AdminDatabaseSignaturesResume(service)
		if err != nil {
			return err
		}
		for e := range resume {
			entities = append(entities, e)
		}
	}

	sort.Strings(entities)
	for _, e := range entities {
		pks, err := client.AdminDatabaseListTuples(service, e)
		if err != nil {
			return err
		}
		reportPath := path.Join(dir, service+"."+e+".signature.json")
		existingReport := make(map[string]int64)
		if _, err := os.Stat(reportPath); err == nil {
			bs, err := os.ReadFile(reportPath)
			if err != nil {
				return err
			}
			var report map[int64][]string
			if err := json.Unmarshal(bs, &report); err != nil {
				return err
			}
			for t, ks := range report {
				for _, k := range ks {
					existingReport[k] = t
				}
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		var display = new(cli.Display)
		display.Printf("Getting info %v...", e)
		display.Do(ctx)

		report := make(map[int64][]string)
		for i, pk := range pks {
			display.Printf("Getting info %v (%d/%d)...", e, i+1, len(pks))
			var keyTimestamp int64
			if t, ok := existingReport[pk]; ok {
				keyTimestamp = t
			} else {
				keyTimestamp, err = client.AdminDatabaseInfoSignedEntity(service, e, pk)
				if err != nil {
					cancel()
					return err
				}
			}
			if _, ok := report[keyTimestamp]; !ok {
				report[keyTimestamp] = nil
			}
			report[keyTimestamp] = append(report[keyTimestamp], pk)
		}
		display.Printf("Getting info %v (%d/%d) - DONE\n", e, len(pks), len(pks))
		cancel()

		fmt.Printf("Entity %s: found %d signature keys used\n", e, len(report))
		for ts, pks := range report {
			fmt.Printf("	%d: %d tuple(s)\n", ts, len(pks))
		}
		buf, err := json.Marshal(report)
		if err != nil {
			return err
		}
		if err := os.WriteFile(reportPath, buf, 0644); err != nil {
			return err
		}
		fmt.Printf("Report file created at %s\n", reportPath)
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
	if err != nil {
		return err
	}
	sort.Strings(entities)
	for _, e := range entities {
		fmt.Println(e)
	}
	return err
}

var adminDatabaseEncryptionRoll = cli.Command{
	Name:  "roll-encrypted-data",
	Short: "Roll encrypted data in database that are using specific encryption key by timestamp",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
		{
			Name: "timestamp",
			IsValid: func(s string) bool {
				_, err := strconv.Atoi(s)
				return err == nil
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name:       "entity",
		AllowEmpty: true,
	},
	Flags: []cli.Flag{
		{
			Name:    "report-dir",
			Usage:   "Path to load report for entities",
			Default: "./report",
		},
	},
}

func adminDatabaseEncryptionRollFunc(args cli.Values) error {
	service := args.GetString(argServiceName)
	timestamp, _ := args.GetInt64("timestamp")

	dir := strings.TrimSpace(args.GetString("report-dir"))
	if dir == "" {
		dir = "."
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		var err error
		entities, err = client.AdminDatabaseListEncryptedEntities(service)
		if err != nil {
			return err
		}
	}

	sort.Strings(entities)
	for _, e := range entities {
		reportPath := path.Join(dir, service+"."+e+".encryption.json")
		buf, err := os.ReadFile(reportPath)
		if err != nil {
			return err
		}
		var report map[int64][]string
		if err := json.Unmarshal(buf, &report); err != nil {
			return err
		}
		if _, ok := report[timestamp]; !ok {
			continue
		}
		if err := client.AdminDatabaseRollEncryptedEntity(service, e, report[timestamp]); err != nil {
			return err
		}
	}

	return nil
}

var adminDatabaseEncryptionInfo = cli.Command{
	Name:  "info-encrypted-data",
	Short: "Get info for encrypted data in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
	VariadicArgs: cli.Arg{
		Name:       "entity",
		AllowEmpty: true,
	},
	Flags: []cli.Flag{
		{
			Name:    "report-dir",
			Usage:   "Path to save report for entities",
			Default: "./report",
		},
	},
}

func adminDatabaseEncryptionInfoFunc(args cli.Values) error {
	service := args.GetString(argServiceName)

	dir := strings.TrimSpace(args.GetString("report-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return cli.WrapError(err, "Unable to create directory %s", args.GetString("output-dir"))
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		var err error
		entities, err = client.AdminDatabaseListEncryptedEntities(service)
		if err != nil {
			return err
		}
	}

	sort.Strings(entities)
	for _, e := range entities {
		pks, err := client.AdminDatabaseListTuples(service, e)
		if err != nil {
			return err
		}
		reportPath := path.Join(dir, service+"."+e+".encryption.json")
		existingReport := make(map[string]int64)
		if _, err := os.Stat(reportPath); err == nil {
			bs, err := os.ReadFile(reportPath)
			if err != nil {
				return err
			}
			var report map[int64][]string
			if err := json.Unmarshal(bs, &report); err != nil {
				return err
			}
			for t, ks := range report {
				for _, k := range ks {
					existingReport[k] = t
				}
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		var display = new(cli.Display)
		display.Printf("Getting info %v...", e)
		display.Do(ctx)

		report := make(map[int64][]string)
		for i, pk := range pks {
			display.Printf("Getting info %v (%d/%d)...", e, i+1, len(pks))
			var keyTimestamp int64
			if t, ok := existingReport[pk]; ok {
				keyTimestamp = t
			} else {
				keyTimestamp, err = client.AdminDatabaseInfoEncryptedEntity(service, e, pk)
				if err != nil {
					cancel()
					return err
				}
			}
			if _, ok := report[keyTimestamp]; !ok {
				report[keyTimestamp] = nil
			}
			report[keyTimestamp] = append(report[keyTimestamp], pk)
		}
		display.Printf("Getting info %v (%d/%d) - DONE\n", e, len(pks), len(pks))
		cancel()

		fmt.Printf("Entity %s: found %d encryption keys used\n", e, len(report))
		for ts, pks := range report {
			fmt.Printf("	%d: %d tuple(s)\n", ts, len(pks))
		}
		buf, err := json.Marshal(report)
		if err != nil {
			return err
		}
		if err := os.WriteFile(reportPath, buf, 0644); err != nil {
			return err
		}
		fmt.Printf("Report file created at %s\n", reportPath)
	}

	return nil
}
