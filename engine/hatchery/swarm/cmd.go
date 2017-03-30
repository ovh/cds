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

	Cmd.Flags().IntVar(&hatcherySwarm.ratioService, "ratio-service", 75, "Percent reserved for spwaning worker with service requirement")
	viper.BindPFlag("ratio-service", Cmd.Flags().Lookup("ratio-service"))

	Cmd.Flags().IntVar(&hatcherySwarm.maxContainers, "max-containers", 10, "")
	viper.BindPFlag("max-containers", Cmd.Flags().Lookup("max-containers"))

	Cmd.Flags().IntVar(&hatcherySwarm.defaultMemory, "worker-memory", 1024, "Worker default memory")
	viper.BindPFlag("worker-memory", Cmd.Flags().Lookup("worker-memory"))

	Cmd.Flags().IntVar(&hatcherySwarm.workerTTL, "worker-ttl", 10, "Worker TTL (minutes)")
	viper.BindPFlag("worker-ttl", Cmd.Flags().Lookup("worker-ttl"))

	Cmd.Flags().Int("spawn-threshold-critical", 20, "log critical if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-critical", Cmd.Flags().Lookup("spawn-threshold-critical"))

	Cmd.Flags().Int("spawn-threshold-warning", 4, "log warning if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-warning", Cmd.Flags().Lookup("spawn-threshold-warning"))
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
		hatchery.Create(hatcherySwarm,
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
	PreRun: func(cmd *cobra.Command, args []string) {
		if viper.GetInt("max-containers") <= 0 {
			sdk.Exit("max-containers must be > 0")
		}
		if viper.GetInt("worker-ttl") <= 0 {
			sdk.Exit("worker-ttl must be > 0")
		}
		if viper.GetInt("worker-memory") <= 1 {
			sdk.Exit("worker-memory must be > 1")
		}

		hatcherySwarm.ratioService = viper.GetInt("ratio-service")
		hatcherySwarm.maxContainers = viper.GetInt("max-containers")
		hatcherySwarm.defaultMemory = viper.GetInt("worker-memory")
		hatcherySwarm.workerTTL = viper.GetInt("worker-ttl")

		if os.Getenv("DOCKER_HOST") == "" {
			sdk.Exit("Please export docker client env variables DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH")
		}
	},
}
