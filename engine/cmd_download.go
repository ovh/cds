package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/mholt/archiver"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	downloadCmd.AddCommand(downloadWorkersCmd)

	downloadWorkersCmd.Flags().StringVar(&flagDownloadWorkersURLAPI, "api", "", "Update binary from a CDS Engine API")
	downloadWorkersCmd.Flags().StringVarP(&flagDownloadWorkersOS, "os", "", "", "Download only for this os")
	downloadWorkersCmd.Flags().StringVarP(&flagDownloadWorkersArch, "arch", "", "", "Download only for this arch")
	downloadWorkersCmd.Flags().StringVar(&flagDownloadWorkersConfigFile, "config", "", "config file")
	downloadWorkersCmd.Flags().StringVar(&flagDownloadWorkersRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	downloadWorkersCmd.Flags().StringVar(&flagDownloadWorkersRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
}

var (
	flagDownloadWorkersURLAPI          string
	flagDownloadWorkersOS              string
	flagDownloadWorkersArch            string
	flagDownloadWorkersConfigFile      string
	flagDownloadWorkersRemoteConfig    string
	flagDownloadWorkersRemoteConfigKey string
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download binaries",
	Long:  "Download binaries",
}

var downloadWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "Download workers binaries from the latest release on GitHub",
	Long: `Download workers binaries from the latest release on GitHub

You can also indicate a specific os or architecture to not download all binaries available with flag --os and --arch`,
	Example: "engine download workers",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize config
		conf := configImport(nil, flagDownloadWorkersConfigFile, flagDownloadWorkersRemoteConfig, flagDownloadWorkersRemoteConfigKey, "", "", false)

		config := cdsclient.Config{Host: flagDownloadWorkersURLAPI}
		client := cdsclient.New(config)

		if conf.API == nil {
			sdk.Exit("Invalid configuration file")
		}

		if ok, err := sdk.DirectoryExists(conf.API.Directories.Download); !ok {
			if err := os.MkdirAll(conf.API.Directories.Download, os.FileMode(0700)); err != nil {
				sdk.Exit("Unable to create directory %s: %v", conf.API.Directories.Download, err)
			}
			log.Info(context.Background(), "Directory %s has been created", conf.API.Directories.Download)
		} else if err != nil {
			sdk.Exit("Invalid download directory %s: %v", conf.API.Directories.Download, err)
		}

		filename := "cds-worker-all.tar.gz"
		urlBinary, err := client.DownloadURLFromGithub(filename)
		if err != nil {
			sdk.Exit("Error while getting %s from err:%s\n", filename, urlBinary, err)
		}

		fmt.Printf("Downloading workers into %s...\n", conf.API.Directories.Download)
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

		fullpath := path.Join(conf.API.Directories.Download, filename)
		if err := ioutil.WriteFile(fullpath, body, 0755); err != nil {
			sdk.Exit("Error while write file content for %s in %s", filename, conf.API.Directories.Download)
		}

		if err := archiver.DefaultTarGz.Unarchive(fullpath, conf.API.Directories.Download); err != nil {
			sdk.Exit("Unarchive %s failed: %v", filename, err)
		}
		fmt.Printf("Unarchive to %s\n", conf.API.Directories.Download)

		// delete file cds-worker-all.tar.gz
		if err := os.Remove(fullpath); err != nil {
			sdk.Exit("Error while deleting file %s: %v", fullpath, err)
		}

		fmt.Println("Download workers binaries done.")
	},
}
