package config

import (
	"fmt"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdConfigShow = &cobra.Command{
	Use:   "show",
	Short: "Show Configuration: tatcli config show",
	Run: func(cmd *cobra.Command, args []string) {
		show()
	},
}

func show() {
	internal.ReadConfig()
	fmt.Printf("username:%s\n", viper.GetString("username"))
	fmt.Printf("password:%s\n", viper.GetString("password"))
	fmt.Printf("url:%s\n", viper.GetString("url"))
}
