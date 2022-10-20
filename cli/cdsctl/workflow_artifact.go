package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowArtifactCmd = cli.Command{
	Name:    "artifact",
	Aliases: []string{"artifacts"},
	Short:   "Manage Workflow Artifact",
}

func workflowArtifact() *cobra.Command {
	return cli.NewCommand(workflowArtifactCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowArtifactDownloadCmd, workflowArtifactDownloadRun, nil, withAllCommandModifiers()...),
	})
}

var workflowArtifactDownloadCmd = cli.Command{
	Name:  "download",
	Short: "Download artifacts of one Workflow Run",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "number"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "artifact-name"},
	},
	Flags: []cli.Flag{
		{
			Name:    "exclude",
			Usage:   "exclude files from download - could be a regex: *.log",
			Default: "",
		},
		{
			Name:    "cdn-url",
			Usage:   "overwrite cdn url (deprecated)",
			Default: "",
		},
	},
}

func workflowArtifactDownloadRun(v cli.Values) error {
	number, err := strconv.ParseInt(v.GetString("number"), 10, 64)
	if err != nil {
		return cli.NewError("number parameter have to be an integer")
	}

	cdnURL := v.GetString("cdn-url")
	if cdnURL != "" {
		fmt.Printf("Flag cdn-url is deprecated, use CDS_CDN_URL env variable instead\n")
	}

	cdnURL, err = client.CDNURL()
	if err != nil {
		return err
	}

	// Search in result
	results, err := client.WorkflowRunResultsList(context.Background(), v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
	if err != nil {
		return err
	}

	var reg *regexp.Regexp
	if len(v.GetString("exclude")) > 0 {
		var err error
		reg, err = regexp.Compile(v.GetString("exclude"))
		if err != nil {
			return cli.WrapError(err, "exclude parameter is not valid")
		}
	}
	var ok bool
	for _, runResult := range results {
		if runResult.Type != sdk.WorkflowRunResultTypeArtifact {
			continue
		}
		artifactData, err := runResult.GetArtifact()
		if err != nil {
			return err
		}
		if v.GetString("artifact-name") != "" && v.GetString("artifact-name") != artifactData.Name {
			continue
		}
		if v.GetString("exclude") != "" && reg.MatchString(artifactData.Name) {
			fmt.Printf("File %s is excluded from download\n", artifactData.Name)
			continue
		}
		var toDownload bool
		if _, err := os.Stat(artifactData.Name); os.IsNotExist(err) {
			toDownload = true
		} else {

			// file exists, check sha512
			var errf error
			f, errf := os.OpenFile(artifactData.Name, os.O_RDWR|os.O_CREATE, os.FileMode(artifactData.Perm))
			if errf != nil {
				_ = f.Close()
				return errf
			}
			md5Sum, err := sdk.FileMd5sum(artifactData.Name)
			if err != nil {
				_ = f.Close()
				return err
			}

			if md5Sum != artifactData.MD5 {
				toDownload = true
			} else {
				fmt.Printf("File %s already downloaded, checksum OK\n", f.Name())
			}
			if err := f.Close(); err != nil {
				return err
			}
		}

		if toDownload {
			fmt.Printf("Downloading %s...\n", artifactData.Name)
			f, err := os.OpenFile(artifactData.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(artifactData.Perm))
			if err != nil {
				return cli.NewError("unable to open file %s: %v", artifactData.Name, err)
			}
			if err := client.CDNItemDownload(context.Background(), cdnURL, artifactData.CDNRefHash, sdk.CDNTypeItemRunResult, artifactData.MD5, f); err != nil {
				_ = f.Close()
				return err
			}
			fmt.Printf("File %s created, checksum OK\n", f.Name())
			if err := f.Close(); err != nil {
				return cli.NewError("unable to close file %s: %v", artifactData.Name, err)
			}
		}
		ok = true
	}

	if !ok {
		return cli.NewError("no artifact downloaded")
	}
	return nil
}
