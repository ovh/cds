package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/inconshreveable/go-update"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/cobra"
)

var updateFromGithub bool
var updateURLAPI string

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update engine binary",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CDS engine version:%s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)

		if !updateFromGithub && updateURLAPI == "" {
			sdk.Exit(`You have to use "./engine update --from-github" or "./engine update --api http://intance/of/your/cds/api"`)
		}

		var urlBinary string
		conf := cdsclient.Config{Host: updateURLAPI}
		client := cdsclient.New(conf)
		if updateFromGithub {
			// no need to have apiEndpoint here
			var errGH error
			urlBinary, errGH = client.DownloadURLFromGithub("engine", runtime.GOOS, runtime.GOARCH)
			if errGH != nil {
				sdk.Exit("Error while getting URL from Github url:%s err:%s\n", urlBinary, errGH)
			}
			fmt.Printf("Updating binary from Github on %s...\n", urlBinary)
		} else {
			urlBinary = client.DownloadURLFromAPI("engine", runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Updating binary from CDS API on %s...\n", urlBinary)
		}

		resp, err := http.Get(urlBinary)
		if err != nil {
			sdk.Exit("Error while getting binary from CDS API: %s\n", err)
		}
		defer resp.Body.Close()

		if err := sdk.CheckContentTypeBinary(resp); err != nil {
			sdk.Exit(err.Error())
		}

		if resp.StatusCode != 200 {
			sdk.Exit("Error http code: %d, url called: %s\n", resp.StatusCode, urlBinary)
		}

		if err := update.Apply(resp.Body, update.Options{}); err != nil {
			sdk.Exit("Error while updating binary from CDS API: %s\n", err)
		}
		fmt.Println("Update engine done.")
	},
}
