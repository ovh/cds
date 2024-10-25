package gitlab

import (
	"context"
	"strings"

	"github.com/rockbears/log"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

type statusData struct {
	status       string
	branchName   string
	url          string
	desc         string
	repoFullName string
	hash         string
}

func getGitlabStateFromStatus(s string) gitlab.BuildStateValue {
	switch s {
	case sdk.StatusWaiting:
		return gitlab.Pending
	case sdk.StatusChecking:
		return gitlab.Pending
	case sdk.StatusSuccess:
		return gitlab.Success
	case sdk.StatusFail:
		return gitlab.Failed
	case sdk.StatusDisabled:
		return gitlab.Canceled
	case sdk.StatusNeverBuilt:
		return gitlab.Canceled
	case sdk.StatusUnknown:
		return gitlab.Failed
	case sdk.StatusSkipped:
		return gitlab.Canceled
	}

	return gitlab.Failed
}

func (c *gitlabClient) CreateInsightReport(ctx context.Context, repo string, sha string, insightKey string, vcsReport sdk.VCSInsight) error {
	// not implemented
	return nil
}

// SetStatus set build status on Gitlab
func (c *gitlabClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	if c.disableStatus {
		log.Warn(ctx, "disableStatus.SetStatus> ⚠ Gitlab statuses are disabled")
		return nil
	}

	cds := "CDS"
	opt := &gitlab.SetCommitStatusOptions{
		Name:        &buildStatus.Title,
		Context:     &cds,
		State:       getGitlabStateFromStatus(buildStatus.Status),
		Ref:         &buildStatus.GitHash,
		TargetURL:   &buildStatus.URLCDS,
		Description: &buildStatus.Description,
	}

	val, _, err := c.client.Commits.GetCommitStatuses(buildStatus.RepositoryFullname, buildStatus.GitHash, nil)
	if err != nil {
		return sdk.WrapError(err, "unable to get commit statuses - repo:%s hash:%s", buildStatus.RepositoryFullname, buildStatus.GitHash)
	}

	log.Debug(ctx, "gitlabClient.SetStatus> existing nb statuses: %d", len(val))

	found := false
	for _, s := range val {
		sameRequest := s.TargetURL == *opt.TargetURL && // Comparing TargetURL as there is the workflow run number inside
			s.Status == string(opt.State) && // Comparing Status to avoid duplicate entries
			s.Ref == *opt.Ref && // Comparing branches name
			s.SHA == buildStatus.GitHash && // Comparing commit SHA to match the right commit
			s.Name == *opt.Name && // Comparing app name (CDS)
			s.Description == *opt.Description // Comparing Description as there are the pipelines names inside

		if sameRequest {
			log.Debug(ctx, "gitlabClient.SetStatus> Duplicate commit status, ignoring request - repo:%s hash:%s", buildStatus.RepositoryFullname, buildStatus.GitHash)
			found = true
			break
		}
	}
	if !found {
		log.Debug(ctx, "gitlabClient.SetStatus> gitlab set status on %v hash:%v status:%v", buildStatus.RepositoryFullname, buildStatus.GitHash, buildStatus.Status)
		if _, _, err := c.client.Commits.SetCommitStatus(buildStatus.RepositoryFullname, buildStatus.GitHash, opt); err != nil {
			return sdk.WrapError(err, "cannot process event repo:%s hash:%s", buildStatus.RepositoryFullname, buildStatus.GitHash)
		}
	}
	return nil
}

func (c *gitlabClient) ListStatuses(ctx context.Context, repo string, ref string) ([]sdk.VCSCommitStatus, error) {
	ss, _, err := c.client.Commits.GetCommitStatuses(repo, ref, nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get commit statuses hash:%s", ref)
	}

	vcsStatuses := []sdk.VCSCommitStatus{}
	for _, s := range ss {
		if !strings.HasPrefix(s.Description, "CDS/") {
			continue
		}
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  *s.CreatedAt,
			Decription: s.Description,
			Ref:        ref,
			State:      processGitlabState(*s),
		})
	}

	return vcsStatuses, nil
}

func processGitlabState(s gitlab.CommitStatus) string {
	switch s.Status {
	case string(gitlab.Success):
		return sdk.StatusSuccess
	case string(gitlab.Failed):
		return sdk.StatusFail
	case string(gitlab.Canceled):
		return sdk.StatusSkipped
	default:
		return sdk.StatusDisabled
	}
}
