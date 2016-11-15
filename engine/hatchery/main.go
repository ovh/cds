package main

import (
	"fmt"
	"os"

	"github.com/ovh/cds/engine/hatchery/docker"
	"github.com/ovh/cds/engine/hatchery/local"
	"github.com/ovh/cds/engine/hatchery/mesos"
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
	rootCmd.AddCommand(mesos.Cmd)
	rootCmd.AddCommand(swarm.Cmd)
	rootCmd.AddCommand(openstack.Cmd)
}

func addFlags() {
	viper.SetEnvPrefix("cds")
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().String("log-level", "noticea", "Log Level: debug, info, warning, notice, critical")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	rootCmd.PersistentFlags().String("api", "", "CDS api endpoint")
	viper.BindPFlag("api", rootCmd.PersistentFlags().Lookup("api"))

	rootCmd.PersistentFlags().String("token", "", "CDS token")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))

	rootCmd.PersistentFlags().Int("request-api-timeout", 10, "Request CDS API: timeout in seconds")
	viper.BindPFlag("request-api-timeout", rootCmd.PersistentFlags().Lookup("request-api-timeout"))

	rootCmd.PersistentFlags().Int("provision", 0, "Allowed worker model provisioning")
	viper.BindPFlag("provision", rootCmd.PersistentFlags().Lookup("provision"))

	rootCmd.PersistentFlags().Int("max-worker", 10, "Maximum allowed simultaenous workers")
	viper.BindPFlag("max-worker", rootCmd.PersistentFlags().Lookup("max-worker"))
}
