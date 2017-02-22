package project

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
		Short: "cds project metadata show <projectKey>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			proj, err := sdk.GetProject(key)
			if err != nil {
				sdk.Exit("Error: cannot get project %s (%s)\n", key, err)
			}
			if proj.Metadata != nil && len(proj.Metadata) > 0 {
				fmt.Printf("Metadata:\n")
				for k, v := range proj.Metadata {
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
		Short:  "cds project metadata create <projectKey> <key> [key [...]]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			proj, err := sdk.GetProject(key)
			if err != nil {
				sdk.Exit("Error: cannot get project %s (%s)\n", key, err)
			}

			news := []string{}
			values := args[1:]
			for _, v := range values {
				news = append(news, v)
			}

			if proj.Metadata == nil {
				proj.Metadata = make(map[string]string)
			}
			for _, v := range news {
				proj.Metadata[v] = ""
			}

			if err := sdk.UpdateProject(&proj); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")

		},
	}

	cmdSet := &cobra.Command{
		Use:   "set",
		Short: "cds project metadata set <projectKey> <key=value> [key=value [...]]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			proj, err := sdk.GetProject(key)
			if err != nil {
				sdk.Exit("Error: cannot get project %s (%s)\n", key, err)
			}

			news := map[string]string{}
			values := args[1:]
			for _, v := range values {
				tuple := strings.SplitN(v, "=", 2)
				if len(tuple) != 2 {
					sdk.Exit("Wrong usage: %s\n", cmd.Short)
				}
				news[tuple[0]] = tuple[1]
			}

			if proj.Metadata == nil {
				proj.Metadata = make(map[string]string)
			}
			for k, v := range news {
				proj.Metadata[k] = v
			}

			if err := sdk.UpdateProject(&proj); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")

		},
	}

	cmdUnset := &cobra.Command{
		Use:   "unset",
		Short: "cds project metadata unset <projectKey> <key> [key [...]]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			proj, err := sdk.GetProject(key)
			if err != nil {
				sdk.Exit("Error: cannot get project %s (%s)\n", key, err)
			}

			if proj.Metadata == nil {
				proj.Metadata = make(map[string]string)
			}

			values := args[1:]
			for _, v := range values {
				delete(proj.Metadata, v)
			}

			if err := sdk.UpdateProject(&proj); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			fmt.Println("OK")
		},
	}

	cmdDelAll := &cobra.Command{
		Use:   "delete",
		Short: "cds project metadata delete <projectKey>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			key := args[0]
			proj, err := sdk.GetProject(key)
			if err != nil {
				sdk.Exit("Error: cannot get project %s (%s)\n", key, err)
			}

			proj.Metadata = make(map[string]string)

			if err := sdk.UpdateProject(&proj); err != nil {
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
