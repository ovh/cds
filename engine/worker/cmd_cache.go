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
	"github.com/ovh/cds/sdk/log"
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
			sdk.Exit("worker cache push > Cannot parse '%s' as a port number : %s\n", portS, errPort)
		}

		if len(args) < 2 {
			sdk.Exit("worker cache push > Wrong usage: Example : worker cache push myTagValue filea fileb filec\n")
		}

		// check tag name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(args[0]) {
			sdk.Exit("worker cache push > Wrong tag pattern, must satisfy %s\n", sdk.NamePattern)
		}

		files := make([]string, len(args)-1)
		for i, arg := range args[1:] {
			absPath, err := filepath.Abs(arg)
			if err != nil {
				sdk.Exit("worker cache push > cannot have absolute path for (%s) : %s\n", absPath, err)
			}
			files[i] = absPath
		}

		c := sdk.Cache{
			Tag:   args[0],
			Files: files,
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
		errRead = sdk.Error{
			Message: "worker cache push > Cannot read body : " + errRead.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%s", errRead)
		writeError(w, r, errRead)
		return
	}

	var c sdk.Cache
	if err := json.Unmarshal(data, &c); err != nil {
		err = sdk.Error{
			Message: "worker cache push > Cannot unmarshall body : " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%s", err)
		writeError(w, r, err)
		return
	}

	res, errTar := sdk.CreateTarFromPaths(c.Files)
	if errTar != nil {
		errTar = sdk.Error{
			Message: "worker cache push > Cannot tar : " + errTar.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%s", errTar)
		writeError(w, r, errTar)
		return
	}
	if wk.currentJob.wJob == nil {
		errW := sdk.Error{
			Message: "worker cache push > Cannot find workflow job info",
			Status:  http.StatusInternalServerError,
		}
		log.Error("%s", errW)
		writeError(w, r, errW)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	if err := wk.client.WorkflowCachePush(projectKey, vars["tag"], res); err != nil {
		err = sdk.Error{
			Message: "worker cache push > Cannot push cache : " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%s", err)
		writeError(w, r, err)
		return
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
			sdk.Exit("worker cache pull > cannot parse '%s' as a port number: %s", portS, errPort)
		}

		if len(args) < 1 {
			sdk.Exit("worker cache pull > Wrong usage: Example : worker cache pull myTagValue")
		}

		fmt.Printf("Worker cache pull in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/cache/%s/pull", port, args[0]), nil)
		if errRequest != nil {
			sdk.Exit("worker cache pull > cannot post worker cache pull with tag %s (Request): %s\n", args[0], errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 10 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("worker cache pull > cannot post worker cache pull (Do): %s\n", errDo)
		}

		if resp.StatusCode >= 300 {
			var errorMsg string
			defer resp.Body.Close()
			if data, errRead := ioutil.ReadAll(resp.Body); errRead == nil {
				var httpErr sdk.Error
				if err := json.Unmarshal(data, &httpErr); err == nil {
					errorMsg = httpErr.Message
				}
			}

			sdk.Exit("worker cache pull > Cannot cache pull HTTP ERROR %d : %s\n", resp.StatusCode, errorMsg)
		}

		fmt.Printf("Worker cache pull with success (tag: %s)\n", args[0])
	}
}

func (wk *currentWorker) cachePullHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if wk.currentJob.wJob == nil {
		errW := fmt.Errorf("worker cache pull > Cannot find workflow job info")
		writeError(w, r, errW)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	bts, err := wk.client.WorkflowCachePull(projectKey, vars["tag"])
	if err != nil {
		err = sdk.Error{
			Message: "worker cache pull > Cannot push cache : " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		writeError(w, r, err)
		return
	}

	tr := tar.NewReader(bts)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			err = sdk.Error{
				Message: "worker cache pull > Unable to read tar file : " + err.Error(),
				Status:  http.StatusBadRequest,
			}
			writeError(w, r, err)
			return
		}

		if hdr == nil {
			continue
		}

		currentDir, err := os.Getwd()
		if err != nil {
			err = sdk.Error{
				Message: "worker cache pull > Unable to get current directory : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			writeError(w, r, err)
			return
		}
		// the target location where the dir/file should be created
		target := filepath.Join(currentDir, hdr.Name)

		if _, errS := os.Stat(filepath.Dir(target)); errS != nil {
			if errM := os.MkdirAll(filepath.Dir(target), 0755); errM != nil {
				errM = sdk.Error{
					Message: "worker cache pull > Cannot create directory : " + errM.Error(),
					Status:  http.StatusInternalServerError,
				}
				writeError(w, r, errM)
				return
			}
		}

		f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
		if err != nil {
			err = sdk.Error{
				Message: "worker cache pull > Cannot create file : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			writeError(w, r, err)
			return
		}

		// copy over contents
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			err = sdk.Error{
				Message: "worker cache pull > Cannot copy content file : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			writeError(w, r, err)
			return
		}
		f.Close()
	}
}
