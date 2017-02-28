package application

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdMetadata() *cobra.Command {

	cmd := &cobra.Command{
		Use: "metadata",
	}

	cmdShow := &cobra.Command{
		Use:   "show",
		Short: "cds application metadata show <projectKey> <app>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			appname := args[1]
			app, err := sdk.GetApplication(key, appname)
			if err != nil {
				sdk.Exit("Error: cannot get application %s %s (%s)\n", key, appname, err)
			}
			if app.Metadata != nil && len(app.Metadata) > 0 {
				fmt.Printf("Metadata:\n")
				for k, v := range app.Metadata {
					fmt.Printf(" - %s: %s\n", k, v)
				}
			} else {
				sdk.Exit("No metadata found.")
			}
		},
	}

	cmdCreate := &cobra.Command{
		Use:    "create",
		Hidden: true,
		Short:  "cds application metadata create <projectKey> <app> <key> [key [...]]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 3 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}

			key := args[0]
			appname := args[1]
			app, err := sdk.GetApplication(key, appname)
			if err != nil {
				sdk.Exit("Error: cannot get application %s %s (%s)\n", key, appname, err)
			}

			news := []string{}
			values := args[2:]
			for _, v := range values {
				news = append(news, v)
			}

			if app.Metadata == nil {
				app.Metadata = make(map[string]string)
			}
			for _, v := range news {
				app.Metadata[v] = ""
			}

			if err := sdk.UpdateApplication(app); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")
		},
	}

	cmdSet := &cobra.Command{
		Use:   "set",
		Short: "cds application metadata set <projectKey> <app> <key=value> [key=value [...]]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 3 {
				fmt.Println(args)
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			appname := args[1]
			app, err := sdk.GetApplication(key, appname)
			if err != nil {
				sdk.Exit("Error: cannot get application %s %s (%s)\n", key, appname, err)
			}

			news := map[string]string{}
			values := args[2:]
			for _, v := range values {
				tuple := strings.SplitN(v, "=", 2)
				if len(tuple) != 2 {
					sdk.Exit("Wrong usage: %s\n", cmd.Short)
				}
				news[tuple[0]] = tuple[1]
			}

			if app.Metadata == nil {
				app.Metadata = make(map[string]string)
			}
			for k, v := range news {
				app.Metadata[k] = v
			}

			if err := sdk.UpdateApplication(app); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")

		},
	}

	cmdUnset := &cobra.Command{
		Use:   "unset",
		Short: "cds application metadata unset <projectKey> <app> <key> [key [...]]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 3 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			appname := args[1]
			app, err := sdk.GetApplication(key, appname)
			if err != nil {
				sdk.Exit("Error: cannot get application %s %s (%s)\n", key, appname, err)
			}

			if app.Metadata == nil {
				app.Metadata = make(map[string]string)
			}

			values := args[2:]
			for _, v := range values {
				delete(app.Metadata, v)
			}

			if err := sdk.UpdateApplication(app); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")
		},
	}

	cmdDelAll := &cobra.Command{
		Use:   "delete",
		Short: "cds application metadata delete <projectKey> <app>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}

			key := args[0]
			appname := args[1]
			app, err := sdk.GetApplication(key, appname)
			if err != nil {
				sdk.Exit("Error: cannot get application %s %s (%s)\n", key, appname, err)
			}

			app.Metadata = make(map[string]string)

			if err := sdk.UpdateApplication(app); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")
		},
	}

	cmd.AddCommand(cmdShow)
	cmd.AddCommand(cmdSet)
	cmd.AddCommand(cmdCreate)
	cmd.AddCommand(cmdUnset)
	cmd.AddCommand(cmdDelAll)

	return cmd

}
