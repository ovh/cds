package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/cli/cdsctl/internal"
)

var contextCmd = cli.Command{
	Name:    "context",
	Aliases: []string{"ctx"},
	Short:   "cdsctl context",
}

func contexts() *cobra.Command {
	return cli.NewCommand(contextCmd, contextRun, []*cobra.Command{
		cli.NewListCommand(contextListCmd, contextListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(contextCurrentCmd, contextCurrentRun, nil, withAllCommandModifiers()...),
	})
}

var contextListCmd = cli.Command{
	Name:  "list",
	Short: "List cdsctl contexts",
}

var contextCurrentCmd = cli.Command{
	Name:  "current",
	Short: "show the current context",
}

func contextListRun(v cli.Values) (cli.ListResult, error) {
	fi, err := os.Open(configFilePath)
	defer fi.Close()
	cdsConfigFile, err := internal.GetConfigFile(fi)
	if err != nil {
		return nil, fmt.Errorf("error while reading config file %s: %v", configFilePath, err)
	}
	m := make([]internal.CDSContext, len(cdsConfigFile.Contexts))
	var i int
	for _, v := range cdsConfigFile.Contexts {
		m[i] = v
		i++
	}

	return cli.AsListResult(m), nil
}

func contextCurrentRun(v cli.Values) error {
	fi, err := os.Open(configFilePath)
	defer fi.Close()
	if err != nil {
		return fmt.Errorf("error while reading config file %s: %v", configFilePath, err)
	}
	current, err := internal.GetCurrentContextName(fi)
	if err != nil {
		return fmt.Errorf("error while getting current context: %v", err)
	}
	fmt.Println(current)
	return nil
}

func contextRun(v cli.Values) error {
	fi, err := os.Open(configFilePath)
	defer fi.Close()
	cdsConfigFile, err := internal.GetConfigFile(fi)
	if err != nil {
		return fmt.Errorf("error while reading config file %s: %v", configFilePath, err)
	}

	if len(cdsConfigFile.Contexts) > 0 {
		opts := make([]string, len(cdsConfigFile.Contexts))
		var i int
		for v := range cdsConfigFile.Contexts {
			opts[i] = v
			i++
		}
		selected := cli.AskChoice(fmt.Sprintf("%s - Choose a context", configFilePath), opts...)
		fi, err = os.OpenFile(configFilePath, os.O_RDONLY, 0600)
		if err != nil {
			return fmt.Errorf("Error while opening file %s: %v", configFilePath, err)
		}
		defer fi.Close()
		fiWrite, err := os.OpenFile(configFilePath, os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("Error while opening file %s: %v", configFilePath, err)
		}
		defer fiWrite.Close()
		if err := internal.SetCurrentContext(fi, fiWrite, opts[selected]); err != nil {
			return err
		}
	} else {
		fmt.Printf("%s - no context - please use cdsctl login\n", configFilePath)
	}

	return nil
}
