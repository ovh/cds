package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var projectAnalyzeCmd = cli.Command{
	Name:    "analyze",
	Aliases: []string{"a"},
	Short:   "Manage repository analyze",
}

func projectRepositoryAnalyze() *cobra.Command {
	return cli.NewCommand(projectAnalyzeCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectRepositoryAnalyzeListCmd, projectRepositoryAnalyzeListFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectRepositoryGetCmd, projectRepositoryGetFunc, nil, withAllCommandModifiers()...),
	})
}

var projectRepositoryAnalyzeListCmd = cli.Command{
	Name:  "list",
	Short: "List all repository analyze",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "vcs-name"},
		{Name: "repository-name"},
	},
}

func projectRepositoryAnalyzeListFunc(v cli.Values) (cli.ListResult, error) {
	analyzes, err := client.ProjectRepositoryAnalyzeList(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-name"), v.GetString("repository-name"))
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(analyzes), nil
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
		{Name: "analyze-id"},
	},
}

func projectRepositoryGetFunc(v cli.Values) error {
	analyze, err := client.ProjectRepositoryAnalyzeGet(context.Background(), v.GetString(_ProjectKey), v.GetString("vcs-name"), v.GetString("repository-name"), v.GetString("analyze-id"))
	if err != nil {
		return err
	}
	type AnalyzeFile struct {
		File string `json:"file"`
	}
	type AnalyzeCli struct {
		ID          string    `json:"id"`
		Created     time.Time `json:"created"`
		Branch      string    `json:"branch"`
		Commit      string    `json:"commit"`
		Status      string    `json:"status"`
		Error       string    `json:"error,omitempty"`
		CommitCheck bool      `json:"commit_check"`
		KeySignID   string    `json:"key_sign_id"`
	}

	resp := AnalyzeCli{
		Branch:      analyze.Branch,
		ID:          analyze.ID,
		Error:       analyze.Data.Error,
		Commit:      analyze.Commit,
		Created:     analyze.Created,
		Status:      analyze.Status,
		CommitCheck: analyze.Data.CommitCheck,
		KeySignID:   analyze.Data.SignKeyID,
	}
	analyzeYaml, _ := yaml.Marshal(resp)
	fmt.Println(string(analyzeYaml))

	if len(analyze.Data.Entities) > 0 {
		fmt.Println("Files found:")
		files := make([]AnalyzeFile, 0, len(analyze.Data.Entities))
		for _, f := range analyze.Data.Entities {
			files = append(files, AnalyzeFile{
				File: f.Path + f.FileName,
			})
		}
		fileYaml, _ := yaml.Marshal(files)
		fmt.Println(string(fileYaml))
	}
	return nil
}
