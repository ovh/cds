package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var flagDownloadOS string
var flagDownloadArch string

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download binaries",
	Long:  "Download binaries",
}

var downloadWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Download workers binaries from latest release on Github",
	Long: `Download workers binaries from latest release on Github

You can also indicate a specific os or architecture to not download all binaries available with flag --os and --arch`,
	Example: "engine download workers",
	Run: func(cmd *cobra.Command, args []string) {
		//Initialize config
		configBootstrap(args)
		configSetDefaults()
		config(args)

		config := cdsclient.Config{Host: updateURLAPI}
		client := cdsclient.New(config)
		resources := sdk.AllDownloadableResources()

		var workersResources []sdk.DownloadableResource
		for _, resource := range resources {
			if resource.Name == "worker" {
				goodArch := flagDownloadArch == resource.Arch
				goodOS := flagDownloadOS == resource.OS

				switch {
				case flagDownloadArch == "" && flagDownloadOS == "":
					workersResources = append(workersResources, resource)
				case flagDownloadArch != "" && flagDownloadOS == "":
					if goodArch {
						workersResources = append(workersResources, resource)
					}
				case flagDownloadArch == "" && flagDownloadOS != "":
					if goodOS {
						workersResources = append(workersResources, resource)
					}
				default:
					if goodArch && goodOS {
						workersResources = append(workersResources, resource)
					}
				}
			}
		}

		for _, workerResource := range workersResources {
			filename := sdk.GetArtifactFilename("worker", workerResource.OS, workerResource.Arch, "")
			// no need to have apiEndpoint here
			urlBinary, errGH := client.DownloadURLFromGithub(filename)
			if errGH != nil {
				if flagDownloadArch != "" && flagDownloadOS != "" {
					sdk.Exit("Error while getting URL from Github url:%s err:%s\nIf it's not available on Github release you should consider compile yourself\n", urlBinary, errGH)
				}
				continue
			}

			fmt.Printf("Downloading worker for os %s and arch %s into %s...\n", workerResource.OS, workerResource.Arch, conf.API.Directories.Download)
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

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				sdk.Exit("Error while reading file content for %s", filename)
			}

			if err := ioutil.WriteFile(path.Join(conf.API.Directories.Download, filename), body, 0755); err != nil {
				sdk.Exit("Error while write file content for %s in %s", filename, conf.API.Directories.Download)
			}
		}

		fmt.Println("Download workers binaries done.")
	},
}

func init() {
	downloadCmd.AddCommand(downloadWorkersCmd)

	downloadWorkersCmd.Flags().StringVarP(&flagDownloadOS, "os", "", "", "Download only for this os")
	downloadWorkersCmd.Flags().StringVarP(&flagDownloadArch, "arch", "", "", "Download only for this arch")
	downloadWorkersCmd.Flags().StringVar(&cfgFile, "config", "", "config file")
	downloadWorkersCmd.Flags().StringVar(&remoteCfg, "remote-config", "", "(optional) consul configuration store")
	downloadWorkersCmd.Flags().StringVar(&remoteCfgKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
}
