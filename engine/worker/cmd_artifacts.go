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

func cmdArtifacts() *cobra.Command {
	c := &cobra.Command{
		Use:   "artifacts",
		Short: "worker artifacts [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]",
		Long: `
Inside a job, you can list artifacts of a workflow:

	worker artifacts --pattern="files.*.yml"

	#theses two commands have the same result:
	worker artifacts
	worker artifacts --workflow={{.cds.workflow}} --number={{.cds.run.number}}

		`,
		Run: artifactsCmd(),
	}
	c.Flags().StringVar(&cmdDownloadWorkflowName, "workflow", "", "Workflow name. Optional, default: current workflow")
	c.Flags().StringVar(&cmdDownloadNumber, "number", "", "Workflow Number. Optional, default: current workflow run")
	c.Flags().StringVar(&cmdDownloadArtifactName, "pattern", "", "Pattern matching files to list. Optional, default: *")
	c.Flags().StringVar(&cmdDownloadTag, "tag", "", "Tag matching files to list. Optional")

	return c
}

func artifactsCmd() func(cmd *cobra.Command, args []string) {
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

		a := workerruntime.DownloadArtifact{
			Workflow: cmdDownloadWorkflowName,
			Number:   number,
			Pattern:  cmdDownloadArtifactName,
			Tag:      cmdDownloadTag,
		}

		data, errMarshal := json.Marshal(a)
		if errMarshal != nil {
			sdk.Exit("internal error (%s)\n", errMarshal)
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/artifacts", port), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("cannot post worker artifacts (Request): %s\n", errRequest)
		}

		client := http.DefaultClient

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("cannot post worker artifacts (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cannot list artifacts HTTP %v\n", err)
			}
			sdk.Exit("artifacts failed: %s\n", string(body))
		}

		// step: read the response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("artifacts failed ReadAll: %v\n", err)
		}
		fmt.Println(string(respBody))
	}
}
