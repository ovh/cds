package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdTag(w *currentWorker) *cobra.Command {
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
		Run: tagCmd(w),
	}
	return c
}

func tagCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

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
			sdk.Exit("cannot post worker tag (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 5 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("command failed: %v\n", errDo)
		}

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("tag failed: unable to read body %v\n", err)
			}
			defer resp.Body.Close()
			cdsError := sdk.DecodeError(body)
			sdk.Exit("tag failed: %v\n", cdsError)
		}
	}
}

func (wk *currentWorker) tagHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // Parses the request body
	tags := []sdk.WorkflowRunTag{}
	for k := range r.Form {
		tags = append(tags, sdk.WorkflowRunTag{
			Tag:   k,
			Value: r.Form.Get(k),
		})
	}

	if err := wk.client.QueueJobTag(wk.currentJob.wJob.ID, tags); err != nil {
		writeError(w, r, err)
		return
	}
}
