package pipeline

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func init() {
	pipelineHookCmd.AddCommand(pipelineAddHookCmd())
	pipelineHookCmd.AddCommand(pipelineDeleteHookCmd())
	pipelineHookCmd.AddCommand(pipelineListHookCmd())
}

var pipelineHookCmd = &cobra.Command{
	Use:   "hook",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func pipelineAddHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline hook add <projectKey> <applicationName> <pipelineName> [<host>/<project>/<slug>]",
		Long:  ``,
		Run:   addPipelineHook,
	}

	return cmd
}

func pipelineDeleteHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "cds pipeline hook delete <projectKey> <applicationName> <pipelineName> [<host>/<project>/<slug> if application non connected to a repository] [<idHook> if application connected to a repository]",
		Long:  ``,
		Run:   deletePipelineHook,
	}

	return cmd
}

var showURLOnly bool

func pipelineListHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds pipeline hook list <projectKey> <applicationName> <pipelineName>",
		Long:  ``,
		Run:   listPipelineHook,
	}

	cmd.Flags().BoolVarP(&showURLOnly, "show-url-only", "", false, "Shows only URL")

	return cmd
}

func addPipelineHook(cmd *cobra.Command, args []string) {

	if len(args) < 3 {
		sdk.Exit("Wrong usage: See %s\n", cmd.Short)
	}

	pipelineProject := args[0]
	appName := args[1]
	pipelineName := args[2]

	p, err := sdk.GetPipeline(pipelineProject, pipelineName)
	if err != nil {
		sdk.Exit("✘ Error: Cannot retrieve pipeline %s-%s (%s)\n", pipelineProject, pipelineName, err)
	}

	a, err := sdk.GetApplication(pipelineProject, appName)
	if err != nil {
		sdk.Exit("✘ Error: Cannot retrieve application %s-%s (%s)\n", pipelineProject, appName, err)
	}

	//If the application is attached to a repositories manager, parameter <host>/<project>/<slug> aren't taken in account
	if a.VCSServer != "" {
		err = sdk.AddHookOnRepositoriesManager(pipelineProject, appName, a.VCSServer, a.RepositoryFullname, pipelineName)
		if err != nil {
			sdk.Exit("✘ Error: Cannot add hook to pipeline %s-%s-%s (%s)\n", pipelineProject, appName, pipelineName, err)
		}
		fmt.Println("✔ Success")
	} else {
		if len(args) != 4 {
			sdk.Exit("✘ Error: Your application has to be attached to a repositories manager. Try : cds application reposmanager attach")
		}
		t := strings.Split(args[3], "/")
		if len(t) != 3 {
			sdk.Exit("✘ Error: Expected repository like <host>/<project>/<slug>. Got %d elements\n", len(t))
		}
		h, err := sdk.AddHook(a, p, t[0], t[1], t[2])
		if err != nil {
			sdk.Exit("✘ Error: Cannot add hook to pipeline %s-%s-%s (%s)\n", pipelineProject, appName, pipelineName, err)
		}
		if strings.Contains(t[0], "stash") {
			fmt.Printf(`Hook created on CDS.
	You now need to configure hook on stash. Use "Http Request Post Receive Hook" to create:
	POST https://<url-to-cds>/hook?&uid=%s&project=%s&name=%s&branch=${refChange.name}&hash=${refChange.toHash}&message=${refChange.type}&author=${user.name}

	`, h.UID, t[1], t[2])
		}
	}
}

func deletePipelineHook(cmd *cobra.Command, args []string) {

	if len(args) < 3 {
		sdk.Exit("Wrong usage: See %s\n", cmd.Short)
	}

	pipelineProject := args[0]
	appName := args[1]
	pipelineName := args[2]

	a, err := sdk.GetApplication(pipelineProject, appName)
	if err != nil {
		sdk.Exit("Cannot retrieve application %s-%s (%s)\n", pipelineProject, appName, err)
	}

	//If the application is attached to a repositories manager, parameter <host>/<project>/<slug> aren't taken in account
	if a.VCSServer != "" {
		if len(args) != 4 {
			sdk.Exit("Wrong usage: See %s\n", cmd.Short)
		}

		hookIDString := args[3]
		hookID, err := strconv.ParseInt(hookIDString, 10, 64)
		if err != nil {
			sdk.Exit("Hook id must be a number (%s)\n", err)
		}

		err = sdk.DeleteHookOnRepositoriesManager(pipelineProject, appName, hookID)
		if err != nil {
			sdk.Exit("Cannot delete on pipeline %s-%s-%s (%s)\n", pipelineProject, appName, pipelineName, err)
		}
		fmt.Println("✔ Success")
	} else {

		t := strings.Split(args[3], "/")
		if len(t) != 3 {
			sdk.Exit("Expected repository like <host>/<project>/<slug>. Got %d elements\n", len(t))
		}

		hooks, err := sdk.GetHooks(pipelineProject, appName, pipelineName)
		if err != nil {
			sdk.Exit("Cannot retrieve hooks from %s/%s/%s (%s)\n", pipelineProject, appName, pipelineName, err)
		}

		for _, h := range hooks {
			if h.Host == t[0] && h.Project == t[1] && h.Repository == t[2] {
				err = sdk.DeleteHook(pipelineProject, appName, pipelineName, h.ID)
				if err != nil {
					sdk.Exit("Cannot delete hook from %s/%s/%s (%s)", pipelineProject, appName, pipelineName, err)
				}
				return
			}
		}
	}
}

func listPipelineHook(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage: See %s\n", cmd.Short)
	}

	pipelineProject := args[0]
	appName := args[1]
	pipelineName := args[2]

	hooks, err := sdk.GetHooks(pipelineProject, appName, pipelineName)
	if err != nil {
		sdk.Exit("Cannot retrieve hooks from %s/%s/%s (%s)\n", pipelineProject, appName, pipelineName, err)
	}

	if showURLOnly {
		for _, h := range hooks {
			fmt.Printf("%s\n", h.Link)
		}
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Repository", "URL"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for _, h := range hooks {
		table.Append([]string{
			fmt.Sprintf("%d", h.ID),
			fmt.Sprintf("%s/%s/%s", h.Host, h.Project, h.Repository),
			h.Link,
		})
	}
	table.Render()

}
