package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectAnalysisCmd = cli.Command{
	Name:    "analysis",
	Aliases: []string{"a"},
	Short:   "Manage repository analysis",
}

func projectRepositoryAnalysis() *cobra.Command {
	return cli.NewCommand(projectAnalysisCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectRepositoryAnalysisListCmd, projectRepositoryAnalysisListFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectRepositoryAnalysisGetCmd, projectRepositoryAnalysisGetFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectRepositoryAnalysisTriggerCmd, projectRepositoryAnalysisTriggerFunc, nil, withAllCommandModifiers()...),
	})
}

var projectRepositoryAnalysisTriggerCmd = cli.Command{
	Name:    "trigger",
	Aliases: []string{"run", "start"},
	Short:   "Trigger an analysis on the given branch",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
	Flags: []cli.Flag{
		{
			Name: "branch",
		},
		{
			Name: "tag",
		},
		{
			Name: "commit",
		},
	},
}

func projectRepositoryAnalysisTriggerFunc(v cli.Values) (interface{}, error) {
	analysisRequest := sdk.AnalysisRequest{
		ProjectKey: v.GetString(_ProjectKey),
		VcsName:    v.GetString("vcs-name"),
		RepoName:   v.GetString("repository-name"),
	}
	if v.GetString("branch") != "" {
		analysisRequest.Ref = sdk.GitRefBranchPrefix + v.GetString("branch")
	}
	if v.GetString("tag") != "" {
		analysisRequest.Ref = sdk.GitRefTagPrefix + v.GetString("tag")
	}
	if v.GetString("commit") != "" {
		analysisRequest.Commit = v.GetString("commit")
	}
	analysisResponse, err := client.ProjectRepositoryAnalysis(context.Background(), analysisRequest)
	if err != nil {
		return nil, err
	}
	return analysisResponse, nil
}

var projectRepositoryAnalysisListCmd = cli.Command{
	Name:  "list",
	Short: "List all repository analysis",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
}

func projectRepositoryAnalysisListFunc(v cli.Values) (cli.ListResult, error) {
	analyses, err := client.ProjectRepositoryAnalysisList(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-name"), v.GetString("repository-name"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(analyses), nil
}

var projectRepositoryAnalysisGetCmd = cli.Command{
	Name:  "show",
	Short: "Get the given analysis",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
		{Name: "analysis-id"},
	},
}

func projectRepositoryAnalysisGetFunc(v cli.Values) error {
	analysis, err := client.ProjectRepositoryAnalysisGet(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-name"), v.GetString("repository-name"), v.GetString("analysis-id"))
	if err != nil {
		return err
	}
	type AnalysisFile struct {
		File string `json:"file"`
	}
	type AnalysisCli struct {
		ID          string    `json:"id"`
		Created     time.Time `json:"created"`
		Ref         string    `json:"ref"`
		Commit      string    `json:"commit"`
		Status      string    `json:"status"`
		Error       string    `json:"error,omitempty"`
		CommitCheck bool      `json:"commit_check"`
		KeySignID   string    `json:"key_sign_id"`
	}

	resp := AnalysisCli{
		Ref:         analysis.Ref,
		ID:          analysis.ID,
		Error:       analysis.Data.Error,
		Commit:      analysis.Commit,
		Created:     analysis.Created,
		Status:      analysis.Status,
		CommitCheck: analysis.Data.CommitCheck,
		KeySignID:   analysis.Data.SignKeyID,
	}
	analysisYaml, _ := yaml.Marshal(resp)
	fmt.Println(string(analysisYaml))

	if len(analysis.Data.Entities) > 0 {
		fmt.Println("Files found:")
		files := make([]AnalysisFile, 0, len(analysis.Data.Entities))
		for _, f := range analysis.Data.Entities {
			files = append(files, AnalysisFile{
				File: f.Path + f.FileName,
			})
		}
		fileYaml, _ := yaml.Marshal(files)
		fmt.Println(string(fileYaml))
	}
	return nil
}
