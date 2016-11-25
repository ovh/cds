package pipeline

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineCommitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commits",
		Short: "cds pipeline commits <projectKey> <appName> <pipelineName> [envName] <buildNumber> ",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 4 {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}

			pk := args[0]
			app := args[1]
			name := args[2]
			var env string
			var bnS string
			if len(args) > 4 {
				env = args[3]
				bnS = args[4]
			} else {
				bnS = args[3]
			}

			bn, err := strconv.Atoi(bnS)
			if err != nil {
				sdk.Exit("%s is not a valid build number (%s)\n", bnS, err)
			}

			commits, err := sdk.GetPipelineCommits(pk, app, name, env, bn)
			if err != nil {
				sdk.Exit("Error : %s", err)
			}
			for _, c := range commits {
				date := time.Unix(c.Timestamp/1000, 0)
				message := strings.Split(c.Message, "\n")[0]
				fmt.Printf("\nCommit:\n - Date: %s \n - Hash: %s\n - Author: %s<%s>\n - Message: %s\n", date.Format(time.RFC1123), c.Hash, c.Author.Name, c.Author.Email, message)
			}
		},
	}

	return cmd
}
