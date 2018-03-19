package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var uptodateCmd = &cobra.Command{
	Use:   "uptodate",
	Short: "check if engine is uptodate",
	Long: `check if engine is uptodate with latest release on github (--from-github) or from an existing API. 

This command exit 0 if current binary is uptodate.
`,
	Example: "engine uptodate --from-github",
	Run: func(cmd *cobra.Command, args []string) {
		if !updateFromGithub && updateURLAPI == "" {
			sdk.Exit(`You have to use "./engine uptodate --from-github" or "./engine uptodate --api http://intance/of/your/cds/api"`)
		}

		conf := cdsclient.Config{Host: updateURLAPI}
		client := cdsclient.New(conf)

		var versionTxt string
		if updateFromGithub {
			urlVersionFile, errGH := client.DownloadURLFromGithub("VERSION")
			if errGH != nil {
				sdk.Exit("Error while getting URL from Github url:%s err:%s\n", urlVersionFile, errGH)
			}
			resp, errG := http.Get(urlVersionFile)
			if errG != nil {
				sdk.Exit("Error while getting binary from CDS API: %s\n", errG)
			}
			defer resp.Body.Close()
			respB, errR := ioutil.ReadAll(resp.Body)
			if errR != nil {
				sdk.Exit("Error while reading VERSION file: %v\n", errR)
			}
			versionTxt = strings.TrimSpace(string(respB))
		} else {
			remoteVersion, errv := client.MonVersion()
			if errv != nil {
				sdk.Exit("Error while getting version from GithubAPI err:%s\n", errv)
			}
			versionTxt = remoteVersion.Version
		}

		if versionTxt == sdk.VERSION {
			fmt.Println("uptodate:true")
			os.Exit(0)
		}
		fmt.Println("uptodate:false")
		os.Exit(1)
	},
}
