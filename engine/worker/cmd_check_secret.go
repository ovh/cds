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

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/ovh/cds/engine/worker/internal"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdCheckSecret() *cobra.Command {
	c := &cobra.Command{
		Use:   "check-secret",
		Short: "worker check-secret fileA fileB",
		Long: `

Inside a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), you can add check if a file contains a CDS variable of type password or private key:

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
		Run: tmplCheckSecretCmd(),
	}
	return c
}

func tmplCheckSecretCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if len(args) == 0 {
			sdk.Exit("Wrong usage: Example: worker check-secret filea fileb filec*")
		}

		for _, file := range args {
			a := workerruntime.FilePath{
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
				body, err := io.ReadAll(resp.Body)
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
