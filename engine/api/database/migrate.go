package database

import "github.com/spf13/cobra"

var DBCmd = &cobra.Command{
	Use: "db",
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "",
	Long:  "",
	Run:   upgradeCmdFunc,
}

var downgradeCmd = &cobra.Command{
	Use:   "downgrade",
	Short: "",
	Long:  "",
	Run:   downgradeCmdFunc,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	Long:  "",
	Run:   statusCmdFunc,
}

func init() {
	DBCmd.AddCommand(upgradeCmd)
}

func upgradeCmdFunc(cmd *cobra.Command, args []string) {

}

func downgradeCmdFunc(cmd *cobra.Command, args []string) {

}

func statusCmdFunc(cmd *cobra.Command, args []string) {

}
