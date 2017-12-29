package main

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var cmdDownloadFromGithub bool

func cmdUpdate(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update worker from CDS API",
		Run:   updateCmd(w),
	}
	c.Flags().BoolVar(&cmdDownloadFromGithub, "from-github", false, "Update binary from latest github release")
	return c
}

func updateCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fmt.Printf("CDS Worker version:%s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)

		var urlBinary string

		if !cmdDownloadFromGithub {
			w.apiEndpoint = viper.GetString("api")
			if w.apiEndpoint == "" {
				sdk.Exit("--api not provided, aborting.")
			}
			w.client = cdsclient.NewWorker(w.apiEndpoint, "download")

			urlBinary = w.client.DownloadURLFromAPI("worker", runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Updating worker binary from CDS API on %s...\n", urlBinary)
		} else {
			// no need to have apiEndpoint here
			w.client = cdsclient.NewWorker("", "download")

			var errGH error
			urlBinary, errGH = w.client.DownloadURLFromGithub("worker", runtime.GOOS, runtime.GOARCH)
			if errGH != nil {
				sdk.Exit("Error while getting URL from Github: %s", errGH)
			}
			fmt.Printf("Updating worker binary from Github on %s...\n", urlBinary)
		}

		resp, err := http.Get(urlBinary)
		if err != nil {
			sdk.Exit("Error while getting binary from CDS API: %s\n", err)
		}
		defer resp.Body.Close()

		if contentType := getContentType(resp); contentType != "application/octet-stream" {
			sdk.Exit("Invalid Binary (Content-Type: %s). Please try again or download it manually from %s\n", contentType, sdk.URLGithubReleases)
		}

		if resp.StatusCode != 200 {
			sdk.Exit("Error http code: %d, url called: %s\n", resp.StatusCode, urlBinary)
		}

		if err := update.Apply(resp.Body, update.Options{}); err != nil {
			sdk.Exit("Error while getting updating worker from CDS API: %s\n", err)
		}
		fmt.Printf("Update worker done. Please restart your worker\n")
	}
}

func getContentType(resp *http.Response) string {
	for k, v := range resp.Header {
		if k == "Content-Type" && len(v) >= 1 {
			return v[0]
		}
	}
	return ""
}
