package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
)

// MatchesRun answers, without reading anything, the question a websocket client asks of every run
// event: is this run one of those it asked to see. It stands for a search, so it has to agree with
// it, and this is what holds the two together — every filter is run through both.
func TestMatchesRun_AgreesWithTheSearch(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	otherRepo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	newRun := func(workflowName string, mutate func(*sdk.V2WorkflowRun)) sdk.V2WorkflowRun {
		wr := sdk.V2WorkflowRun{
			ProjectKey:   proj.Key,
			VCSServerID:  vcsServer.ID,
			VCSServer:    vcsServer.Name,
			RepositoryID: repo.ID,
			Repository:   repo.Name,
			WorkflowName: workflowName,
			WorkflowSha:  "123",
			WorkflowRef:  "refs/heads/master",
			RunNumber:    1,
			Started:      time.Now(),
			LastModified: time.Now(),
			Status:       sdk.V2WorkflowRunStatusSuccess,
			Initiator:    &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
			RunEvent:     sdk.V2WorkflowRunEvent{},
			Contexts: sdk.WorkflowRunContext{
				Git: sdk.GitContext{
					Server:        "github",
					Repository:    "ovh/cds",
					Ref:           "refs/heads/main",
					Sha:           "abcdef123456",
					Author:        "jdoe",
					CommitMessage: "a commit that says something",
				},
				CDS: sdk.CDSContext{Version: "1.2.3+cds.1"},
			},
			WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{}},
		}
		if mutate != nil {
			mutate(&wr)
		}
		require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))
		return wr
	}

	runs := []sdk.V2WorkflowRun{
		newRun("wf-plain", nil),
		newRun("wf-failed", func(r *sdk.V2WorkflowRun) { r.Status = sdk.V2WorkflowRunStatusFail }),
		newRun("wf-dev", func(r *sdk.V2WorkflowRun) {
			r.Contexts.Git.Ref = "refs/heads/dev"
			r.Contexts.Git.Sha = "999999999999"
			r.Contexts.Git.Author = "asmith"
			r.Contexts.Git.CommitMessage = "another message entirely"
		}),
		newRun("wf-other-repo", func(r *sdk.V2WorkflowRun) {
			r.RepositoryID = otherRepo.ID
			r.Repository = otherRepo.Name
			r.Contexts.Git.Repository = "ovh/other"
		}),
		newRun("wf-workflow-ref", func(r *sdk.V2WorkflowRun) { r.WorkflowRef = "refs/heads/dev" }),
		newRun("wf-templated", func(r *sdk.V2WorkflowRun) {
			r.Contexts.CDS.WorkflowTemplateVCSServer = "github"
			r.Contexts.CDS.WorkflowTemplateRepository = "ovh/templates"
			r.Contexts.CDS.WorkflowTemplate = "my-template"
		}),
		newRun("wf-annotated", func(r *sdk.V2WorkflowRun) {
			r.Annotations = sdk.WorkflowRunAnnotations{"release": "1.2.3"}
		}),
		newRun("wf-from-vcs", func(r *sdk.V2WorkflowRun) {
			r.Initiator = &sdk.V2Initiator{VCS: "github", VCSUsername: "mybot"}
		}),
	}

	for _, tc := range []struct {
		name    string
		filters workflow_v2.SearchRunsFilters
	}{
		{"no filter", workflow_v2.SearchRunsFilters{}},
		{"workflow", workflow_v2.SearchRunsFilters{Workflows: []string{vcsServer.Name + "/" + repo.Name + "/wf-plain"}}},
		{"several workflows", workflow_v2.SearchRunsFilters{Workflows: []string{
			vcsServer.Name + "/" + repo.Name + "/wf-plain",
			vcsServer.Name + "/" + repo.Name + "/wf-failed",
		}}},
		{"status", workflow_v2.SearchRunsFilters{Status: []string{string(sdk.V2WorkflowRunStatusFail)}}},
		{"git ref", workflow_v2.SearchRunsFilters{Refs: []string{"refs/heads/dev"}}},
		{"workflow ref", workflow_v2.SearchRunsFilters{WorkflowRefs: []string{"refs/heads/dev"}}},
		{"run repository", workflow_v2.SearchRunsFilters{Repositories: []string{"github/ovh/cds"}}},
		{"workflow repository", workflow_v2.SearchRunsFilters{WorkflowRepositories: []string{vcsServer.Name + "/" + otherRepo.Name}}},
		{"commit author", workflow_v2.SearchRunsFilters{Authors: []string{"asmith"}}},
		{"commit", workflow_v2.SearchRunsFilters{Commits: []string{"abcdef123456"}}},
		{"unmatched commit", workflow_v2.SearchRunsFilters{Commits: []string{"000000000000"}}},
		{"template", workflow_v2.SearchRunsFilters{Templates: []string{"github/ovh/templates/my-template"}}},
		{"annotation", workflow_v2.SearchRunsFilters{Annotations: []string{"release:1.2.3"}}},
		{"unmatched annotation", workflow_v2.SearchRunsFilters{Annotations: []string{"release:9.9.9"}}},
		{"actor", workflow_v2.SearchRunsFilters{Actors: []string{admin.Username}}},
		{"actor of a run started from a vcs", workflow_v2.SearchRunsFilters{Actors: []string{"mybot"}}},
		{"two filters", workflow_v2.SearchRunsFilters{
			Status: []string{string(sdk.V2WorkflowRunStatusSuccess)},
			Refs:   []string{"refs/heads/main"},
		}},

		// Free text, which the database matches over a concatenation of the run it builds itself.
		{"free text on the workflow name", workflow_v2.SearchRunsFilters{Query: "wf-dev"}},
		{"free text on a commit message", workflow_v2.SearchRunsFilters{Query: "entirely"}},
		{"free text on the version", workflow_v2.SearchRunsFilters{Query: "1.2.3+cds.1"}},
		{"free text on an annotation", workflow_v2.SearchRunsFilters{Query: "release:1.2.3"}},
		{"free text on the run number", workflow_v2.SearchRunsFilters{Query: "wf-plain#1"}},
		{"free text of several words", workflow_v2.SearchRunsFilters{Query: "wf- message"}},
		{"free text matching nothing", workflow_v2.SearchRunsFilters{Query: "nowhere"}},
		{"free text next to a filter", workflow_v2.SearchRunsFilters{
			Status: []string{string(sdk.V2WorkflowRunStatusSuccess)},
			Query:  "wf-",
		}},
		{"free text wildcards are matched literally", workflow_v2.SearchRunsFilters{Query: "%"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Every run of the project fits in the page: what is compared is the filters, not the
			// paging, which the match leaves to the list.
			found, err := workflow_v2.SearchRuns(context.Background(), db, proj.Key, tc.filters, 0, 100, "")
			require.NoError(t, err)

			selected := make(map[string]bool, len(found))
			for i := range found {
				selected[found[i].ID] = true
			}

			for i := range runs {
				payload, err := sdk.NewEventWorkflowRunPayload(runs[i], nil, nil)
				require.NoError(t, err)
				require.Equal(t, selected[runs[i].ID], tc.filters.MatchesRun(*payload),
					"run %s", runs[i].WorkflowName)
			}
		})
	}
}
