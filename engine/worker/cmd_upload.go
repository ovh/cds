package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdUploadTag string

func cmdUpload(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "upload",
		Short: "worker upload --tag=<tag> <path>",
		Run:   uploadCmd(w),
	}
	c.Flags().StringVar(&cmdUploadTag, "tag", "", "Tag for artifact Upload. Tag is mandatory")
	return c
}

func uploadCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if cmdUploadTag == "" {
			sdk.Exit("worker upload: invalid tag. %s\n", cmd.Short)
		}

		if len(args) == 0 {
			sdk.Exit("Wrong usage: Example : worker upload --tag={{.cds.version}} filea fileb filec*")
		}

		for _, arg := range args {
			a := sdk.Artifact{
				Name: arg,
				Tag:  cmdUploadTag,
			}

			data, errMarshal := json.Marshal(a)
			if errMarshal != nil {
				sdk.Exit("internal error (%s)\n", errMarshal)
			}

			req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/upload", port), bytes.NewReader(data))
			if errRequest != nil {
				sdk.Exit("cannot post worker upload (Request): %s\n", errRequest)
			}

			client := http.DefaultClient
			client.Timeout = 5 * time.Minute

			resp, errDo := client.Do(req)
			if errDo != nil {
				sdk.Exit("cannot post worker upload (Do): %s\n", errDo)
			}

			if resp.StatusCode >= 300 {
				sdk.Exit("cannot artefact upload HTTP %d\n", resp.StatusCode)
			}
		}

	}
}

func (wk *currentWorker) uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var a sdk.Artifact
	if err := json.Unmarshal(data, &a); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	action := sdk.Action{
		Parameters: []sdk.Parameter{
			{
				Name:  "path",
				Type:  sdk.StringParameter,
				Value: a.Name,
			},
			{
				Name:  "tag",
				Type:  sdk.StringParameter,
				Value: a.Tag,
			},
		},
	}

	sendLog := getLogger(wk, wk.currentJob.pbJob.ID, wk.currentJob.currentStep)

	if result := runArtifactUpload(wk)(context.Background(), &action, wk.currentJob.pbJob.ID, &wk.currentJob.pbJob.Parameters, sendLog); result.Status != sdk.StatusSuccess.String() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
