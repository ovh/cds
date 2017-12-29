package main

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"

	"github.com/inconshreveable/go-update"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var updateCmd = cli.Command{
	Name:  "update",
	Short: "Update cdsctl from CDS API or from CDS Release",
	Flags: []cli.Flag{
		{
			Name:    "from-github",
			Usage:   "Update binary from latest github release",
			Default: "false",
			Kind:    reflect.Bool,
		},
	},
}

func updateRun(v cli.Values) error {
	fmt.Printf("CDS cdsctl version:%s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)

	var urlBinary string
	if v.GetBool("from-github") {
		// no need to have apiEndpoint here
		var errGH error
		urlBinary, errGH = client.DownloadURLFromGithub("cdsctl", runtime.GOOS, runtime.GOARCH)
		if errGH != nil {
			return fmt.Errorf("Error while getting URL from Github url:%s err:%s", urlBinary, errGH)
		}
		fmt.Printf("Updating binary from Github on %s...\n", urlBinary)
	} else {
		urlBinary = client.DownloadURLFromAPI("cdsctl", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Updating binary from CDS API on %s...\n", urlBinary)
	}

	resp, err := http.Get(urlBinary)
	if err != nil {
		return fmt.Errorf("Error while getting binary from CDS API: %s\n", err)
	}
	defer resp.Body.Close()

	if err := sdk.CheckContentTypeBinary(resp); err != nil {
		return fmt.Errorf(err.Error())
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Error http code: %d, url called: %s", resp.StatusCode, urlBinary)
	}

	if err := update.Apply(resp.Body, update.Options{}); err != nil {
		return fmt.Errorf("Error while updating binary from CDS API: %s", err)
	}
	fmt.Println("Update cdsctl done.")
	return nil
}
