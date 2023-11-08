package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdTag() *cobra.Command {
	c := &cobra.Command{
		Use:   "tag",
		Short: "worker tag key=value key=value",
		Long: `
On the workflow view, the sidebar on the left displays a select box to filter on CDS Tags.

So, what's a tag? A tag is a CDS Variable, exported as a tag. There are default tags as git.branch, git.hash, tiggered_by and environment.

Inside a job, you can add a Tag with the worker command:

	# worker tag <key>=<value> <key>=<value>
	worker tag tagKey=tagValue anotherTagKey=anotherTagValue


Tags are useful to add indication on the sidebar about the context of a Run.

You can select the tags displayed on the sidebar Workflow → Advanced → "Tags to display in the sidebar".

![Tag](/images/worker.commands.tag.png)
		`,
		Run: tagCmd(),
	}
	return c
}

func tagCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		port := MustGetWorkerHTTPPort()

		if len(args) == 0 {
			sdk.Exit("Wrong usage: Example : worker tag <key>=<value>")
		}

		formValues := url.Values{}
		for _, s := range args {
			t := strings.SplitN(s, "=", 2)
			if len(t) != 2 {
				sdk.Exit("Wrong usage: Example : worker tag <key>=<value>")
			}
			formValues.Set(t[0], t[1])
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/tag", port), strings.NewReader(formValues.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		if errRequest != nil {
			sdk.Exit("cannot post worker tag (Request): %v\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 5 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("http call failed: %v\n", errDo)
		}
		if resp.StatusCode >= 300 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("tag failed: unable to read body %v\n", err)
			}
			defer resp.Body.Close()
			sdk.Exit("tag failed: %s\n", string(body))
		}
	}
}
