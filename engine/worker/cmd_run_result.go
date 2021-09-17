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
	c.AddCommand(cmdRunResultAddStaticFile())
	return c
}

func cmdRunResultAddStaticFile() *cobra.Command {
	c := &cobra.Command{
		Use:   "static-file",
		Short: "worker run-result add static-file <name> <remote_url>",
		Long: `Inside a job, add a run result of type static-file:
Worker Command:

	worker run-result add static-file <name> <remote_url>

Example:

	worker run-result add static-file the-title https://the-remote-url/somewhere/index.html
`,
		Run: addStaticFileRunResultCmd(),
	}
	return c
}

func addStaticFileRunResultCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			sdk.Exit("missing arguments. Cmd: worker run-result add static-file <name> <remote_url>")
		}

		name := args[0]
		remoteURL := args[1]

		payload := sdk.WorkflowRunResultStaticFile{
			Name:      name,
			RemoteURL: remoteURL,
		}
		data, _ := json.Marshal(payload)
		addRunResult(data, sdk.WorkflowRunResultTypeStaticFile)
	}
}

func cmdRunResultAddArtifactIntegration() *cobra.Command {
	c := &cobra.Command{
		Use:   "artifact-manager",
		Short: "worker run-result add artifact-manager <artifact_name> <repository_name> <path_inside_repository>",
		Long: `Inside a job, add a run result of type artifact manager:
Worker Command:

	worker run-result add artifact-manager <artifact_name> <repository_name> <path_inside_repository>

Example:

	worker run-result add artifact-manager custom/debian:10 my-docker-repository-name custom/debian/10
`,
		Run: addArtifactManagerRunResultCmd(),
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

		payload := sdk.WorkflowRunResultArtifactManager{
			Name:     fileName,
			Perm:     0,
			Path:     filePath,
			RepoName: repositoryName,
		}
		data, _ := json.Marshal(payload)

		addRunResult(data, sdk.WorkflowRunResultTypeArtifactManager)
	}
}

func addRunResult(data []byte, stype sdk.WorkflowRunResultType) {
	portS := os.Getenv(internal.WorkerServerPort)
	if portS == "" {
		sdk.Exit("%s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
	}

	port, errPort := strconv.Atoi(portS)
	if errPort != nil {
		sdk.Exit("cannot parse '%s' as a port number", portS)
	}

	req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/run-result/add/%s", port, stype), bytes.NewBuffer(data))
	if errRequest != nil {
		sdk.Exit("cannot add run result (Request): %s\n", errRequest)
	}
	client := http.DefaultClient
	resp, errDo := client.Do(req)
	if errDo != nil {
		sdk.Exit("cannot post worker run-result (Do): %s\n", errDo)
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
