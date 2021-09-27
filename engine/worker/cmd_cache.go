package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
)

func cmdCache() *cobra.Command {
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
	}
	cmdCacheRoot.AddCommand(cmdCachePush(), cmdCachePull())

	return cmdCacheRoot
}

var cmdStorageIntegrationName string

func cmdCachePush() *cobra.Command {
	c := &cobra.Command{
		Use:     "push",
		Aliases: []string{"upload"},
		Short:   "worker cache push tagValue {{.cds.workspace}}/pathToUpload",
		Long: `
Inside a project, you can create a cache from your worker with a tag (useful for vendors for example)
	worker cache push <tagValue> dir/file

You can use you storage integration:
	worker cache push --destination=MyStorageIntegration  <tagValue> dir/file
		`,
		Example: "worker cache push {{.cds.workflow}}-{{.cds.version}} ./pathToUpload",
		Run:     cachePushCmd(),
	}
	c.Flags().StringVar(&cmdStorageIntegrationName, "destination", "", "optional. Your storage integration name")
	return c
}

func cachePushCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("worker cache push > %s not found, are you running inside a CDS worker job?", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("worker cache push > Cannot parse '%s' as a port number : %s", portS, errPort)
		}

		if len(args) < 2 {
			sdk.Exit("worker cache push > Wrong usage: Example : worker cache push myTagValue filea fileb filec")
		}

		files := make([]string, len(args)-1)
		for i, arg := range args[1:] {
			files[i] = arg
		}

		cwd, err := os.Getwd()
		if err != nil {
			sdk.Exit("worker cache push > Cannot find working directory : %s", err)
		}

		c := sdk.Cache{
			Tag:              base64.RawURLEncoding.EncodeToString([]byte(args[0])),
			Files:            files,
			WorkingDirectory: cwd,
			IntegrationName:  cmdStorageIntegrationName,
		}

		data, errMarshal := json.Marshal(c)
		if errMarshal != nil {
			sdk.Exit("worker cache push > internal error (%s)", errMarshal)
		}

		fmt.Printf("Worker cache push in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest(
			"POST",
			fmt.Sprintf("http://127.0.0.1:%d/cache/push", port),
			bytes.NewReader(data),
		)
		if errRequest != nil {
			sdk.Exit("worker cache push > cannot post worker cache push (Request): %s", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 30 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("worker cache push > cannot post worker cache push (Do): %s", errDo)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cache push HTTP %d error %v", resp.StatusCode, err)
			}
			var sdkErr sdk.Error
			if sdk.JSONUnmarshal(body, &sdkErr); err != nil {
				sdk.Exit("unable to read error: %s: %v", string(body), err)
			}
			sdk.Exit("%v", sdkErr)
		}

		fmt.Printf("Worker cache push with success (tag: %s)\n", args[0])
	}
}

func cmdCachePull() *cobra.Command {
	c := &cobra.Command{
		Use:     "pull",
		Aliases: []string{"download"},
		Short:   "worker cache pull tagValue",
		Long: `
Inside a project, you can fetch a cache from your worker with a tag

	worker cache pull <tagValue>

If you push a cache with:

	worker cache push latest {{.cds.workspace}}/pathToUpload

The command:

	worker cache pull latest

will create the directory {{.cds.workspace}}/pathToUpload with the content of the cache

If you want to push a cache into a storage integration:

	worker cache push latest --from=MyStorageIntegration {{.cds.workspace}}/pathToUpload

		`,
		Run: cachePullCmd(),
	}
	c.Flags().StringVar(&cmdStorageIntegrationName, "from", "", "optional. Your storage integration name")
	return c
}

func cachePullCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("worker cache pull > %s not found, are you running inside a CDS worker job?", internal.WorkerServerPort)
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
			sdk.Exit("worker cache pull > cannot get current path: %s", err)
		}

		fmt.Printf("Worker cache pull in progress... (tag: %s)\n", args[0])
		req, errRequest := http.NewRequest(
			"GET",
			fmt.Sprintf("http://127.0.0.1:%d/cache/%s/pull?path=%s&integration=%s", port,
				base64.RawURLEncoding.EncodeToString([]byte(args[0])),
				url.QueryEscape(dir),
				url.QueryEscape(cmdStorageIntegrationName)),
			nil,
		)
		if errRequest != nil {
			sdk.Exit("worker cache pull > cannot post worker cache pull with tag %s (Request): %s", args[0], errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 10 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("worker cache pull > cannot post worker cache pull (Do): %s", errDo)
		}

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("cache pull HTTP %d error %v", resp.StatusCode, err)
			}
			var sdkErr sdk.Error
			if sdk.JSONUnmarshal(body, &sdkErr); err != nil {
				sdk.Exit("unable to read error: %s: %v", string(body), err)
			}
			sdk.Exit("%v", sdkErr)
		}

		fmt.Printf("Worker cache pull with success (tag: %s)\n", args[0])
	}
}
