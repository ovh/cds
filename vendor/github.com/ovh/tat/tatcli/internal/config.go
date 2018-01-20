package internal

import (
	"fmt"

	"github.com/spf13/viper"
)

var (
	// Verbose display return from Tat Engine
	Verbose bool

	// Debug display Request and response from Tat Engine
	Debug bool

	// ConfigFile is $HOME/.tatcli/config.json per default
	// contains user, password and url of tat
	ConfigFile string

	// SSLInsecureSkipVerify Skip certificate check with SSL connection
	SSLInsecureSkipVerify bool

	// Pretty prints json return in pretty format
	Pretty bool

	// ShowStackTrace prints stacktrace on tatcli panic
	ShowStackTrace bool

	// URL of tat engine
	URL string

	// TatwebuiURL of tat Web UI, used only by tatcli ui and facultative
	TatwebuiURL string

	// Username of tat user
	Username string

	// Password of tat user
	Password string
)

// ReadConfig reads config in .tatcli/config per default
func ReadConfig() {
	if ConfigFile != "" {
		viper.SetConfigFile(ConfigFile)
		viper.ReadInConfig() // Find and read the config file
		if Debug {
			fmt.Printf("Using config file %s\n", ConfigFile)
		}
	}
}
