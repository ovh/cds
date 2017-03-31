package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ovh/cds/engine/hatchery/docker"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/marathon"
	"github.com/ovh/cds/engine/hatchery/openstack"
	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "hatchery",
	Short: "hatchery <mode> --api=<cds.domain> --token=<token>",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Initialize()
		sdk.SetAgent(sdk.HatcheryAgent)

		if viper.GetInt("max-worker") < 1 {
			sdk.Exit("max-worker have to be > 0\n")
		}

		if viper.GetInt("provision") < 0 {
			sdk.Exit("provision have to be >= 0\n")
		}

		if viper.GetString("api") == "" {
			sdk.Exit("CDS api endpoint not provided. See help on flag --api\n")
		}

		if viper.GetString("token") == "" {
			sdk.Exit("Worker token not provided. See help on flag --token\n")
		}
	},
}

func main() {
	addFlags()
	addCommands()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func addCommands() {
	rootCmd.AddCommand(local.Cmd)
	rootCmd.AddCommand(docker.Cmd)
	rootCmd.AddCommand(marathon.Cmd)
	rootCmd.AddCommand(swarm.Cmd)
	rootCmd.AddCommand(openstack.Cmd)
}

// Cannot rely on viper.AutomaticEnv here because of the presence of hyphen '-'
// in the flag (ie. for openstack cmd).
// AutomaticEnv will only lookup for a strings.ToUpper
// which will not replace hypen '-' with underscore '_'.
// i.e "openstack-tenant" -> "CDS_OPENSTACK-TENANT" instead of "CDS_OPENSTACK_TENANT"
// Hence SetEnvKeyReplacer.
// Sadly, this isn't used by Flags().*Var functions, so no automatic binding on variables.
func addFlags() {
	viper.SetEnvPrefix("cds")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	rootCmd.PersistentFlags().String("log-level", "notice", "Log Level: debug, info, warning, notice, critical")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	rootCmd.PersistentFlags().String("api", "", "CDS api endpoint")
	viper.BindPFlag("api", rootCmd.PersistentFlags().Lookup("api"))

	rootCmd.PersistentFlags().String("token", "", "CDS token")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))

	rootCmd.PersistentFlags().Int("request-api-timeout", 10, "Request CDS API: timeout in seconds")
	viper.BindPFlag("request-api-timeout", rootCmd.PersistentFlags().Lookup("request-api-timeout"))

	rootCmd.PersistentFlags().Int("provision", 0, "Allowed worker model provisioning")
	viper.BindPFlag("provision", rootCmd.PersistentFlags().Lookup("provision"))

	rootCmd.PersistentFlags().Int("provision-seconds", 30, "Check provisioning each n Seconds")
	viper.BindPFlag("provision-seconds", rootCmd.PersistentFlags().Lookup("provision-seconds"))

	rootCmd.PersistentFlags().Int("max-worker", 10, "Maximum allowed simultaenous workers")
	viper.BindPFlag("max-worker", rootCmd.PersistentFlags().Lookup("max-worker"))

	rootCmd.PersistentFlags().Int("max-failures-heartbeat", 10, "Maximum allowed consecutives failures on heatbeat routine")
	viper.BindPFlag("max-failures-heartbeat", rootCmd.PersistentFlags().Lookup("max-failures-heartbeat"))

	rootCmd.PersistentFlags().BoolP("insecure", "k", false, `(SSL) This option explicitly allows hatchery to perform "insecure" SSL connections on CDS API.`)
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))

	rootCmd.PersistentFlags().String("name", "", "The name for hatchery <name>-<type>")
	viper.BindPFlag("name", rootCmd.PersistentFlags().Lookup("name"))

	rootCmd.PersistentFlags().Int64("grace-time-queued", 4, "if worker is queued less than this value (seconds), hatchery does not take care of it")
	viper.BindPFlag("grace-time-queued", rootCmd.PersistentFlags().Lookup("grace-time-queued"))
}
