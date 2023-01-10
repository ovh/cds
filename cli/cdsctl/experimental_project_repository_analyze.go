package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var projectAnalysisCmd = cli.Command{
	Name:    "analysis",
	Aliases: []string{"a"},
	Short:   "Manage repository analysis",
}

func projectRepositoryAnalysis() *cobra.Command {
	return cli.NewCommand(projectAnalysisCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectRepositoryAnalysisListCmd, projectRepositoryAnalysisListFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectRepositoryGetCmd, projectRepositoryGetFunc, nil, withAllCommandModifiers()...),
	})
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

var projectRepositoryGetCmd = cli.Command{
	Name:  "show",
	Short: "List available repositories on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
		{Name: "analysis-id"},
	},
}

func projectRepositoryGetFunc(v cli.Values) error {
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
		Branch      string    `json:"branch"`
		Commit      string    `json:"commit"`
		Status      string    `json:"status"`
		Error       string    `json:"error,omitempty"`
		CommitCheck bool      `json:"commit_check"`
		KeySignID   string    `json:"key_sign_id"`
	}

	resp := AnalysisCli{
		Branch:      analysis.Branch,
		ID:          analysis.ID,
		Error:       strings.Join(analysis.Data.Errors, "\n"),
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
