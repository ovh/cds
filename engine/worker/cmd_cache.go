package main

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
Inside a project, you can create or retrieve a cache from your worker with a tag (useful for vendors for example).

You can access to this cache from any workflow inside a project. You just have to choose a tag that fits with your needs.

For example if you need a different cache for each workflow so choose a tag scoped with your workflow name and workflow version (example of tag value: {{.cds.workflow}}-{{.cds.version}})

## Use Case
Java Developers often use maven to manage dependencies. The mvn install command could be long because all the maven dependencies have to be downloaded on a fresh CDS Job workspace.
With the worker cache feature, you don't have to download the dependencies if they haven't been updated since the last run of the job.


- cache push: take the current .m2/ directory and set it as a cache
- cache pull: download a cache of .m2 directory

Here, an example of a script inside a CDS Job using the cache feature:

	#!/bin/bash

	tag=($(md5sum pom.xml))

	# download the cache of .m2/
	if worker cache pull $tag; then
		echo ".m2/ getted from cache";
	fi

	# update the directory .m2/
	# as there is a cache, mvn does not need to download all dependencies
	# if they are not updated on upstream
	mvn install

	# put in cache the updated .m2/ directory
	worker cache push $tag .m2/


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
	worker push <tagValue> dir/file
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

		files := make([]string, len(args)-1)
		for i, arg := range args[1:] {
			absPath, err := filepath.Abs(arg)
			if err != nil {
				sdk.Exit("worker cache push > cannot have absolute path for (%s) : %s\n", absPath, err)
			}
			files[i] = absPath
		}

		cwd, err := os.Getwd()
		if err != nil {
			sdk.Exit("worker cache push > Cannot find working directory : %s\n", err)
		}

		c := sdk.Cache{
			Tag:              args[0],
			Files:            files,
			WorkingDirectory: cwd,
		}

		data, errMarshal := json.Marshal(c)
		if errMarshal != nil {
			sdk.Exit("worker cache push > internal error (%s)\n", errMarshal)
		}

		fmt.Printf("Worker cache push in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest(
			"POST",
			fmt.Sprintf("http://127.0.0.1:%d/cache/%s/push", port, base64.RawURLEncoding.EncodeToString([]byte(args[0]))),
			bytes.NewReader(data),
		)
		if errRequest != nil {
			sdk.Exit("worker cache push > cannot post worker cache push (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 30 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("worker cache push > cannot post worker cache push (Do): %s\n", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cache push HTTP error %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("Error: http code %d : %v\n", resp.StatusCode, cdsError)
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
		log.Error("%v", errRead)
		writeError(w, r, errRead)
		return
	}

	var c sdk.Cache
	if err := json.Unmarshal(data, &c); err != nil {
		err = sdk.Error{
			Message: "worker cache push > Cannot unmarshall body : " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", err)
		writeError(w, r, err)
		return
	}

	res, errTar := sdk.CreateTarFromPaths(c.WorkingDirectory, c.Files)
	if errTar != nil {
		errTar = sdk.Error{
			Message: "worker cache push > Cannot tar : " + errTar.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", errTar)
		writeError(w, r, errTar)
		return
	}
	if wk.currentJob.wJob == nil {
		errW := sdk.Error{
			Message: "worker cache push > Cannot find workflow job info",
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", errW)
		writeError(w, r, errW)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	if projectKey == "" {
		errP := sdk.Error{
			Message: "worker cache push > Cannot find project",
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", errP)
		writeError(w, r, errP)
		return
	}

	var errPush error
	for i := 0; i < 10; i++ {
		if errPush = wk.client.WorkflowCachePush(projectKey, vars["ref"], res); errPush == nil {
			return
		}
		time.Sleep(3 * time.Second)
		log.Error("worker cache push > cannot push cache (retry x%d) : %v", i, errPush)
	}

	err := sdk.Error{
		Message: "worker cache push > Cannot push cache : " + errPush.Error(),
		Status:  http.StatusInternalServerError,
	}
	log.Error("%v", err)
	writeError(w, r, err)
}

func cmdCachePull(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:     "pull",
		Aliases: []string{"download"},
		Short:   "worker cache pull tagValue",
		Long: `
Inside a project, you can fetch a cache from your worker with a tag

	worker pull <tagValue>

If you push a cache with:

	worker cache push latest {{.cds.workspace}}/pathToUpload

The command:

	worker cache pull latest

will create the directory {{.cds.workspace}}/pathToUpload with the content of the cache

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

		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			sdk.Exit("worker cache pull > cannot get current path: %s\n", err)
		}

		fmt.Printf("Worker cache pull in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest(
			"GET",
			fmt.Sprintf("http://127.0.0.1:%d/cache/%s/pull?path=%s", port, base64.RawURLEncoding.EncodeToString([]byte(args[0])), url.QueryEscape(dir)),
			nil,
		)
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
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cache pull HTTP error %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("Error: %v -> %s\n", cdsError, string(body))
		}

		fmt.Printf("Worker cache pull with success (tag: %s)\n", args[0])
	}
}

func (wk *currentWorker) cachePullHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	path := r.FormValue("path")

	if wk.currentJob.wJob == nil {
		errW := fmt.Errorf("worker cache pull > Cannot find workflow job info")
		writeError(w, r, errW)
		return
	}
	params := wk.currentJob.wJob.Parameters
	projectKey := sdk.ParameterValue(params, "cds.project")
	bts, err := wk.client.WorkflowCachePull(projectKey, vars["ref"])
	if err != nil {
		err = sdk.Error{
			Message: "worker cache pull > Cannot pull cache : " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		writeError(w, r, err)
		return
	}

	tr := tar.NewReader(bts)
	for {
		header, errH := tr.Next()
		if errH == io.EOF {
			break
		}

		if errH != nil {
			errH = sdk.Error{
				Message: "worker cache pull > Unable to read tar file : " + errH.Error(),
				Status:  http.StatusBadRequest,
			}
			writeJSON(w, errH, http.StatusBadRequest)
			return
		}

		if header == nil {
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(path, header.Name)

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				fmt.Println("create directory ", target)
				if err := os.MkdirAll(target, 0755); err != nil {
					err = sdk.Error{
						Message: "worker cache pull > Unable to mkdir all files : " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
					writeJSON(w, err, http.StatusInternalServerError)
					return
				}
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				err = sdk.Error{
					Message: "worker cache pull > Unable to create symlink : " + err.Error(),
					Status:  http.StatusInternalServerError,
				}
				writeJSON(w, err, http.StatusInternalServerError)
				return
			}

			// if it's a file create it
		case tar.TypeReg, tar.TypeLink:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				err = sdk.Error{
					Message: "worker cache pull > Unable to open file : " + err.Error(),
					Status:  http.StatusInternalServerError,
				}
				writeJSON(w, err, http.StatusInternalServerError)
				return
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				err = sdk.Error{
					Message: "worker cache pull > Cannot copy content file : " + err.Error(),
					Status:  http.StatusInternalServerError,
				}
				writeJSON(w, err, http.StatusInternalServerError)
				return
			}

			_ = f.Close()
		}
	}
}
