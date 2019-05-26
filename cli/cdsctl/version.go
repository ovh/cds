package main

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
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
	var apiVersion *sdk.Version
	var err error
	if client != nil {
		apiVersion, err = client.Version()
	} else {
		err = errors.New("no configuration file found")
	}

	m := map[string]interface{}{
		"version":  sdk.VersionString(),
		"keychain": keychainEnabled,
	}

	if apiVersion != nil {
		m["api-version"] = apiVersion.Version
		m["api-url"] = client.APIURL()
	} else if err != nil {
		m["api-version"] = err.Error()
		m["api-url"] = "-"
	}

	var baseURL string
	configUser, err := client.ConfigUser()
	if err != nil {
		return err
	}

	if b, ok := configUser[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	if baseURL == "" {
		fmt.Println("Unable to retrieve UI URI")
		return nil
	}

	m["ui-url"] = baseURL

	format := v.GetString("format")
	if format == "" {
		fmt.Println(m["version"])
		fmt.Printf("CDS api version: %s\n", m["api-version"])
		fmt.Printf("CDS URL: %s\n", m["api-url"])
		fmt.Printf("CDS UI: %s\n", m["ui-url"])
		fmt.Printf("keychain support: %v\n", m["keychain"])
		return nil
	}

	var buf []byte

	switch format {
	case "json":
		buf, err = json.Marshal(m)
	case "yaml":
		buf, err = yaml.Marshal(m)
	default:
		return fmt.Errorf("invalid given format")
	}
	if err != nil {
		return err
	}

	fmt.Println(string(buf))

	return nil
}
