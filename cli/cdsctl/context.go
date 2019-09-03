package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/cli/cdsctl/internal"
)

var contextCmd = cli.Command{
	Name:    "context",
	Aliases: []string{"ctx"},
	Short:   "Manage cdsctl config file",
	Flags: []cli.Flag{
		{
			Name:  "context-name",
			Usage: "A cdsctl context name",
		},
	},
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
	Short: "Show the current context",
}

func contextListRun(v cli.Values) (cli.ListResult, error) {
	fi, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error while opening config file %s: %v", configFilePath, err)
	}
	defer fi.Close() // nolint
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
	if err != nil {
		return fmt.Errorf("error while opening config file %s: %v", configFilePath, err)
	}
	defer fi.Close() // nolint
	current, err := internal.GetCurrentContextName(fi)
	if err != nil {
		return fmt.Errorf("error while getting current context: %v", err)
	}
	fmt.Println(current)
	return nil
}

func contextRun(v cli.Values) error {
	fi, err := os.Open(configFilePath)
	if err != nil {
		return fmt.Errorf("error while opening config file %s: %v", configFilePath, err)
	}
	defer fi.Close() // nolint
	cdsConfigFile, err := internal.GetConfigFile(fi)
	if err != nil {
		return fmt.Errorf("error while reading config file %s: %v", configFilePath, err)
	}

	if v.GetBool("no-interactive") {
		if v.GetString("context-name") == "" {
			return fmt.Errorf("you must use a context name with no-interactive flag. Example: cdsctl context --context-name my-context")
		}
		wdata := &bytes.Buffer{}
		if err := internal.SetCurrentContext(fi, wdata, v.GetString("context-name")); err != nil {
			return err
		}
		if err := fi.Close(); err != nil {
			return fmt.Errorf("Error while closing file %s: %v", configFilePath, err)
		}
		if err := writeConfigFile(configFilePath, wdata); err != nil {
			return err
		}
		return nil
	}

	// interactive: let user choose the context
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
		defer fi.Close() // nolint

		wdata := &bytes.Buffer{}
		if err := internal.SetCurrentContext(fi, wdata, opts[selected]); err != nil {
			return err
		}
		if err := fi.Close(); err != nil {
			return fmt.Errorf("Error while closing file %s: %v", configFilePath, err)
		}
		if err := writeConfigFile(configFilePath, wdata); err != nil {
			return err
		}
	} else {
		fmt.Printf("%s - no context - please use cdsctl login\n", configFilePath)
	}

	return nil
}
