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
}

func contexts() *cobra.Command {
	return cli.NewCommand(contextCmd, contextRun, []*cobra.Command{
		cli.NewListCommand(contextListCmd, contextListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(contextGetCurrentCmd, contextGetCurrentRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(contextSetCurrentCmd, contextSetCurrentRun, nil, withAllCommandModifiers()...),
	})
}

var contextListCmd = cli.Command{
	Name:  "list",
	Short: "List cdsctl contexts",
}

var contextGetCurrentCmd = cli.Command{
	Name:  "current",
	Short: "Show the current context",
}

var contextSetCurrentCmd = cli.Command{
	Name:  "set-current",
	Short: "Set the current context",
	Args: []cli.Arg{
		{
			Name: "name",
		},
	},
}

func contextListRun(v cli.Values) (cli.ListResult, error) {
	fi, err := os.Open(configFilePath)
	if err != nil {
		return nil, cli.WrapError(err, "error while opening config file %s", configFilePath)
	}
	defer fi.Close() // nolint
	cdsConfigFile, err := internal.GetConfigFile(fi)
	if err != nil {
		return nil, cli.WrapError(err, "error while reading config file %s", configFilePath)
	}
	m := make([]internal.CDSContext, len(cdsConfigFile.Contexts))
	var i int
	for _, v := range cdsConfigFile.Contexts {
		m[i] = v
		i++
	}

	return cli.AsListResult(m), nil
}

func contextGetCurrentRun(v cli.Values) error {
	fi, err := os.Open(configFilePath)
	if err != nil {
		return cli.WrapError(err, "error while opening config file %s", configFilePath)
	}
	defer fi.Close() // nolint
	current, err := internal.GetCurrentContextName(fi)
	if err != nil {
		return cli.WrapError(err, "error while getting current context")
	}
	fmt.Println(current)
	return nil
}

func contextSetCurrentRun(v cli.Values) error {
	fi, err := os.OpenFile(configFilePath, os.O_RDONLY, 0600)
	if err != nil {
		return cli.WrapError(err, "Error while opening file %s", configFilePath)
	}

	wdata := &bytes.Buffer{}
	if err := internal.SetCurrentContext(fi, wdata, v.GetString("name")); err != nil {
		fi.Close() // nolint
		return err
	}
	if err := fi.Close(); err != nil {
		return cli.WrapError(err, "Error while closing file %s", configFilePath)
	}
	if err := writeConfigFile(configFilePath, wdata); err != nil {
		return err
	}
	return nil
}

func contextRun(v cli.Values) error {
	fi, err := os.Open(configFilePath)
	if err != nil {
		return cli.WrapError(err, "error while opening config file %s", configFilePath)
	}
	cdsConfigFile, err := internal.GetConfigFile(fi)
	if err != nil {
		return cli.WrapError(err, "error while reading config file %s", configFilePath)
	}

	if v.GetBool("no-interactive") {
		fi.Close() // nolint
		return cli.NewError("you must use the set subcommand with no-interactive flag. Example: cdsctl context set my-context")
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
			return cli.WrapError(err, "Error while opening file %s", configFilePath)
		}

		wdata := &bytes.Buffer{}
		if err := internal.SetCurrentContext(fi, wdata, opts[selected]); err != nil {
			fi.Close() // nolint
			return err
		}
		if err := fi.Close(); err != nil {
			return cli.WrapError(err, "Error while closing file %s", configFilePath)
		}
		if err := writeConfigFile(configFilePath, wdata); err != nil {
			return err
		}
	} else {
		fmt.Printf("%s - no context - please use cdsctl login\n", configFilePath)
	}

	return nil
}
