package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdRequirementsFormat string

func cmdRequirements(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "requirements",
		Short: "worker requirements --format json|csv|keyval",
		Run:   requirementsCmd(w),
	}
	c.Flags().StringVar(&cmdRequirementsFormat, "format", "json", "Output format. json, csv, keyval")
	return c
}

func requirementsCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		req, errRequest := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/requirements", port), nil)
		req.Header.Add("Content-Type", "application/json")
		if errRequest != nil {
			sdk.Exit("cannot post worker tag (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 1 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("command failed: %v\n", errDo)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			sdk.Exit("Get Requirements failed: unable to read body %v\n", err)
		}
		defer resp.Body.Close()

		var reqs []sdk.Requirement
		if err := json.Unmarshal(body, &reqs); err != nil {
			sdk.Exit("Get Requirements failed: unable to unmarshal body %v\n", err)
		}

		if resp.StatusCode < 300 {
			switch cmdRequirementsFormat {
			case "json":
				fmt.Printf("body: %+v\n", string(body))
			case "csv":
				printCSVRequirements(reqs)
			case "keyval":
				printKeyValRequirements(reqs)
			}
			return
		}
		cdsError := sdk.DecodeError(body)
		sdk.Exit("Get Requirements failed: %v\n", cdsError)
	}
}

func printCSVRequirements(requirements []sdk.Requirement) {
	for _, req := range requirements {
		fmt.Printf("%s;%s;%s\n", req.Type, req.Name, req.Value)
	}
}

func printKeyValRequirements(requirements []sdk.Requirement) {
	for _, req := range requirements {
		fmt.Printf("%s=%s\n", req.Name, req.Value)
	}
}

func (wk *currentWorker) requirementsHandler(w http.ResponseWriter, r *http.Request) {
	requirements, errR := wk.client.Requirements()
	if errR != nil {
		writeError(w, nil, errR)
		return
	}
	writeJSON(w, requirements, 200)
}
