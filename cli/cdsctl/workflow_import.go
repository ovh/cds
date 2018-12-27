package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var workflowImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a workflow",
	Long: `
In case you want to import just your workflow. Instead of use a local file you can also use an URL to your yaml file.
		
If you want to update also dependencies likes pipelines, applications or environments at same time you have to use workflow push instead workflow import.

	`,
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "path"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Override workflow if exists",
			Default: "false",
		},
	},
}

func workflowImportRun(c cli.Values) error {
	var contentFile io.Reader
	path := c.GetString("path")
	format := "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	}

	if isURL, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, path); isURL {
		var errF error
		contentFile, _, errF = exportentities.OpenURL(path, format)
		if errF != nil {
			return errF
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		contentFile = f
	}

	msgs, err := client.WorkflowImport(c.GetString(_ProjectKey), contentFile, format, c.GetBool("force"))
	if err != nil {
		return err
	}

	for _, s := range msgs {
		fmt.Println(s)
	}

	return nil
}
