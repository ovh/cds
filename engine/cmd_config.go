package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	toml "github.com/yesnault/go-toml"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/hatchery/kubernetes"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/hatchery/vsphere"
	"github.com/ovh/cds/engine/hooks"
	"github.com/ovh/cds/engine/ui"
	"github.com/ovh/cds/engine/vcs"
	"github.com/ovh/cds/sdk"
)

func init() {
	configCmd.AddCommand(configNewCmd)
	configCmd.AddCommand(configCheckCmd)
	configCmd.AddCommand(configRegenCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInitTokenCmd)

	configNewCmd.Flags().BoolVar(&flagConfigNewAsEnv, "env", false, "Print configuration as environment variable")
	configRegenCmd.Flags().BoolVar(&flagConfigRegenAsEnv, "env", false, "Print configuration as environment variable")
	configEditCmd.Flags().BoolVar(&flagConfigRegenAsEnv, "env", false, "Print configuration as environment variable")

	configInitTokenCmd.Flags().StringVar(&flagInitTokenConfigFile, "config", "", "config file")
	configInitTokenCmd.Flags().StringVar(&flagInitTokenRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	configInitTokenCmd.Flags().StringVar(&flagInitTokenRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
	configInitTokenCmd.Flags().StringVar(&flagInitTokenVaultAddr, "vault-addr", "", "(optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)")
	configInitTokenCmd.Flags().StringVar(&flagInitTokenVaultToken, "vault-token", "", "(optional) Vault token to fetch secrets from vault")
}

var (
	flagConfigNewAsEnv           bool
	flagConfigRegenAsEnv         bool
	flagInitTokenConfigFile      string
	flagInitTokenRemoteConfig    string
	flagInitTokenRemoteConfigKey string
	flagInitTokenVaultAddr       string
	flagInitTokenVaultToken      string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CDS Configuration",
}

var configNewCmd = &cobra.Command{
	Use:   "new",
	Short: "CDS configuration file assistant",
	Long: `
Generate the whole configuration file
	$ engine config new > conf.toml

you can compose your file configuration
this will generate a file configuration containing
api and hatchery:local µService
	$ engine config new api hatchery:local

For advanced usage, Debug and Tracing section can be generated as:
	$ engine config new debug tracing [µService(s)...]

All options
	$ engine config new [debug] [tracing] [api] [hatchery:local] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate]

`,

	Run: func(cmd *cobra.Command, args []string) {
		conf := configBootstrap(args)
		initToken, err := configSetStartupData(&conf)
		if err != nil {
			sdk.Exit("%v", err)
		}

		if !flagConfigNewAsEnv {
			btes, err := toml.Marshal(conf)
			if err != nil {
				sdk.Exit("%v", err)
			}
			fmt.Println(string(btes))
		} else {
			configPrintToEnv(conf, os.Stdout)
		}

		fmt.Println("# On first login, you will be asked to enter the following token:")
		fmt.Println("# " + initToken)
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check CDS configuration file",
	Long:  `$ engine config check <path>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.Help() // nolint
			sdk.Exit("Wrong usage")
		}

		// Initialize config from given path
		conf := configImport(nil, args[0], "", "", "", "", false)

		var hasError bool
		if conf.API != nil && conf.API.URL.API != "" {
			fmt.Printf("checking api configuration...\n")
			if err := api.New().CheckConfiguration(*conf.API); err != nil {
				fmt.Printf("api Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.UI != nil && conf.UI.API.HTTP.URL != "" {
			fmt.Printf("checking UI configuration...\n")
			if err := ui.New().CheckConfiguration(*conf.UI); err != nil {
				fmt.Printf("ui Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.DatabaseMigrate != nil && conf.DatabaseMigrate.API.HTTP.URL != "" {
			fmt.Printf("checking migrate configuration...\n")
			if err := api.New().CheckConfiguration(*conf.DatabaseMigrate); err != nil {
				fmt.Printf("migrate Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Local != nil && conf.Hatchery.Local.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:local configuration...\n")
			if err := local.New().CheckConfiguration(*conf.Hatchery.Local); err != nil {
				fmt.Printf("hatchery:local Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Marathon != nil && conf.Hatchery.Marathon.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:marathon configuration...\n")
			if err := marathon.New().CheckConfiguration(*conf.Hatchery.Marathon); err != nil {
				fmt.Printf("hatchery:marathon Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Openstack != nil && conf.Hatchery.Openstack.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:openstack configuration...\n")
			if err := openstack.New().CheckConfiguration(*conf.Hatchery.Openstack); err != nil {
				fmt.Printf("hatchery:openstack Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Kubernetes != nil && conf.Hatchery.Kubernetes.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:kubernetes configuration...\n")
			if err := kubernetes.New().CheckConfiguration(*conf.Hatchery.Kubernetes); err != nil {
				fmt.Printf("hatchery:kubernetes Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.Swarm != nil && conf.Hatchery.Swarm.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:swarm configuration...\n")
			if err := swarm.New().CheckConfiguration(*conf.Hatchery.Swarm); err != nil {
				fmt.Printf("hatchery:swarm Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hatchery != nil && conf.Hatchery.VSphere != nil && conf.Hatchery.VSphere.API.HTTP.URL != "" {
			fmt.Printf("checking hatchery:vsphere configuration...\n")
			if err := vsphere.New().CheckConfiguration(*conf.Hatchery.VSphere); err != nil {
				fmt.Printf("hatchery:vsphere Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.VCS != nil && conf.VCS.API.HTTP.URL != "" {
			fmt.Printf("checking vcs configuration...\n")
			if err := vcs.New().CheckConfiguration(*conf.VCS); err != nil {
				fmt.Printf("vcs Configuration: %v\n", err)
				hasError = true
			}
		}

		if conf.Hooks != nil && conf.Hooks.API.HTTP.URL != "" {
			fmt.Printf("checking hooks configuration...\n")
			if err := hooks.New().CheckConfiguration(*conf.Hooks); err != nil {
				fmt.Printf("hooks Configuration: %v\n", err)
				hasError = true
			}
		}

		if !hasError {
			fmt.Println("Configuration file OK")
		}
	},
}

var configRegenCmd = &cobra.Command{
	Use:   "regen",
	Short: "Regen tokens and keys for given CDS configuration file",
	Long:  `$ engine config regen <input-path> <output-path>`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help() // nolint
			sdk.Exit("Wrong usage")
		}

		oldConf := configImport(nil, args[0], "", "", "", "", true)

		initToken, err := configSetStartupData(&oldConf)
		if err != nil {
			sdk.Exit("%v", err)
		}

		writer := os.Stdout
		if len(args) == 2 {
			output := args[1]
			if _, err := os.Stat(output); err == nil {
				if err := os.Remove(output); err != nil {
					sdk.Exit("%v", err)
				}
			}
			writer, err = os.Create(output)
			if err != nil {
				sdk.Exit("%v", err)
			}
		}
		defer writer.Close()

		if !flagConfigRegenAsEnv {
			btes, err := toml.Marshal(oldConf)
			if err != nil {
				sdk.Exit("%v", err)
			}
			fmt.Fprintln(writer, string(btes))
		} else {
			configPrintToEnv(oldConf, writer)
		}

		fmt.Fprintln(writer, "# On first login, you will be asked to enter the following token:")
		fmt.Fprintln(writer, "# "+initToken)
	},
}

var configEditCmd = &cobra.Command{
	Use:     "edit",
	Short:   "Edit a CDS configuration file",
	Long:    `$ engine config edit <path-toml-file> key=value key=value`,
	Example: `$ engine config edit conf.toml log.level=debug hatchery.swarm.commonConfiguration.name=hatchery-swarm-name`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Help()
			sdk.Exit("Wrong usage")
		}

		cfgFile := args[0]

		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			sdk.Exit("File %s doesn't exist", cfgFile)
		}

		btes, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			sdk.Exit("Error while read content of file %s - err:%v", cfgFile, err)
		}

		tomlConf, err := toml.Load(string(btes))
		if err != nil {
			sdk.Exit("Error while load toml content of file %s - err:%v", cfgFile, err)
		}

		for _, vk := range args[1:] {
			t := strings.Split(vk, "=")
			if len(t) != 2 {
				sdk.Exit("Invalid key=value: %v", vk)
			}
			// check if value is bool, float, int or else string
			if v, err := strconv.ParseBool(t[1]); err == nil {
				tomlConf.Set(t[0], "", false, "", v)
			} else if v, err := strconv.ParseInt(t[1], 10, 64); err == nil {
				tomlConf.Set(t[0], "", false, "", v)
			} else {
				tomlConf.Set(t[0], "", false, "", t[1])
			}
		}

		tmpFile := "cds.tmp.toml"
		if err := ioutil.WriteFile(tmpFile, []byte(tomlConf.String()), os.FileMode(0640)); err != nil {
			sdk.Exit("Error while create tempfile: %v", err)
		}

		defer os.Remove(tmpFile)

		viper.SetConfigFile(tmpFile)
		if err := viper.ReadInConfig(); err != nil {
			sdk.Exit(err.Error())
		}

		var conf Configuration
		if err := viper.Unmarshal(&conf); err != nil {
			sdk.Exit("Unable to parse config: %v", err.Error())
		}

		btesOutput, err := toml.Marshal(conf)
		if err != nil {
			sdk.Exit("%v", err)
		}
		fmt.Println(string(btesOutput))

	},
}

var configInitTokenCmd = &cobra.Command{
	Use:   "init-token",
	Short: "Generate/Regenerate an init token for a given CDS configuration file",
	Long:  `$ engine config init-token`,
	Run: func(cmd *cobra.Command, args []string) {

		conf := configImport(args, flagInitTokenConfigFile, flagInitTokenRemoteConfig, flagInitTokenRemoteConfigKey, flagInitTokenVaultAddr, flagInitTokenVaultToken, false)

		initToken, err := getInitTokenFromExistingConfiguration(conf)
		if err != nil {
			sdk.Exit("error: %v", err)
		}

		fmt.Println(initToken)
	},
}