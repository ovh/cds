package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	workflowArtifactCmd = cli.Command{
		Name:  "artifact",
		Short: "Manage Workflow Artifact",
	}

	workflowArtifact = cli.NewCommand(workflowArtifactCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(workflowArtifactListCmd, workflowArtifactListRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(workflowArtifactDownloadCmd, workflowArtifactDownloadRun, nil, withAllCommandModifiers()...),
		})
)

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
	number, err := strconv.ParseInt(v["number"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("number parameter have to be an integer")
	}
	workflowArtifacts, err := client.WorkflowRunArtifacts(v[_ProjectKey], v[_WorkflowName], number)
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
		{Name: "artefact-name"},
	},
}

func workflowArtifactDownloadRun(v cli.Values) error {
	number, err := strconv.ParseInt(v["number"], 10, 64)
	if err != nil {
		return fmt.Errorf("number parameter have to be an integer")
	}

	artifacts, err := client.WorkflowRunArtifacts(v[_ProjectKey], v[_WorkflowName], number)
	if err != nil {
		return err
	}

	var ok bool
	for _, a := range artifacts {
		if v["artefact-name"] != "" && v["artefact-name"] != a.Name {
			continue
		}
		f, err := os.OpenFile(a.Name, os.O_RDWR|os.O_CREATE, os.FileMode(a.Perm))
		if err != nil {
			return err
		}
		fmt.Printf("Downloading %s...\n", a.Name)
		if err := client.WorkflowNodeRunArtifactDownload(v[_ProjectKey], v[_WorkflowName], a, f); err != nil {
			return err
		}

		sha512sum, err512 := sdk.FileSHA512sum(a.Name)
		if err512 != nil {
			return err512
		}

		if err := f.Close(); err != nil {
			return err
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

		fmt.Printf("File %s created, checksum OK\n", f.Name())
		ok = true
	}

	if !ok {
		return fmt.Errorf("No artifact downloaded")
	}
	return nil
}
