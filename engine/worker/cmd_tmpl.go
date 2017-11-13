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

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdTmpl(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "tmpl",
		Short: "worker tmpl <input file> <output file>",
		Run:   tmplCmd(w),
	}
	return c
}

type tmplPath struct {
	Path        string `json:"path"`
	Destination string `json:"destination"`
}

func tmplCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if len(args) != 2 {
			sdk.Exit("Wrong usage: Example : worker tmpl filea fileb")
		}

		a := tmplPath{args[0], args[1]}

		data, errMarshal := json.Marshal(a)
		if errMarshal != nil {
			sdk.Exit("internal error (%s)\n", errMarshal)
		}

		req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/tmpl", port), bytes.NewReader(data))
		if errRequest != nil {
			sdk.Exit("cannot post worker tmpl (Request): %s\n", errRequest)
		}

		client := http.DefaultClient
		client.Timeout = 5 * time.Minute

		resp, errDo := client.Do(req)
		if errDo != nil {
			sdk.Exit("tmpl call failed: %v", errDo)
		}

		if resp.StatusCode >= 300 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("tmpl failed: unable to read body %v\n", err)
			}
			cdsError := sdk.DecodeError(body)
			sdk.Exit("tmpl failed: %v\n", cdsError)
		}
	}
}

func (wk *currentWorker) tmplHandler(w http.ResponseWriter, r *http.Request) {
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
		writeError(w, r, newError)
		return
	}

	var a tmplPath
	if err := json.Unmarshal(data, &a); err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}

	btes, err := ioutil.ReadFile(a.Path)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}

	vars := sdk.ParametersToMap(wk.currentJob.params)

	res, err := sdk.Interpolate(string(btes), vars)
	if err != nil {
		log.Error("Unable to interpolate: %v", err)
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}

	if err := ioutil.WriteFile(a.Destination, []byte(res), os.FileMode(0644)); err != nil {
		writeError(w, r, err)
		return
	}
}
