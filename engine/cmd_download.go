package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/rockbears/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
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
		downloadTarGzFromGithub(context.Background(), conf.API.Download.Directory, "cds-worker-all.tar.gz")
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
		if conf.UI == nil {
			sdk.Exit("Invalid configuration file - missing ui section")
		}
		downloadTarGzFromGithub(context.Background(), conf.UI.Staticdir, "ui.tar.gz")
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
		if conf.DatabaseMigrate == nil {
			sdk.Exit("Invalid configuration file - missing databaseMigrate section")
		}
		downloadTarGzFromGithub(context.Background(), conf.DatabaseMigrate.Directory, "sql.tar.gz")
	},
}

func downloadTarGzFromGithub(ctx context.Context, confPath, filename string) {
	if ok, err := sdk.DirectoryExists(confPath); !ok {
		if err := os.MkdirAll(confPath, os.FileMode(0700)); err != nil {
			sdk.Exit("Unable to create directory %s: %v", confPath, err)
		}
		log.Info(context.Background(), "Directory %s has been created", confPath)
	} else if err != nil {
		sdk.Exit("Invalid download directory %s: %v", confPath, err)
	}

	if err := sdk.DownloadFromGitHub(ctx, confPath, filename, "latest"); err != nil {
		sdk.Exit("Downloading %s failed: %v", filename, err)
	}

	fullpath := path.Join(confPath, filename)
	src, err := os.Open(fullpath)
	if err != nil {
		sdk.Exit("Unable to open source file %s failed: %v", fullpath, err)
	}
	defer src.Close()

	if err := sdk.UntarGz(afero.NewOsFs(), confPath, src); err != nil {
		sdk.Exit("Unarchive %s failed: %v", filename, err)
	}
	fmt.Printf("Unarchive to %s\n", confPath)

	// delete file cds-worker-all.tar.gz
	if err := os.Remove(fullpath); err != nil {
		sdk.Exit("Error while deleting file %s: %v", fullpath, err)
	}

	fmt.Println("Download done.")
}
