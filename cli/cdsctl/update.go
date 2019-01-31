package main

import (
	"fmt"
	"net/http"

	goUpdate "github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

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
			Type:    cli.FlagBool,
		},
	},
}

func update() *cobra.Command {
	return cli.NewCommand(updateCmd, updateRun, nil, cli.CommandWithoutExtraFlags)
}

func updateRun(v cli.Values) error {
	fmt.Println(sdk.VersionString())

	var urlBinary string
	if v.GetBool("from-github") {
		// no need to have apiEndpoint here
		var errGH error
		urlBinary, errGH = client.DownloadURLFromGithub(sdk.GetArtifactFilename("cdsctl", sdk.GOOS, sdk.GOARCH))
		if errGH != nil {
			return fmt.Errorf("Error while getting URL from Github url:%s err:%s", urlBinary, errGH)
		}
		fmt.Printf("Updating binary from Github on %s...\n", urlBinary)
	} else {
		urlBinary = client.DownloadURLFromAPI("cdsctl", sdk.GOOS, sdk.GOARCH)
		fmt.Printf("Updating binary from CDS API on %s...\n", urlBinary)
	}

	resp, err := http.Get(urlBinary)
	if err != nil {
		return fmt.Errorf("error while getting binary from CDS API: %v", err)
	}
	defer resp.Body.Close()

	if err := sdk.CheckContentTypeBinary(resp); err != nil {
		return fmt.Errorf(err.Error())
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Error http code: %d, url called: %s", resp.StatusCode, urlBinary)
	}

	if err := goUpdate.Apply(resp.Body, goUpdate.Options{}); err != nil {
		return fmt.Errorf("Error while updating binary from CDS API: %s", err)
	}
	fmt.Println("Update cdsctl done.")
	return nil
}
