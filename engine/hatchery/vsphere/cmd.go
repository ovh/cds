package vsphere

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func init() {
	hatcheryVSphere = &HatcheryVSphere{}

	Cmd.Flags().StringVar(&hatcheryVSphere.host, "vsphere-host", "", "")
	viper.BindPFlag("vsphere-host", Cmd.Flags().Lookup("vsphere-host"))

	Cmd.Flags().StringVar(&hatcheryVSphere.user, "vsphere-user", "", "")
	viper.BindPFlag("vsphere-user", Cmd.Flags().Lookup("vsphere-user"))

	Cmd.Flags().StringVar(&hatcheryVSphere.endpoint, "vsphere-endpoint", "", "")
	viper.BindPFlag("vsphere-endpoint", Cmd.Flags().Lookup("vsphere-endpoint"))

	Cmd.Flags().StringVar(&hatcheryVSphere.password, "vsphere-password", "", "")
	viper.BindPFlag("vsphere-password", Cmd.Flags().Lookup("vsphere-password"))

	Cmd.Flags().StringVar(&hatcheryVSphere.datacenterString, "vsphere-datacenter", "", "")
	viper.BindPFlag("vsphere-datacenter", Cmd.Flags().Lookup("vsphere-datacenter"))

	Cmd.Flags().StringVar(&hatcheryVSphere.datastoreString, "vsphere-datastore", "", "")
	viper.BindPFlag("vsphere-datastore", Cmd.Flags().Lookup("vsphere-datastore"))

	Cmd.Flags().StringVar(&hatcheryVSphere.networkString, "vsphere-network", "VM Network", "")
	viper.BindPFlag("vsphere-network", Cmd.Flags().Lookup("vsphere-network"))

	Cmd.Flags().StringVar(&hatcheryVSphere.cardName, "vsphere-ethernet-card", "e1000", "Name of the virtual ethernet card")
	viper.BindPFlag("vsphere-ethernet-card", Cmd.Flags().Lookup("vsphere-ethernet-card"))

	Cmd.Flags().String("vsphere-ip-range", "", "")
	viper.BindPFlag("vsphere-ip-range", Cmd.Flags().Lookup("vsphere-ip-range"))

	Cmd.Flags().IntVar(&hatcheryVSphere.workerTTL, "worker-ttl", 30, "Worker TTL (minutes)")
	viper.BindPFlag("worker-ttl", Cmd.Flags().Lookup("worker-ttl"))

	Cmd.Flags().Int("spawn-threshold-critical", 480, "log critical if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-critical", Cmd.Flags().Lookup("spawn-threshold-critical"))

	Cmd.Flags().Int("spawn-threshold-warning", 360, "log warning if spawn take more than this value (in seconds)")
	viper.BindPFlag("spawn-threshold-warning", Cmd.Flags().Lookup("spawn-threshold-warning"))

	Cmd.Flags().BoolVar(&hatcheryVSphere.disableCreateImage, "disable-create-image", false, `if true: hatchery does not create vsphere image when a worker model is updated`)
	viper.BindPFlag("disable-create-image", Cmd.Flags().Lookup("disable-create-image"))

	Cmd.Flags().IntVar(&hatcheryVSphere.createImageTimeout, "create-image-timeout", 180, `max wait for create a vsphere image (in seconds)`)
	viper.BindPFlag("create-image-timeout", Cmd.Flags().Lookup("create-image-timeout"))
}

// Cmd configures comamnd for HatcheryVSphere
var Cmd = &cobra.Command{
	Use:   "vsphere",
	Short: "Hatchery Vsphere commands: hatchery vsphere --help",
	Long: `Hatchery Vsphere commands: hatchery vsphere <command>
Start worker on a docker vsphere cluster.

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ CDS_VSPHERE_USER=<user> \
	CDS_VSPHERE_PASSWORD=<password> \
	CDS_VSPHERE_HOST=pcc-11-222-333-444 \
	CDS_VSPHERE_ENDPOINT=pcc-11-222-333-444.ovh.com \
	CDS_VSPHERE_DATACENTER=pcc-11-222-333-444_datacenter1234 \
  CDS_API=https://api.domain \
  CDS_MAX-worker=10 \
  CDS_MODE=vsphere \
  CDS_TOKEN=2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4 \
	./hatchery
	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatchery.Create(hatcheryVSphere,
			viper.GetString("name"),
			viper.GetString("api"),
			viper.GetString("token"),
			viper.GetInt64("max-worker"),
			viper.GetBool("provision-disabled"),
			viper.GetInt("request-api-timeout"),
			viper.GetInt("max-failures-heartbeat"),
			viper.GetBool("insecure"),
			viper.GetInt("provision-seconds"),
			viper.GetInt("register-seconds"),
			viper.GetInt("spawn-threshold-warning"),
			viper.GetInt("spawn-threshold-critical"),
			viper.GetInt("grace-time-queued"),
		)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		hatcheryVSphere.host = viper.GetString("vsphere-host")
		if hatcheryVSphere.host == "" {
			sdk.Exit("flag or environment variable vsphere-host not provided, aborting\n")
		}

		hatcheryVSphere.user = viper.GetString("vsphere-user")
		if hatcheryVSphere.user == "" {
			sdk.Exit("flag or environment variable vsphere-user not provided, aborting\n")
		}

		hatcheryVSphere.endpoint = viper.GetString("vsphere-endpoint")
		if hatcheryVSphere.endpoint == "" {
			sdk.Exit("flag or environment variable vsphere-endpoint not provided, aborting\n")
		}

		hatcheryVSphere.password = viper.GetString("vsphere-password")
		if hatcheryVSphere.password == "" {
			sdk.Exit("flag or environment variable vsphere-password not provided, aborting\n")
		}

		hatcheryVSphere.datacenterString = viper.GetString("vsphere-datacenter")
		if hatcheryVSphere.datacenterString == "" {
			sdk.Exit("flag or environment variable vsphere-datacenter not provided, aborting\n")
		}
	},
}
