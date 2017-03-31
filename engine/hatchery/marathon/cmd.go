package marathon

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/gambol99/go-marathon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func init() {
	hatcheryMarathon = &HatcheryMarathon{}

	Cmd.Flags().StringVar(&hatcheryMarathon.marathonHost, "marathon-host", "", "marathon-host")
	viper.BindPFlag("marathon-host", Cmd.Flags().Lookup("marathon-host"))

	Cmd.Flags().StringVar(&hatcheryMarathon.marathonID, "marathon-id", "", "marathon-id")
	viper.BindPFlag("marathon-id", Cmd.Flags().Lookup("marathon-id"))

	Cmd.Flags().StringVar(&hatcheryMarathon.marathonUser, "marathon-user", "", "marathon-user")
	viper.BindPFlag("marathon-user", Cmd.Flags().Lookup("marathon-user"))

	Cmd.Flags().StringVar(&hatcheryMarathon.marathonPassword, "marathon-password", "", "marathon-password")
	viper.BindPFlag("marathon-password", Cmd.Flags().Lookup("marathon-password"))

	Cmd.Flags().StringVar(&hatcheryMarathon.marathonLabelsString, "marathon-labels", "", "marathon-labels")
	viper.BindPFlag("marathon-labels", Cmd.Flags().Lookup("marathon-labels"))

	Cmd.Flags().IntVar(&hatcheryMarathon.defaultMemory, "worker-memory", 1024, "Worker default memory")
	viper.BindPFlag("worker-memory", Cmd.Flags().Lookup("worker-memory"))

	Cmd.Flags().IntVar(&hatcheryMarathon.workerTTL, "worker-ttl", 10, "Worker TTL (minutes)")
	viper.BindPFlag("worker-ttl", Cmd.Flags().Lookup("worker-ttl"))

	Cmd.Flags().IntVar(&hatcheryMarathon.workerSpawnTimeout, "worker-spawn-timeout", 120, "Worker Timeout Spawning (seconds)")
	viper.BindPFlag("worker-spawn-timeout", Cmd.Flags().Lookup("worker-spawn-timeout"))

	Cmd.Flags().Int("spawn-threshold-critical", 10, "log critical if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-critical", Cmd.Flags().Lookup("spawn-threshold-critical"))

	Cmd.Flags().Int("spawn-threshold-warning", 4, "log warning if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-warning", Cmd.Flags().Lookup("spawn-threshold-warning"))
}

// Cmd configures comamnd for HatcheryLocal
var Cmd = &cobra.Command{
	Use:   "marathon",
	Short: "Hatchery marathon commands: hatchery marathon --help",
	Long: `Hatchery marathon commands: hatchery marathon <command>
Start worker model instances on a marathon cluster

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ hatchery marathon --api=https://<api.domain> --token=<token>

	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatchery.Create(hatcheryMarathon,
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
		hatcheryMarathon.token = viper.GetString("token")

		if viper.GetString("marathon-host") == "" {
			sdk.Exit("flag or environmnent variable marathon-host not provided, aborting\n")
		}

		hatcheryMarathon.marathonID = viper.GetString("marathon-id")
		if hatcheryMarathon.marathonID == "" {
			sdk.Exit("flag or environmnent variable marathon-id not provided, aborting\n")
		}

		if viper.GetString("marathon-user") == "" {
			sdk.Exit("flag or environmnent variable marathon-user not provided, aborting\n")
		}

		if viper.GetString("marathon-password") == "" {
			sdk.Exit("flag or environmnent variable marathon-password not provided, aborting\n")
		}

		hatcheryMarathon.marathonLabelsString = viper.GetString("marathon-labels")
		hatcheryMarathon.marathonLabels = map[string]string{}
		if hatcheryMarathon.marathonLabelsString != "" {
			array := strings.Split(hatcheryMarathon.marathonLabelsString, ",")
			for _, s := range array {
				if !strings.Contains(s, "=") {
					continue
				}
				tuple := strings.Split(s, "=")
				if len(tuple) != 2 {
					sdk.Exit("malformatted flag or environmnent variable marathon-labels")
				}
				hatcheryMarathon.marathonLabels[tuple[0]] = tuple[1]
			}
		}

		//Custom http client with 3 retries
		httpClient := &http.Client{
			Transport: &httpcontrol.Transport{
				RequestTimeout:  time.Minute,
				MaxTries:        3,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: viper.GetBool("insecure")},
			},
		}

		config := marathon.NewDefaultConfig()
		config.URL = hatcheryMarathon.marathonHost
		config.HTTPBasicAuthUser = hatcheryMarathon.marathonUser
		config.HTTPBasicPassword = hatcheryMarathon.marathonPassword
		config.HTTPClient = httpClient

		client, err := marathon.NewClient(config)
		if err != nil {
			sdk.Exit("Connection failed on %s\n", viper.GetString("marathon-host"))
		}

		hatcheryMarathon.client = client

	},
}
