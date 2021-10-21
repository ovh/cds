package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

var cmdUploadTag string

func cmdUpload() *cobra.Command {
	c := &cobra.Command{
		Use:   "upload",
		Short: "worker upload {{.cds.workspace}}/fileToUpload",
		Long: `
Inside a job, there are two ways to upload an artifact:

* with a step using action Upload Artifacts
* with a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), using the worker command: ` + "`worker upload <path>`" + `

` + "`worker upload --tag={{.cds.version}} {{.cds.workspace}}/files*.yml`" + `

You can use you storage integration:
	worker upload --destination="yourStorageIntegrationName"
		`,
		Run: uploadCmd(),
	}
	c.Flags().StringVar(&cmdUploadTag, "tag", "", "Tag for artifact Upload - Tag is mandatory")
	c.Flags().StringVar(&cmdStorageIntegrationName, "destination", "", "optional. Your storage integration name")
	return c
}

func uploadCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if len(args) == 0 {
			sdk.Exit("Wrong usage: Example : worker upload --tag={{.cds.version}} filea fileb filec*")
		}

		cwd, _ := os.Getwd()
		for _, arg := range args {
			a := workerruntime.UploadArtifact{
				Name:             arg,
				Tag:              cmdUploadTag,
				WorkingDirectory: cwd,
			}

			data, errMarshal := json.Marshal(a)
			if errMarshal != nil {
				sdk.Exit("internal error (%s)\n", errMarshal)
			}

			req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/upload?integration=%s", port, url.QueryEscape(cmdStorageIntegrationName)), bytes.NewReader(data))
			if errRequest != nil {
				sdk.Exit("cannot post worker upload (Request): %s\n", errRequest)
			}

			client := http.DefaultClient
			client.Timeout = 30 * time.Minute

			resp, errDo := client.Do(req)
			if errDo != nil {
				sdk.Exit("cannot post worker upload (Do): %s\n", errDo)
			}

			if resp.StatusCode >= 300 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					sdk.Exit("cannot artifact upload HTTP %v\n", err)
				}
				cdsError := sdk.DecodeError(body)
				sdk.Exit("artifact upload failed: %v\n", cdsError)
			}
		}
	}
}
