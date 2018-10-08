package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func cmdUpdate(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "worker update [flags]",
		Long: `Update worker from CDS API or from CDS Release

Update from Github:

		worker update --from-github

Update from your CDS API:

		worker update --api https://your-cds-api.localhost
		`,
		Run: updateCmd(w),
	}
	c.Flags().Bool(flagFromGithub, false, "Update binary from latest github release")
	c.Flags().String(flagAPI, "", "URL of CDS API")
	c.Flags().Bool(flagInsecure, false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)
	return c
}

func updateCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fmt.Println(sdk.VersionString())
		var urlBinary string
		if !FlagBool(cmd, "from-github") {
			w.apiEndpoint = FlagString(cmd, flagAPI)
			if w.apiEndpoint == "" {
				sdk.Exit("--api not provided, aborting update.")
			}
			w.client = cdsclient.NewWorker(w.apiEndpoint, "download", cdsclient.NewHTTPClient(time.Second*360, FlagBool(cmd, flagInsecure)))
			urlBinary = w.client.DownloadURLFromAPI("worker", sdk.GOOS, sdk.GOARCH)
			fmt.Printf("Updating worker binary from CDS API on %s...\n", urlBinary)
		} else {
			// no need to have apiEndpoint here
			w.client = cdsclient.NewWorker("", "download", nil)

			var errGH error
			urlBinary, errGH = w.client.DownloadURLFromGithub(sdk.GetArtifactFilename("worker", sdk.GOOS, sdk.GOARCH))
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
		fmt.Println("Update worker done.")
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
