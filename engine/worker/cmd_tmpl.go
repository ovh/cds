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

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
)

func cmdTmpl() *cobra.Command {
	c := &cobra.Command{
		Use:   "tmpl",
		Short: "worker tmpl inputFile outputFile",
		Long: `

Inside a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), you can add a replace CDS variables with the real value into a file:

	# create a file
	cat << EOF > myFile
	this a a line in the file, with a CDS variable {{.cds.version}}
	EOF

	# worker tmpl <input file> <output file>
	worker tmpl {{.cds.workspace}}/myFile {{.cds.workspace}}/outputFile


The file ` + "`outputFile`" + ` will contain the string:

	this a a line in the file, with a CDS variable 2


if it's the RUN n°2 of the current workflow.
		`,
		Run: tmplCmd(),
	}
	return c
}

func tmplCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		portS := os.Getenv(internal.WorkerServerPort)
		if portS == "" {
			sdk.Exit("%s not found, are you running inside a CDS worker job?\n", internal.WorkerServerPort)
		}

		port, errPort := strconv.Atoi(portS)
		if errPort != nil {
			sdk.Exit("cannot parse '%s' as a port number", portS)
		}

		if len(args) != 2 {
			sdk.Exit("Wrong usage: Example : worker tmpl filea fileb")
		}

		a := workerruntime.TmplPath{args[0], args[1]}

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
