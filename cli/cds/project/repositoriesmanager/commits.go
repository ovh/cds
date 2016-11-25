package repositoriesmanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func getCommitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commits",
		Short: "cds project reposmanager commits <projectKey> <repositories manager> <repoFullname> [<since> [<until>]]",
		Long:  ``,
		Run:   getCommits,
	}

	return cmd
}

func getCommits(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	rmName := args[1]
	repoFullname := args[2]

	var since, until string
	if len(args) > 3 {
		since = args[3]
	}
	if len(args) > 4 {
		until = args[4]
	}

	commits, err := sdk.GetCommits(projectKey, rmName, repoFullname, since, until)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	for _, c := range commits {
		date := time.Unix(c.Timestamp/1000, 0)
		message := strings.Split(c.Message, "\n")[0]
		fmt.Printf("\nCommit:\n - Date: %s \n - Hash: %s\n - Author: %s<%s>\n - Message: %s\n", date.Format(time.RFC1123), c.Hash, c.Author.Name, c.Author.Email, message)
	}

}
