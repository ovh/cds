package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

var (
	cmdDownloadWorkflowName string
	cmdDownloadNumber       string
	cmdDownloadArtifactName string
	cmdDownloadTag          string
)

func cmdDownload() *cobra.Command {
	c := &cobra.Command{
		Use:   "download",
		Short: "worker download [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]",
		Long: `
Inside a job, there are two ways to download an artifact:

* with a step using action Download Artifacts
* with a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), using the worker command.

Worker Command:

	worker download --tag=<tag> <path>

Example:

	worker download --pattern="files.*.yml"

Theses two commands have the same result:

	worker download
	worker download --workflow={{.cds.workflow}} --number={{.cds.run.number}}

		`,
		Run: downloadCmd(),
	}
	c.Flags().StringVar(&cmdDownloadWorkflowName, "workflow", "", "Workflow name to download from. Optional, default: current workflow")
	c.Flags().StringVar(&cmdDownloadNumber, "number", "", "Workflow Number to download from. Optional, default: current workflow run")
	c.Flags().StringVar(&cmdDownloadArtifactName, "pattern", "", "Pattern matching files to download. Optional, default: *")
	c.Flags().StringVar(&cmdDownloadTag, "tag", "", "Tag matching files to download. Optional")
	return c
}

func downloadCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		var number int64
		if cmdDownloadNumber != "" {
			var errN error
			number, errN = strconv.ParseInt(cmdDownloadNumber, 10, 64)
			if errN != nil {
				sdk.Exit("number parameter have to be an integer")
			}
		}

		wd, _ := os.Getwd()
		a := workerruntime.DownloadArtifact{
			Workflow:    cmdDownloadWorkflowName,
			Number:      number,
			Pattern:     cmdDownloadArtifactName,
			Tag:         cmdDownloadTag,
			Destination: wd,
		}

		data, errMarshal := json.Marshal(a)
		if errMarshal != nil {
			sdk.Exit("internal error (%s)\n", errMarshal)
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/download", port), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("cannot post worker download (Request): %s\n", errRequest)
		}

		client := http.DefaultClient

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("cannot post worker download (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cannot artifact download HTTP %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("download failed: %v\n", cdsError)
		}
	}
}
