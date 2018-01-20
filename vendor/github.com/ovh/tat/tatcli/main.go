package main

import (
	"fmt"
	"os"

	"github.com/ovh/tat/tatcli/config"
	"github.com/ovh/tat/tatcli/group"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/ovh/tat/tatcli/message"
	"github.com/ovh/tat/tatcli/presence"
	"github.com/ovh/tat/tatcli/stats"
	"github.com/ovh/tat/tatcli/system"
	"github.com/ovh/tat/tatcli/topic"
	"github.com/ovh/tat/tatcli/ui"
	"github.com/ovh/tat/tatcli/update"
	"github.com/ovh/tat/tatcli/user"
	"github.com/ovh/tat/tatcli/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var home = os.Getenv("HOME")

var rootCmd = &cobra.Command{
	Use:   "tatcli",
	Short: "Text And Tags - Command Line Tool",
	Long:  `Text And Tags - Command Line Tool`,
}

func main() {
	addCommands()
	rootCmd.PersistentFlags().BoolVarP(&internal.Verbose, "verbose", "v", false, "verbose output - display response from tat engine")
	rootCmd.PersistentFlags().BoolVarP(&internal.Debug, "debug", "", false, "debug output - display request and response")
	rootCmd.PersistentFlags().BoolVarP(&internal.Pretty, "pretty", "t", false, "Pretty Print Json Output")
	rootCmd.PersistentFlags().BoolVarP(&internal.ShowStackTrace, "showStackTrace", "", false, "Show Stack Trace if tatcli panic")
	rootCmd.PersistentFlags().BoolVarP(&internal.SSLInsecureSkipVerify, "sslInsecureSkipVerify", "k", false, "Skip certificate check with SSL connection")
	rootCmd.PersistentFlags().StringVarP(&internal.URL, "url", "", "", "URL Tat Engine, facultative if you have a "+home+"/.tatcli/config.json file")
	rootCmd.PersistentFlags().StringVarP(&internal.TatwebuiURL, "tatwebui-url", "", "", "URL of Tat WebUI, facultative")
	rootCmd.PersistentFlags().StringVarP(&internal.Username, "username", "u", "", "username, facultative if you have a "+home+"/.tatcli/config.json file")
	rootCmd.PersistentFlags().StringVarP(&internal.Password, "password", "p", "", "password, facultative if you have a "+home+"/.tatcli/config.json file")
	rootCmd.PersistentFlags().StringVarP(&internal.ConfigFile, "configFile", "c", home+"/.tatcli/config.json", "configuration file, default is "+home+"/.tatcli/config.json")

	viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("tatwebui-url", rootCmd.PersistentFlags().Lookup("tatwebui-url"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("sslInsecureSkipVerify", rootCmd.PersistentFlags().Lookup("sslInsecureSkipVerify"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//AddCommands adds child commands to the root command rootCmd.
func addCommands() {
	rootCmd.AddCommand(config.Cmd)
	rootCmd.AddCommand(group.Cmd)
	rootCmd.AddCommand(message.Cmd)
	rootCmd.AddCommand(presence.Cmd)
	rootCmd.AddCommand(stats.Cmd)
	rootCmd.AddCommand(topic.Cmd)
	rootCmd.AddCommand(update.Cmd)
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(ui.Cmd)
	rootCmd.AddCommand(system.Cmd)
	rootCmd.AddCommand(version.Cmd)
}
