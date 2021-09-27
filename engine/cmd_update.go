package main

import (
	"fmt"
	"net/http"

	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func init() {
	updateCmd.Flags().BoolVar(&flagUpdateFromGithub, "from-github", false, "Update binary from latest github release")
	updateCmd.Flags().StringVar(&flagUpdateURLAPI, "api", "", "Update binary from a CDS Engine API")
}

var (
	flagUpdateFromGithub bool
	flagUpdateURLAPI     string
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update engine binary",
	Example: "engine update --from-github",
	Run: func(cmd *cobra.Command, args []string) {

		if !flagUpdateFromGithub && flagUpdateURLAPI == "" {
			sdk.Exit(`You have to use "./engine update --from-github" or "./engine update --api http://intance/of/your/cds/api"`)
		}

		var urlBinary string
		conf := cdsclient.Config{Host: flagUpdateURLAPI}
		client := cdsclient.New(conf)

		fmt.Println(sdk.VersionString())

		if flagUpdateFromGithub {
			// no need to have apiEndpoint here
			var errGH error
			urlBinary, errGH = sdk.DownloadURLFromGithub(sdk.BinaryFilename("engine", sdk.GOOS, sdk.GOARCH, ""), "latest")
			if errGH != nil {
				sdk.Exit("Error while getting URL from Github url:%s err:%s\n", urlBinary, errGH)
			}
			fmt.Printf("Updating binary from Github on %s...\n", urlBinary)
		} else {
			urlBinary = client.DownloadURLFromAPI("engine", sdk.GOOS, sdk.GOARCH, "")
			fmt.Printf("Updating binary from CDS API on %s...\n", urlBinary)
		}

		resp, err := http.Get(urlBinary)
		if err != nil {
			sdk.Exit("Error while getting binary from: %s err:%s\n", urlBinary, err)
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
