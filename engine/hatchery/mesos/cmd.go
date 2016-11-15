package mesos

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	hatcheryMesos = &HatcheryMesos{}
	Cmd.Flags().StringVar(&hatcheryMesos.marathonHost, "marathon-host", "", "marathon-host")
	viper.BindPFlag("marathon-host", Cmd.Flags().Lookup("marathon-host"))

	Cmd.Flags().StringVar(&hatcheryMesos.marathonID, "marathon-id", "", "marathon-id")
	viper.BindPFlag("marathon-id", Cmd.Flags().Lookup("marathon-id"))

	Cmd.Flags().StringVar(&hatcheryMesos.marathonVHOST, "marathon-vhost", "", "marathon-vhost")
	viper.BindPFlag("marathon-vhost", Cmd.Flags().Lookup("marathon-vhost"))

	Cmd.Flags().StringVar(&hatcheryMesos.marathonUser, "marathon-user", "", "marathon-user")
	viper.BindPFlag("marathon-user", Cmd.Flags().Lookup("marathon-user"))

	Cmd.Flags().StringVar(&hatcheryMesos.marathonPassword, "marathon-password", "", "marathon-password")
	viper.BindPFlag("marathon-password", Cmd.Flags().Lookup("marathon-password"))
}

// Cmd configures comamnd for HatcheryLocal
var Cmd = &cobra.Command{
	Use:   "mesos",
	Short: "Hatchery mesos commands: hatchery mesos --help",
	Long: `Hatchery mesos commands: hatchery mesos <command>
Start worker model instances on a mesos cluster

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ hatchery mesos --api=https://<api.domain> --token=<token>

	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatchery.Born(hatcheryMesos, viper.GetString("api"), viper.GetString("token"), viper.GetInt("provision"), viper.GetInt("request-api-timeout"))
	},
	PreRun: func(cmd *cobra.Command, args []string) {

		if hatcheryMesos.marathonHost == "" {
			sdk.Exit("flag or environmnent variable marathon-host not provided, aborting\n")
		}

		if hatcheryMesos.marathonID == "" {
			sdk.Exit("flag or environmnent variable marathon-id not provided, aborting\n")
		}

		if hatcheryMesos.marathonVHOST == "" {
			sdk.Exit("flag or environmnent variable marathon-vhost not provided, aborting\n")
		}

		if hatcheryMesos.marathonUser == "" {
			sdk.Exit("flag or environmnent variable marathon-user not provided, aborting\n")
		}

		if hatcheryMesos.marathonPassword == "" {
			sdk.Exit("flag or environmnent variable marathon-password not provided, aborting\n")
		}
	},
}
