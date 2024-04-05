package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
)

var experimentalWorkflowResultCmd = cli.Command{
	Name:    "results",
	Short:   "CDS Experimental workflow results commands",
	Aliases: []string{"result", "rs"},
}

func experimentalWorkflowResult() *cobra.Command {
	return cli.NewCommand(experimentalWorkflowResultCmd, nil, []*cobra.Command{
		cli.NewListCommand(workflowV2RunResultListCmd, workflowV2RunResultListFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowV2RunResultDownloadCmd, workflowV2RunResultDownloadFunc, nil, withAllCommandModifiers()...),
	})
}

var workflowV2RunResultDownloadCmd = cli.Command{
	Name:    "download",
	Aliases: []string{"dl", "get"},
	Short:   "Download a run result",
	Example: "cdsctl experimental workflow results dl <project_key> <run_identifier> ",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
	Flags: []cli.Flag{
		{Name: "pattern"},
	},
}

func workflowV2RunResultDownloadFunc(v cli.Values) error {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	results, err := client.WorkflowV2RunResultList(context.Background(), projKey, runIdentifier)
	if err != nil {
		return err
	}

	pattern := v.GetString("pattern")

	for _, r := range results {

		if pattern != "" {
			g := glob.New(pattern)
			result, err := g.MatchString(r.Name())
			if err != nil {
				return err
			}
			if result == nil {
				fmt.Printf("Ignoring %s\n", r.Name())
				continue
			}
		}

		var fileName string
		var perm fs.FileMode = fs.ModePerm
		var md5 string
		var artifactManagerPath, artifactManagerRepo string
		var projectIntegrationName string
		var cdnHTTPUrl, cdnAPIRefHash string

		if r.ArtifactManagerIntegrationName != nil {
			projectIntegrationName = *r.ArtifactManagerIntegrationName
			pi, err := client.ProjectIntegrationGet(projKey, projectIntegrationName, false)
			if err != nil {
				return err
			}
			if pi.Model.Name != sdk.ArtifactoryIntegration.Name {
				return fmt.Errorf("unable to download an artifact from integration %s\n", projectIntegrationName)
			}
		}

		switch r.Type {
		case sdk.V2WorkflowRunResultTypeArsenalDeployment, sdk.V2WorkflowRunResultTypeDocker, sdk.V2WorkflowRunResultTypeVariable:
			fmt.Printf("Run result %s of type %s cannot be downloaded\n", r.Name(), r.Type)
			return nil
		default:
			cdnHTTPUrl = r.ArtifactManagerMetadata.Get("cdn_http_url")
			cdnAPIRefHash = r.ArtifactManagerMetadata.Get("cdn_api_ref_hash")
			artifactManagerPath = r.ArtifactManagerMetadata.Get("path")
			artifactManagerRepo = r.ArtifactManagerMetadata.Get("repository")

			if artifactManagerRepo != "" {
				fileName = r.ArtifactManagerMetadata.Get("name")
				md5 = r.ArtifactManagerMetadata.Get("md5")
			} else {
				detail, _ := r.GetDetailAsV2WorkflowRunResultGenericDetail()
				if detail != nil {
					perm = detail.Mode
					if fileName == "" {
						fileName = detail.Name
					}
					md5 = detail.MD5
				}
			}
			if fileName == "" {
				return fmt.Errorf("enable to download result %s. Missing filename property", r.Name())
			}
		}

		var f *os.File
		var toDownload bool
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			toDownload = true
		} else {
			// file exists, check md5
			var errf error
			f, errf = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, os.FileMode(perm))
			if errf != nil {
				return errf
			}
			md5Sum, err := sdk.FileMd5sum(fileName)
			if err != nil {
				return err
			}
			if md5Sum != md5 {
				toDownload = true
			}
			_ = f.Close()
		}

		if !toDownload {
			fmt.Printf("File %s already downloaded, checksum OK\n", f.Name())
			continue
		}

		if projectIntegrationName != "" {
			_, err := exec.LookPath("jfrog")
			if err != nil {
				fmt.Printf("# File is available on repository %s: %s\n", artifactManagerRepo, artifactManagerPath)
				fmt.Printf("# to download the file use the following command\n")
				fmt.Printf("jfrog rt download %s %q\n", artifactManagerRepo, artifactManagerPath)
				return err
			} else {
				fmt.Printf("Downloading file %s from %s\n", artifactManagerPath, artifactManagerRepo)
			}
			cmd := exec.Command("jfrog", "rt", "download", "--flat", artifactManagerRepo+"/"+strings.TrimPrefix(artifactManagerPath, "/"))
			output, err := cmd.CombinedOutput()
			fmt.Println(string(output))
			if err != nil {
				return err
			}
		} else if cdnHTTPUrl != "" {
			var err error
			f, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(perm))
			if err != nil {
				return cli.NewError("unable to open file %s: %s", fileName, err)
			}
			fmt.Printf("Downloading %s...\n", fileName)
			if err := client.CDNItemDownload(context.Background(), cdnHTTPUrl, cdnAPIRefHash, sdk.CDNTypeItemRunResultV2, md5, f); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return cli.NewError("unable to close file %s: %v", fileName, err)
			}
			fmt.Printf("File %s created, checksum OK\n", f.Name())
		} else {
			fmt.Printf("unable to download artifact %s of type %s. The artifact is not uploaded oo CDN or into artifactory.\n", r.Name(), r.Type)
		}
	}
	return nil
}

var workflowV2RunResultListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List run result",
	Example: "cdsctl experimental workflow results list <project_key> <run_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "proj_key"},
		{Name: "run_identifier"},
	},
}

func workflowV2RunResultListFunc(v cli.Values) (cli.ListResult, error) {
	projKey := v.GetString("proj_key")
	runIdentifier := v.GetString("run_identifier")
	results, err := client.WorkflowV2RunResultList(context.Background(), projKey, runIdentifier)
	if err != nil {
		return nil, err
	}
	type runResult struct {
		ID   string `cli:"id"`
		Name string `cli:"name"`
		Type string `cli:"type"`
	}
	cliResults := make([]runResult, 0, len(results))
	for _, r := range results {
		cliResults = append(cliResults, runResult{
			ID:   r.ID,
			Name: r.Name(),
			Type: string(r.Type),
		})
	}
	return cli.AsListResult(cliResults), nil
}
