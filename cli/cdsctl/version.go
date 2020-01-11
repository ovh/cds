package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/cli/cdsctl/internal"
	"github.com/ovh/cds/sdk"
)

var versionCmd = cli.Command{
	Name:  "version",
	Short: "show cdsctl version",
	Flags: []cli.Flag{
		{
			Type:  cli.FlagString,
			Name:  "format",
			Usage: "Specify out format (json or yaml)",
		},
	},
}

func version() *cobra.Command {
	return cli.NewCommand(versionCmd, versionRun, nil, cli.CommandWithoutExtraFlags)
}

func versionRun(v cli.Values) error {
	format := v.GetString("format")
	if format == "" {
		fmt.Println(sdk.VersionString())
		fmt.Printf("keychain support: %t\n", internal.IsKeychainEnabled())
		return nil
	}

	type versionWithKeychain struct {
		sdk.Version
		Keychain bool `json:"keychain"`
	}

	version := versionWithKeychain{
		Version:  sdk.VersionCurrent(),
		Keychain: internal.IsKeychainEnabled(),
	}

	var buf []byte
	var err error
	switch format {
	case "json":
		buf, err = json.Marshal(version)
	case "yaml":
		buf, err = yaml.Marshal(version)
	default:
		return fmt.Errorf("invalid given format")
	}
	if err != nil {
		return err
	}

	fmt.Println(string(buf))

	return nil
}
