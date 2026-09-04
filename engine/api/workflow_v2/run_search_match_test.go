package workflow_v2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func runEventPayload() sdk.EventWorkflowRunPayload {
	return sdk.EventWorkflowRunPayload{
		ProjectKey:   "KEY",
		VCSServer:    "github",
		Repository:   "ovh/cds",
		WorkflowName: "my-workflow",
		WorkflowRef:  "refs/heads/master",
		Status:       sdk.V2WorkflowRunStatusSuccess,
		RunNumber:    42,
		UserID:       "user-1",
		Username:     "jdoe",
		Annotations:  sdk.WorkflowRunAnnotations{"release": "1.2.3"},
		Contexts: sdk.EventWorkflowRunPayloadContexts{
			Git: sdk.GitContext{
				Server:        "github",
				Repository:    "ovh/cds",
				Ref:           "refs/heads/main",
				Sha:           "abcdef123456",
				Author:        "asmith",
				CommitMessage: "fix the thing that was broken",
			},
			CDS: sdk.CDSContext{
				Version:                    "1.2.3+cds.1",
				WorkflowTemplateVCSServer:  "github",
				WorkflowTemplateRepository: "ovh/templates",
				WorkflowTemplate:           "tmpl",
			},
		},
	}
}

func asVCSRun(r *sdk.EventWorkflowRunPayload) {
	r.UserID = ""
	r.Username = "github/mybot"
	r.VCSUsername = "mybot"
}

func TestSearchRunsFilters_MatchesRun(t *testing.T) {
	for _, tc := range []struct {
		name    string
		filters SearchRunsFilters
		mutate  func(*sdk.EventWorkflowRunPayload)
		want    bool
	}{
		{name: "a search without filter takes every run", want: true},
		{name: "workflow path", filters: SearchRunsFilters{Workflows: []string{"github/ovh/cds/my-workflow"}}, want: true},
		{name: "another workflow path", filters: SearchRunsFilters{Workflows: []string{"github/ovh/cds/other"}}},
		{name: "any of the values of a filter", filters: SearchRunsFilters{Workflows: []string{"github/ovh/cds/other", "github/ovh/cds/my-workflow"}}, want: true},
		{name: "status", filters: SearchRunsFilters{Status: []string{string(sdk.V2WorkflowRunStatusSuccess)}}, want: true},
		{name: "another status", filters: SearchRunsFilters{Status: []string{string(sdk.V2WorkflowRunStatusFail)}}},
		{name: "git ref", filters: SearchRunsFilters{Refs: []string{"refs/heads/main"}}, want: true},
		{name: "workflow ref", filters: SearchRunsFilters{WorkflowRefs: []string{"refs/heads/master"}}, want: true},
		{name: "run repository", filters: SearchRunsFilters{Repositories: []string{"github/ovh/cds"}}, want: true},
		{name: "workflow repository", filters: SearchRunsFilters{WorkflowRepositories: []string{"github/ovh/cds"}}, want: true},
		{name: "commit author", filters: SearchRunsFilters{Authors: []string{"asmith"}}, want: true},
		{name: "commit", filters: SearchRunsFilters{Commits: []string{"abcdef123456"}}, want: true},
		{name: "template path", filters: SearchRunsFilters{Templates: []string{"github/ovh/templates/tmpl"}}, want: true},
		{name: "annotation", filters: SearchRunsFilters{Annotations: []string{"release:1.2.3"}}, want: true},
		{name: "every annotation asked for is required", filters: SearchRunsFilters{Annotations: []string{"release:1.2.3", "env:prod"}}},
		{name: "the actor of a run started by a user", filters: SearchRunsFilters{Actors: []string{"jdoe"}}, want: true},
		{name: "the actor of a run started from a vcs", filters: SearchRunsFilters{Actors: []string{"mybot"}}, mutate: asVCSRun, want: true},
		{
			name:    "the vcs actor is searched on its own name, not on the flattened one",
			filters: SearchRunsFilters{Actors: []string{"github/mybot"}},
			mutate:  asVCSRun,
		},
		{
			name:    "the vcs name of a run started by a user is not its actor",
			filters: SearchRunsFilters{Actors: []string{"mybot"}},
			mutate:  func(r *sdk.EventWorkflowRunPayload) { r.VCSUsername = "mybot" },
		},
		{
			name:    "filters of different keys are all required",
			filters: SearchRunsFilters{Status: []string{string(sdk.V2WorkflowRunStatusSuccess)}, Authors: []string{"jdoe"}},
		},
		{
			// The database reads those from the jsonb contexts of a run, where a field that is not
			// set is absent, and no value is matched against an absent field.
			name:    "a context field a run does not carry matches nothing",
			filters: SearchRunsFilters{Commits: []string{""}},
			mutate:  func(r *sdk.EventWorkflowRunPayload) { r.Contexts.Git.Sha = "" },
		},
		{
			name:    "a template path missing one of its parts matches nothing",
			filters: SearchRunsFilters{Templates: []string{"github/ovh/templates/"}},
			mutate:  func(r *sdk.EventWorkflowRunPayload) { r.Contexts.CDS.WorkflowTemplate = "" },
		},

		// Free text
		{name: "a word of the workflow name", filters: SearchRunsFilters{Query: "workflow"}, want: true},
		{name: "free text is case insensitive", filters: SearchRunsFilters{Query: "MY-WORKFLOW"}, want: true},
		{name: "the run number", filters: SearchRunsFilters{Query: "my-workflow#42"}, want: true},
		{name: "a word of the commit message", filters: SearchRunsFilters{Query: "broken"}, want: true},
		{name: "the version", filters: SearchRunsFilters{Query: "1.2.3+cds.1"}, want: true},
		{name: "an annotation value", filters: SearchRunsFilters{Query: "1.2.3"}, want: true},
		{name: "the initiator", filters: SearchRunsFilters{Query: "jdoe"}, want: true},
		{name: "every word must be found", filters: SearchRunsFilters{Query: "workflow broken"}, want: true},
		{name: "words can be given in any order", filters: SearchRunsFilters{Query: "broken workflow"}, want: true},
		{name: "one word not found excludes the run", filters: SearchRunsFilters{Query: "workflow nowhere"}},
		{name: "like wildcards are matched literally", filters: SearchRunsFilters{Query: "%"}},
		{name: "free text is required next to a filter", filters: SearchRunsFilters{Status: []string{string(sdk.V2WorkflowRunStatusSuccess)}, Query: "nowhere"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := runEventPayload()
			if tc.mutate != nil {
				tc.mutate(&run)
			}
			require.Equal(t, tc.want, tc.filters.MatchesRun(run))
		})
	}
}
