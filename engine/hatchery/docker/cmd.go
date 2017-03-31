package docker

import (
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	hatcheryDocker = &HatcheryDocker{}

	Cmd.Flags().StringVarP(&hatcheryDocker.addhost, "docker-add-host", "", "", "Start worker with a custom host-to-IP mapping (host:ip)")
	viper.BindPFlag("docker-add-host", Cmd.Flags().Lookup("docker-add-host"))

	Cmd.Flags().Int("spawn-threshold-critical", 10, "log critical if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-critical", Cmd.Flags().Lookup("spawn-threshold-critical"))

	Cmd.Flags().Int("spawn-threshold-warning", 4, "log warning if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-warning", Cmd.Flags().Lookup("spawn-threshold-warning"))
}

// Cmd configures comamnd for HatcheryLocal
var Cmd = &cobra.Command{
	Use:   "docker",
	Short: "Hatchery docker commands: hatchery docker --help",
	Long: `Hatchery docker commands: hatchery docker <command>
Start worker model instances on a single host.

Hatchery in docker mode looks for worker models of type 'docker' to start.

We will add a worker model to build Go applications:

$ cds worker model add golang docker --image=golang:latest
Add Go binary capability to model:

$ cds worker model capability add golang go binary go

You can generate a token for a given group using the CLI:

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ hatchery docker --api=https://<api.domain> --token=<token>

	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatcheryDocker.addhost = viper.GetString("docker-add-host")
		hatchery.Create(hatcheryDocker,
			viper.GetString("api"),
			viper.GetString("token"),
			viper.GetInt("max-worker"),
			viper.GetInt("provision"),
			viper.GetInt("request-api-timeout"),
			viper.GetInt("max-failures-heartbeat"),
			viper.GetBool("insecure"),
			viper.GetInt("provision-seconds"),
			viper.GetInt("spawn-threshold-warning"),
			viper.GetInt("spawn-threshold-critical"),
			viper.GetInt("grace-time-queued"),
		)
	},
}
