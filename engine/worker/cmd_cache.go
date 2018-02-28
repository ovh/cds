package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdCache(w *currentWorker) *cobra.Command {
	cmdCacheRoot := &cobra.Command{
		Use: "cache",
		Long: `
  Inside a project, you can create or retrieve a cache from your worker with a tag
    `,
		Short: "Inside a project, you can create or retrieve a cache from your worker with a tag",
	}
	cmdCacheRoot.AddCommand(cmdCachePush(w))

	return cmdCacheRoot
}

func cmdCachePush(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "push",
		Aliases: []string{"upload"},
		Short:   "worker cache push tagValue {{.cds.workspace}}/pathToUpload",
		Long: `
Inside a project, you can create a cache from your worker with a tag
		`,
		Run: cachePushCmd(w),
	}
	return c
}

func cachePushCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if len(args) < 2 {
			sdk.Exit("Wrong usage: Example : worker cache push myTagValue filea fileb filec*")
		}

		c := sdk.Cache{
			Tag:   args[0],
			Files: args[1:],
		}

		data, errMarshal := json.Marshal(c)
		if errMarshal != nil {
			sdk.Exit("internal error (%s)\n", errMarshal)
		}

		fmt.Printf("Worker cache push in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/cache/%s/push", port, args[0]), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("cannot post worker cache push (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 10 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("cannot post worker cache push (Do): %s\n", errDo)
		}

		if resp.StatusCode >= 300 {
			sdk.Exit("cannot cache push HTTP %d\n", resp.StatusCode)
		}

		fmt.Printf("Worker cache push with success (tag: %s)\n", args[0])
	}
}

func (wk *currentWorker) cachePushHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var c sdk.Cache
	if err := json.Unmarshal(data, &c); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sendLog := getLogger(wk, wk.currentJob.wJob.ID, wk.currentJob.currentStep)

	sendLog(fmt.Sprintf("%v", c))

	res, errTar := sdk.CreateTarFromPaths(c.Files)
	if errTar != nil {
		sendLog("Error: cannot tar : " + errTar.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if wk.currentJob.wJob == nil {
		sendLog("Error: cannot find workflow job info")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	sendLog(fmt.Sprintf("%s %s", projectKey, vars["tag"]))
	if err := wk.client.WorkflowCachePush(projectKey, vars["tag"], res); err != nil {
		sendLog(fmt.Sprintf("Cannot push cache: %s", err))
	}
}
