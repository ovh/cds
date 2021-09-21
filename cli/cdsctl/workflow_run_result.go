package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowRunResultCmd = cli.Command{
	Name:    "result",
	Aliases: []string{"results"},
	Short:   "Manage Workflow Run Result",
}

type RunResultCli struct {
	ID   string `cli:"id"`
	Type string `cli:"type"`
	Name string `cli:"name"`
}

func workflowRunResult() *cobra.Command {
	return cli.NewCommand(workflowRunResultCmd, nil, []*cobra.Command{
		cli.NewListCommand(workflowRunResultListCmd, workflowRunResultList, nil, withAllCommandModifiers()...),
		cli.NewCommand(workflowRunResultGetCmd, workflowRunResultGet, nil, withAllCommandModifiers()...),
	})
}

var workflowRunResultGetCmd = cli.Command{
	Name:  "download",
	Short: "download workflow run result",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{
			Name: "run-number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
		},
		{
			Name: "result-id",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
		},
	},
}

func workflowRunResultGet(v cli.Values) error {
	ctx := context.Background()
	runNumber, err := v.GetInt64("run-number")
	if err != nil {
		return err
	}
	resultID := v.GetString("result-id")

	confCDN, err := client.ConfigCDN()
	if err != nil {
		return err
	}

	runResults, err := client.WorkflowRunResultsList(ctx, v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return err
	}
	for _, r := range runResults {
		if r.ID != resultID {
			continue
		}
		var cdnHash string
		var fileName string
		var perm uint32
		var md5 string
		switch r.Type {
		case sdk.WorkflowRunResultTypeArtifact:
			art, err := r.GetArtifact()
			if err != nil {
				return err
			}
			cdnHash = art.CDNRefHash
			fileName = art.Name
			perm = art.Perm
			md5 = art.MD5
		case sdk.WorkflowRunResultTypeCoverage:
			cov, err := r.GetCoverage()
			if err != nil {
				return err
			}
			cdnHash = cov.CDNRefHash
			fileName = cov.Name
			perm = cov.Perm
			md5 = cov.MD5
		default:
			return cli.NewError("cannot get result of type %s", r.Type)
		}

		var f *os.File
		var toDownload bool
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			toDownload = true
		} else {
			// file exists, check sha512
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
		}

		if toDownload {
			var err error
			f, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(perm))
			if err != nil {
				return cli.NewError("unable to open file %s: %s", fileName, err)
			}
			fmt.Printf("Downloading %s...\n", fileName)
			if err := client.CDNItemDownload(context.Background(), confCDN.HTTPURL, cdnHash, sdk.CDNTypeItemRunResult, md5, f); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return cli.NewError("unable to close file %s: %v", fileName, err)
			}
		}
		if toDownload {
			fmt.Printf("File %s created, checksum OK\n", f.Name())
		} else {
			fmt.Printf("File %s already downloaded, checksum OK\n", f.Name())
		}
	}
	return nil
}

var workflowRunResultListCmd = cli.Command{
	Name:  "list",
	Short: "List workflow run result",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Args: []cli.Arg{
		{
			Name: "run-number",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
		},
	},
	Flags: []cli.Flag{
		{
			Name:      "type",
			ShortHand: "t",
			Usage:     "List only result of one type",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
		},
	},
}

func workflowRunResultList(v cli.Values) (cli.ListResult, error) {
	ctx := context.Background()

	runNumber, err := v.GetInt64("run-number")
	if err != nil {
		return nil, err
	}

	results, err := client.WorkflowRunResultsList(ctx, v.GetString(_ProjectKey), v.GetString(_WorkflowName), runNumber)
	if err != nil {
		return nil, err
	}

	if v.GetString("type") == "" {
		cliResults, err := toCLIRunResult(results)
		return cli.AsListResult(cliResults), err
	}

	resultsFiltered := make([]sdk.WorkflowRunResult, 0)
	for _, r := range results {
		if string(r.Type) != v.GetString("type") {
			continue
		}
		resultsFiltered = append(resultsFiltered, r)
	}
	cliResults, err := toCLIRunResult(resultsFiltered)
	return cli.AsListResult(cliResults), err
}

func toCLIRunResult(results []sdk.WorkflowRunResult) ([]RunResultCli, error) {
	cliresults := make([]RunResultCli, 0, len(results))
	for _, r := range results {
		artiType := string(r.Type)
		var name string
		switch r.Type {
		case sdk.WorkflowRunResultTypeCoverage:
			covResult, err := r.GetCoverage()
			if err != nil {
				return nil, err
			}
			name = covResult.Name
		case sdk.WorkflowRunResultTypeArtifact:
			artiResult, err := r.GetArtifact()
			if err != nil {
				return nil, err
			}
			name = artiResult.Name
		case sdk.WorkflowRunResultTypeArtifactManager:
			artiResult, err := r.GetArtifactManager()
			if err != nil {
				return nil, err
			}
			name = artiResult.Name
			artiType = artiResult.RepoType
		}

		cliresults = append(cliresults, RunResultCli{
			ID:   r.ID,
			Type: artiType,
			Name: name,
		})
	}
	return cliresults, nil
}
