package swarm

import (
	"os"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	hatcherySwarm = &HatcherySwarm{}

	Cmd.Flags().BoolVar(&hatcherySwarm.onlyWithServiceReq, "only-with-service-req", false, "")
	viper.BindPFlag("only-with-service-req", Cmd.Flags().Lookup("only-with-service-req"))

	Cmd.Flags().IntVar(&hatcherySwarm.maxContainers, "max-containers", 10, "")
	viper.BindPFlag("max-containers", Cmd.Flags().Lookup("max-containers"))

	Cmd.Flags().IntVar(&hatcherySwarm.defaultMemory, "worker-memory", 1024, "Worker default memory")
	viper.BindPFlag("worker-memory", Cmd.Flags().Lookup("worker-memory"))

	Cmd.Flags().IntVar(&hatcherySwarm.defaultMemory, "worker-ttl", 1, "Worker TTL (hours)")
	viper.BindPFlag("worker-ttl", Cmd.Flags().Lookup("worker-ttl"))
}

// Cmd configures comamnd for HatcherySwarm
var Cmd = &cobra.Command{
	Use:   "swarm",
	Short: "Hatchery Swarm commands: hatchery swarm --help",
	Long: `Hatchery Swarm commands: hatchery swarm <command>
Start worker on a docker swarm cluster.

You have to export DOCKER_HOST
You should export DOCKER_TLS_VERIFY and DOCKER_CERT_PATH

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ DOCKER_HOST="tcp://localhost:2375" hatchery swarm --api=https://<api.domain> --token=<token> 

	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatchery.Born(hatcherySwarm, viper.GetString("api"), viper.GetString("token"), viper.GetInt("provision"), viper.GetInt("request-api-timeout"))
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		hatcherySwarm.onlyWithServiceReq = viper.GetBool("only-with-service-req")
		hatcherySwarm.maxContainers = viper.GetInt("max-containers")
		hatcherySwarm.defaultMemory = viper.GetInt("worker-memory")
		hatcherySwarm.workerTTL = viper.GetInt("worker-ttl")

		if os.Getenv("DOCKER_HOST") == "" {
			sdk.Exit("Please export docker client env variables DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH")
		}
	},
}
