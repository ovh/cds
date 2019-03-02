package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdCheckSecret(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "check-secret",
		Short: "worker check-secret fileA fileB",
		Long: `

Inside a step script (https://ovh.github.io/cds/manual/actions/script/), you can add check if a file contains a CDS variable of type password or private key:

` + "```bash" + `
#!/bin/bash

set -ex

# create a file
cat << EOF > myFile
this a a line in the file, with a CDS variable of type password {{.cds.app.password}}
EOF

# worker check-secret myFile
worker check-secret {{.cds.workspace}}/myFile
` + "```" + `

This command will exit 1 and a log is displayed, as:

	variable cds.app.password is used in file myFile

The command will exit 0 if no variable of type password or key is found.

		`,
		Run: tmplCheckSecretCmd(w),
	}
	return c
}

type filePath struct {
	Path string `json:"path"`
}

func tmplCheckSecretCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if len(args) == 0 {
			sdk.Exit("Wrong usage: Example: worker check-secret filea fileb filec*")
		}

		for _, file := range args {
			a := filePath{
				Path: file,
			}

			data, errMarshal := json.Marshal(a)
			if errMarshal != nil {
				sdk.Exit("internal error (%s)\n", errMarshal)
			}

			req, errRequest := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/checksecret", port), bytes.NewReader(data))
			if errRequest != nil {
				sdk.Exit("cannot post worker check-secret (Request): %s\n", errRequest)
			}

			client := http.DefaultClient
			client.Timeout = 10 * time.Minute

			resp, errDo := client.Do(req)
			if errDo != nil {
				sdk.Exit("cannot post worker check-secret (Do): %s\n", errDo)
			}

			if resp.StatusCode >= 300 {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					sdk.Exit("cannot read response body %v\n", err)
				}
				cdsError := sdk.DecodeError(body)
				if cdsError != nil {
					sdk.Exit("%v\n", cdsError)
				}
				sdk.Exit("%v\n", string(body))
			}
		}
	}
}

func (wk *currentWorker) checkSecretHandler(w http.ResponseWriter, r *http.Request) {
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sendLog := getLogger(wk, wk.currentJob.wJob.ID, wk.currentJob.currentStep)

	var a filePath
	if err := json.Unmarshal(data, &a); err != nil {
		sendLog(fmt.Sprintf("failed to unmarshal %s", data))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	btes, err := ioutil.ReadFile(a.Path)
	if err != nil {
		sendLog(fmt.Sprintf("failed to read file %s", a.Path))
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}
	sbtes := string(btes)

	var varFound string
	for _, p := range wk.currentJob.params {
		if (p.Type == sdk.SecretVariable || p.Type == sdk.KeyVariable) && len(p.Value) >= sdk.SecretMinLength && strings.Contains(sbtes, p.Value) {
			varFound = p.Name
			break
		}
	}

	if varFound != "" {
		writeByteArray(w, []byte(fmt.Sprintf("secret variable %s is used in file %s", varFound, a.Path)), http.StatusExpectationFailed)
		return
	}
	sendLog(fmt.Sprintf("no secret found in file %s", a.Path))
}
