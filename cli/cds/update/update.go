package update

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/inconshreveable/go-update"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/cli/cds/internal"
)

// used by CI to inject at build time
var urlUpdateRelease string

// Cmd update
var Cmd = &cobra.Command{
	Use:     "update",
	Short:   "Update cds to the latest release version: cds update",
	Long:    `cds update`,
	Aliases: []string{"up"},
	Run: func(cmd *cobra.Command, args []string) {
		doUpdate(urlUpdateRelease, internal.Architecture)
	},
}

func getContentType(resp *http.Response) string {
	for k, v := range resp.Header {
		if k == "Content-Type" && len(v) >= 1 {
			return v[0]
		}
	}
	return ""
}

func guessArchitecture() string {
	cmd := exec.Command("uname", "-m")
	line, err := cmd.Output()
	if err != nil {
		fmt.Printf("Cannot guess architecture: %s\n", err)
		return ""
	}

	return string(line)
}

func doUpdate(url, architecture string) {
	if architecture == "" {
		fmt.Printf("You seem to have a custom build of cds\n")
		guess := guessArchitecture()
		if guess == "" {
			sdk.Exit("Please download latest release on %s\n", url)
		}

		fmt.Printf("Assuming architecture being %s\n", guess)
		architecture = guess
		url = fmt.Sprintf("%s/download/cli/%s", sdk.Host, architecture)
	}

	if internal.Verbose {
		fmt.Printf("Url to update cds: %s\n", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		sdk.Exit("Error when downloading cds: %s\n", err.Error())
		fmt.Printf("Url: %s\n", url)
		os.Exit(1)
	}

	contentType := getContentType(resp)
	if contentType != "application/octet-stream" {
		sdk.Exit("Invalid Binary (Content-Type: %s). Please try again or download it manually from %s\n", contentType, url)
		fmt.Printf("Url: %s\n", url)
		os.Exit(1)
	}

	if resp.StatusCode != 200 {
		sdk.Exit("Error http code: %d, url called: %s\n", resp.StatusCode, url)
		os.Exit(1)
	}

	fmt.Printf("Getting latest release from : %s ...\n", url)
	defer resp.Body.Close()
	if err = update.Apply(resp.Body, update.Options{}); err != nil {
		sdk.Exit("Error when updating cds: %s\n", err.Error())
		sdk.Exit("Url: %s\n", url)
		os.Exit(1)
	}
	fmt.Println("Update done.")
}
