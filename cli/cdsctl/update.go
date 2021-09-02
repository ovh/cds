package main

import (
	"fmt"
	"net/http"

	goUpdate "github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/cli/cdsctl/internal"
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
	var variant string
	if !internal.IsKeychainEnabled() {
		variant = "nokeychain"
	}
	var urlBinary string
	if v.GetBool("from-github") {
		// no need to have apiEndpoint here
		var errGH error
		urlBinary, errGH = sdk.DownloadURLFromGithub(sdk.BinaryFilename("cdsctl", sdk.GOOS, sdk.GOARCH, variant), "latest")
		if errGH != nil {
			return cli.NewError("Error while getting URL from GitHub url:%s err:%s", urlBinary, errGH)
		}
		fmt.Printf("Updating binary from GitHub on %s...\n", urlBinary)
	} else {
		urlBinary = client.DownloadURLFromAPI("cdsctl", sdk.GOOS, sdk.GOARCH, variant)
		fmt.Printf("Updating binary from CDS API on %s...\n", urlBinary)
	}

	resp, err := http.Get(urlBinary)
	if err != nil {
		return cli.WrapError(err, "error while getting binary from CDS API")
	}
	defer resp.Body.Close()

	if err := sdk.CheckContentTypeBinary(resp); err != nil {
		return cli.NewError(err.Error())
	}

	if resp.StatusCode != 200 {
		return cli.NewError("Error http code: %d, url called: %s", resp.StatusCode, urlBinary)
	}

	if err := goUpdate.Apply(resp.Body, goUpdate.Options{}); err != nil {
		return cli.WrapError(err, "Error while updating binary from CDS API")
	}
	fmt.Println("Update cdsctl done.")
	return nil
}
