package forgejo

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func (f *forgejoClient) CreateInsightReport(ctx context.Context, repo string, sha string, insightKey string, vcsReport sdk.VCSInsight) error {
	// Ignore this call on forgejo, like github
	return nil
}

func (f *forgejoClient) SetStatus(ctx context.Context, buildStatus sdk.VCSBuildStatus) error {
	if buildStatus.Status == "" {
		log.Debug(ctx, "forgejo.SetStatus> Do not process event for empty status")
		return nil
	}

	forgejoStatus := CreateStatusOption{
		Context:   buildStatus.Context,
		TargetURL: buildStatus.URLCDS,
	}

	// forgejo display on the UI, the context contact with the description.
	// so that, we remove the context from the description:
	// description":"WorkflowNotificationsLog:Success","context":"ITV2WFNOTIF-WorkflowNotificationsLog
	// we want only ITV2WFNOTIF-WorkflowNotificationsLog:Success display
	td := strings.Split(buildStatus.Description, ":")
	if len(td) == 2 {
		forgejoStatus.Description = td[1]
	} else {
		forgejoStatus.Description = buildStatus.Description
	}

	switch buildStatus.Status {
	case sdk.StatusChecking, sdk.StatusPending, sdk.StatusBuilding:
		forgejoStatus.State = StatusPending
	case sdk.StatusSuccess:
		forgejoStatus.State = StatusSuccess
	case sdk.StatusFail:
		forgejoStatus.State = StatusFailure
	default:
		forgejoStatus.State = StatusPending
	}

	owner, repoName, err := getRepo(buildStatus.RepositoryFullname)
	if err != nil {
		return err
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/statuses/%s", owner, repoName, url.PathEscape(buildStatus.GitHash))
	log.Debug(ctx, "SetStatus> forgejo post on %v", apiPath)

	var s Status
	if _, err := f.client.post(ctx, apiPath, forgejoStatus, &s); err != nil {
		return sdk.WrapError(err, "unable to post forgejo status")
	}

	log.Debug(ctx, "SetStatus> Status %d %s created at %v", s.ID, s.URL, s.Created)
	return nil
}

func (f *forgejoClient) ListStatuses(ctx context.Context, fullname string, ref string) ([]sdk.VCSCommitStatus, error) {
	owner, repoName, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	var statuses []*Status
	apiPath := fmt.Sprintf("/repos/%s/%s/commits/%s/statuses", owner, repoName, url.PathEscape(ref))
	if _, err := f.client.get(ctx, apiPath, &statuses); err != nil {
		return nil, err
	}

	var vcsStatuses []sdk.VCSCommitStatus
	for _, s := range statuses {
		vcsStatuses = append(vcsStatuses, sdk.VCSCommitStatus{
			CreatedAt:  s.Created,
			Decription: s.Context,
			Ref:        ref,
			State:      processForgejoState(s.State),
		})
	}

	return vcsStatuses, nil
}

func processForgejoState(s CommitStatusState) string {
	switch s {
	case StatusSuccess:
		return sdk.StatusSuccess
	case StatusError, StatusFailure:
		return sdk.StatusFail
	case StatusPending:
		return sdk.StatusBuilding
	default:
		return sdk.StatusDisabled
	}
}
