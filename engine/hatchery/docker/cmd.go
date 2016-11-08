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
		hatchery.Born(hatcheryDocker, viper.GetString("api"), viper.GetString("token"), viper.GetInt("provision"), viper.GetInt("request-api-timeout"))
	},
}
