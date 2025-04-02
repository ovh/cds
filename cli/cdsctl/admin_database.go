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
	"github.com/ovh/cds/sdk/cdsclient"
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
		cli.NewListCommand(adminDatabaseEntityList, adminDatabaseEntityListFunc, nil),
		cli.NewCommand(adminDatabaseEntityInfo, adminDatabaseEntityInfoFunc, nil),
		cli.NewCommand(adminDatabaseEntityRoll, adminDatabaseEntityRollFunc, nil),
	})
}

const argServiceName = "service"
const flagBatchSize = "batch-size"

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
	migrations, err := client.AdminDatabaseMigrationList(args.GetString(argServiceName))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(migrations), nil
}

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
	res := make(map[string][]sdk.DatabaseCanonicalForm)

	es, err := client.AdminDatabaseEntityList(args.GetString(argServiceName))
	if err != nil {
		return nil, err
	}
	for _, r := range es {
		res[r.Name] = r.CanonicalForms
	}

	return res, nil
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
	Flags: []cli.Flag{
		{
			Name:    flagBatchSize,
			Default: "1",
			IsValid: func(value string) bool {
				v, err := strconv.Atoi(value)
				return err == nil && v >= 0
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
	batchSize, _ := args.GetInt64(flagBatchSize)
	noInteractive := args.GetBool("no-interactive")

	es, err := client.AdminDatabaseEntityList(service)
	if err != nil {
		return err
	}

	mEs := make(map[string]sdk.DatabaseEntity)
	for _, e := range es {
		mEs[e.Name] = e
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		for _, e := range es {
			if !e.Signed {
				continue
			}
			entities = append(entities, e.Name)
		}
	}

	sort.Strings(entities)
	for _, e := range entities {
		for _, c := range mEs[e].CanonicalForms {
			if c.Latest {
				continue
			}
			pks, err := client.AdminDatabaseEntity(service, e, cdsclient.Signer(c.Signer))
			if err != nil {
				return err
			}
			fmt.Printf("%s: roll signer: found %d tuple(s) to roll for signer %q\n", e, len(pks), c.Signer)
			if len(pks) == 0 {
				continue
			}

			if !noInteractive && !cli.AskConfirm(fmt.Sprintf("%s: roll signer: confirm rolling for %d tuple(s)", e, len(pks))) {
				return cli.NewError("operation aborted")
			}
			if err := adminDatabaseRollEntity(client, service, e, pks, int(batchSize), nil); err != nil {
				return err
			}
			fmt.Printf("%s: roll signer: (%d/%d)\n", e, len(pks), len(pks))
		}
	}

	return nil
}

var adminDatabaseEntityList = cli.Command{
	Name:  "list-entities",
	Short: "List all entitites in database",
	Args: []cli.Arg{
		{
			Name: argServiceName,
			IsValid: func(s string) bool {
				return s == sdk.TypeCDN || s == sdk.TypeAPI
			},
		},
	},
}

func adminDatabaseEntityListFunc(args cli.Values) (cli.ListResult, error) {
	res, err := client.AdminDatabaseEntityList(args.GetString(argServiceName))
	if err != nil {
		return nil, err
	}

	type DatabaseEntityCLI struct {
		Name      string `json:"name" cli:"name,key"`
		Encrypted bool   `json:"encrypted,omitempty" cli:"encrypted"`
		Signed    bool   `json:"signed,omitempty" cli:"signed"`
	}

	var es []DatabaseEntityCLI
	for _, i := range res {
		es = append(es, DatabaseEntityCLI{
			Name:      i.Name,
			Encrypted: i.Encrypted,
			Signed:    i.Signed,
		})
	}

	return cli.AsListResult(es), nil
}

var adminDatabaseEntityInfo = cli.Command{
	Name:  "info-entity",
	Short: "Get info for entity in database",
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
		{
			Name:    flagBatchSize,
			Default: "1",
			IsValid: func(value string) bool {
				v, err := strconv.Atoi(value)
				return err == nil && v >= 0
			},
		},
		{
			Name:    "no-cache",
			Default: "false",
			Type:    cli.FlagBool,
		},
	},
}

func adminDatabaseEntityInfoFunc(args cli.Values) error {
	service := args.GetString(argServiceName)

	batchSize, _ := args.GetInt64(flagBatchSize)
	noCache := args.GetBool("no-cache")

	dir := strings.TrimSpace(args.GetString("report-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return cli.WrapError(err, "unable to create directory: %s", args.GetString("output-dir"))
	}

	es, err := client.AdminDatabaseEntityList(service)
	if err != nil {
		return err
	}
	mEntites := make(map[string]sdk.DatabaseEntity)
	for _, e := range es {
		mEntites[e.Name] = e
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		for e := range mEntites {
			entities = append(entities, e)
		}
	} else {
		for _, e := range entities {
			if _, ok := mEntites[e]; !ok {
				return cli.WrapError(err, "invalid given entity name: %s", e)
			}
		}
	}
	sort.Strings(entities)

	for _, e := range entities {
		report := NewDatabaseEntityStorage(service, e)

		if !noCache {
			if err := report.Load(dir); err != nil {
				return err
			}
		}

		pks, err := client.AdminDatabaseEntity(service, e)
		if err != nil {
			return err
		}

		var filteredPks []string
		for _, pk := range pks {
			if i, ok := report.MInfo[pk]; ok && i.Signed == mEntites[e].Signed && i.Encrypted == mEntites[e].Encrypted {
				continue
			}
			filteredPks = append(filteredPks, pk)
		}

		if err := adminDatabaseInfoEntity(client, service, e, filteredPks, int(batchSize), func(is []sdk.DatabaseEntityInfo) error {
			for _, i := range is {
				report.MInfo[i.PK] = i
			}
			return report.Save(dir, true)
		}); err != nil {
			return err
		}

		fmt.Printf("%s: info: (%d/%d)\n", e, len(pks), len(pks))

		if err := report.Save(dir, false); err != nil {
			return err
		}
		report.PrintReport()
	}

	return nil
}

var adminDatabaseEntityRoll = cli.Command{
	Name:  "roll-entity",
	Short: "Roll signed and encrypted data for given entity in database that are using specific signature or encryption key by timestamp",
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
		{
			Name:    flagBatchSize,
			Default: "1",
			IsValid: func(value string) bool {
				v, err := strconv.Atoi(value)
				return err == nil && v >= 0
			},
		},
	},
}

func adminDatabaseEntityRollFunc(args cli.Values) error {
	service := args.GetString(argServiceName)
	timestamp, _ := args.GetInt64("timestamp")
	batchSize, _ := args.GetInt64(flagBatchSize)
	noInteractive := args.GetBool("no-interactive")

	dir := strings.TrimSpace(args.GetString("report-dir"))
	if dir == "" {
		dir = "."
	}

	es, err := client.AdminDatabaseEntityList(service)
	if err != nil {
		return err
	}
	mEntites := make(map[string]sdk.DatabaseEntity)
	for _, e := range es {
		mEntites[e.Name] = e
	}

	entities := args.GetStringSlice("entity")
	if len(entities) == 0 {
		for e := range mEntites {
			entities = append(entities, e)
		}
	} else {
		for _, e := range entities {
			if _, ok := mEntites[e]; !ok {
				return cli.WrapError(err, "invalid given entity name: %s", e)
			}
		}
	}
	sort.Strings(entities)

	for _, e := range entities {
		report := NewDatabaseEntityStorage(service, e)
		if err := report.Load(dir); err != nil {
			return err
		}

		pks := report.ComputePKsFromKeyTimestamp(timestamp)
		fmt.Printf("%s: roll: found %d tuple(s) to roll for given key timestamp\n", e, len(pks))
		if len(pks) == 0 {
			continue
		}

		if !noInteractive && !cli.AskConfirm(fmt.Sprintf("%s: roll: confirm rolling for %d tuple(s)", e, len(pks))) {
			return cli.NewError("operation aborted")
		}

		if err := adminDatabaseRollEntity(client, service, e, pks, int(batchSize), func(is []sdk.DatabaseEntityInfo) error {
			for _, i := range is {
				report.MInfo[i.PK] = i
			}
			return report.Save(dir, true)
		}); err != nil {
			return err
		}

		fmt.Printf("%s: roll: (%d/%d)\n", e, len(pks), len(pks))

		if err := report.Save(dir, false); err != nil {
			return err
		}
		report.PrintReport()
	}

	return nil
}

func adminDatabaseInfoEntity(client cdsclient.Interface, service string, entity string, pks []string, batchSize int, batchCallback func([]sdk.DatabaseEntityInfo) error) error {
	if batchSize <= 0 {
		batchSize = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var display = new(cli.Display)
	display.Printf("%s: getting info...\n", entity)
	display.Do(ctx)
	for i := 0; i < len(pks); i += batchSize {
		j := i + batchSize
		if j >= len(pks) {
			j = len(pks)
		}
		is, err := client.AdminDatabaseEntityInfo(service, entity, pks[i:j])
		if err != nil {
			return err
		}
		display.Printf("%s: getting info (%d/%d)...\n", entity, j, len(pks))
		if batchCallback != nil {
			if err := batchCallback(is); err != nil {
				return err
			}
		}
	}
	return nil
}

func adminDatabaseRollEntity(client cdsclient.Interface, service string, entity string, pks []string, batchSize int, batchCallback func([]sdk.DatabaseEntityInfo) error) error {
	if batchSize <= 0 {
		batchSize = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var display = new(cli.Display)
	display.Printf("%s: rolling...\n", entity)
	display.Do(ctx)
	for i := 0; i < len(pks); i += batchSize {
		j := i + batchSize
		if j >= len(pks) {
			j = len(pks)
		}
		is, err := client.AdminDatabaseEntityRoll(service, entity, pks[i:j])
		if err != nil {
			return err
		}
		display.Printf("%s: rolling (%d/%d)...\n", entity, j, len(pks))
		if batchCallback != nil {
			if err := batchCallback(is); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewDatabaseEntityStorage(service, entity string) *DatabaseEntityStorage {
	return &DatabaseEntityStorage{
		service: service,
		entity:  entity,
		MInfo:   make(map[string]sdk.DatabaseEntityInfo),
	}
}

type DatabaseEntityStorage struct {
	service string
	entity  string
	MInfo   map[string]sdk.DatabaseEntityInfo
}

func (d *DatabaseEntityStorage) Load(reportDir string) error {
	signatureReportPath := path.Join(reportDir, d.service+"."+d.entity+".signature.json")
	if _, err := os.Stat(signatureReportPath); err == nil {
		bs, err := os.ReadFile(signatureReportPath)
		if err != nil {
			return err
		}
		var report map[int64][]string
		if err := json.Unmarshal(bs, &report); err != nil {
			return err
		}
		for t, ks := range report {
			for _, k := range ks {
				if i, ok := d.MInfo[k]; !ok {
					d.MInfo[k] = sdk.DatabaseEntityInfo{
						PK:          k,
						Signed:      true,
						SignatureTS: t,
					}
				} else {
					i.Signed = true
					i.SignatureTS = t
					d.MInfo[k] = i
				}
			}
		}
		fmt.Printf("%s: load: signature report files loaded from %s\n", d.entity, signatureReportPath)
	}
	encryptionReportPath := path.Join(reportDir, d.service+"."+d.entity+".encryption.json")
	if _, err := os.Stat(encryptionReportPath); err == nil {
		bs, err := os.ReadFile(encryptionReportPath)
		if err != nil {
			return err
		}
		var report map[int64][]string
		if err := json.Unmarshal(bs, &report); err != nil {
			return err
		}
		for t, ks := range report {
			for _, k := range ks {
				if i, ok := d.MInfo[k]; !ok {
					d.MInfo[k] = sdk.DatabaseEntityInfo{
						PK:           k,
						Encrypted:    true,
						EncryptionTS: t,
					}
				} else {
					i.Encrypted = true
					i.EncryptionTS = t
					d.MInfo[k] = i
				}
			}
		}
		fmt.Printf("%s: load: encryption report files loaded from %s\n", d.entity, encryptionReportPath)
	}
	return nil
}

func (d *DatabaseEntityStorage) Save(reportDir string, silent bool) error {
	signatureReportPath := path.Join(reportDir, d.service+"."+d.entity+".signature.json")
	signatureReport := d.ComputeSignatureReport()
	bufSig, err := json.Marshal(signatureReport)
	if err != nil {
		return err
	}
	if err := os.WriteFile(signatureReportPath, bufSig, 0644); err != nil {
		return err
	}
	if !silent {
		fmt.Printf("%s: save: signature report files created at %s\n", d.entity, signatureReportPath)
	}
	encryptionReportPath := path.Join(reportDir, d.service+"."+d.entity+".encryption.json")
	encryptionReport := d.ComputeEncryptionReport()
	bufEnc, err := json.Marshal(encryptionReport)
	if err != nil {
		return err
	}
	if err := os.WriteFile(encryptionReportPath, bufEnc, 0644); err != nil {
		return err
	}
	if !silent {
		fmt.Printf("%s: save: encryption report files created at %s\n", d.entity, encryptionReportPath)
	}
	return nil
}

func (d *DatabaseEntityStorage) ComputeSignatureReport() map[int64][]string {
	report := make(map[int64][]string)
	for _, e := range d.MInfo {
		if e.Signed {
			if _, ok := report[e.SignatureTS]; !ok {
				report[e.SignatureTS] = nil
			}
			report[e.SignatureTS] = append(report[e.SignatureTS], e.PK)
		}
	}
	return report
}

func (d *DatabaseEntityStorage) ComputeEncryptionReport() map[int64][]string {
	report := make(map[int64][]string)
	for _, e := range d.MInfo {
		if e.Encrypted {
			if _, ok := report[e.EncryptionTS]; !ok {
				report[e.EncryptionTS] = nil
			}
			report[e.EncryptionTS] = append(report[e.EncryptionTS], e.PK)
		}
	}
	return report
}

func (d *DatabaseEntityStorage) ComputePKsFromKeyTimestamp(ts int64) []string {
	var res []string
	for _, e := range d.MInfo {
		if e.Encrypted && e.EncryptionTS == ts || e.Signed && e.SignatureTS == ts {
			res = append(res, e.PK)
		}
	}
	return res
}

func (d *DatabaseEntityStorage) PrintReport() {
	fmt.Printf("%s: report: found %d tuple(s)\n", d.entity, len(d.MInfo))
	signatureReport := d.ComputeSignatureReport()
	if len(signatureReport) > 0 {
		fmt.Printf("	%d signature key(s) found:\n", len(signatureReport))
		for ts, pks := range signatureReport {
			fmt.Printf("		%d: %d tuple(s)\n", ts, len(pks))
		}
	}
	encryptionReport := d.ComputeEncryptionReport()
	if len(encryptionReport) > 0 {
		fmt.Printf("	%d encryption key(s) found:\n", len(encryptionReport))
		for ts, pks := range encryptionReport {
			fmt.Printf("		%d: %d tuple(s)\n", ts, len(pks))
		}
	}
}
