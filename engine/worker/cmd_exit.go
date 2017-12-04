package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdExit(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "exit",
		Short: "worker exit",
		Long:  "worker exit command lets job finish current step and disabled all further steps",
		Run:   exitCmd(w),
	}
	return c
}

func exitCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/exit", port), nil)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		if errRequest != nil {
			sdk.Exit("cannot post worker exit (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 5 * time.Second

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("command failed: %v\n", errDo)
		}

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("tag failed: unable to read body %v\n", err)
			}
			defer resp.Body.Close()
			cdsError := sdk.DecodeError(body)
			sdk.Exit("exit failed: %v\n", cdsError)
		}
	}
}

func (wk *currentWorker) exitHandler(w http.ResponseWriter, r *http.Request) {
	wk.manualExit = true
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
