package local

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	hatcheryLocal = &HatcheryLocal{}
	Cmd.Flags().StringVarP(&hatcheryLocal.basedir, "basedir", "", "/tmp", "BaseDir for worker workspace")
	viper.BindPFlag("basedir", Cmd.Flags().Lookup("basedir"))
}

// Cmd configures comamnd for HatcheryLocal
var Cmd = &cobra.Command{
	Use:   "local",
	Short: "Hatchery Local commands: hatchery local --help",
	Long: `Hatchery Local commands: hatchery local <command>
Hatchery starts workers directly as local process.

$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4

$ hatchery docker --api=https://<api.domain> --token=<token> --basedir=/tmp

	`,
	Run: func(cmd *cobra.Command, args []string) {
		hatchery.Born(hatcheryLocal, viper.GetString("api"), viper.GetString("token"), viper.GetInt("provision"), viper.GetInt("request-api-timeout"), viper.GetBool("insecure"))
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if hatcheryLocal.basedir == "" {
			sdk.Exit("basedir not provided, aborting. See flag --basedir hatchery local -h\n")
		}
	},
}
