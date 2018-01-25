package ui

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/config"
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/viper"
)

func (ui *tatui) loadConfig() {
	internal.ReadConfig()
	filters := viper.GetStringSlice("filters")

	// no range to keep order
	for index := 0; index < len(filters); index++ {
		filter := filters[index]
		tuples := strings.Split(filter, " ")
		if len(tuples) <= 2 {
			continue
		}
		topic := tuples[1]
		if _, ok := ui.currentFilterMessages[topic]; !ok {
			ui.currentFilterMessages[topic] = make(map[int]*tat.MessageCriteria)
			ui.currentFilterMessagesText[topic] = make(map[int]string)
		}
		c, criteriaText := ui.prepareFilterMessages(strings.Join(tuples[2:], " "), tuples[0], topic)
		ui.currentFilterMessages[topic][len(ui.currentFilterMessages[topic])] = c
		ui.currentFilterMessagesText[topic][len(ui.currentFilterMessagesText[topic])] = criteriaText
	}

	commands := viper.GetStringSlice("commands")
	// no range to keep order
	for index := 0; index < len(commands); index++ {
		commandsOnTopic := commands[index]
		tuples := strings.Split(strings.TrimSpace(commandsOnTopic), " ")
		if len(tuples) <= 1 {
			continue
		}
		topic := tuples[0]
		ui.uiTopicCommands[topic] = commandsOnTopic[len(topic):]
	}

	var conf config.TemplateJSONType
	err := viper.Unmarshal(&conf)
	if err != nil {
		internal.Exit("unable to decode confif file, err: %v", err)
	}

	ui.hooks = conf.Hooks
}

func (ui *tatui) setTatWebUIURL(str string) {
	str = strings.Replace(str, "/set-tatwebui-url ", "", 1)
	if str == "" {
		return
	}
	validURL := govalidator.IsURL(str)
	if !validURL {
		ui.msg.Text = "You entered an invalid URL"
		ui.render()
		return
	}
	viper.Set("tatwebui-url", str)
	ui.saveConfig()
}

func (ui *tatui) saveConfig() {
	var templateJSON config.TemplateJSONType

	if viper.GetString("url") == "" ||
		viper.GetString("username") == "" ||
		viper.GetString("password") == "" {
		ui.msg.Text = " Conf not saved: url, username or password empty"
		return
	}

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
	if viper.GetString("hook-run-action") != "" {
		templateJSON.PostHookRunAction = viper.GetString("post-hook-run-action")
	}

	templateJSON.Hooks = ui.hooks

	var filters []string
	for _, criteriasForTopic := range ui.currentFilterMessagesText {
		for index := 0; index < len(criteriasForTopic); index++ {
			// not this, for keeping order for _, criteria := range criteriasForTopic
			criteria := criteriasForTopic[index]
			toAdd := strings.TrimSpace(criteria)
			if toAdd != "" {
				filters = append(filters, toAdd)
			}
		}
	}
	templateJSON.Filters = filters

	var commands []string
	for topic, topicCommands := range ui.uiTopicCommands {
		if strings.TrimSpace(topicCommands) != "" {
			commands = append(commands, topic+topicCommands)
		}
	}
	templateJSON.Commands = commands

	jsonStr, err := json.MarshalIndent(templateJSON, "", "  ")
	if err != nil {
		ui.msg.Text = " Error while preparing config.json"
		return
	}
	jsonStr = append(jsonStr, '\n')
	filename := internal.ConfigFile

	dir := path.Dir(filename)
	if _, e := os.Stat(dir); os.IsNotExist(e) {
		err2 := os.Mkdir(dir, 0740)
		ui.msg.Text = " Error while saving config.json " + err2.Error()
		return
	}

	if err := ioutil.WriteFile(filename, jsonStr, 0600); err != nil {
		ui.msg.Text = " Error while saving config.json " + err.Error()
		return
	}

	ui.msg.Text = "Config file is saved"
}
