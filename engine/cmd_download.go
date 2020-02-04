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
	downloadCmd.AddCommand(downloadUICmd)
	downloadCmd.AddCommand(downloadSQLCmd)

	downloadWorkersCmd.Flags().StringVar(&flagDownloadURLAPI, "api", "", "Update binary from a CDS Engine API")
	downloadWorkersCmd.Flags().StringVarP(&flagDownloadOS, "os", "", "", "Download only for this os")
	downloadWorkersCmd.Flags().StringVarP(&flagDownloadArch, "arch", "", "", "Download only for this arch")
	downloadWorkersCmd.Flags().StringVar(&flagDownloadConfigFile, "config", "", "config file")
	downloadWorkersCmd.Flags().StringVar(&flagDownloadRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	downloadWorkersCmd.Flags().StringVar(&flagDownloadRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")

	downloadUICmd.Flags().StringVar(&flagDownloadConfigFile, "config", "", "config file")
	downloadUICmd.Flags().StringVar(&flagDownloadRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	downloadUICmd.Flags().StringVar(&flagDownloadRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")

	downloadSQLCmd.Flags().StringVar(&flagDownloadConfigFile, "config", "", "config file")
	downloadSQLCmd.Flags().StringVar(&flagDownloadRemoteConfig, "remote-config", "", "(optional) consul configuration store")
	downloadSQLCmd.Flags().StringVar(&flagDownloadRemoteConfigKey, "remote-config-key", "cds/config.api.toml", "(optional) consul configuration store key")
}

var (
	flagDownloadURLAPI          string
	flagDownloadOS              string
	flagDownloadArch            string
	flagDownloadConfigFile      string
	flagDownloadRemoteConfig    string
	flagDownloadRemoteConfigKey string
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
		conf := configImport(nil, flagDownloadConfigFile, flagDownloadRemoteConfig, flagDownloadRemoteConfigKey, "", "", false)
		if conf.API == nil {
			sdk.Exit("Invalid configuration file")
		}
		downloadTarGzFromGithub(conf.API.Directories.Download, "cds-worker-all.tar.gz")
	},
}

var downloadUICmd = &cobra.Command{
	Use:     "ui",
	Short:   "Download ui files from the latest release on GitHub",
	Long:    `Download ui files from the latest release on GitHub`,
	Example: "engine download ui",
	Run: func(cmd *cobra.Command, args []string) {
		conf := configImport(nil, flagDownloadConfigFile, flagDownloadRemoteConfig, flagDownloadRemoteConfigKey, "", "", false)
		if conf.API == nil {
			sdk.Exit("Invalid configuration file")
		}
		downloadTarGzFromGithub(conf.UI.Staticdir, "ui.tar.gz")
	},
}

var downloadSQLCmd = &cobra.Command{
	Use:     "sql",
	Short:   "Download sql files from the latest release on GitHub",
	Long:    `Download sql files from the latest release on GitHub`,
	Example: "engine download sql",
	Run: func(cmd *cobra.Command, args []string) {
		conf := configImport(nil, flagDownloadConfigFile, flagDownloadRemoteConfig, flagDownloadRemoteConfigKey, "", "", false)
		if conf.API == nil {
			sdk.Exit("Invalid configuration file")
		}
		downloadTarGzFromGithub(conf.DatabaseMigrate.Directory, "sql.tar.gz")
	},
}

func downloadTarGzFromGithub(confPath, filename string) {
	config := cdsclient.Config{Host: flagDownloadURLAPI}
	client := cdsclient.New(config)

	if ok, err := sdk.DirectoryExists(confPath); !ok {
		if err := os.MkdirAll(confPath, os.FileMode(0700)); err != nil {
			sdk.Exit("Unable to create directory %s: %v", confPath, err)
		}
		log.Info(context.Background(), "Directory %s has been created", confPath)
	} else if err != nil {
		sdk.Exit("Invalid download directory %s: %v", confPath, err)
	}

	urlBinary, err := client.DownloadURLFromGithub(filename)
	if err != nil {
		sdk.Exit("Error while getting %s from err:%s\n", filename, urlBinary, err)
	}

	fmt.Printf("Downloading into %s...\n", confPath)
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

	fullpath := path.Join(confPath, filename)
	if err := ioutil.WriteFile(fullpath, body, 0755); err != nil {
		sdk.Exit("Error while write file content for %s in %s", filename, confPath)
	}

	if err := archiver.DefaultTarGz.Unarchive(fullpath, confPath); err != nil {
		sdk.Exit("Unarchive %s failed: %v", filename, err)
	}
	fmt.Printf("Unarchive to %s\n", confPath)

	// delete file cds-worker-all.tar.gz
	if err := os.Remove(fullpath); err != nil {
		sdk.Exit("Error while deleting file %s: %v", fullpath, err)
	}

	fmt.Println("Download done.")
}
