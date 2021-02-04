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

	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	feature, err := client.FeatureEnabled(sdk.FeatureCDNArtifact, map[string]string{
		"project_key": projectKey,
	})
	if err != nil {
		return nil, err
	}

	if !feature.Enabled {
		workflowArtifacts, err := client.WorkflowRunArtifacts(v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
		if err != nil {
			return nil, err
		}
		return cli.AsListResult(workflowArtifacts), nil
	}

	cdnLinks, err := client.WorkflowRunArtifactsLinks(projectKey, workflowName, number)
	if err != nil {
		return nil, err
	}

	type Artifact struct {
		Name string `cli:"name"`
		Md5  string `cli:"md5"`
	}
	arts := make([]Artifact, 0, len(cdnLinks.Items))
	for _, item := range cdnLinks.Items {
		arts = append(arts, Artifact{
			Name: item.APIRef.ToFilename(),
			Md5:  item.MD5,
		})
	}
	return cli.AsListResult(arts), nil
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

	feature, err := client.FeatureEnabled(sdk.FeatureCDNArtifact, map[string]string{
		"project_key": v.GetString(_ProjectKey),
	})
	if err != nil {
		return err
	}

	if !feature.Enabled {
		ok, err := downloadFromCDSAPI(v, number)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("No artifact downloaded")
		}
		return nil
	}

	cdnLinks, err := client.WorkflowRunArtifactsLinks(v.GetString(_ProjectKey), v.GetString(_WorkflowName), number)
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

	for _, item := range cdnLinks.Items {
		if v.GetString("artifact-name") != "" && v.GetString("artifact-name") != item.APIRef.ToFilename() {
			continue
		}
		if v.GetString("exclude") != "" && reg.MatchString(item.APIRef.ToFilename()) {
			fmt.Printf("File %s is excluded from download\n", item.APIRef.ToFilename())
			continue
		}
		apiRef, _ := item.GetCDNArtifactApiRef()
		var f *os.File
		var toDownload bool
		if _, err := os.Stat(item.APIRef.ToFilename()); os.IsNotExist(err) {
			toDownload = true
		} else {

			// file exists, check sha512
			var errf error
			f, errf = os.OpenFile(item.APIRef.ToFilename(), os.O_RDWR|os.O_CREATE, os.FileMode(apiRef.Perm))
			if errf != nil {
				return errf
			}
			md5Sum, err := sdk.FileMd5sum(item.APIRef.ToFilename())
			if err != nil {
				return err
			}

			if md5Sum != item.MD5 {
				toDownload = true
			}
		}

		if toDownload {
			var err error
			f, err = os.OpenFile(item.APIRef.ToFilename(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(apiRef.Perm))
			if err != nil {
				return err
			}
			fmt.Printf("Downloading %s...\n", item.APIRef.ToFilename())
			r, err := client.CDNItemDownload(context.Background(), cdnLinks.CDNHttpURL, item.APIRefHash, sdk.CDNTypeItemArtifact)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, r); err != nil {
				return sdk.WrapError(err, "unable to write file")
			}
			if err := f.Close(); err != nil {
				return err
			}
		}

		md5Sum, err := sdk.FileMd5sum(item.APIRef.ToFilename())
		if err != nil {
			return err
		}

		if md5Sum != item.MD5 {
			return fmt.Errorf("Invalid sha512sum \ndownloaded file:%s\n%s:%s", md5Sum, f.Name(), item.MD5)
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
			return false, fmt.Errorf("exclude parameter is not valid: %v", err)
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
