package main

import (
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
		cli.NewListCommand(workflowArtifactListCmd, workflowArtifactListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowArtifactDownloadCmd, workflowArtifactDownloadRun, nil, withAllCommandModifiers()...),
	})
}

var workflowArtifactListCmd = cli.Command{
	Name:  "list",
	Short: "List artifacts of one Workflow Run",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{Name: "number"},
	},
}

func workflowArtifactListRun(v cli.Values) (cli.ListResult, error) {
	number, err := strconv.ParseInt(v.GetString("number"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("number parameter have to be an integer")
	}
	workflowArtifacts, err := client.WorkflowRunArtifacts(v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workflowArtifacts), nil
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
	},
}

func workflowArtifactDownloadRun(v cli.Values) error {
	number, err := strconv.ParseInt(v.GetString("number"), 10, 64)
	if err != nil {
		return fmt.Errorf("number parameter have to be an integer")
	}

	artifacts, err := client.WorkflowRunArtifacts(v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
	if err != nil {
		return err
	}

	var reg *regexp.Regexp
	if len(v.GetString("exclude")) > 0 {
		var err error
		reg, err = regexp.Compile(v.GetString("exclude"))
		if err != nil {
			return fmt.Errorf("exclude parameter is not valid: %v", err)
		}
	}

	var ok bool
	for _, a := range artifacts {
		if v.GetString("artifact-name") != "" && v.GetString("artifact-name") != a.Name {
			continue
		}
		if v.GetString("exclude") != "" && reg.MatchString(a.Name) {
			fmt.Printf("File %s is excluded from download\n", a.Name)
			continue
		}

		var f *os.File
		var toDownload bool
		if _, err := os.Stat(a.Name); os.IsNotExist(err) {
			toDownload = true
		} else {
			// file exists, check sha512
			var errf error
			f, errf = os.OpenFile(a.Name, os.O_RDWR|os.O_CREATE, os.FileMode(a.Perm))
			if errf != nil {
				return errf
			}
			sha512sum, err512 := sdk.FileSHA512sum(a.Name)
			if err512 != nil {
				return err512
			}

			if sha512sum != a.SHA512sum {
				toDownload = true
			}
		}

		if toDownload {
			var errf error
			f, errf = os.OpenFile(a.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
			if errf != nil {
				return errf
			}
			fmt.Printf("Downloading %s...\n", a.Name)
			if err := client.WorkflowNodeRunArtifactDownload(v.GetString(_ProjectKey), v.GetString(_WorkflowName), a, f); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}

		sha512sum, err512 := sdk.FileSHA512sum(a.Name)
		if err512 != nil {
			return err512
		}

		if sha512sum != a.SHA512sum {
			return fmt.Errorf("Invalid sha512sum \ndownloaded file:%s\n%s:%s", sha512sum, f.Name(), a.SHA512sum)
		}

		md5sum, errmd5 := sdk.FileMd5sum(a.Name)
		if errmd5 != nil {
			return errmd5
		}

		if md5sum != a.MD5sum {
			return fmt.Errorf("Invalid md5sum \ndownloaded file:%s\n%s:%s", md5sum, f.Name(), a.MD5sum)
		}

		if toDownload {
			fmt.Printf("File %s created, checksum OK\n", f.Name())
		} else {
			fmt.Printf("File %s already downloaded, checksum OK\n", f.Name())
		}

		ok = true
	}

	if !ok {
		return fmt.Errorf("No artifact downloaded")
	}
	return nil
}
