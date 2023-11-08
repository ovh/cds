package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdExport = &cobra.Command{
	Use:   "export",
	Short: "worker export <varname> <value>",
	Long: `
Inside a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), you can create a build variable with the worker command:

	worker export foo bar


then, you can use new build variable:

	echo "{{.cds.build.foo}}"


## Scope

You can use the build variable in :

* another step of the current job with ` + "`{{.cds.build.varname}}`" + `
* the next stages in same pipeline ` + "`{{.cds.build.varname}}`" + `
* the next pipelines ` + "`{{.workflow.pipelineName.build.varname}}`" + ` with ` + "`pipelineName`" + ` the name of the pipeline in your workflow

	`,
	Run: exportCmd,
}

func exportCmd(cmd *cobra.Command, args []string) {
	port := MustGetWorkerHTTPPort()

	if len(args) != 2 {
		sdk.Exit("Wrong usage: See '%s'\n", cmd.Short)
	}

	v := sdk.Variable{
		Name:  args[0],
		Type:  sdk.StringVariable,
		Value: args[1],
	}

	data, err := json.Marshal(v)
	if err != nil {
		sdk.Exit("internal error (%v)\n", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/var", port), bytes.NewReader(data))
	if err != nil {
		sdk.Exit("cannot add variable: %v\n", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		sdk.Exit("cannot add variable: %v\n", err)
	}

	if resp.StatusCode >= 300 {
		sdk.Exit("cannot add variable: HTTP %d\n", resp.StatusCode)
	}
}
