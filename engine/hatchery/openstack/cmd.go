package openstack

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	hatcheryOpenStack = &HatcheryCloud{}

	Cmd.Flags().StringVar(&hatcheryOpenStack.tenant, "openstack-tenant", "", "")
	viper.BindPFlag("openstack-tenant", Cmd.Flags().Lookup("openstack-tenant"))

	Cmd.Flags().StringVar(&hatcheryOpenStack.user, "openstack-user", "", "")
	viper.BindPFlag("openstack-user", Cmd.Flags().Lookup("openstack-user"))

	Cmd.Flags().StringVar(&hatcheryOpenStack.address, "openstack-auth-endpoint", "", "")
	viper.BindPFlag("openstack-auth-endpoint", Cmd.Flags().Lookup("openstack-auth-endpoint"))

	Cmd.Flags().StringVar(&hatcheryOpenStack.password, "openstack-password", "", "")
	viper.BindPFlag("openstack-password", Cmd.Flags().Lookup("openstack-password"))

	Cmd.Flags().StringVar(&hatcheryOpenStack.region, "openstack-region", "", "")
	viper.BindPFlag("openstack-region", Cmd.Flags().Lookup("openstack-region"))

	Cmd.Flags().StringVar(&hatcheryOpenStack.network, "openstack-network", "Ext-Net", "")
	viper.BindPFlag("openstack-network", Cmd.Flags().Lookup("openstack-network"))

	Cmd.Flags().String("openstack-ip-range", "Ext-Net", "")
	viper.BindPFlag("openstack-ip-range", Cmd.Flags().Lookup("openstack-ip-range"))

	Cmd.Flags().IntVar(&hatcheryOpenStack.workerTTL, "worker-ttl", 30, "Worker TTL (minutes)")
	viper.BindPFlag("worker-ttl", Cmd.Flags().Lookup("worker-ttl"))
}

// Cmd configures comamnd for HatcheryCloud
var Cmd = &cobra.Command{
	Use:   "cloud",
	Short: "Hatchery Cloud commands: hatchery cloud --help",
	Long: `Hatchery Cloud commands: hatchery cloud <command>
Start worker on a docker openstack cluster.

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ CDS_OPENSTACK_USER=<user> CDS_OPENSTACK_TENANT=<tenant> CDS_OPENSTACK_AUTH_ENDPOINT=https://auth.cloud.ovh.net CDS_OPENSTACK_PASSWORD=<password> CDS_OPENSTACK_REGION=SBG1 hatchery \
        --api=https://api.domain \
        --max-worker=10 \
        --mode=openstack \
        --provision=1 \
        --token=2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatchery.Born(hatcheryOpenStack, viper.GetString("api"), viper.GetString("token"), viper.GetInt("provision"), viper.GetInt("request-api-timeout"))
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		hatcheryOpenStack.tenant = viper.GetString("openstack-tenant")
		if hatcheryOpenStack.tenant == "" {
			sdk.Exit("flag or environmnent variable openstack-tenant not provided, aborting\n")
		}

		hatcheryOpenStack.user = viper.GetString("openstack-user")
		if hatcheryOpenStack.user == "" {
			sdk.Exit("flag or environmnent variable openstack-user not provided, aborting\n")
		}

		hatcheryOpenStack.address = viper.GetString("openstack-auth-endpoint")
		if hatcheryOpenStack.address == "" {
			sdk.Exit("flag or environmnent variable openstack-auth-endpoint not provided, aborting\n")
		}

		hatcheryOpenStack.password = viper.GetString("openstack-password")
		if hatcheryOpenStack.password == "" {
			sdk.Exit("flag or environmnent variable openstack-password not provided, aborting\n")
		}

		hatcheryOpenStack.region = viper.GetString("openstack-region")
		if hatcheryOpenStack.region == "" {
			sdk.Exit("flag or environmnent variable openstack-region not provided, aborting\n")
		}

		var err error
		if viper.GetString("openstack-ip-range") != "" {
			hatcheryOpenStack.ips, err = IPinRanges(viper.GetString("openstack-ip-range"))
			if err != nil {
				sdk.Exit("flag or environmnent variable openstack-ip-range error: %s\n", err)
			}
		}
	},
}
