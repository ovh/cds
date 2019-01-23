package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

func adminCurl() *cobra.Command {
	return cli.NewCommand(adminCurlCmd, adminCurlFunc, nil)
}

var adminCurlCmd = cli.Command{
	Name:  "curl",
	Short: "Execute request to CDS api",
	Args: []cli.Arg{
		{
			Name: "path",
		},
	},
	Flags: []cli.Flag{
		{
			Type:      cli.FlagString,
			Name:      "request",
			ShortHand: "X",
			Default:   http.MethodGet,
		},
		{
			Type:      cli.FlagString,
			Name:      "data",
			ShortHand: "d",
		},
	},
}

func adminCurlFunc(v cli.Values) error {
	var rdata io.Reader

	data := v.GetString("data")
	if data != "" {
		rdata = bytes.NewReader([]byte(data))
	}

	res, _, _, err := client.(cdsclient.Raw).Request(context.Background(), v.GetString("request"), v.GetString("path"), rdata)
	if err != nil {
		return err
	}

	fmt.Println(string(res))

	return nil
}
