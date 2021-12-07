package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func cmdCDSVersionSet() *cobra.Command {
	c := &cobra.Command{
		Use:   "set-version",
		Short: "Override {{.cds.version}} value with given string. This value should be unique for the workflow and can't be changed when set.",
		Run:   cdsVersionSetCmd(),
	}
	return c
}

func cdsVersionSetCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || args[0] == "" {
			sdk.Exit("invalid given value for CDS version")
		}

		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?", internal.WorkerServerPort)
		}

		port, err := strconv.Atoi(portS)
		if err != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		a := workerruntime.CDSVersionSet{
			Value: args[0],
		}

		data, err := json.Marshal(a)
		if err != nil {
			sdk.Exit("%v", err)
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/version", port), bytes.NewReader(data))
		if err != nil {
			sdk.Exit("cannot post set version (Request): %s", err)
		}

		client := http.DefaultClient
		client.Timeout = 5 * time.Minute

		resp, err := client.Do(req)
		if err != nil {
			sdk.Exit("command failed: %v", err)
		}

		if resp.StatusCode >= 300 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("set version failed: unable to read body %v", err)
			}
			defer resp.Body.Close()
			sdk.Exit("SetVersion failed: %s", string(body))
		}
		fmt.Printf("CDS version set to %s", a.Value)
	}
}
