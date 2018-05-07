package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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

	artifactsFiltered := sdk.ArtifactsGetUniqueNameAndLatest(artifacts)

	var ok bool
	for _, a := range artifactsFiltered {
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
		if err := f.Close(); err != nil {
			return err
		}
		fileForMD5, errop := os.Open(a.GetName())
		if errop != nil {
			return errop
		}
		//Compute md5sum
		hash := md5.New()
		if _, errcopy := io.Copy(hash, fileForMD5); errcopy != nil {
			return errcopy
		}
		hashInBytes := hash.Sum(nil)[:16]
		md5sumStr := hex.EncodeToString(hashInBytes)
		fileForMD5.Close()
		if md5sumStr != a.MD5sum {
			return fmt.Errorf("Invalid md5sum \ndownloaded file:%s\n%s:%s", md5sumStr, f.Name(), a.MD5sum)
		}

		fmt.Printf("File %s created, checksum OK\n", f.Name())
		ok = true
	}

	if !ok {
		return fmt.Errorf("No artifact downloaded")
	}
	return nil
}
