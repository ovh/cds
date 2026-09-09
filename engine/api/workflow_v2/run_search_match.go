package workflow_v2

import (
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
)

// MatchesRun reports whether the given run is one the filters select. It is the in-memory twin of
// runQueryFilters and runQueryAnnotationFilters, written in the same order so that the two can be
// read side by side, and held to them by TestMatchesRun_AgreesWithTheSearch.
//
// An event carries the whole run, so answering whether that run belongs to the list a websocket
// client is watching asks nothing of the database. Which is all that is being asked there: whether
// the client may read the project at all was settled when it registered its filter.
func (s SearchRunsFilters) MatchesRun(run sdk.EventWorkflowRunPayload) bool {
	if len(s.Workflows) > 0 && !matches(s.Workflows, strings.Join([]string{run.VCSServer, run.Repository, run.WorkflowName}, "/")) {
		return false
	}
	if len(s.Actors) > 0 && !matchesContext(s.Actors, actorOf(run)) {
		return false
	}
	if len(s.Status) > 0 && !matches(s.Status, string(run.Status)) {
		return false
	}
	if len(s.Refs) > 0 && !matchesContext(s.Refs, run.Contexts.Git.Ref) {
		return false
	}
	if len(s.WorkflowRefs) > 0 && !matches(s.WorkflowRefs, run.WorkflowRef) {
		return false
	}
	if len(s.Repositories) > 0 && !matchesContext(s.Repositories, contextPath(run.Contexts.Git.Server, run.Contexts.Git.Repository)) {
		return false
	}
	if len(s.WorkflowRepositories) > 0 && !matches(s.WorkflowRepositories, strings.Join([]string{run.VCSServer, run.Repository}, "/")) {
		return false
	}
	if len(s.Authors) > 0 && !matchesContext(s.Authors, run.Contexts.Git.Author) {
		return false
	}
	if len(s.Commits) > 0 && !matchesContext(s.Commits, run.Contexts.Git.Sha) {
		return false
	}
	if len(s.Templates) > 0 && !matchesContext(s.Templates, contextPath(run.Contexts.CDS.WorkflowTemplateVCSServer, run.Contexts.CDS.WorkflowTemplateRepository, run.Contexts.CDS.WorkflowTemplate)) {
		return false
	}
	// Every annotation asked for has to be carried by the run, which is what the array containment
	// of the query means. The pairs are compared whole, as they are there, so a value holding a
	// colon is no different from any other.
	for _, annotation := range s.Annotations {
		var found bool
		for k, v := range run.Annotations {
			if k+":"+v == annotation {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// Every word of a free text search has to be found, in any order, anywhere in the text the
	// search reads of a run.
	if words := sdk.SearchQueryWords(s.Query); len(words) > 0 {
		text := strings.ToLower(runSearchableText(run))
		for _, w := range words {
			if !strings.Contains(text, w) {
				return false
			}
		}
	}
	return true
}

// runSearchableText is the text a free text search reads of a run, the twin of
// runQuerySearchableText. The parts are joined the same way and in the same order; a part the run
// does not carry is empty here where the query leaves it out, which changes nothing to the words
// found in the result.
func runSearchableText(run sdk.EventWorkflowRunPayload) string {
	return strings.Join([]string{
		run.ProjectKey,
		run.VCSServer + "/" + run.Repository + "/" + run.WorkflowName,
		run.WorkflowName + "#" + strconv.FormatInt(run.RunNumber, 10),
		run.WorkflowRef,
		string(run.Status),
		run.Contexts.Git.Server + "/" + run.Contexts.Git.Repository,
		run.Contexts.Git.Ref,
		run.Contexts.Git.Sha,
		run.Contexts.Git.Author,
		run.Contexts.Git.CommitMessage,
		run.Contexts.CDS.Version,
		run.Username,
		run.VCSUsername,
		annotationsText(run.Annotations),
	}, " ")
}

func annotationsText(annotations sdk.WorkflowRunAnnotations) string {
	pairs := make([]string, 0, len(annotations))
	for k, v := range annotations {
		pairs = append(pairs, k+":"+v)
	}
	return strings.Join(pairs, " ")
}

// actorOf is the initiator of a run in the form the actor filter is written in. The query reads the
// user name and the VCS name of the initiator as two values and matches either; a run carries only
// one of them, so there is only ever one to compare.
func actorOf(run sdk.EventWorkflowRunPayload) string {
	if run.UserID != "" {
		return run.Username
	}
	return run.VCSUsername
}

// matches mirrors "column = ANY(:values)" over a column the table declares NOT NULL.
func matches(values []string, value string) bool {
	return sdk.StringSlice(values).Contains(value)
}

// matchesContext mirrors the same comparison over a value read from the jsonb of a run, where a
// field that is not set is absent rather than empty. NULL matches no value, so neither does an
// empty one here.
func matchesContext(values []string, value string) bool {
	return value != "" && sdk.StringSlice(values).Contains(value)
}

// contextPath joins the parts of a path read from the jsonb of a run. Concatenating an absent field
// gives NULL in SQL whatever the other parts, so one part missing leaves nothing to match.
func contextPath(parts ...string) string {
	for _, p := range parts {
		if p == "" {
			return ""
		}
	}
	return strings.Join(parts, "/")
}
