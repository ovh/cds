package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	defaults "github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// config reads in config file and ENV variables if set.
func config() {
	for k := range AsEnvVariables(conf, "") {
		viper.BindEnv(strings.ToLower(strings.Replace(k, "_", ".", -1)), "CDS_"+k)
	}

	if cfgFile != "" {
		//If the config file doesn't exists, let's exit
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			sdk.Exit("File doesn't exist")
		}
		fmt.Println("Reading configuration file", cfgFile)

		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			sdk.Exit(err.Error())
		}

	} else if remoteCfg != "" {
		fmt.Println("Reading configuration from consul @", remoteCfg)
		viper.AddRemoteProvider("consul", remoteCfg, remoteCfgKey)
		viper.SetConfigType("toml")

		if err := viper.ReadRemoteConfig(); err != nil {
			sdk.Exit(err.Error())
		}
	} else if vaultAddr != "" && vaultToken != "" {
		//I hope one day viper will be a standard viper remote provider
		fmt.Println("Reading configuration from vault @", vaultAddr)

		s, errS := secret.New(vaultToken, vaultAddr)
		if errS != nil {
			sdk.Exit("Error when getting config from vault: %v", errS)
		}
		// Get raw config file from vault
		cfgFileContent, errV := s.GetFromVault(vaultConfKey)
		if errV != nil {
			sdk.Exit("Error when fetching config from vault: %v", errV)
		}

		// Put the content in a buffer and ask viper to read the buffer
		cfgBuffer := bytes.NewBufferString(cfgFileContent)
		if err := viper.ReadConfig(cfgBuffer); err != nil {
			sdk.Exit("Unable to read config: %v", err.Error())
		}
	}

	if err := viper.Unmarshal(conf); err != nil {
		sdk.Exit("Unable to parse config: %v", err.Error())
	}
	defaults.SetDefaults(conf)
}
