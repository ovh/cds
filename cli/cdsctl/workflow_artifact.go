package main

import (
	"context"
	"fmt"
	"io"
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

	results, err := client.WorkflowRunResultsList(context.Background(), v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
	if err != nil {
		return nil, err
	}

	type Artifact struct {
		Name string `cli:"name"`
		Md5  string `cli:"md5"`
	}

	artifacts := make([]Artifact, 0, len(workflowArtifacts))
	for _, art := range workflowArtifacts {
		artifacts = append(artifacts, Artifact{Name: art.Name, Md5: art.MD5sum})
	}
	for _, runResult := range results {
		if runResult.Type != sdk.WorkflowRunResultTypeArtifact {
			continue
		}
		artiData, err := runResult.GetArtifact()
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{Name: artiData.Name, Md5: artiData.MD5})
	}

	return cli.AsListResult(artifacts), nil
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

	confCDN, err := client.ConfigCDN()
	if err != nil {
		return err
	}
	ok, err := downloadFromCDSAPI(v, number)
	if err != nil {
		return err
	}
	if ok {
		return nil
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
		var f *os.File
		var toDownload bool
		if _, err := os.Stat(artifactData.Name); os.IsNotExist(err) {
			toDownload = true
		} else {

			// file exists, check sha512
			var errf error
			f, errf = os.OpenFile(artifactData.Name, os.O_RDWR|os.O_CREATE, os.FileMode(artifactData.Perm))
			if errf != nil {
				return errf
			}
			md5Sum, err := sdk.FileMd5sum(artifactData.Name)
			if err != nil {
				return err
			}

			if md5Sum != artifactData.MD5 {
				toDownload = true
			}
		}

		if toDownload {
			var err error
			f, err = os.OpenFile(artifactData.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(artifactData.Perm))
			if err != nil {
				return err
			}
			fmt.Printf("Downloading %s...\n", artifactData.Name)
			r, err := client.CDNItemDownload(context.Background(), confCDN.HTTPURL, artifactData.CDNRefHash, sdk.CDNTypeItemRunResult)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, r); err != nil {
				return cli.WrapError(err, "unable to write file")
			}
			if err := f.Close(); err != nil {
				return cli.WrapError(err, "unable to close file")
			}
		}

		md5Sum, err := sdk.FileMd5sum(artifactData.Name)
		if err != nil {
			return err
		}

		if md5Sum != artifactData.MD5 {
			return fmt.Errorf("Invalid md5Sum \ndownloaded file:%s\n%s:%s", md5Sum, f.Name(), artifactData.MD5)
		}

		if toDownload {
			fmt.Printf("File %s created, checksum OK\n", f.Name())
		} else {
			fmt.Printf("File %s already downloaded, checksum OK\n", f.Name())
		}

		ok = true
	}

	if !ok {
		return fmt.Errorf("no artifact downloaded")
	}
	return nil
}

func downloadFromCDSAPI(v cli.Values, number int64) (bool, error) {
	artifacts, err := client.WorkflowRunArtifacts(v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
	if err != nil {
		return false, err
	}

	var reg *regexp.Regexp
	if len(v.GetString("exclude")) > 0 {
		var err error
		reg, err = regexp.Compile(v.GetString("exclude"))
		if err != nil {
			return false, cli.WrapError(err, "exclude parameter is not valid")
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
				return ok, errf
			}
			sha512sum, err512 := sdk.FileSHA512sum(a.Name)
			if err512 != nil {
				return ok, err512
			}

			if sha512sum != a.SHA512sum {
				toDownload = true
			}
		}

		if toDownload {
			var errf error
			f, errf = os.OpenFile(a.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(a.Perm))
			if errf != nil {
				return ok, errf
			}
			fmt.Printf("Downloading %s...\n", a.Name)
			if err := client.WorkflowNodeRunArtifactDownload(v.GetString(_ProjectKey), v.GetString(_WorkflowName), a, f); err != nil {
				return ok, err
			}
			if err := f.Close(); err != nil {
				return ok, err
			}
		}

		sha512sum, err512 := sdk.FileSHA512sum(a.Name)
		if err512 != nil {
			return ok, err512
		}

		if sha512sum != a.SHA512sum {
			return ok, fmt.Errorf("Invalid sha512sum \ndownloaded file:%s\n%s:%s", sha512sum, f.Name(), a.SHA512sum)
		}

		md5sum, errmd5 := sdk.FileMd5sum(a.Name)
		if errmd5 != nil {
			return ok, errmd5
		}

		if md5sum != a.MD5sum {
			return ok, fmt.Errorf("Invalid md5sum \ndownloaded file:%s\n%s:%s", md5sum, f.Name(), a.MD5sum)
		}

		if toDownload {
			fmt.Printf("File %s created, checksum OK\n", f.Name())
		} else {
			fmt.Printf("File %s already downloaded, checksum OK\n", f.Name())
		}

		ok = true
	}
	return ok, nil
}
