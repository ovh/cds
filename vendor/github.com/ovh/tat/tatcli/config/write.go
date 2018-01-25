package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdConfigTemplate = &cobra.Command{
	Use:   "template",
	Short: "Write a template configuration file in $HOME/.tatcli/config.json: tatcli config template",
	Run: func(cmd *cobra.Command, args []string) {
		writeTemplate()
	},
}

// TemplateJSONType describes .tatcli/config.json file
type TemplateJSONType struct {
	Username          string   `json:"username"`
	Password          string   `json:"password"`
	URL               string   `json:"url"`
	TatwebuiURL       string   `json:"tatwebui-url"`
	PostHookRunAction string   `json:"post-hook-run-action,omitempty"`
	Filters           []string `json:"filters,omitempty"`
	Commands          []string `json:"commands,omitempty"`
	Hooks             []Hook   `json:"hooks"`
}

type Hook struct {
	Shortcut string   `json:"shortcut"`
	Command  string   `json:"command"`
	Exec     string   `json:"exec"`
	Topics   []string `json:"topics"`
}

func writeTemplate() {
	var templateJSON TemplateJSONType

	if viper.GetString("username") != "" {
		templateJSON.Username = viper.GetString("username")
	}
	if viper.GetString("password") != "" {
		templateJSON.Password = viper.GetString("password")
	}
	if viper.GetString("url") != "" {
		templateJSON.URL = viper.GetString("url")
	}
	if viper.GetString("tatwebui-url") != "" {
		templateJSON.TatwebuiURL = viper.GetString("tatwebui-url")
	}

	jsonStr, err := json.MarshalIndent(templateJSON, "", "  ")
	internal.Check(err)
	jsonStr = append(jsonStr, '\n')
	filename := internal.ConfigFile

	dir := path.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		internal.Check(os.Mkdir(dir, 0740))
	}

	internal.Check(ioutil.WriteFile(filename, jsonStr, 0600))
	fmt.Printf("%s is written\n", filename)
}
