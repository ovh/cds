package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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
  Inside a project, you can create or retrieve a cache from your worker with a tag (useful for vendors for example)

	You can access to this cache from any workflow inside a project. You just have to choose a tag that fits with your needs.

	For example if you need a different cache for each workflow so choose a tag scoped with your workflow name and workflow version (example of tag value: {{.cds.workflow}}-{{.cds.version}})
    `,
		Short: "Inside a project, you can create or retrieve a cache from your worker with a tag",
	}
	cmdCacheRoot.AddCommand(cmdCachePush(w), cmdCachePull(w))

	return cmdCacheRoot
}

func cmdCachePush(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "push",
		Aliases: []string{"upload"},
		Short:   "worker cache push tagValue {{.cds.workspace}}/pathToUpload",
		Long: `
Inside a project, you can create a cache from your worker with a tag (useful for vendors for example)
		`,
		Example: "worker cache push {{.cds.workflow}}-{{.cds.version}} {{.cds.workspace}}/pathToUpload",
		Run:     cachePushCmd(w),
	}
	return c
}

func cachePushCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("worker cache push > %s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("worker cache push > Cannot parse '%s' as a port number", portS)
		}

		if len(args) < 2 {
			sdk.Exit("worker cache push > Wrong usage: Example : worker cache push myTagValue filea fileb filec")
		}

		c := sdk.Cache{
			Tag:   args[0],
			Files: args[1:],
		}

		data, errMarshal := json.Marshal(c)
		if errMarshal != nil {
			sdk.Exit("worker cache push > internal error (%s)\n", errMarshal)
		}

		fmt.Printf("Worker cache push in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/cache/%s/push", port, args[0]), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("worker cache push > cannot post worker cache push (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 10 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("worker cache push > cannot post worker cache push (Do): %s\n", errDo)
		}

		if resp.StatusCode >= 300 {
			sdk.Exit("worker cache push > Cannot cache push HTTP ERROR %d\n", resp.StatusCode)
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

	res, errTar := sdk.CreateTarFromPaths(c.Files)
	if errTar != nil {
		sendLog("worker cache push > Cannot tar : " + errTar.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if wk.currentJob.wJob == nil {
		sendLog("worker cache push > Cannot find workflow job info")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	if err := wk.client.WorkflowCachePush(projectKey, vars["tag"], res); err != nil {
		sendLog(fmt.Sprintf("worker cache push > Cannot push cache: %s", err))
	}
}

func cmdCachePull(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "pull",
		Aliases: []string{"download"},
		Short:   "worker cache pull tagValue",
		Long: `
Inside a project, you can fetch a cache from your worker with a tag
		`,
		Run: cachePullCmd(w),
	}
	return c
}

func cachePullCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("worker cache pull > %s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("worker cache pull > cannot parse '%s' as a port number", portS)
		}

		if len(args) < 1 {
			sdk.Exit("worker cache pull > Wrong usage: Example : worker cache pull myTagValue")
		}

		fmt.Printf("Worker cache pull in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/cache/%s/pull", port, args[0]), nil)
		if errRequest != nil {
			sdk.Exit("worker cache pull > cannot post worker cache pull (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 10 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("worker cache pull > cannot post worker cache pull (Do): %s\n", errDo)
		}

		if resp.StatusCode >= 300 {
			sdk.Exit("worker cache pull > Cannot cache pull HTTP ERROR %d\n", resp.StatusCode)
		}

		fmt.Printf("Worker cache pull with success (tag: %s)\n", args[0])
	}
}

func (wk *currentWorker) cachePullHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sendLog := getLogger(wk, wk.currentJob.wJob.ID, wk.currentJob.currentStep)

	if wk.currentJob.wJob == nil {
		sendLog("worker cache pull > Cannot find workflow job info")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	bts, err := wk.client.WorkflowCachePull(projectKey, vars["tag"])
	if err != nil {
		sendLog(fmt.Sprintf("worker cache pull > Cannot push cache: %s", err))
	}

	tr := tar.NewReader(bts)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			sendLog(sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("worker cache pull > Unable to read tar file")).Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if hdr == nil {
			continue
		}

		currentDir, err := os.Getwd()
		if err != nil {
			sendLog("worker cache pull > Unable to get current directory")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// the target location where the dir/file should be created
		target := filepath.Join(currentDir, hdr.Name)

		if _, err := os.Stat(filepath.Dir(target)); err != nil {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				sendLog(sdk.WrapError(err, "worker cache pull > Cannot create directory %s ", target).Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
		if err != nil {
			sendLog(sdk.WrapError(err, "worker cache pull > Cannot create file %s ", target).Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// copy over contents
		if _, err := io.Copy(f, tr); err != nil {
			sendLog(sdk.WrapError(err, "worker cache pull > Cannot copy content file ").Error())
			w.WriteHeader(http.StatusInternalServerError)
			f.Close()
			return
		}
		f.Close()
	}
}
