package main

import (
	"regexp"

	"github.com/ovh/cds/cli"
)

var workflowHistoryCmd = cli.Command{
	Name:  "history",
	Short: "Display CDS workflow runs history",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	OptionalArgs: []cli.Arg{
		{
			Name: "offset",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 1,
		},
		{
			Name: "limit",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`[0-9]?`, s)
				return match
			},
			Weight: 2,
		},
	},
}

func workflowHistoryRun(v cli.Values) (cli.ListResult, error) {
	var offset int64
	if v.GetString("offset") != "" {
		var errn error
		offset, errn = v.GetInt64("offset")
		if errn != nil {
			return nil, errn
		}
	}

	var limit int64
	if v.GetString("limit") != "" {
		var errl error
		limit, errl = v.GetInt64("limit")
		if errl != nil {
			return nil, errl
		}
	}

	w, err := client.WorkflowRunList(v[_ProjectKey], v[_WorkflowName], offset, limit)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(w), nil
}
