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
	type versionCDSCTL struct {
		Version      string `json:"version" yaml:"version"`
		Architecture string `json:"architecture" yaml:"architecture"`
		OS           string `json:"os" yaml:"os"`
		GitHash      string `json:"git_hash" yaml:"git_hash"`
		BuildTime    string `json:"build_time" yaml:"build_time"`
		Keychain     bool   `json:"keychain"`
	}

	vers := sdk.VersionCurrent()
	cdsctlVers := versionCDSCTL{
		Version:      vers.Version,
		Architecture: vers.Architecture,
		OS:           vers.OS,
		GitHash:      vers.GitHash,
		BuildTime:    vers.BuildTime,
		Keychain:     internal.IsKeychainEnabled(),
	}

	var versionAPI *sdk.Version
	if cfg.Host != "" {
		var err error
		versionAPI, err = client.Version()
		if err != nil {
			return fmt.Errorf("error while getting API version: %v", err)
		}
	}

	type allVersion struct {
		CDSCTLVersion versionCDSCTL `json:"cdsctl,omitempty" yaml:"cdsctl,omitempty"`
		APIVersion    *sdk.Version  `json:"api,omitempty" yaml:"api,omitempty"`
	}

	versions := allVersion{
		CDSCTLVersion: cdsctlVers,
		APIVersion:    versionAPI,
	}

	var buf []byte
	var err error
	switch format {
	case "json":
		buf, err = json.Marshal(versions)
	case "yaml":
		buf, err = yaml.Marshal(versions)
	default:
		buf, err = yaml.Marshal(versions)
	}
	if err != nil {
		return err
	}

	fmt.Println(string(buf))

	return nil
}
