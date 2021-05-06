package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
)

func cmdRunResult() *cobra.Command {
	c := &cobra.Command{
		Use:   "run-result",
		Short: "worker run-result",
		Long:  `Inside a job, manage run result`,
	}
	c.AddCommand(cdmAddRunResult())
	return c
}

func cdmAddRunResult() *cobra.Command {
	c := &cobra.Command{
		Use:   "add",
		Short: "worker run-result add",
		Long:  `Inside a job, add a run result`,
	}
	c.AddCommand(cmdRunResultAddArtifactIntegration())
	return c
}

func cmdRunResultAddArtifactIntegration() *cobra.Command {
	c := &cobra.Command{
		Use:   "artifact-manager",
		Short: "worker run-result add",
		Long:  `Inside a job, add a run result of type artifact manager`,
		Run:   addArtifactManagerRunResultCmd(),
	}
	return c
}

func addArtifactManagerRunResultCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if len(args) != 3 {
			sdk.Exit("missing arguments. Cmd: worker run-result add artifact-manager <fileName> <repo-name> <file-path>")
		}

		fileName := args[0]
		repositoryName := args[1]
		filePath := args[2]

		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		payload := sdk.WorkflowRunResultArtifactManager{
			Name:     fileName,
			Perm:     0,
			Path:     filePath,
			RepoName: repositoryName,
		}
		data, _ := json.Marshal(payload)

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/run-result/add", port), bytes.NewBuffer(data))
		if errRequest != nil {
			sdk.Exit("cannot add run result (Request): %s\n", errRequest)
		}
		client := http.DefaultClient
		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("cannot post worker artifacts (Do): %s\n", errDo)
		}
		defer resp.Body.Close() // nolint

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cannot add run result HTTP %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			if cdsError != nil {
				sdk.Exit("adding run result failed: %v\n", cdsError)
			} else {
				sdk.Exit("adding run result failed: %s\n", body)
			}
		}

		// step: read the response body
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("add run result failed ReadAll: %v\n", err)
		}
		fmt.Println(string(respBody))
	}
}
